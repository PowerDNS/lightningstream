package lmdbstats

import (
	"context"
	"log/slog"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
)

const (
	DefaultCheckInterval   = time.Second
	DefaultPersistDBIName  = "_sync_history"
	DefaultPersistInterval = time.Minute
)

type Config struct {
	// CheckInterval determines how often we check the LastTxnID and readers
	// to keep track of reader lifetimes. This check is cheap.
	CheckInterval time.Duration `yaml:"check_interval"`

	// LogInterval determines how often periodic LMDB stats need to be logged
	// Disabled by default (0).
	LogInterval time.Duration `yaml:"log_interval"`
	// LogLevel is the level wat which to log periodic LMDB stats.
	// Defaults to INFO.
	LogLevel slog.Level `yaml:"log_level"`

	Persist        bool   `yaml:"persist"`
	PersistDBIName string `yaml:"persist_dbi_name"`

	// PersistInterval determines how often we commit the stats we have to the
	// LMDB when not triggered by the application, if enabled.
	PersistInterval time.Duration `yaml:"persist_interval"`

	// PersistErrors will, when a write txn fails, try to record this in a
	// separate write-transaction. The transaction that errored will always be
	// rolled back.
	PersistErrors bool `yaml:"persist_errors"`

	// PersistEntryLimit limits the number of persisted entries, removing
	// the older ones as we add new ones.
	// By default, all entries are kept forever (value 0).
	PersistEntryLimit int `yaml:"persist_entry_limit"`

	// MemorySmaps enables parsing of /proc/self/smaps on Linux for detailed
	// kernel memory page statistics. That can cause a latency spike for large
	// mappings.
	// This is only useful within the process doing the actual LMDB work.
	// Currently only supported on Linux, no effect on other platforms.
	MemoryDetails bool `yaml:"memory_details"`
}

func (c Config) WithDefaults() Config {
	if c.CheckInterval == 0 {
		c.CheckInterval = DefaultCheckInterval
	}
	if c.LogLevel == 0 {
		c.LogLevel = slog.LevelInfo
	}
	if c.PersistDBIName == "" {
		c.PersistDBIName = DefaultPersistDBIName
	}
	return c
}

type Options struct {
	Logger *slog.Logger
	Config Config
}

func NewMonitor(env *lmdb.Env, opt Options) (*Monitor, error) {
	l := opt.Logger
	if l == nil {
		l = slog.Default()
	}
	m := &Monitor{
		opt:  opt,
		conf: opt.Config.WithDefaults(),
		l:    l.With("component", "lmdbstats-monitor"),

		readers: make(map[ReaderInfo]*ReaderTracking),
	}
	return m, nil
}

// Monitor monitors one or more LMDB databases in the background.
type Monitor struct {
	opt     Options
	conf    Config
	l       *slog.Logger
	env     *lmdb.Env
	readers map[ReaderInfo]*ReaderTracking
}

// ReaderTracking keeps track of then a reader could have been created.
// Readers can be created at any time and do not create TxnIDs, but they
// always use the latest TxnID that was committed at the time of creation.
// When the monitor is just started, we do not have any records of when
// a read transaction could have started. If persisted, we can probably get
// it from the LMDB.
type ReaderTracking struct {
	FirstSeen    time.Time // first time we have seen this reader
	EarliestTime time.Time // the earliest time it could have been created
}

func (m *Monitor) Run(ctx context.Context) error {
	ticker := time.NewTicker(m.conf.CheckInterval)
	defer ticker.Stop()
	// FIXME: also do a run immediately

	var lastTime time.Time // starts as zero
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			now := time.Now()
			rl, err := ParsedReaderList(m.env)
			if err != nil {
				return err
			}
			seen := make(map[ReaderInfo]bool)
			for _, r := range rl {
				_, exists := m.readers[r]
				if !exists {
					// TODO: narrow down using internal txn info? needed?
					// TODO: narrow down using log DBI if persisted
					m.readers[r] = &ReaderTracking{
						FirstSeen:    now,
						EarliestTime: lastTime,
					}
				}
				seen[r] = true
			}
			// Remove readers that are gone
			for r := range m.readers {
				if !seen[r] {
					delete(m.readers, r) // safe in loop in Go
				}
			}
			lastTime = now

		}
	}
}
