package receiver

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/utils"
)

type Downloader struct {
	r        *Receiver
	l        logrus.FieldLogger
	instance string
	last     snapshot.NameInfo

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
				if err := utils.SleepContext(ctx, time.Second); err != nil {
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
	t0 := time.Now()

	// Fetch the blob from the storage
	data, err := d.r.Storage.Load(ctx, ni.FullName)
	if err != nil {
		return err
	}
	t1 := time.Now()

	// TODO: Distinguish between storage load errors and unpack errors

	// Uncompress
	dataBuffer := bytes.NewBuffer(data)
	g, err := gzip.NewReader(dataBuffer)
	if err != nil {
		return err
	}
	pbData, err := io.ReadAll(g)
	if err != nil {
		return err
	}
	if err := g.Close(); err != nil {
		return err
	}

	// Load protobuf
	msg := new(snapshot.Snapshot)
	if err := msg.Unmarshal(pbData); err != nil {
		return err
	}

	// Make snapshot available to the syncer, replacing any previous one
	// that has not been loaded yet.
	d.r.mu.Lock()
	d.r.snapshotsByInstance[d.instance] = msg
	d.r.mu.Unlock()

	t2 := time.Now()
	d.l.WithFields(logrus.Fields{
		"filename":          ni.FullName,
		"time_load_storage": utils.TimeDiff(t1, t0),
		"time_load_total":   utils.TimeDiff(t2, t0),
	}).Info("Snapshot downloaded")

	return nil
}
