package syncer

import (
	"context"
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

	r := receiver.New(
		s.st,
		s.c,
		s.name,
		s.l,
		s.instanceID(),
	)
	go func() {
		err := r.Run(ctx)
		s.l.WithError(err).Info("Receiver exited")
	}()

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
	for {
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

		// Wait for change
		s.l.Debug("Waiting for a new transaction")
		for {
			if err := utils.SleepContext(ctx, time.Second); err != nil { // TODO: config
				return err
			}

			// Check for new remote snapshot to load
			instance, snap := r.Next()
			if instance != "" {
				// New remote snapshot to load
				actualTxnID, localChanged, err := s.LoadOnce(ctx, env, instance, snap, lastTxnID)
				if err != nil {
					return err
				}
				if !localChanged {
					// Prevent triggering a local snapshot if there were no local
					// changes by bumping the transaction ID we consider synced.
					lastTxnID = actualTxnID
				}
			}

			// Wait for change
			info, err := env.Info()
			if err != nil {
				return err
			}
			logrus.WithFields(logrus.Fields{
				"info.LastTxnID": info.LastTxnID,
				"lastTxnID":      lastTxnID,
			}).Trace("Checking if TxnID changed")
			if info.LastTxnID > lastTxnID {
				lastTxnID = info.LastTxnID
				break // dump new version
			}
		}
		s.l.WithField("LastTxnID", lastTxnID).Debug("LMDB changed locally, syncing")
	}
}

func (s *Syncer) LoadOnce(ctx context.Context, env *lmdb.Env, instance string, snap *snapshot.Snapshot, lastTxnID int64) (txnID int64, localChanged bool, err error) {

	t0 := time.Now() // for performance measurements

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
			"txnID":        txnID,
			"lastTxnID":    lastTxnID,
			"instance":     instance,
			"timestamp":    snapshot.TimestampFromNano(snap.Meta.TimestampNano),
			"localChanged": localChanged,
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
			err = strategy.Put(txn, targetDBI, it)
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

	s.l.WithFields(logrus.Fields{
		"time_acquire":      utils.TimeDiff(tTxnAcquire, t0),
		"time_copy_shadow1": utils.TimeDiff(tShadow1End, tShadow1Start),
		"time_copy_shadow2": utils.TimeDiff(tShadow2End, tShadow2Start),
		"time_load":         utils.TimeDiff(tLoadEnd, tLoadStart),
		"time_total":        utils.TimeDiff(tLoaded, t0),
		"txnID":             txnID,
		"instance":          instance,
		"timestamp":         snapshot.TimestampFromNano(snap.Meta.TimestampNano),
	}).Info("Loaded remote snapshot")

	return txnID, localChanged, nil
}
