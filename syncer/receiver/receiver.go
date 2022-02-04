package receiver

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/storage"
	"powerdns.com/platform/lightningstream/utils"
)

func New(st storage.Interface, c config.Config, dbname string, l logrus.FieldLogger, inst string) *Receiver {
	r := &Receiver{
		st:                     st,
		c:                      c,
		prefix:                 dbname + "__", // snapshot filename prefix
		l:                      l.WithField("component", "receiver"),
		ownInstance:            inst,
		lastNotifiedByInstance: make(map[string]snapshot.NameInfo),
		ignoredFilenames:       make(map[string]bool),
		snapshotsByInstance:    make(map[string]*snapshot.Snapshot),
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
	st          storage.Interface
	c           config.Config
	prefix      string
	l           logrus.FieldLogger
	ownInstance string

	// Only accessed by Run goroutine
	lastNotifiedByInstance map[string]snapshot.NameInfo
	ignoredFilenames       map[string]bool

	// The following fields are protected by this mutex, because they
	// are accessed by multiple goroutines.
	mu                    sync.Mutex
	snapshotsByInstance   map[string]*snapshot.Snapshot
	lastSeenByInstance    map[string]snapshot.NameInfo
	downloadersByInstance map[string]*Downloader
}

// Next returns the next remote snapshot to process if there is one
func (r *Receiver) Next() (instance string, snap *snapshot.Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for instance, snap = range r.snapshotsByInstance {
		break // first is assigned to return values now
	}
	if instance != "" {
		// Consider handled
		delete(r.snapshotsByInstance, instance)
	}
	return instance, snap
}

func (r *Receiver) Run(ctx context.Context) error {
	for {
		if err := r.RunOnce(ctx); err != nil {
			r.l.WithError(err).Error("Fetch error")
		}

		if err := utils.SleepContext(ctx, time.Second); err != nil { // TODO: config
			return err
		}
	}
}

func (r *Receiver) RunOnce(ctx context.Context) error {
	//r.l.Debug("RunOnce")
	st := r.st
	prefix := r.prefix

	// The result is ordered lexicographically
	ls, err := st.List(ctx, prefix)
	if err != nil {
		return err
	}
	names := ls.Names()
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
		lastSeenByInstance[ni.InstanceID] = ni
	}

	now := time.Now()

	r.mu.Lock()
	r.lastSeenByInstance = lastSeenByInstance
	r.mu.Unlock()

	for inst, ni := range lastSeenByInstance {
		//r.l.WithField("filename", ni.FullName).Debug("Considering")
		lastNotified := r.lastNotifiedByInstance[inst]
		if ni.FullName == lastNotified.FullName {
			//r.l.WithField("filename", ni.FullName).Debug("Already handled")
			continue // no change
		}

		if inst == r.ownInstance && lastNotified.FullName != "" {
			// Own instance and already notified once.
			// We only want to load our own snapshot once on startup, and ignore
			// any further snapshots.
			//r.l.WithField("filename", ni.FullName).Debug("Skipping own instance")
			continue
		}

		r.l.WithFields(logrus.Fields{
			"instance":   inst,
			"timestamp":  ni.TimestampString,
			"generation": ni.GenerationID,
			"age":        now.Sub(ni.Timestamp).Round(10 * time.Millisecond),
		}).Debug("New snapshot detected")

		d := r.getDownloader(ctx, inst)
		d.NotifyNewSnapshot()
		r.lastNotifiedByInstance[inst] = ni
	}

	return nil
}

func (r *Receiver) getDownloader(ctx context.Context, instance string) *Downloader {
	r.mu.Lock()
	d, exists := r.downloadersByInstance[instance]
	r.mu.Unlock()

	if exists {
		return d
	}

	d = &Downloader{
		r: r,
		l: r.l.WithFields(logrus.Fields{
			"component": "downloader",
			"instance":  instance,
		}),
		instance:          instance,
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

	r.mu.Lock()
	r.downloadersByInstance[instance] = d
	r.mu.Unlock()

	return d
}
