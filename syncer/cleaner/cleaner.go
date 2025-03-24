package cleaner

import (
	"context"
	"sync"
	"time"

	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/PowerDNS/lightningstream/utils"
	"github.com/PowerDNS/simpleblob"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

func New(name string, st simpleblob.Interface, cc config.Cleanup, logger logrus.FieldLogger) *Worker {
	return &Worker{
		st:               st,
		name:             name,
		prefix:           name + "__",
		l:                logger.WithField("component", "cleaner"),
		conf:             cc,
		ignoredFilenames: map[string]bool{},
		snapFirstSeen:    map[string]time.Time{},
		mu:               sync.Mutex{},
		lastByInstance:   map[string]time.Time{},
	}
}

// Worker performs a periodic cleanup of old snapshots. It cleans snapshots of
// all instances, not just itself.
type Worker struct {
	st               simpleblob.Interface
	name             string
	prefix           string
	l                logrus.FieldLogger
	ignoredFilenames map[string]bool
	snapFirstSeen    map[string]time.Time
	conf             config.Cleanup

	// mu protects lastByInstance
	mu sync.Mutex
	// lastByInstance tracks the last snapshot loaded by instance and
	// successfully committed to a snapshot, so that the cleaner can make safe
	// decisions about when to remove stale snapshots.
	lastByInstance map[string]time.Time
}

// SetCommitted records the snapshot time of the last snapshots loaded
// by instance that was subsequently incorporated in one of our own snapshots.
func (w *Worker) SetCommitted(last map[string]time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for instance, t := range last {
		w.lastByInstance[instance] = t
	}
}

// GetCommitted retrieves the snapshot time of the last snapshot loaded
// by instance that was subsequently incorporated in one of our own snapshots.
// If no snapshot was loaded for this instance, it will return a zero time.
func (w *Worker) GetCommitted(instance string) time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastByInstance[instance]
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.conf.Enabled {
		// If disabled, simply wait for the context to close
		<-ctx.Done()
		return context.Canceled
	}
	for {
		err := w.RunOnce(ctx, time.Now())
		if err != nil {
			w.l.WithError(err).Warn("Clean run failed")
		}
		if err = utils.SleepContextPerturb(ctx, w.conf.Interval); err != nil {
			return err
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context, now time.Time) error {
	if !w.conf.Enabled {
		return nil
	}

	ls, err := w.st.List(ctx, w.prefix)
	metricListCalls.Inc()
	if err != nil {
		metricListFailed.Inc()
		return err
	}
	names := ls.Names()

	// Get a list of snapshots, ignoring files that are not snapshots
	var removalCandidates []snapshot.NameInfo // candidates for deletion
	seen := make(map[string]bool)
	for _, name := range names {
		if w.ignoredFilenames[name] {
			//r.l.WithField("filename", name).Debug("Ignored")
			continue
		}
		ni, err := snapshot.ParseName(name)
		if err != nil {
			w.l.WithError(err).WithField("filename", name).
				Debug("Skipping invalid filename")
			w.ignoredFilenames[name] = true
			continue
		}

		removalCandidates = append(removalCandidates, ni)
		seen[name] = true
	}
	nTotal := len(removalCandidates)

	// Clean old entries from the snapFirstSeen map (files that no longer appear
	// in the listing)
	var removeFromFirstSeen []string
	for name := range w.snapFirstSeen {
		if !seen[name] {
			removeFromFirstSeen = append(removeFromFirstSeen, name)
		}
	}
	for _, name := range removeFromFirstSeen {
		delete(w.snapFirstSeen, name)
	}

	const (
		doNotDelete        = false
		continueEvaluation = true
	)

	// Sort from newest to oldest, so that the first snapshot we see for an
	// instance is its most recent one.
	slices.SortFunc(removalCandidates, func(a, b snapshot.NameInfo) int {
		switch {
		case a.Timestamp.After(b.Timestamp):
			return 1

		case a.Timestamp.Before(b.Timestamp):
			return -1
		}

		return 0
	})

	// Protect the newest snapshots, in case an instance is still downloading it.
	// We do not use the snapshot time, because the appearance of a snapshot can be
	// delayed due to multi-site syncing. Instead, we keep track when we have
	// first seen a snapshot in the listing, and only consider it for deletion
	// after that interval exceeds the threshold.
	seenInstances := make(map[string]bool)
	removalCandidates = lo.Filter(removalCandidates, func(ni snapshot.NameInfo, index int) bool {
		firstSeenTime, exists := w.snapFirstSeen[ni.FullName]
		if !exists {
			w.snapFirstSeen[ni.FullName] = now
			// Not setting seenInstances so that the previous newest snapshot
			// remains for at least one config Cleanup.Interval when a new one
			// just arrived.
			return doNotDelete
		}
		if now.Sub(firstSeenTime) <= w.conf.MustKeepInterval {
			seenInstances[ni.InstanceID] = true
			return doNotDelete
		}
		return continueEvaluation
	})

	// Remove older snapshots if we have seen very recent snapshots for that instance
	var tooOld []snapshot.NameInfo
	removalCandidates = lo.Filter(removalCandidates, func(ni snapshot.NameInfo, index int) bool {
		if !seenInstances[ni.InstanceID] {
			// Do not delete newest (first in list) snapshot for this instance
			seenInstances[ni.InstanceID] = true
			if now.Sub(ni.Timestamp) > w.conf.RemoveOldInstancesInterval {
				// Move to tooOld list to consider for stale instance cleanup below
				tooOld = append(tooOld, ni)
			}
			return doNotDelete
		}
		// This instance has newer snapshots that we keep.
		return continueEvaluation
	})

	// Everything that is still in the removalCandidates can now be deleted,
	// because we have determined there are newer snapshots for those instances,
	// and we are skipping all the very recent ones.
	nCleaned := 0
	nError := 0
	for _, ni := range removalCandidates {
		l := w.l.WithField("snapshot", ni.FullName)
		l.Debug("Cleaning old snapshot")
		metricDeleteCalls.WithLabelValues(w.name, "newer snapshot").Inc()
		if err := w.st.Delete(ctx, ni.FullName); err != nil {
			l.WithError(err).Warn("Could not delete old snapshot")
			metricDeleteFailed.Inc()
			nError++
			continue
		}
		nCleaned++
	}

	// Now consider the tooOld list with instances that are likely dead.
	// Removing them is only safe if recent snapshots have incorporated their
	// changes. The only safe way to determine if this is the case,
	// is to check if _this_ instance has both loaded and merged the snapshot,
	// and subsequently successfully committed a snapshot of its own.
	for _, ni := range tooOld {
		l := w.l.WithField("snapshot", ni.FullName)
		lastCommitted := w.GetCommitted(ni.InstanceID)
		if ni.Timestamp.After(lastCommitted) {
			// Newer than any snapshots we have merged and committed that
			// includes this version, so do not delete.
			l.Debug("Not cleaning stale snapshot, merge not proven yet")
			continue
		}
		metricDeleteCalls.WithLabelValues(w.name, "stale instance").Inc()
		if err := w.st.Delete(ctx, ni.FullName); err != nil {
			l.WithError(err).Warn("Could not delete old snapshot")
			metricDeleteFailed.Inc()
			nError++
			continue
		}
		l.WithField("instance", ni.InstanceID).Info(
			"Cleaning stale instance snapshot, merge proven")
		nCleaned++
	}

	w.l.WithFields(logrus.Fields{
		"cleaned": nCleaned,
		"failed":  nError,
		"total":   nTotal,
	}).Debug("Cleaning stats")

	return nil
}
