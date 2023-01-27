package syncer

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/lmdbenv/strategy"
	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/syncer/receiver"
	"powerdns.com/platform/lightningstream/utils"
)

// Sync opens the env and starts the two-way sync loop.
func (s *Syncer) Sync(ctx context.Context) error {
	// Open the env
	env, err := s.openEnv()
	if err != nil {
		return err
	}
	defer s.closeEnv(env)

	s.startStatsLogger(ctx, env)
	s.registerCollector(env)

	r := receiver.New(
		s.st,
		s.c,
		s.name,
		s.l,
		s.instanceID(),
	)

	return s.syncLoop(ctx, env, r)
}

// syncLoop enters a two-way sync-loop and only returns when an error that cannot be
// handled occurs.
// TODO: merge with sendLoop
func (s *Syncer) syncLoop(ctx context.Context, env *lmdb.Env, r *receiver.Receiver) error {
	info, err := env.Info()
	if err != nil {
		return err
	}
	lastTxnID := info.LastTxnID
	warnedEmpty := false

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Run cleaner in background to clean old snapshots
	go func() {
		err := s.cleaner.Run(ctx)
		s.l.WithError(err).Info("Cleaner exited")
	}()

	// Wait for an initial snapshot listing
	for {
		err := r.RunOnce(ctx, true) // including own snapshots, only during startup
		if err == nil {
			break
		}
		s.l.WithError(err).Info("Waiting for initial receiver listing")
		time.Sleep(time.Second) // TODO: Configurable?
	}

	hasSnapshots := r.HasSnapshots()
	hasData := lastTxnID > 0
	waitingForInstances := make(map[string]bool)
	for _, instance := range r.SeenInstances() {
		waitingForInstances[instance] = true
	}

	if hasData && !s.lc.SchemaTracksChanges {
		// Sync to shadow using a time in the past to not overwrite newer data.
		// At least is allows us to save newer entries that were added
		// while the syncer was not running. it will not save updated entries.
		s.l.Info("Syncing main to shadow, in case data was changed before start")
		err := env.Update(func(txn *lmdb.Txn) error {
			// TODO: use const
			past := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
			pastNano := uint64(past.UnixNano())
			return s.mainToShadow(ctx, txn, pastNano)
		})
		if err != nil {
			return err
		}
	}

	// Store a snapshot of current data if there are no snapshots yet
	if hasData && !hasSnapshots {
		s.l.Info("Performing initial snapshot, because none exists yet")
		actualTxnID, err := s.SendOnce(ctx, env)
		if err != nil {
			return err
		}
		lastTxnID = actualTxnID
	} else {
		s.l.Debug("Not performing a snapshot of current data before sync " +
			"(empty database or some snapshot already exist)")
	}

	// Run receiver in background to get newer snapshot after loading the
	// initial batch of snapshots.
	go func() {
		err := r.Run(ctx)
		s.l.WithError(err).Info("Receiver exited")
	}()

	// There is no guarantee that the snapshots listed before have already been
	// downloaded and are available for loading, but this is fine.
	// The update loop will not cause any issues, even if a snapshot is generated.

	for {
		// Keep checking for new remote snapshots until we have local changes
		for {
			// Load all new snapshots that are available now
			// This will not starve the syncer from sending local changes,
			// because every load will implicitly trigger a snapshot when local
			// changes are detected.
			for {
				instance, update := r.Next()
				if instance == "" {
					break // no more remote snapshots
				}
				// New remote snapshot to load
				delete(waitingForInstances, instance)
				actualTxnID, localChanged, err := s.LoadOnce(ctx, env, instance, update, lastTxnID)
				if err != nil {
					return err
				}
				if !localChanged {
					// Prevent triggering a local snapshot if there were no local
					// changes by bumping the transaction ID we consider synced.
					lastTxnID = actualTxnID
				}
			}

			// Check if any of the instances we are waiting for have disappeared,
			// which can happen when a cleaner removes stale snapshots.
			if len(waitingForInstances) > 0 {
				// Check if the instance still has snapshots
				current := make(map[string]bool)
				for _, instance := range r.SeenInstances() {
					current[instance] = true
				}
				// No need to wait for these any longer, remove them from the map
				var toDelete []string
				for instance := range waitingForInstances {
					if !current[instance] {
						toDelete = append(toDelete, instance)
					}
				}
				for _, instance := range toDelete {
					delete(waitingForInstances, instance)
				}
				// Sort alphabetically for log message
				var instances []string
				for instance := range waitingForInstances {
					instances = append(instances, instance)
				}
				sort.Strings(instances)
				istr := strings.Join(instances, " ")
				s.l.WithField("instances", istr).Info("Still waiting for snapshots from instances")
			}

			// If set, we are done now
			if s.c.OnlyOnce && len(waitingForInstances) == 0 {
				s.l.Info("Stopping, because requested to only do a single pass")
				return nil
			}

			// Wait for change in local LMDB
			info, err := env.Info()
			if err != nil {
				return err
			}
			s.l.WithFields(logrus.Fields{
				"info.LastTxnID": info.LastTxnID,
				"lastTxnID":      lastTxnID,
			}).Trace("Checking if TxnID changed")
			if info.LastTxnID > lastTxnID {
				lastTxnID = info.LastTxnID
				break // create new snapshot
			}

			// Sleep before next check for snapshots and local changes
			s.l.Debug("Waiting for a new transaction")
			if err := utils.SleepContext(ctx, s.c.LMDBPollInterval); err != nil {
				return err
			}
		}

		s.l.WithField("LastTxnID", lastTxnID).Debug("LMDB changed locally, syncing")
		// Store snapshot
		// If lastTxnID == 0, the LMDB is empty, so we do not store anything
		if lastTxnID > 0 {
			actualTxnID, err := s.SendOnce(ctx, env)
			if err != nil {
				return err
			}
			lastTxnID = actualTxnID
		} else if !warnedEmpty {
			s.l.Warn("LMDB is empty, waiting for data")
			warnedEmpty = true
		}

	}
}

func (s *Syncer) LoadOnce(ctx context.Context, env *lmdb.Env, instance string, update snapshot.Update, lastTxnID int64) (txnID int64, localChanged bool, err error) {

	t0 := time.Now() // for performance measurements
	snap := update.Snapshot

	var tTxnAcquire time.Time
	var tShadow1Start time.Time
	var tShadow1End time.Time
	var tShadow2Start time.Time
	var tShadow2End time.Time
	var tLoadStart time.Time
	var tLoadEnd time.Time

	schemaTracksChanges := s.lc.SchemaTracksChanges

	err = env.Update(func(txn *lmdb.Txn) error {
		ts := time.Now()
		tTxnAcquire = ts
		tsNano := uint64(ts.UnixNano())
		txnID = int64(txn.ID())

		// There was a local change if the update transaction ID was more than 1
		// higher than the last transaction ID we took a snapshot of.
		// If nothing had changed since the last snapshot, we would get the next
		// transaction ID in sequence.
		localChanged = lastTxnID < (txnID - 1)

		// TODO: Would be useful to have the NameInfo here
		l := s.l.WithFields(logrus.Fields{
			"txnID":             txnID,
			"lastTxnID":         lastTxnID,
			"snapshot_instance": instance,
			"timestamp":         snapshot.TimestampFromNano(snap.Meta.TimestampNano),
			"localChanged":      localChanged,
		})
		l.Debug("Started load")

		// First update the shadow dbs to reflect the latest local state
		tShadow1Start = time.Now()
		if !schemaTracksChanges && localChanged {
			err := s.mainToShadow(ctx, txn, tsNano)
			if err != nil {
				return err
			}
		}
		tShadow1End = time.Now()

		// Apply snapshot
		tLoadStart = time.Now()
		for _, dbiMsg := range snap.Databases {
			dbiName := dbiMsg.Name
			ld := l.WithField("dbi", dbiName)

			if strings.HasPrefix(dbiName, SyncDBIPrefix) {
				ld.Warn("Remote snapshot contains private DBI, ignoring")
				continue // skip our own special dbs
			}

			ld.Debug("Starting merge of snapshot into DBI")
			targetDBIName := dbiName
			if !schemaTracksChanges {
				targetDBIName = SyncDBIShadowPrefix + dbiName

				// We need to create the actual data DBI too in this case
				_, err := txn.OpenDBI(dbiName, lmdb.Create)
				if err != nil {
					return err
				}
			}

			targetDBI, err := txn.OpenDBI(targetDBIName, lmdb.Create)
			if err != nil {
				return err
			}

			it := &TimestampedIterator{
				Entries: dbiMsg.Entries,
			}
			err = strategy.Update(txn, targetDBI, it)
			if err != nil {
				return err
			}
			ld.Debug("Merge successful")

			if utils.IsCanceled(ctx) {
				return context.Canceled
			}
		}
		tLoadEnd = time.Now()

		// Apply state of shadow dbs to main data
		tShadow2Start = time.Now()
		if !schemaTracksChanges {
			err := s.shadowToMain(ctx, txn)
			if err != nil {
				return err
			}
		}
		tShadow2End = time.Now()

		return nil
	})
	if err != nil {
		// We always return LMDB reading errors, as these are really unexpected
		return -1, false, err
	}
	tLoaded := time.Now()

	// If no actual changes were made, LMDB will not record the transaction
	// and reuse the ID the next time, so we need to adjust the txnID we return.
	info, err := env.Info()
	if err != nil {
		return -1, false, err
	}
	if info.LastTxnID < txnID {
		// Transaction was empty, no changes
		s.l.WithField("prevTxnID", txnID).WithField("txnID", info.LastTxnID).
			Debug("Adjusting TxnID (no changes)")
		txnID = info.LastTxnID
	}

	ts := snapshot.TimestampFromNano(snap.Meta.TimestampNano)
	l := s.l.WithFields(logrus.Fields{
		"time_total":        utils.TimeDiff(tLoaded, t0),
		"txnID":             txnID,
		"snapshot_instance": instance,
		"shorthash":         snapshot.ShortHash(snap.Meta.InstanceID, ts),
		"timestamp":         ts,
	})
	l.Info("Loaded remote snapshot")

	l.WithFields(logrus.Fields{
		"time_acquire":      utils.TimeDiff(tTxnAcquire, t0),
		"time_copy_shadow1": utils.TimeDiff(tShadow1End, tShadow1Start),
		"time_copy_shadow2": utils.TimeDiff(tShadow2End, tShadow2Start),
		"time_load":         utils.TimeDiff(tLoadEnd, tLoadStart),
	}).Debug("Loaded remote snapshot (with timings)")

	s.lastByInstance[instance] = update.NameInfo.Timestamp

	return txnID, localChanged, nil
}
