package receiver

import (
	"context"
	"sync"
	"time"

	"github.com/PowerDNS/simpleblob"
	"github.com/sirupsen/logrus"

	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/utils"
)

func New(st simpleblob.Interface, c config.Config, dbname string, l logrus.FieldLogger, inst string) *Receiver {
	r := &Receiver{
		st:                     st,
		c:                      c,
		lmdbname:               dbname,
		prefix:                 dbname + "__", // snapshot filename prefix
		l:                      l.WithField("component", "receiver"),
		ownInstance:            inst,
		lastNotifiedByInstance: make(map[string]snapshot.NameInfo),
		ignoredFilenames:       make(map[string]bool),
		snapshotsByInstance:    make(map[string]snapshot.Update),
		lastSeenByInstance:     make(map[string]snapshot.NameInfo),
		downloadersByInstance:  make(map[string]*Downloader),
	}
	return r
}

// Receiver monitors a storage backend, downloads updates and notifies the
// syncer.Syncer of new snapshots. When the Syncer gets around to handle a
// snapshot, it will offer the latest version available instead of the version
// that was available at the time of notification.
// It spawns per-instance Downloader goroutines to take care of the actual
// downloading.
type Receiver struct {
	st          simpleblob.Interface
	c           config.Config
	lmdbname    string
	prefix      string
	l           logrus.FieldLogger
	ownInstance string

	// Only accessed by Run goroutine
	lastNotifiedByInstance map[string]snapshot.NameInfo
	ignoredFilenames       map[string]bool

	// The following fields are protected by this mutex, because they
	// are accessed by multiple goroutines.
	mu                    sync.Mutex
	snapshotsByInstance   map[string]snapshot.Update
	lastSeenByInstance    map[string]snapshot.NameInfo
	downloadersByInstance map[string]*Downloader
	hasSnapshots          bool
}

// Next returns the next remote snapshot.Update to process if there is one
// It is to be called by the Syncer.
func (r *Receiver) Next() (instance string, update snapshot.Update) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for instance, update = range r.snapshotsByInstance {
		break // first is assigned to return values now
	}
	if instance != "" {
		// Consider handled
		delete(r.snapshotsByInstance, instance)
	}
	return instance, update
}

// HasSnapshots indicates if there are any snapshots in the storage backend
// for our prefix.
func (r *Receiver) HasSnapshots() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hasSnapshots
}

// SeenInstances returns all seen instance names.
// This includes our own instance, even if RunOnce was called with includingOwn
// set to false.
func (r *Receiver) SeenInstances() (names []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name := range r.lastSeenByInstance {
		names = append(names, name)
	}
	return names
}

func (r *Receiver) Run(ctx context.Context) error {
	for {
		if err := r.RunOnce(ctx, false); err != nil {
			r.l.WithError(err).Error("Fetch error")
		}

		if err := utils.SleepContext(ctx, r.c.StoragePollInterval); err != nil {
			return err
		}
	}
}

func (r *Receiver) RunOnce(ctx context.Context, includingOwn bool) error {
	//r.l.Debug("RunOnce")
	st := r.st
	prefix := r.prefix

	// The result is ordered lexicographically
	ls, err := st.List(ctx, prefix)
	metricSnapshotsListCalls.Inc()
	if err != nil {
		metricSnapshotsListFailed.WithLabelValues(r.lmdbname).Inc()
		return err
	}
	names := ls.Names()

	// Create a new map of the latest snapshots by instance to replace the old map.
	// Note that this always includes our own instance, even if includingOwn is false,
	// which is important during startup in the sync loop.
	lastSeenByInstance := make(map[string]snapshot.NameInfo)
	for _, name := range names {
		if r.ignoredFilenames[name] {
			//r.l.WithField("filename", name).Debug("Ignored")
			continue
		}
		ni, err := snapshot.ParseName(name)
		if err != nil {
			r.l.WithError(err).WithField("filename", name).
				Debug("Skipping invalid filename")
			r.ignoredFilenames[name] = true
			continue
		}
		// Since the names are sorted alphabetically, this newer ones will
		// always overwrite older ones.
		lastSeenByInstance[ni.InstanceID] = ni
	}

	now := time.Now()

	// It is safe to continue using the map after this, because the map is not
	// mutated from this point on.
	// This map is read by the Downloader.
	r.mu.Lock()
	r.lastSeenByInstance = lastSeenByInstance
	r.hasSnapshots = len(lastSeenByInstance) > 0
	r.mu.Unlock()

	for inst, ni := range lastSeenByInstance {
		//r.l.WithField("filename", ni.FullName).Debug("Considering")
		lastNotified := r.lastNotifiedByInstance[inst]
		if ni.FullName == lastNotified.FullName {
			//r.l.WithField("filename", ni.FullName).Debug("Already handled")
			continue // no change
		}

		if !includingOwn && inst == r.ownInstance {
			// Own instance. We only want these during startup.
			continue
		}

		age := now.Sub(ni.Timestamp)
		r.l.WithFields(logrus.Fields{
			"snapshot_instance": inst,
			"timestamp":         ni.TimestampString,
			"generation":        ni.GenerationID,
			"age":               age.Round(10 * time.Millisecond),
		}).Debug("New snapshot detected")

		metricSnapshotsLastReceivedTimestamp.WithLabelValues(r.lmdbname, inst).
			Set(float64(ni.Timestamp.UnixNano()) / 1e9)
		metricSnapshotsLastReceivedAge.WithLabelValues(r.lmdbname, inst).
			Observe(float64(age) / float64(time.Second))

		d := r.getDownloader(ctx, inst)
		d.NotifyNewSnapshot()
		r.lastNotifiedByInstance[inst] = ni
	}

	return nil
}

func (r *Receiver) getDownloader(ctx context.Context, instance string) *Downloader {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, exists := r.downloadersByInstance[instance]

	if exists {
		return d
	}

	d = &Downloader{
		r: r,
		l: r.l.WithFields(logrus.Fields{
			"component":         "downloader",
			"instance":          r.ownInstance,
			"snapshot_instance": instance,
		}),
		c:                 r.c,
		instance:          instance,
		lmdbname:          r.lmdbname,
		last:              snapshot.NameInfo{},
		newSnapshotSignal: make(chan struct{}, 1),
	}

	go func() {
		err := d.Run(ctx)
		if err != nil && err != context.Canceled {
			d.l.WithError(err).Warn("Run returned with an error")
		}
		d.l.Debug("Run exited")

		// Technically there is a race condition where the caller of getDownloader
		// would receive a Downloader that is in the process of exiting, but that
		// should never happen, because Downloaders are not expected to exit, unless
		// cancelled.
		r.mu.Lock()
		delete(r.downloadersByInstance, instance)
		r.mu.Unlock()
	}()

	r.downloadersByInstance[instance] = d

	return d
}
