package syncer

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/c2h5oh/datasize"
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/snapshot"
)

func (s *Syncer) Send(ctx context.Context) error {
	// TODO: First fetch last snapshot and compare

	// Open the env
	s.l.WithField("lmdbpath", s.lc.Path).Info("Opening LMDB for reading")
	env, err := lmdbenv.NewWithOptions(s.lc.Path, s.lc.Options)
	if err != nil {
		return err
	}
	defer func() {
		if err := env.Close(); err != nil {
			s.l.WithError(err).Warn("Env close returned error")
		}
	}()

	// Print some env info
	info, err := env.Info()
	if err != nil {
		return err
	}
	s.l.WithFields(logrus.Fields{
		"MapSize":   datasize.ByteSize(info.MapSize).HumanReadable(),
		"LastTxnID": info.LastTxnID,
	}).Info("Env info")

	warnedEmpty := false
	lastTxnID := info.LastTxnID
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
			if err := sleepContext(ctx, time.Second); err != nil { // TODO: config
				return err
			}

			// Wait for change
			info, err := env.Info()
			if err != nil {
				return err
			}
			if info.LastTxnID != lastTxnID {
				lastTxnID = info.LastTxnID
				break
			}
		}
		s.l.WithField("LastTxnID", lastTxnID).Debug("LMDB changed, syncing")
	}
}

func (s *Syncer) SendOnce(ctx context.Context, env *lmdb.Env) (txnID int64, err error) {
	// TODO: other meta fields (generationID, instanceID, previousSnapshot)
	var msg = new(snapshot.Snapshot)
	msg.FormatVersion = 1
	msg.Meta.DatabaseName = s.name
	msg.Meta.Hostname = hostname

	t0 := time.Now() // for performance measurements

	// Snapshot timestamp determined within transaction
	var ts time.Time

	err = env.View(func(txn *lmdb.Txn) error {
		// Determine snapshot timestamp after we opened the transaction
		ts = time.Now()
		msg.Meta.TimestampNSec = ts.UnixNano()

		// Get the actual transaction ID we ended up opening, which could be
		// higher than the one we received from env.Info() if a new one was
		// created in the meantime.
		// int64 matches the type we get from env.Info()
		txnID = int64(txn.ID())
		s.l.WithField("txnID", txnID).Debug("Started dump of transaction")
		msg.Meta.LmdbTxnID = txnID

		// List of DBIs to dump
		dbiNames, err := lmdbenv.ReadDBINames(txn)
		if err != nil {
			return err
		}

		// Dump all DBIs
		for _, dbiName := range dbiNames {
			dbiMsg, err := s.readDBI(txn, dbiName)
			if err != nil {
				return err
			}
			msg.Databases = append(msg.Databases, dbiMsg)
		}
		return nil
	})
	if err != nil {
		// We always return LMDB reading errors, as these are really unexpected
		return -1, err
	}
	tDumped := time.Now()

	// Snapshot complete, serialize it
	pb, err := msg.Marshal()
	if err != nil {
		return -1, err
	}
	tMarshaled := time.Now()

	// Compress it
	out := bytes.NewBuffer(make([]byte, 0, datasize.MB))
	gw, err := gzip.NewWriterLevel(out, gzip.BestSpeed)
	if err != nil {
		return -1, err
	}
	if _, err = gw.Write(pb); err != nil {
		return -1, err
	}
	if err = gw.Close(); err != nil {
		return -1, err
	}
	tCompressed := time.Now()

	// Send it to storage
	fileTimestamp := strings.Replace(
		ts.UTC().Format("20060102-150405.000000000"),
		".", "-", 1)
	name := fmt.Sprintf("%s__%s__%s__%s.pb.gz",
		s.name,
		fileTimestamp,
		"TODO-instance",
		"TODO-generation",
	)
	for i := 0; i < 100; i++ { // TODO: config
		err = s.st.Store(ctx, name, out.Bytes())
		if err != nil {
			s.l.WithError(err).Warn("Store failed, retrying")
			if err := sleepContext(ctx, time.Second); err != nil { // TODO: config
				return -1, err
			}
			continue
		}
		s.l.Debug("Store succeeded")
		break
	}
	if err != nil {
		s.l.WithError(err).Warn("Store failed too many times, giving up")
		return -1, err
	}
	tStored := time.Now()

	s.l.WithFields(logrus.Fields{
		"time_dump":     tDumped.Sub(t0).Round(time.Millisecond),
		"time_marshal":  tMarshaled.Sub(tDumped).Round(time.Millisecond),
		"time_compress": tCompressed.Sub(tMarshaled).Round(time.Millisecond),
		"time_store":    tStored.Sub(tCompressed).Round(time.Millisecond),
		"time_total":    tStored.Sub(t0).Round(time.Millisecond),
		"snapshot_size": datasize.ByteSize(out.Len()).HumanReadable(),
		"snapshot_name": name,
		"txnID":         txnID,
	}).Info("Stored snapshot")

	return txnID, nil
}

func (s *Syncer) readDBI(txn *lmdb.Txn, dbiName string) (dbiMsg *snapshot.DBI, err error) {
	l := s.l.WithField("dbi", dbiName)

	l.Debug("Opening DBI")
	dbi, err := txn.OpenDBI(dbiName, 0)
	if err != nil {
		return nil, err
	}

	stat, err := txn.Stat(dbi)
	if err != nil {
		return nil, err
	}
	l.WithField("entries", stat.Entries).Debug("Reading DBI")

	dbiMsg = new(snapshot.DBI)
	dbiMsg.Name = dbiName
	dbiMsg.Entries = make([]snapshot.KV, 0, stat.Entries)
	// TODO: directly read it into the right structure
	items, err := lmdbenv.ReadDBI(txn, dbi)
	if err != nil {
		return nil, err
	}

	var prev []byte
	for _, item := range items {
		if prev != nil && bytes.Compare(prev, item.Key) >= 0 {
			return nil, fmt.Errorf(
				"non-default key order detected in DBI %q, refusing to continue",
				dbiName)
		}
		prev = item.Key
		dbiMsg.Entries = append(dbiMsg.Entries, snapshot.KV{
			Key:   item.Key,
			Value: item.Val,
		})
	}

	return dbiMsg, nil
}
