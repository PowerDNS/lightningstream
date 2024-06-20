// Package stats implements a Prometheus Collector for LMDBs
package stats

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/PowerDNS/lightningstream/lmdbenv"
)

// Collector implements an LMDB stats collector for Prometheus.
// It must be registered with Prometheus before it actually works.
// It can only be registered once!
type Collector struct {
	mu      sync.Mutex
	targets map[string]Target
	smaps   bool
}

type Target struct {
	Name    string
	DBNames []string
	Env     *lmdb.Env
}

// NewCollector creates a new Collector.
// 'withSmaps' indicates if we need to collect /proc/self/smaps if available (Linux).
// Note that smaps collection could potentially be expensive.
func NewCollector(withSmaps bool) *Collector {
	return &Collector{
		targets: make(map[string]Target),
		smaps:   withSmaps,
	}
}

func (c *Collector) AddTarget(name string, dbnames []string, env *lmdb.Env) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.targets[name] = Target{
		Name:    name,
		DBNames: dbnames,
		Env:     env,
	}
}

func (c *Collector) EnableSmaps(enabled bool) {
	c.mu.Lock()
	c.smaps = enabled
	c.mu.Unlock()
}

// Describe is part of the prometheus.Collect interface
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- envMapSizeDesc
	ch <- envCurrentReadersDesc
	ch <- envMaxReadersDesc
	ch <- envLastTxnID
	ch <- envFileSizeDesc
	ch <- statUsageBytesDesc
	ch <- statTotalUsageBytesDesc
	ch <- statTotalUsageFractionDesc
	ch <- statEntriesDesc
	ch <- statPagesDesc
	ch <- statDepthDesc
	ch <- smapsDesc
}

// Collect is part of the prometheus.Collect interface. It fetches statistics
// from LMDB.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	var targets []Target
	c.mu.Lock()
	for _, t := range c.targets {
		targets = append(targets, t)
	}
	c.mu.Unlock()
	for _, t := range targets {
		c.doCollect(ch, t)
	}
}

func (c *Collector) doCollect(ch chan<- prometheus.Metric, t Target) {
	do := func() error {
		// Collect env info
		info, err := t.Env.Info()
		if err != nil {
			return errors.Wrap(err, "env info")
		}
		ch <- prometheus.MustNewConstMetric(
			envMapSizeDesc,
			prometheus.GaugeValue,
			float64(info.MapSize),
			t.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			envCurrentReadersDesc,
			prometheus.GaugeValue,
			float64(info.NumReaders),
			t.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			envLastTxnID,
			prometheus.GaugeValue,
			float64(info.LastTxnID),
			t.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			envMaxReadersDesc,
			prometheus.GaugeValue,
			float64(info.MaxReaders),
			t.Name,
		)

		// Collect file size
		path, err := t.Env.Path()
		if err != nil {
			return errors.Wrap(err, "env path")
		}
		filesize, err := lmdbFileSize(path)
		if err != nil {
			return errors.Wrap(err, "file size")
		}
		ch <- prometheus.MustNewConstMetric(
			envFileSizeDesc,
			prometheus.GaugeValue,
			float64(filesize),
			t.Name,
		)

		// Collect per database stat
		err = t.Env.View(func(txn *lmdb.Txn) error {
			var totalUsedBytes uint64
			var err error
			dbnames := t.DBNames
			if dbnames == nil {
				dbnames, err = lmdbenv.ReadDBINames(txn)
				if err != nil {
					return err
				}
			}
			for _, dbname := range dbnames {
				dbi, err := txn.OpenDBI(dbname, 0)
				if err != nil {
					return errors.Wrap(err, "opendbi "+dbname)
				}

				stat, err := txn.Stat(dbi)
				if err != nil {
					return errors.Wrap(err, "stat "+dbname)
				}

				usedBytes := PageUsageBytes(stat)
				ch <- prometheus.MustNewConstMetric(
					statUsageBytesDesc,
					prometheus.GaugeValue,
					float64(usedBytes),
					t.Name,
					dbname,
				)
				totalUsedBytes += usedBytes

				ch <- prometheus.MustNewConstMetric(
					statEntriesDesc,
					prometheus.GaugeValue,
					float64(stat.Entries),
					t.Name,
					dbname,
				)
				ch <- prometheus.MustNewConstMetric(
					statDepthDesc,
					prometheus.GaugeValue,
					float64(stat.Depth),
					t.Name,
					dbname,
				)
				ch <- prometheus.MustNewConstMetric(
					statPagesDesc,
					prometheus.GaugeValue,
					float64(stat.BranchPages),
					t.Name,
					dbname, "branch",
				)
				ch <- prometheus.MustNewConstMetric(
					statPagesDesc,
					prometheus.GaugeValue,
					float64(stat.LeafPages),
					t.Name,
					dbname, "leaf",
				)
				ch <- prometheus.MustNewConstMetric(
					statPagesDesc,
					prometheus.GaugeValue,
					float64(stat.OverflowPages),
					t.Name,
					dbname, "overflow",
				)
			}
			ch <- prometheus.MustNewConstMetric(
				statTotalUsageBytesDesc,
				prometheus.GaugeValue,
				float64(totalUsedBytes),
				t.Name,
			)
			// Should always be true
			if info.MapSize > 0 {
				ch <- prometheus.MustNewConstMetric(
					statTotalUsageFractionDesc,
					prometheus.GaugeValue,
					float64(totalUsedBytes)/float64(info.MapSize),
					t.Name,
				)
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "open view")
		}

		// Collect memory info from smaps (only available on Linux)
		c.mu.Lock()
		smaps := c.smaps
		c.mu.Unlock()
		if smaps {
			fullpath, err := lmdbFullPath(path)
			if err != nil {
				return errors.Wrap(err, "full path")
			}

			data, err := os.ReadFile("/proc/self/smaps")
			if err != nil {
				if !os.IsNotExist(err) {
					return errors.Wrap(err, "read smaps")
				}
			} else {
				m := getMemoryStats(string(data), fullpath)
				for k, v := range m {
					ch <- prometheus.MustNewConstMetric(
						smapsDesc,
						prometheus.GaugeValue,
						float64(v),
						t.Name,
						fullpath, k,
					)
				}
			}
		}

		return nil
	}
	if err := do(); err != nil {
		logrus.Errorf("Collector: %v", err)
	}
}

// Verify that Collector correctly implements the interface
var _ prometheus.Collector = (*Collector)(nil)

var (
	// These names were picked to be the same as for the Python services that
	// sync LMDBs.
	envMapSizeDesc = prometheus.NewDesc(
		"lmdb_mapsize_bytes",
		"Map size of LMDB database",
		[]string{"lmdb"},
		nil,
	)
	envCurrentReadersDesc = prometheus.NewDesc(
		"lmdb_env_readers_current",
		"Number of current readers for LMDB database",
		[]string{"lmdb"},
		nil,
	)
	envMaxReadersDesc = prometheus.NewDesc(
		"lmdb_env_readers_max",
		"Maximum number of readers for LMDB database",
		[]string{"lmdb"},
		nil,
	)
	envLastTxnID = prometheus.NewDesc(
		"lmdb_env_last_tnx_id",
		"Last write transaction ID of LMDB database",
		[]string{"lmdb"},
		nil,
	)
	envFileSizeDesc = prometheus.NewDesc(
		"lmdb_filesize_bytes",
		"File size of LMDB database",
		[]string{"lmdb"},
		nil,
	)
	statUsageBytesDesc = prometheus.NewDesc(
		"lmdb_db_usage_bytes",
		"Bytes used in last version by data in databases",
		[]string{"lmdb", "db"},
		nil,
	)
	statTotalUsageBytesDesc = prometheus.NewDesc(
		"lmdb_total_usage_bytes",
		"Bytes used in last version by data in all databases",
		[]string{"lmdb"},
		nil,
	)
	statTotalUsageFractionDesc = prometheus.NewDesc(
		"lmdb_total_usage_fraction",
		"Bytes used in last version by data in all databases as fraction (0-1) of map size",
		[]string{"lmdb"},
		nil,
	)
	statEntriesDesc = prometheus.NewDesc(
		"lmdb_stat_entries",
		"Number of entries in named LMDB database",
		[]string{"lmdb", "db"},
		nil,
	)
	statPagesDesc = prometheus.NewDesc(
		"lmdb_stat_pages",
		"Number of pages (4kB) in named LMDB database per page type (branch, leaf and overflow)",
		[]string{"lmdb", "db", "pagetype"},
		nil,
	)
	statDepthDesc = prometheus.NewDesc(
		"lmdb_stat_depth",
		"Tree depth in named LMDB database",
		[]string{"lmdb", "db"},
		nil,
	)
	smapsDesc = prometheus.NewDesc(
		"lmdb_smaps_bytes",
		"Memory statistics for LMDB database from /proc/self/smaps (Linux only)",
		[]string{"lmdb", "database_path", "smap"},
		nil,
	)
)

func lmdbFullPath(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", errors.Wrap(err, "stat")
	}
	if st.IsDir() {
		path = filepath.Join(path, "data.mdb")
	}
	return filepath.Abs(path)
}

func lmdbFileSize(path string) (int64, error) {
	path, err := lmdbFullPath(path)
	if err != nil {
		return 0, errors.Wrap(err, "full path")
	}
	st, err := os.Stat(path)
	if err != nil {
		return 0, errors.Wrap(err, "stat")
	}
	return st.Size(), nil
}
