package receiver

import (
	"context"
	"time"

	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/PowerDNS/lightningstream/utils"
	"github.com/c2h5oh/datasize"
	"github.com/sirupsen/logrus"
)

type Downloader struct {
	r        *Receiver
	l        logrus.FieldLogger
	instance string
	lmdbname string
	last     snapshot.NameInfo
	c        config.Config

	// for signaling new work
	newSnapshotSignal chan struct{}
}

// NotifyNewSnapshot notifies the downloader that a new snapshot is available.
// This never blocks. The chan has capacity 1. If a signal is already in there,
// we do not need to add another one.
func (d *Downloader) NotifyNewSnapshot() {
	select {
	case d.newSnapshotSignal <- struct{}{}:
	default:
	}
}

// Run keeps downloading and unpacking new snapshots, and offering them to
// the update loop.
// It checks the Receiver for the latest version that it has seen, and downloads
// that snapshot if it has not been loaded yet.
// When a download or load fails, it keeps retrying the latest snapshots with
// a delay in between. Eventually either the load succeeds, or a new snapshot
// becomes available that can be loaded.
func (d *Downloader) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-d.newSnapshotSignal:
			// continue
		}

		// Keep retrying on error.
		// If a newer snapshot shows up, switch to that one.
		// If a snapshot disappears, this will be reflected in the last seen.
		for {
			// Get last one seen by Receiver
			d.r.mu.Lock()
			ni, exists := d.r.lastSeenByInstance[d.instance]
			d.r.mu.Unlock()

			if !exists {
				d.l.Warn("this instance no longer has any snapshots")
				// We could be waiting forever in this case, but that is fine
				break // wait
			}

			if ni.FullName == d.last.FullName {
				break // already processed the most recent one
			}

			// Do one load attempt
			if err := d.LoadOnce(ctx, ni); err != nil {
				d.l.WithError(err).WithField("filename", ni.FullName).Warn("Load error")
				if err := utils.SleepContext(ctx, d.c.StorageRetryInterval); err != nil {
					return err // cancelled
				}
				continue // retry
			}

			// Mark this as the last processed one
			d.last = ni
			break // success
		}
	}
}

func (d *Downloader) LoadOnce(ctx context.Context, ni snapshot.NameInfo) error {
	// Limit number of downloaded compressed snapshots in memory
	downloadToken := d.r.downloadSnapshotLimit.Acquire()
	defer downloadToken.Release()

	// Fetch the blob from the storage
	t0 := time.Now()
	metricSnapshotsLoadCalls.Inc()
	data, err := d.r.st.Load(ctx, ni.FullName)
	if err != nil {
		metricSnapshotsLoadFailed.WithLabelValues(d.lmdbname, d.instance).Inc()

		// Signal failure to health tracker
		d.r.storageLoadHealth.AddFailure(err)

		return err
	}

	// Store the size of the snapshot in bytes
	metricSnapshotsStorageBytes.WithLabelValues(d.r.lmdbname).Set(float64(len(data)))

	metricSnapshotsTimestampString.WithLabelValues(d.r.lmdbname, d.instance, ni.TimestampString).Set(1)

	metricSnapshotsReceiverGenerationID.WithLabelValues(d.r.lmdbname, d.instance, ni.GenerationID).Set(1)
	d.l.Debugf("GenerationID of downloaded snapshot: %s", ni.GenerationID)

	// Signal success to health tracker
	d.r.storageLoadHealth.AddSuccess()

	blobSize := datasize.ByteSize(len(data))
	metricSnapshotsLoadBytes.Add(float64(blobSize))

	// Limit number of decompressed snapshots in memory
	// CAUTION: we cannot defer the Release, check all error paths!
	token := d.r.decompressedSnapshotLimit.Acquire()

	t1 := time.Now()

	msg, err := snapshot.LoadData(data)
	if err != nil {
		d.l.Debug("Returning DecompressedSnapshotToken")
		token.Release()
		// This snapshot is considered corrupt, we will ignore it from now on
		d.r.MarkCorrupt(ni.FullName, err)
		d.last = ni
		return err
	}

	// Release the download token once we have released the downloaded snapshot
	data = nil // allow it to be freed
	_ = data   // silence linter
	downloadToken.Release()

	// Make snapshot available to the syncer, replacing any previous one
	// that has not been loaded yet.
	d.r.mu.Lock()
	// If we are about to replace an Update in the snapshotsByInstance map,
	// we will need to Close it so as to release its concurrency limit token.
	overwrittenSnap, hasOverwrittenSnap := d.r.snapshotsByInstance[d.instance]
	// FIXME: use *snapshot.Update pointer in APIs with new tokens
	d.r.snapshotsByInstance[d.instance] = snapshot.Update{
		Snapshot: msg,
		NameInfo: ni,
		BlobSize: blobSize,
		OnClose: func(u *snapshot.Update) {
			if u.Snapshot == nil {
				return // already called?
			}
			d.l.Debug("Returning DecompressedSnapshotToken")
			// Clear it before returning the token
			u.Snapshot = nil
			utils.GC()
			// Return token
			token.Release()
		},
	}
	d.r.mu.Unlock()

	if hasOverwrittenSnap {
		d.l.WithField("overwritten_snapshot", overwrittenSnap.NameInfo.FullName).
			Debug("Closing overwritten snapshot")
		overwrittenSnap.Close()
	}

	t2 := time.Now()
	d.l.WithFields(logrus.Fields{
		"timestamp":         ni.TimestampString,
		"generation":        ni.GenerationID,
		"shorthash":         ni.ShortHash(),
		"time_load_storage": utils.TimeDiff(t1, t0),
		"time_load_total":   utils.TimeDiff(t2, t0),
	}).Info("Snapshot downloaded")

	// Emit seconds since last snapshot was received as a metric.
	// Datadog doesn't work well with UNIX timestamps, so we use seconds
	snapshotAge := time.Since(ni.Timestamp).Seconds()
	metricSnapshotsLastDownloadedSeconds.WithLabelValues(d.r.lmdbname, d.instance).Set(snapshotAge)

	// Log short hash of the snapshot
	// metricSnapshotsShortHash.WithLabelValues(d.r.lmdbname, d.instance, ni.ShortHash()).Set(1)

	// Log time taken to download the snapshot from storage
	metricSnapshotsTimeToDownloadFromStorage.WithLabelValues(d.r.lmdbname, d.instance).
		Set(utils.TimeDiff(t1, t0).Seconds())

	metricSnapshotsTimeToDownloadTotal.WithLabelValues(d.r.lmdbname, d.instance).
		Set(utils.TimeDiff(t2, t0).Seconds())

	return nil
}
