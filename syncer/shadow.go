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
		if strings.HasPrefix(dbiName, "_sync") {
			continue // skip shadow databases
		}
		dbiMsg, err := s.readDBI(txn, dbiName)
		if err != nil {
			return err
		}

		targetDBI, err := txn.OpenDBI("_sync_"+dbiName, lmdb.Create)
		if err != nil {
			return err
		}

		it := &MainToShadowIterator{
			DBIMsg: dbiMsg,
			TSNano: tsNano,
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
