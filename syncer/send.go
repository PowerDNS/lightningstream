package syncer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/PowerDNS/lightningstream/syncer/events"
	"github.com/PowerDNS/lightningstream/syncer/hooks"
	"github.com/PowerDNS/lightningstream/utils"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/c2h5oh/datasize"
	"github.com/sirupsen/logrus"
)

func (s *Syncer) SendOnce(ctx context.Context, env *lmdb.Env) (txnID header.TxnID, err error) {
	var msg = new(snapshot.Snapshot)
	msg.FormatVersion = snapshot.CurrentFormatVersion
	msg.CompatVersion = snapshot.WriteCompatFormatVersion
	msg.Meta.DatabaseName = s.name
	msg.Meta.Hostname = hostname
	msg.Meta.InstanceID = s.instanceID()
	msg.Meta.GenerationID = s.generationIDWithID(uint64(time.Now().UnixNano()))

	t0 := time.Now() // for performance measurements

	// Snapshot timestamp determined within transaction
	var ts time.Time

	var tTxnAcquire time.Time
	var tShadow time.Time

	schemaTracksChanges := s.lc.SchemaTracksChanges

	txnRawRead := false
	var inTxn func(lmdb.TxnOp) error
	if schemaTracksChanges {
		inTxn = env.View
		txnRawRead = true // []byte will point directly into LMDB, potentially unsafe
	} else {
		inTxn = env.Update
	}

	err = inTxn(func(txn *lmdb.Txn) error {
		// Call hook if defined
		if s.hooks.BeforeRead != nil {
			params := hooks.BeforeReadParams{
				Snapshot: msg,
				Env:      env,
				Txn:      txn,
			}
			if err := s.hooks.BeforeRead(params); err != nil {
				return fmt.Errorf("hooks.BeforeRead: %w", err)
			}
		}

		// We can speed SendOnce up by about 30% by setting txn.RawRead to true
		// if this is env.View, but this is only safe if the returned []byte
		// keys and values are not used outside the transaction, because
		// they point into the LMDB pages.
		// This is safe here, because s.readDBI() allocates an arena []byte and
		// copies all keys and values in there when txn.RawRead is set.
		txn.RawRead = txnRawRead

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

		// Skip the data dump if we are not going to send it anyway
		if s.opt.ReceiveOnly {
			return nil
		}

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
			dbiMsg, err := s.readDBI(txn, readDBIName, dbiName, false)
			if err != nil {
				return fmt.Errorf("dbi %s: %w", dbiNames, err)
			}

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
	// This is only relevant for write transactions, which are only used for
	// a non-native schema with shadow DBIs, not for native schema.
	// In such case there is a race present, where the application could have
	// written new data under the txnID we think we have written.
	// TODO: Further investigate this non-native potential race
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

	// Return before actually writing a snapshot, but after the txnID was adjusted
	// when we are in receive-only mode.
	if s.opt.ReceiveOnly {
		s.l.Info("Snapshot store skipped, because running in receive-only mode")
		return txnID, nil
	}

	// Build a filename
	ni := snapshot.NameInfo{
		Kind:         snapshot.KindSnapshot,
		Extension:    snapshot.DefaultExtension,
		SyncerName:   s.name,
		InstanceID:   s.instanceID(),
		GenerationID: s.generationID(),
		Timestamp:    ts,
	}
	if s.hooks.UpdateSnapshotInfo != nil {
		err := s.hooks.UpdateSnapshotInfo(hooks.SnapshotInfo{
			Snapshot: msg,
			NameInfo: &ni,
		})
		if err != nil {
			return 0, fmt.Errorf("hooks.UpdateSnapshotInfo: %w", err)
		}
	}
	name := ni.BuildName()

	// Compress the snapshot and release memory
	out, dds, err := snapshot.DumpData(msg)
	if err != nil {
		return 0, err
	}
	tDumpedData := time.Now()

	// Used in logging
	generationID := string(msg.Meta.GenerationID)
	nameTimeStamp := snapshot.NameTimestamp(ts)

	meta := msg.Meta // keep a copy of metadata for events
	msg = nil        // no longer needed
	timeGC := utils.GC()

	unixTimestamp := float64(ts.UnixNano()) / 1e9

	metricSnapshotsLoaded.WithLabelValues(s.name).Inc()
	metricSnapshotsLastTimestamp.WithLabelValues(s.name).Set(unixTimestamp)
	// Log the time in seconds since the snapshot was created
	metricSnapshotsLastAge.WithLabelValues(s.name).Set(time.Since(ts).Seconds())
	metricSnapshotsLastSize.WithLabelValues(s.name).Set(float64(len(out)))

	// Calculate SHA-256 hash of the compressed snapshot
	// TODO: Revisit if we need this
	hash := sha256.Sum256(out)
	hashStr := hex.EncodeToString(hash[:])
	hashPrefix := hashStr[:8] // Use first 8 chars for brevity

	// Send it to storage
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

		// Send timestamp string used in the snapshot filename
		metricSnapshotsTimeStamp.WithLabelValues(s.name, s.instanceID(), nameTimeStamp).Set(1)

		// Send generation ID used in the snapshot filename
		metricSnapshotsSyncerGenerationID.WithLabelValues(s.name, s.instanceID(), generationID).Set(1)

		s.l.Debugf("Store succeeded with GenerationID %s", generationID)
		metricSnapshotsStoreBytes.Add(float64(len(out)))

		// UpdateStored hook and event
		updateInfo := events.UpdateInfo{
			NameInfo: ni,
			Meta:     meta,
		}
		if s.hooks.UpdateStored != nil {
			err := s.hooks.UpdateStored(updateInfo)
			if err != nil {
				return 0, err
			}
		}
		s.events.UpdateStored.Publish(updateInfo)

		// Signal success to health tracker
		s.storageStoreHealth.AddSuccess()

		break
	}
	if err != nil {
		s.l.WithError(err).Warn("Store failed too many times, giving up")
		metricSnapshotsStoreFailedPermanently.WithLabelValues(s.name).Inc()
		return 0, err
	}
	tStored := time.Now()

	var compressionRatio string
	if dds.CompressedSize > 0 {
		r := float32(dds.ProtobufSize) / float32(dds.CompressedSize)
		compressionRatio = fmt.Sprintf("1:%.2f", r)
	}

	s.l.WithFields(logrus.Fields{
		"time_copy_shadow":   tShadow.Sub(tTxnAcquire).Round(time.Millisecond),
		"time_dump":          tDumped.Sub(tShadow).Round(time.Millisecond),
		"time_compress":      dds.TCompressed.Round(time.Millisecond),
		"time_store":         tStored.Sub(tDumpedData).Round(time.Millisecond),
		"time_gc":            timeGC,
		"time_total":         tStored.Sub(t0).Round(time.Millisecond),
		"uncompressed_size":  dds.ProtobufSize.HumanReadable(),
		"compression_ratio":  compressionRatio,
		"snapshot_size":      datasize.ByteSize(len(out)).HumanReadable(),
		"snapshot_name":      name,
		"txnID":              txnID,
		"timeStampHash":      hashStr,
		"timeStampShortHash": hashPrefix,
		"timeStamp":          nameTimeStamp,
		"generationID":       generationID,
		"instanceID":         s.instanceID(),
		"unix_timestamp":     unixTimestamp,
		"time_acquire":       utils.TimeDiff(tTxnAcquire, t0),
	}).Info("Stored snapshot")

	if ni.Kind == snapshot.KindSnapshot {
		s.lastSnapshotTime = time.Now()
	}

	// Tell the cleaner which snapshots made by other instances have been
	// incorporated in the last snapshot that we sent.
	s.cleaner.SetCommitted(s.lastByInstance)

	return txnID, nil
}
