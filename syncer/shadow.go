package syncer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/strategy"
	"powerdns.com/platform/lightningstream/utils"
)

// mainToShadow syncs the current databases to shadow databases with timestamps.
// The sync is unidirectional, the state of the main database determines which
// keys will be present in the shadow database.
func (s *Syncer) mainToShadow(ctx context.Context, txn *lmdb.Txn, tsNano uint64) error {
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

		if s.lc.DBIOptions[dbiName].DupSortHack {
			if err = dupSortHackEncode(dbiMsg.Entries); err != nil {
				return fmt.Errorf("dupsort_hack error for DBI %s: %w", dbiName, err)
			}
		}

		if utils.IsCanceled(ctx) {
			return context.Canceled
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

		if utils.IsCanceled(ctx) {
			return context.Canceled
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
func (s *Syncer) shadowToMain(ctx context.Context, txn *lmdb.Txn) error {
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

		dupSortHack := s.lc.DBIOptions[dbiName].DupSortHack

		// Dump associated shadow database. We will ignore the timestamps.
		// At this point the shadow database must exist, as this function call
		// will always be preceded by a mainToShadow call.
		dbiMsg, err := s.readDBI(txn, SyncDBIShadowPrefix+dbiName, false)
		if err != nil {
			return err
		}

		if dupSortHack {
			if err = dupSortHackDecode(dbiMsg.Entries); err != nil {
				return fmt.Errorf("dupsort_hack error for DBI %s: %w", dbiName, err)
			}
		}

		if utils.IsCanceled(ctx) {
			return context.Canceled
		}

		// The target is the current DBI
		var flags uint
		if dupSortHack {
			flags = lmdb.DupSort
		}
		targetDBI, err := txn.OpenDBI(dbiName, flags)
		if err != nil {
			return err
		}

		var stratFunc = strategy.IterUpdate
		if dupSortHack {
			stratFunc = strategy.EmptyPut
		}

		// This iterator will insert the plain items without timestamp header
		it := &PlainIterator{
			Entries: dbiMsg.Entries,
		}
		err = stratFunc(txn, targetDBI, it)
		if err != nil {
			return err
		}

		if utils.IsCanceled(ctx) {
			return context.Canceled
		}
	}

	tStored := time.Now()

	s.l.WithFields(logrus.Fields{
		"time_total": tStored.Sub(t0).Round(time.Millisecond),
		"txnID":      txn.ID(),
	}).Info("Synced data from shadow")
	return nil
}
