package syncer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/c2h5oh/datasize"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
	"powerdns.com/platform/lightningstream/lmdbenv/stats"
	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/utils"
)

const (
	// SyncDBIPrefix is the shared DBI name prefix for all special tables that
	// must not be synced.
	SyncDBIPrefix = "_sync"
	// SyncDBIShadowPrefix is the DBI name prefix of shadow databases.
	SyncDBIShadowPrefix = "_sync_shadow_"
)

// ErrEntry is returned when an entry is invalid, for example due to a missing
// or invalid header.
type ErrEntry struct {
	DBIName string
	Key     []byte
	Err     error
}

func (e ErrEntry) Error() string {
	k := utils.DisplayASCII(e.Key)
	return fmt.Sprintf("invalid entry (dbi %s, key %s): %v", e.DBIName, k, e.Err)
}

func (e ErrEntry) Unwrap() error {
	return e.Err
}

var (
	ErrNoTimestamp = errors.New("no timestamp to use for entry")
	ErrNoTxnID     = errors.New("no TxnID set on iterator")
)

var hostname string

func init() {
	h, err := os.Hostname()
	if err != nil {
		return
	}
	hostname = h
}

var reUnsafe = regexp.MustCompile("[^a-zA-Z0-9-]")

// instanceID returns a safe instance name
func (s *Syncer) instanceID() string {
	n := s.c.Instance
	if n == "" {
		n = hostname
	}
	n = reUnsafe.ReplaceAllString(n, "-")
	return n
}

// instanceID returns a safe instance name
func (s *Syncer) generationID() string {
	return fmt.Sprintf("G-%016x", s.generation)
}

// openEnv opens the LMDB env with the right options
func (s *Syncer) openEnv() (env *lmdb.Env, err error) {
	s.l.WithField("lmdbpath", s.lc.Path).Info("Opening LMDB")
	env, err = lmdbenv.NewWithOptions(s.lc.Path, s.lc.Options)
	if err != nil {
		return nil, err
	}

	// Print some env info
	info, err := env.Info()
	if err != nil {
		return nil, err
	}
	s.l.WithFields(logrus.Fields{
		"MapSize":   datasize.ByteSize(info.MapSize).HumanReadable(),
		"LastTxnID": info.LastTxnID,
	}).Info("Env info")

	// TODO: Perhaps check data if SchemaTracksChanges is set. Check if
	//       the timestamp is in a reasonable range or 0.

	return env, nil
}

// closeEnv closes the LMDB env, logging any unexpected errors for easy defer
func (s *Syncer) closeEnv(env *lmdb.Env) {
	if err := env.Close(); err != nil {
		s.l.WithError(err).Warn("Env close returned error")
	}
}

// readDBI reads a DBI into a snapshot DBI.
// By default, the headers of values will be split out to the corresponding snapshot fields.
// If rawValues is true, the value will be stored as is and the headers will
// not be extracted. This is useful when reading a database without headers.
func (s *Syncer) readDBI(txn *lmdb.Txn, dbiName string, rawValues bool) (dbiMsg *snapshot.DBI, err error) {
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

	dbiFlags, err := txn.Flags(dbi)
	if err != nil {
		return nil, err
	}
	isDupSort := dbiFlags&lmdb.DupSort > 0
	if isDupSort && !s.lc.DupSortHack {
		return nil, fmt.Errorf("readDBI: dupsort db %q found and dupsort_hack disabled", dbiName)
	}
	dbiMsg.Flags = uint64(dbiFlags)

	var prev []byte
	for _, item := range items {
		// Not checking wrong order to support native integer and reverse ordering
		if prev != nil && !isDupSort && bytes.Equal(prev, item.Key) {
			return nil, fmt.Errorf(
				"duplicate key detected in DBI %q without dupsort_hack, refusing to continue",
				dbiName)
		}
		prev = item.Key
		val := item.Val
		var ts header.Timestamp
		var flags header.Flags
		if !rawValues {
			h, appVal, err := header.Parse(val)
			if err != nil {
				return nil, ErrEntry{
					DBIName: dbiName,
					Key:     item.Key,
					Err:     err,
				}
			}
			ts = h.Timestamp
			flags = h.Flags
			val = appVal
		}

		dbiMsg.Entries = append(dbiMsg.Entries, snapshot.KV{
			Key:           item.Key,
			Value:         val,
			TimestampNano: uint64(ts),
			Flags:         uint32(flags.Masked()),
		})
	}

	return dbiMsg, nil
}

func (s *Syncer) startStatsLogger(ctx context.Context, env *lmdb.Env) {
	// Log LMDB stats every configured interval
	interval := s.c.LMDBLogStatsInterval
	if interval <= 0 {
		s.l.Info("LMDB stats logging disabled")
		return
	}
	s.l.WithField("interval", interval).Info("Enabled LMDB stats logging")
	go func() {
		logStatsTicker := time.NewTicker(interval)
		defer logStatsTicker.Stop()
		logStatsCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		for {
			select {
			case <-logStatsCtx.Done():
				return
			case <-logStatsTicker.C:
				// Skip the meta db, not that interesting
				stats.Log(env, nil, s.c.LMDBScrapeSmaps, s.l)
			}
		}
	}()

}

func (s *Syncer) registerCollector(env *lmdb.Env) {
	lmdbCollector.EnableSmaps(s.c.LMDBScrapeSmaps)
	lmdbCollector.AddTarget(s.name, nil, env)
}
