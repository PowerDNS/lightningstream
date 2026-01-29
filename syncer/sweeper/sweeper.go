package sweeper

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/lmdbenv/limitscanner"
	"github.com/PowerDNS/lightningstream/utils"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/sirupsen/logrus"
)

const (
	// SyncDBIPrefix is the shared DBI name prefix for all special tables that
	// must not be synced.
	// TODO: Duplicated from syncer to prevent import loop, move somewhere else
	SyncDBIPrefix = "_sync"
)

func New(name string, conf config.Sweeper, env *lmdb.Env, l logrus.FieldLogger, schemaTracksChanges bool) *Sweeper {
	return &Sweeper{
		name:                name,
		l:                   l.WithField("component", "sweeper"),
		env:                 env,
		conf:                conf,
		schemaTracksChanges: schemaTracksChanges,
	}
}

// Sweeper cleans old stale deleted entries from a single LMDB.
// This is also known as the "tomb sweeper".
// You need one Sweeper per LMDB.
// Do not confuse this with the Cleaner, which cleans snapshots.
type Sweeper struct {
	name string
	l    logrus.FieldLogger
	env  *lmdb.Env
	conf config.Sweeper

	schemaTracksChanges bool // native schema?

	lastStats stats // mainly for tests
}

// Run runs the sweeper according to the configured schedule.
// It only runs until an error occurs or the context is closed.
func (s *Sweeper) Run(ctx context.Context) error {
	wait := s.conf.FirstInterval
	for {
		// Wait
		if err := utils.SleepContext(ctx, wait); err != nil {
			return err // context closed
		}
		wait = s.conf.Interval

		// Do sweep
		err := s.sweep(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			s.l.WithError(err).Warn("Sweep failed")
		}
	}
}

// sweep performs a single full database sweep.
func (s *Sweeper) sweep(ctx context.Context) error {
	t0 := time.Now()

	retention := s.conf.RetentionDuration()
	cutoff := time.Now().Add(-retention)
	cutoffTS := header.TimestampFromTime(cutoff)

	s.l.WithField("cutoff", cutoff).Debug("Sweep started")
	defer s.l.Debug("Sweep finished")

	// Get the list of DBI names to work on.
	// This needs to be done every time, because new DBIs may be added over time.
	var dbiNames []string
	err := s.env.View(func(txn *lmdb.Txn) error {
		var err error
		dbiNames, err = lmdbenv.ReadDBINames(txn)
		return err
	})
	if err != nil {
		return err
	}

	// Stats
	var st stats

	for _, dbiName := range dbiNames {
		// We must not corrupt the data if a non-native schema is used for
		// the main data. In this case we only sweep our own shadow and meta
		// DBIs, which do use our native format.
		if !s.schemaTracksChanges && !strings.HasPrefix(dbiName, SyncDBIPrefix) {
			continue
		}

		l := s.l.WithField("dbi", dbiName)
		l.Debug("Sweep DBI")

		var last limitscanner.LimitCursor
		var limitReached bool
		for {
			err := s.env.Update(func(txn *lmdb.Txn) error {
				st.nTxn++

				dbi, err := txn.OpenDBI(dbiName, 0)
				if err != nil {
					return err
				}

				ls, err := limitscanner.NewLimitScanner(limitscanner.Options{
					Txn:                     txn,
					DBI:                     dbi,
					LimitDuration:           s.conf.LockDuration,
					LimitDurationCheckEvery: limitscanner.LimitDurationCheckEveryDefault,
					Last:                    last,
				})
				if err != nil {
					return err // configuration error
				}
				defer ls.Close()

				// Actual cleaning
				for ls.Scan() {
					h, _, err := header.Parse(ls.Val())
					if err != nil {
						return fmt.Errorf("failed to parse header for key %s: %w",
							utils.DisplayASCII(ls.Key()), err)
					}
					if !h.Flags.IsDeleted() {
						// TODO: Consider keeping track of a histogram of ages,
						//       but the standard Prometheus Observe() may be
						//       too slow to use here.
						st.nEntries++
						continue
					}
					if h.Timestamp >= cutoffTS {
						st.nEntries++
						st.nDeleted++
						continue
					}
					// Too old deleted entry, clean
					st.nCleaned++
					if err := txn.Del(dbi, ls.Key(), ls.Val()); err != nil {
						// Should only happen if the mapsize or disk is full
						return fmt.Errorf("failed to delete key %s: %w",
							utils.DisplayASCII(ls.Key()), err)
					}
				}

				last, limitReached = ls.Cursor()
				return ls.Err()
			})
			if limitReached {
				l.Debug("Sweep limit reached, continuing after pause")
				// Give the app some room to get a write lock before continuing
				if err := utils.SleepContext(ctx, s.conf.ReleaseDuration); err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to sweep dbi %s: %w", dbiName, err)
			}

			// Done with this DBI
			break
		}
	}
	st.timeTaken = time.Since(t0)
	s.lastStats = st
	metricCleanedTotal.WithLabelValues(s.name).Add(float64(st.nCleaned))
	metricStatsTotal.WithLabelValues(s.name).Set(float64(st.nEntries))
	metricStatsDeleted.WithLabelValues(s.name).Set(float64(st.nDeleted))
	metricStatsAvailable.WithLabelValues(s.name).Set(1)
	metricDurationSummary.WithLabelValues(s.name).Observe(st.timeTaken.Seconds())
	s.l.WithFields(st.logFields()).
		Info("Sweep for stale deleted entries completed")
	return nil
}
