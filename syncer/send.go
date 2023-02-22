package syncer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/c2h5oh/datasize"
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/utils"
)

// Send opens the env and starts the send-loop. No data is received from the
// storage.
func (s *Syncer) Send(ctx context.Context) error {
	// Open the env
	env, err := s.openEnv()
	if err != nil {
		return err
	}
	s.startStatsLogger(ctx, env)
	s.registerCollector(env)
	defer s.closeEnv(env)
	return s.sendLoop(ctx, env)
}

// sendLoop enters a send-loop and only returns when an error that cannot be
// handled occurs.
// TODO: evolve this into a more generic sync loop with send-only option
func (s *Syncer) sendLoop(ctx context.Context, env *lmdb.Env) error {
	info, err := env.Info()
	if err != nil {
		return err
	}
	lastTxnID := header.TxnID(info.LastTxnID)
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
			if err := utils.SleepContext(ctx, s.c.LMDBPollInterval); err != nil {
				return err
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
			if header.TxnID(info.LastTxnID) > lastTxnID {
				lastTxnID = header.TxnID(info.LastTxnID)
				break // dump new version
			}
		}
		s.l.WithField("LastTxnID", lastTxnID).Debug("LMDB changed, syncing")
	}
}

func (s *Syncer) SendOnce(ctx context.Context, env *lmdb.Env) (txnID header.TxnID, err error) {
	var msg = new(snapshot.Snapshot)
	msg.FormatVersion = snapshot.CurrentFormatVersion
	msg.Meta.DatabaseName = s.name
	msg.Meta.Hostname = hostname
	msg.Meta.InstanceID = s.instanceID()
	msg.Meta.GenerationID = s.generationID()

	t0 := time.Now() // for performance measurements

	// Snapshot timestamp determined within transaction
	var ts time.Time

	var tTxnAcquire time.Time
	var tShadow time.Time

	schemaTracksChanges := s.lc.SchemaTracksChanges

	var inTxn func(lmdb.TxnOp) error
	if schemaTracksChanges {
		inTxn = env.View
	} else {
		inTxn = env.Update
	}

	err = inTxn(func(txn *lmdb.Txn) error {
		// Determine snapshot timestamp after we opened the transaction
		ts = time.Now()
		tTxnAcquire = ts
		tsNano := header.TimestampFromTime(ts)
		msg.Meta.TimestampNano = uint64(tsNano)

		// Get the actual transaction ID we ended up opening, which could be
		// higher than the one we received from env.Info() if a new one was
		// created in the meantime.
		// int64 matches the type we get from env.Info()
		// If we update, it may be higher than from Info
		txnID = header.TxnID(txn.ID())
		s.l.WithField("txnID", txnID).Debug("Started dump of transaction")

		// First update the shadow dbs
		if !schemaTracksChanges {
			err := s.mainToShadow(ctx, txn, tsNano)
			if err != nil {
				return err
			}
		}
		tShadow = time.Now()

		// List of DBIs to dump
		dbiNames, err := lmdbenv.ReadDBINames(txn)
		if err != nil {
			return err
		}

		// Dump all DBIs using their shadow db
		for _, dbiName := range dbiNames {
			if strings.HasPrefix(dbiName, SyncDBIPrefix) {
				continue // skip our own special dbs
			}

			readDBIName := dbiName
			if !schemaTracksChanges {
				readDBIName = SyncDBIShadowPrefix + dbiName
			}
			dbiMsg, err := s.readDBI(txn, readDBIName, false)
			if err != nil {
				return fmt.Errorf("dbi %s: %w", dbiNames, err)
			}
			dbiMsg.Name = dbiName // replace shadow name if used
			msg.Databases = append(msg.Databases, dbiMsg)

			if utils.IsCanceled(ctx) {
				return context.Canceled
			}
		}
		return nil
	})
	if err != nil {
		// We always return LMDB reading errors, as these are really unexpected
		return 0, err
	}
	tDumped := time.Now()

	// If no actual changes were made, LMDB will not record the transaction
	// and reuse the ID the next time, so we need to adjust the txnID we return.
	info, err := env.Info()
	if err != nil {
		return 0, err
	}
	if header.TxnID(info.LastTxnID) < txnID {
		// Transaction was empty, no changes
		s.l.WithField("prevTxnID", txnID).WithField("txnID", info.LastTxnID).
			Debug("Adjusting TxnID (no changes)")
		txnID = header.TxnID(info.LastTxnID)
	}
	msg.Meta.LmdbTxnID = int64(txnID)

	out, dds, err := snapshot.DumpData(msg)
	if err != nil {
		return 0, err
	}
	tDumpedData := time.Now()

	metricSnapshotsLoaded.WithLabelValues(s.name).Inc()
	metricSnapshotsLastTimestamp.WithLabelValues(s.name).Set(float64(ts.UnixNano()) / 1e9)
	metricSnapshotsLastSize.WithLabelValues(s.name).Set(float64(len(out)))

	// Send it to storage
	name := snapshot.Name(s.name, s.instanceID(), s.generationID(), ts)
	for i := 0; i < s.c.StorageRetryCount || s.c.StorageRetryForever; i++ {
		metricSnapshotsStoreCalls.Inc()
		err = s.st.Store(ctx, name, out)
		if err != nil {
			s.l.WithError(err).Warn("Store failed, retrying")
			metricSnapshotsStoreFailed.WithLabelValues(s.name).Inc()

			// Signal failure to health tracker
			s.storageStoreHealth.AddFailure(err)

			if err := utils.SleepContext(ctx, s.c.StorageRetryInterval); err != nil {
				return 0, err
			}
			continue
		}
		s.l.Debug("Store succeeded")
		metricSnapshotsStoreBytes.Add(float64(len(out)))

		// Signal success to health tracker
		s.storageStoreHealth.AddSuccess()

		break
	}
	if err != nil {
		s.l.WithError(err).Warn("Store failed too many times, giving up")
		metricSnapshotsStoreFailedPermenantly.WithLabelValues(s.name).Inc()
		return 0, err
	}
	tStored := time.Now()

	s.l.WithFields(logrus.Fields{
		"time_acquire":     utils.TimeDiff(tTxnAcquire, t0),
		"time_copy_shadow": tShadow.Sub(tTxnAcquire).Round(time.Millisecond),
		"time_dump":        tDumped.Sub(tShadow).Round(time.Millisecond),
		"time_marshal":     dds.TMarshaled.Round(time.Millisecond),
		"time_compress":    dds.TCompressed.Round(time.Millisecond),
		"time_store":       tStored.Sub(tDumpedData).Round(time.Millisecond),
		"time_total":       tStored.Sub(t0).Round(time.Millisecond),
		"snapshot_size":    datasize.ByteSize(len(out)).HumanReadable(),
		"snapshot_name":    name,
		"txnID":            txnID,
	}).Info("Stored snapshot")

	// Tell the cleaner which snapshots made by other instances have been
	// incorporated in the last snapshot that we sent.
	s.cleaner.SetCommitted(s.lastByInstance)

	return txnID, nil
}
