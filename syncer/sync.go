package syncer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/dbiflags"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
	"powerdns.com/platform/lightningstream/lmdbenv/strategy"
	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/status"
	"powerdns.com/platform/lightningstream/syncer/receiver"
	"powerdns.com/platform/lightningstream/utils"
)

const (
	// MaxConsecutiveSnapshotLoads are the maximum number of snapshot to
	// load before we break for snapshotting local changes, if local changes
	// exist.
	MaxConsecutiveSnapshotLoads = 10
)

// Sync opens the env and starts the two-way sync loop.
func (s *Syncer) Sync(ctx context.Context) error {
	env := s.env
	status.AddLMDBEnv(s.name, env)
	defer status.RemoveLMDBEnv(s.name)

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
func (s *Syncer) syncLoop(ctx context.Context, env *lmdb.Env, r *receiver.Receiver) error {
	info, err := env.Info()
	if err != nil {
		return err
	}

	// The lastSyncedTxnID starts as 0 to force at least one snapshot on startup
	var lastSyncedTxnID header.TxnID
	hasDataAtStart := info.LastTxnID > 0
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
		time.Sleep(time.Second)
	}

	// Start tracker: Initial storage snapshots listed
	s.startTracker.SetPassedInitialListing()

	hasSnapshots := r.HasSnapshots()
	ownInstanceID := s.instanceID()

	// The waitingForInstances are to:
	// - decide when to exit if OnlyOnce is true
	// - decide when to mark this instance as 'ready'
	//
	// There is a risk that we will never finish loading all of these instance.
	// This situation could happen when:
	// - The download of the latest snapshot for an instance keeps failing due
	//   to network errors.
	//
	// The following scenarios are handled appropriately:
	// - Corrupt snapshots will be ignored (see Receiver.MarkCorrupt)
	// - When the last snapshot of an instance has been cleaned/ignored, we will
	//   remove the instance from this set.
	//
	waitingForInstances := NewInstanceSet()
	for _, instance := range r.SeenInstances() {
		if instance == ownInstanceID {
			// This instance name has existing snapshots that we must load before
			// attempting to write new snapshots, to not lose data.
			s.l.Info("This instance has existing snapshots that must be loaded " +
				"before we create new snapshots")
		}
		waitingForInstances.Add(instance)
	}

	if hasDataAtStart && !s.lc.SchemaTracksChanges {
		// Sync to shadow using a time in the past to not overwrite newer data.
		// At least is allows us to save newer entries that were added
		// while the syncer was not running. It will not save updated entries.
		s.l.Info("Syncing main to shadow, in case data was changed before start")
		err := env.Update(func(txn *lmdb.Txn) error {
			// We would like to just use timestamp 0 here, but that
			// would break older clients that explicitly guard against
			// zero timestamps.
			// Previously we would use the year 2000 here, but this more clearly
			// identifies migrated records.
			pastNano := header.Timestamp(1) // 1 ns after UNIX epoch (1970)
			return s.mainToShadow(ctx, txn, pastNano)
		})
		if err != nil {
			return err
		}
	}

	// Store a snapshot of current data if there are no snapshots yet.
	// We do not do this here when a snapshot already exists, because it could
	// be a snapshot from this instance that we do not want to overwrite
	// with an empty one in the LMDB was reset.
	if hasDataAtStart && !hasSnapshots {
		s.l.Info("Performing initial snapshot, because none exists yet")
		actualTxnID, err := s.SendOnce(ctx, env)
		if err != nil {
			return err
		}
		lastSyncedTxnID = actualTxnID
		// Start tracker: Initial snapshot stored
		s.startTracker.SetPassedInitialStore()
	} else {
		s.l.Debug("Not performing a snapshot of current data before sync " +
			"(empty database or some snapshot already exist)")
	}

	if !hasDataAtStart {
		// Start tracker: Initial snapshot stored
		// To make sure it is set
		s.startTracker.SetPassedInitialStore()
	}

	// To force periodic snapshots
	lastSnapshotTime := time.Now()
	forceSnapshotInterval := s.c.StorageForceSnapshotInterval
	forceSnapshotEnabled := forceSnapshotInterval > 0

	// Run receiver in background to get newer snapshot after loading the
	// initial batch of snapshots.
	go func() {
		err := r.Run(ctx)
		s.l.WithError(err).Info("Receiver exited")
	}()

	// There is no guarantee that the snapshots listed before have already been
	// downloaded and are available for loading, but this is fine.
	// The update loop will not cause any issues, even if a snapshot is generated.

	// Keep checking for new remote snapshots and uploading on local changes
	for {
		// Load all new snapshots that are ready (downloaded and unpacked).
		// To not starve the syncer from sending local changes, we break for
		// a local snapshot after MaxConsecutiveSnapshotLoads loads.
		// Additionally, in shadow mode, every load will implicitly trigger a
		// snapshot when local changes are detected.
		nLoads := 0
	loadReadySnapshotsLoop:
		for {
			instance, update := r.Next()
			if instance == "" {
				break loadReadySnapshotsLoop // no more ready remote snapshots
			}
			// New remote snapshot to load
			nLoads++
			if instance == ownInstanceID {
				s.l.Info("Loading snapshot for own instance")
			}
			waitingForInstances.Remove(instance)
			actualTxnID, localChanged, err := s.LoadOnce(
				ctx, env, instance, update, lastSyncedTxnID)
			if err != nil {
				return err
			}
			utils.GC()
			if !localChanged {
				// Prevent triggering a local snapshot if there were no local
				// changes by bumping the transaction ID we consider synced
				// to the one just created by the snapshot load.
				// If there were local changes, we leave it as is to trigger
				// a snapshot below.
				lastSyncedTxnID = actualTxnID
			}
			if localChanged && nLoads > MaxConsecutiveSnapshotLoads {
				break loadReadySnapshotsLoop // allow a local snapshot before proceeding
			}
		}

		// Check if any of the instances we are waiting for have disappeared,
		// which can happen when a cleaner removes stale snapshots.
		if !waitingForInstances.Done() {
			cleaned := waitingForInstances.CleanDisappeared(r.SeenInstances())
			for _, name := range cleaned {
				s.l.WithField("cleaned_instance", name).Info(
					"No longer waiting for instance, because its snapshots disappeared")
			}
			if !waitingForInstances.Done() {
				cur := waitingForInstances.String()
				s.l.WithField("instances", cur).Info(
					"Still waiting for snapshots from instances")
			}
		}

		// Check if we need to do a periodic snapshot
		snapshotOverdue := false
		if dt := time.Since(lastSnapshotTime); forceSnapshotEnabled && dt > forceSnapshotInterval {
			snapshotOverdue = true
			logrus.WithField(
				"last_snapshot_time_passed", dt.Round(time.Second).String(),
			).Info("Snapshot overdue, forcing one")
		}

		// Check for change in local LMDB
		info, err := env.Info()
		if err != nil {
			return err
		}
		s.l.WithFields(logrus.Fields{
			"info.LastTxnID":  info.LastTxnID,
			"lastSyncedTxnID": lastSyncedTxnID,
		}).Trace("Checking if TxnID changed")
		if header.TxnID(info.LastTxnID) > lastSyncedTxnID || snapshotOverdue {
			// We have data to snapshot, or we have not performed a snapshot
			// yet after startup.
			if waitingForInstances.Contains(ownInstanceID) {
				// We must not store a snapshot before we have loaded our own
				// snapshot, because if we started with an empty LMDB, we
				// could write a snapshot that loses data that was only in our
				// own older snapshot. This could happen in this process is
				// terminated after writing this snapshot, and before loading
				// its old one (triggered at beginning with RunOnce) and
				// subsequently writing another snapshot.
				// This guarantee does not hold in shadow mode, because
				// every load there can trigger a snapshot store.
				// TODO: Can we fix this for shadow mode?
				s.l.Info("Waiting to load own old snapshot before writing a new one")
			} else {
				lastSyncedTxnID = header.TxnID(info.LastTxnID)
				s.l.WithField("LastTxnID", lastSyncedTxnID).Debug("LMDB changed locally, syncing")

				// Store snapshot
				if hasDataAtStart || lastSyncedTxnID > 0 {
					actualTxnID, err := s.SendOnce(ctx, env)
					if err != nil {
						return err
					}
					lastSyncedTxnID = actualTxnID
					lastSnapshotTime = time.Now()
					// Start tracker: Initial snapshot stored
					s.startTracker.SetPassedInitialStore()
				} else if !warnedEmpty {
					s.l.Warn("LMDB is empty, waiting for data")
					warnedEmpty = true
				}
			}
		}

		// Update start tracker if pass has completed
		if waitingForInstances.Done() {
			s.startTracker.SetPassCompleted()
		}

		// If set, we are done now.
		// This check is now intentionally after the local snapshot upload.
		if s.c.OnlyOnce && waitingForInstances.Done() {
			s.l.Info("Stopping, because requested to only do a single pass")
			return nil
		}

		// Sleep before next check for snapshots and local changes
		s.l.Debug("Waiting for a new transaction")
		if err := utils.SleepContext(ctx, s.c.LMDBPollInterval); err != nil {
			return err
		}
	}

}

func (s *Syncer) LoadOnce(ctx context.Context, env *lmdb.Env, instance string, update snapshot.Update, lastTxnID header.TxnID) (txnID header.TxnID, localChanged bool, err error) {

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
		tsNano := header.TimestampFromTime(ts)
		txnID = header.TxnID(txn.ID())

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
			"timestamp":         snapshot.NameTimestampFromNano(header.Timestamp(snap.Meta.TimestampNano)),
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
			dbiName := dbiMsg.Name()
			dbiOpt := s.lc.DBIOptions[dbiName]
			ld := l.WithField("dbi", dbiName)

			if strings.HasPrefix(dbiName, SyncDBIPrefix) {
				ld.Warn("Remote snapshot contains private DBI, ignoring")
				continue // skip our own special dbs
			}

			err := dbiMsg.ValidateTransform(snap.FormatVersion, schemaTracksChanges)
			if err != nil {
				return err
			}

			ld.Debug("Starting merge of snapshot into DBI")
			targetDBIName := dbiName
			if !schemaTracksChanges {
				targetDBIName = SyncDBIShadowPrefix + dbiName

				// We need to create the actual data DBI too if it does not
				// exist yet.
				exists, err := lmdbenv.DBIExists(txn, dbiName)
				if err != nil {
					return err
				}
				if !exists {
					if snap.FormatVersion < 3 && dbiOpt.OverrideCreateFlags == nil {
						// Earlier versions stored the DBI flags from the shadow
						// DBI instead of the flags from the original DBI.
						return fmt.Errorf(
							"DBI %s does not exist yet, and we cannot safely "+
								"create it from a formatVersion=%d snapshot, "+
								"only a formatVersion 3+ snapshot contains the "+
								"information we need for this; you can explicitly "+
								"override the flags through `override_create_flags` "+
								"in `dbi_options`, but only attempt this if you "+
								"are sure you need it",
							dbiName, snap.FormatVersion)
					}

					var flags = dbiflags.Flags(dbiMsg.Flags())
					if dbiOpt.OverrideCreateFlags != nil {
						flags = *dbiOpt.OverrideCreateFlags
					}
					ld.WithField("flags", flags).Warn("Creating new DBI from snapshot")
					_, err := txn.OpenDBI(dbiName, lmdb.Create|uint(flags))
					if err != nil {
						return err
					}
				}
			}

			// Create the target DBI if needed
			exists, err := lmdbenv.DBIExists(txn, targetDBIName)
			if err != nil {
				return err
			}
			if !exists {
				// The formatVersion does not matter here, because the DBI flags
				// stored in earlier versions will be the correct ones for the
				// DBI that we are creating here (shadow or native).
				var flags = dbiflags.Flags(dbiMsg.Flags())
				if dbiOpt.OverrideCreateFlags != nil {
					flags = *dbiOpt.OverrideCreateFlags
				}
				if !schemaTracksChanges {
					// Only flags like MDB_INTEGERKEY must be transferred
					// to shadow DBIs.
					flags &= AllowedShadowDBIFlagsMask
				}
				ld.WithField("dbi", targetDBIName).
					WithField("flags", flags).Warn("Creating new DBI from snapshot")
				_, err := txn.OpenDBI(targetDBIName, lmdb.Create|uint(flags))
				if err != nil {
					return err
				}
			}

			// Open the DBI now. It has been created if it did not exist yet.
			targetDBI, err := txn.OpenDBI(targetDBIName, 0)
			if err != nil {
				return err
			}

			it, err := NewNativeIterator(
				snap.FormatVersion,
				snap.CompatVersion,
				dbiMsg,
				0, // no default timestamp
				header.TxnID(txn.ID()),
			)
			if err != nil {
				return fmt.Errorf("create native iterator: %w", err)
			}
			if s.lc.HeaderExtraPaddingBlock {
				it.HeaderPaddingBlock = true
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
		return 0, false, err
	}
	tLoaded := time.Now()

	// If no actual changes were made, LMDB will not record the transaction
	// and reuse the ID the next time, so we need to adjust the txnID we return.
	info, err := env.Info()
	if err != nil {
		return 0, false, err
	}
	if header.TxnID(info.LastTxnID) < txnID {
		// Transaction was empty, no changes
		s.l.WithField("prevTxnID", txnID).WithField("txnID", info.LastTxnID).
			Debug("Adjusting TxnID (no changes)")
		txnID = header.TxnID(info.LastTxnID)
	}

	ts := snapshot.NameTimestampFromNano(header.Timestamp(snap.Meta.TimestampNano))
	l := s.l.WithFields(logrus.Fields{
		"time_total":        utils.TimeDiff(tLoaded, t0),
		"time_write_lock":   utils.TimeDiff(tLoaded, tTxnAcquire),
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
