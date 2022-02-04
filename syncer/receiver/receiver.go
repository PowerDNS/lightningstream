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

// Receiver monitors a storage backend, downloads updates and notifies the
// syncer.Syncer of new snapshots. When the Syncer gets around to handle a
// snapshot, it will offer the latest version available instead of the version
// that was available at the time of notification.
// It spawns per-instance Downloader goroutines to take care of the actual
// downloading.
type Receiver struct {
	Storage     storage.Interface
	Config      config.Config
	Prefix      string
	Logger      logrus.Logger
	OwnInstance string

	// Only accessed by Run goroutine
	lastNotifiedByInstance map[string]snapshot.NameInfo

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
			r.Logger.WithError(err).Error("Fetch error")
		}

		if err := utils.SleepContext(ctx, time.Second); err != nil { // TODO: config
			return err
		}
	}
}

func (r *Receiver) RunOnce(ctx context.Context) error {
	st := r.Storage
	prefix := r.Prefix

	// The result is ordered lexicographically
	ls, err := st.List(ctx, prefix)
	if err != nil {
		return err
	}
	names := ls.Names()
	lastSeenByInstance := make(map[string]snapshot.NameInfo)
	for _, name := range names {
		ni, err := snapshot.ParseName(name)
		if err != nil {
			r.Logger.WithError(err).WithField("filename", name).
				Debug("Skipping invalid filename")
			continue
		}
		if ni.InstanceID == r.OwnInstance {
			// FIXME: we need to load our own snapshot once on startup
			continue // ignore own snapshots
		}
		lastSeenByInstance[ni.InstanceID] = ni
	}

	now := time.Now()

	r.mu.Lock()
	r.lastSeenByInstance = lastSeenByInstance
	r.mu.Unlock()

	for inst, ni := range lastSeenByInstance {
		lastNotified := r.lastNotifiedByInstance[inst]
		if ni.FullName == lastNotified.FullName {
			continue // no change
		}

		if inst == r.OwnInstance && lastNotified.FullName != "" {
			// Own instance and already notified once.
			// We only want to load our own snapshot once on startup, and ignore
			// any further snapshots.
			continue
		}

		r.Logger.WithFields(logrus.Fields{
			"instance": inst,
			"filename": ni.FullName,
			"age":      now.Sub(ni.Timestamp).Round(10 * time.Millisecond),
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
		l: r.Logger.WithFields(logrus.Fields{
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
