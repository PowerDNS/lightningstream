package syncer

import (
	"context"
	"strings"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/strategy"
)

// mainToShadow syncs the current databases to shadow databases with timestamps.
// The sync is unidirectional, the state of the main database determines which
// keys will be present in the shadow database.
func (s *Syncer) mainToShadow(ctx context.Context, env *lmdb.Env, txn *lmdb.Txn, tsNano uint64) error {
	t0 := time.Now()

	// List of DBIs to dump
	dbiNames, err := lmdbenv.ReadDBINames(txn)
	if err != nil {
		return err
	}

	for _, dbiName := range dbiNames {
		if strings.HasPrefix(dbiName, SyncDBIPrefix) {
			continue // skip shadow and other special databases
		}
		// raw dump, because main does not have timestamps
		dbiMsg, err := s.readDBI(txn, dbiName, true)
		if err != nil {
			return err
		}

		targetDBI, err := txn.OpenDBI(SyncDBIShadowPrefix+dbiName, lmdb.Create)
		if err != nil {
			return err
		}

		it := &TimestampedIterator{
			Entries:              dbiMsg.Entries,
			DefaultTimestampNano: tsNano,
		}
		err = strategy.IterUpdate(txn, targetDBI, it)
		if err != nil {
			return err
		}
	}

	tStored := time.Now()

	s.l.WithFields(logrus.Fields{
		"time_total": tStored.Sub(t0).Round(time.Millisecond),
		"txnID":      txn.ID(),
	}).Info("Synced data to shadow")
	return nil
}

// shadowToMain syncs the current databases from shadow databases with timestamps.
// The sync is unidirectional. After the sync the main database will contain
// all the non-deleted key-values present in the shadow database.
func (s *Syncer) shadowToMain(ctx context.Context, env *lmdb.Env, txn *lmdb.Txn) error {
	t0 := time.Now()

	// List of DBIs to dump
	dbiNames, err := lmdbenv.ReadDBINames(txn)
	if err != nil {
		return err
	}

	for _, dbiName := range dbiNames {
		if strings.HasPrefix(dbiName, SyncDBIPrefix) {
			continue // skip shadow and other special databases
		}

		// Dump associated shadow database. We will ignore the timestamps.
		// At this point the shadow database must exist, as this function call
		// will always be preceded by a mainToShadow call.
		dbiMsg, err := s.readDBI(txn, SyncDBIShadowPrefix+dbiName, false)
		if err != nil {
			return err
		}

		// The target is the current DBI
		targetDBI, err := txn.OpenDBI(dbiName, 0)
		if err != nil {
			return err
		}

		// This iterator will insert the plain items without timestamp header
		it := &PlainIterator{
			Entries: dbiMsg.Entries,
		}
		err = strategy.IterUpdate(txn, targetDBI, it)
		if err != nil {
			return err
		}
	}

	tStored := time.Now()

	s.l.WithFields(logrus.Fields{
		"time_total": tStored.Sub(t0).Round(time.Millisecond),
		"txnID":      txn.ID(),
	}).Info("Synced data from shadow")
	return nil
}
