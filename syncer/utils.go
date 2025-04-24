package syncer

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"time"

	"github.com/PowerDNS/lightningstream/lmdbenv/dbiflags"
	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/lmdbenv/stats"
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/PowerDNS/lightningstream/syncer/hooks"
	"github.com/PowerDNS/lightningstream/utils"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// SyncDBIPrefix is the shared DBI name prefix for all special tables that
	// must not be synced.
	SyncDBIPrefix = "_sync"
	// SyncDBIShadowPrefix is the DBI name prefix of shadow databases.
	SyncDBIShadowPrefix = "_sync_shadow_"
)

const (
	// AllowedShadowDBIFlagsMask is the set of LMDB DBI flags that we transfer
	// to shadow DBIs.
	// MDB_INTEGERKEY needs to be transferred for proper ordering of shadow DBIs.
	AllowedShadowDBIFlagsMask = dbiflags.IntegerKey
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
	ErrNoTxnID = errors.New("no TxnID set on iterator")
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

// readDBI reads a DBI into a snapshot DBI.
// By default, the headers of values will be split out to the corresponding snapshot fields.
// If rawValues is true, the value will be stored as is and the headers will
// not be extracted. This is useful when reading a database without headers.
// The origDBIName is used to ensure that the flags stored are those of the original
// DBI, not of the shadow DBI, and to set the name field of DBI.
func (s *Syncer) readDBI(txn *lmdb.Txn, dbiName, origDBIName string, rawValues bool) (dbiMsg *snapshot.DBI, err error) {
	l := s.l.WithField("dbi", dbiName)

	l.Debug("Opening DBI")
	dbi, err := txn.OpenDBI(dbiName, 0)
	if err != nil {
		return nil, err
	}

	// Get some DBI stats for optimisation
	stat, err := txn.Stat(dbi)
	if err != nil {
		return nil, err
	}
	l.WithField("entries", stat.Entries).Debug("Reading DBI")

	// Always enable txn.RawRead so that the slices point directly into the
	// LMDB pages, since we will copy the values into the snapshot.DBI anyway.
	restoreRawRead := txn.RawRead
	txn.RawRead = true
	defer func() {
		txn.RawRead = restoreRawRead
	}()

	// Pre-allocate based on the amount of data the DBI currently
	// takes up as LMDB pages to avoid reallocs later.
	// For native DBIs, we always have a 24 byte header of which we only include
	// a timestamp (9 bytes) and headers (2 bytes) in the protobuf. The keys
	// and values will take 2 bytes each if < 16 bytes long, adding up to
	// 15 bytes out of 24 used, and 9 free for larger values. So this will be
	// almost certainly enough for native data, even under the assumption that
	// LMDB does not have any overhead in the pages (it does) and all pages
	// are tightly packed (they rarely are).
	sizeHint := float64(stats.PageUsageBytes(stat))
	if rawValues {
		// For rawValues, we do not have this header padding, so add a bit more.
		sizeHint = (1.2 * sizeHint) + 4*float64(stat.Entries)
	}
	dbiMsg = snapshot.NewDBISize(int(sizeHint))
	dbiMsg.SetName(origDBIName)

	// Flags of the original DBI (not the shadow DBI)
	var dbiFlags uint
	if dbiName != origDBIName {
		// We are dumping a shadow DBI, but need to store the flags of the
		// original DBI.
		l.Debug("Opening original DBI for flags")
		origDBI, err := txn.OpenDBI(origDBIName, 0)
		if err != nil {
			return nil, err
		}
		dbiFlags, err = txn.Flags(origDBI)
		if err != nil {
			return nil, err
		}
	} else {
		// This is the original DBI
		dbiFlags, err = txn.Flags(dbi)
		if err != nil {
			return nil, err
		}
	}
	isDupSort := dbiFlags&lmdb.DupSort > 0
	if isDupSort {
		if !s.lc.DupSortHack {
			return nil, fmt.Errorf("readDBI: dupsort db %q found and dupsort_hack disabled", dbiName)
		}
		dbiMsg.SetTransform(snapshot.TransformDupSortHackV1)
	}
	dbiMsg.SetFlags(uint64(dbiFlags))

	// Read all entries
	c, err := txn.OpenCursor(dbi)
	if err != nil {
		return nil, errors.Wrap(err, "open cursor")
	}
	defer c.Close()

	filterReadDBI := s.hooks.FilterReadDBI
	filtered := false

	var prev []byte
	var flag uint = lmdb.First
	for {
		key, val, err := c.Get(nil, nil, flag)
		if err != nil {
			if lmdb.IsNotFound(err) {
				break
			} else {
				return nil, errors.Wrap(err, "cursor next")
			}
		}

		// Not checking wrong order to support native integer and reverse ordering
		if prev != nil && !isDupSort && bytes.Equal(prev, key) {
			return nil, fmt.Errorf(
				"duplicate key detected in DBI %q without dupsort_hack, refusing to continue",
				dbiName)
		}
		prev = key

		var ts header.Timestamp
		var txnID header.TxnID
		var flags header.Flags
		if !rawValues {
			h, appVal, err := header.Parse(val)
			if err != nil {
				return nil, ErrEntry{
					DBIName: dbiName,
					Key:     key,
					Err:     err,
				}
			}
			ts = h.Timestamp
			flags = h.Flags
			val = appVal
		}

		flag = lmdb.Next

		// Filter and append
		if filterReadDBI != nil {
			include := filterReadDBI(hooks.FilterReadDBIParams{
				Timestamp: ts,
				TxnID:     txnID,
				Flags:     flags,
			})
			if !include {
				filtered = true
				continue
			}
		}
		dbiMsg.Append(snapshot.KV{
			Key:           key,
			Value:         val,
			TimestampNano: uint64(ts),
			Flags:         uint32(flags.Masked()),
		})
	}

	// Check how close our hint was
	var efficiency float64
	actualSize := dbiMsg.Size()
	if sizeHint > 0 {
		efficiency = math.Round(100*float64(actualSize)/sizeHint) / 100
	}
	s.l.WithFields(logrus.Fields{
		"entries":          dbiMsg.NumWrittenEntries,
		"size_hint_used":   int(sizeHint),
		"actual_data_size": actualSize,
		"hint_efficiency":  efficiency,
		"filtered":         filtered, // if filtered, the estimate is not accurate
	}).Debug("Check our pre-alloc size estimate (<1 is OK)")

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
