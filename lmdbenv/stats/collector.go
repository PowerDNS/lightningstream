// Package stats implements a Prometheus Collector for LMDBs
package stats

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Collector implements an LMDB stats collector for Prometheus.
// It must be registered with Prometheus before it actually works.
type Collector struct {
	env     *lmdb.Env
	dbnames []string
	smaps   bool
}

// NewCollector creates a new Collector.
// 'dbnames' are the LMDB database names to collect stats for.
// 'withSmaps' indicates if we need to collect /proc/self/smaps if available (Linux).
// Note that smaps collection could potentially be expensive.
func NewCollector(env *lmdb.Env, dbnames []string, withSmaps bool) *Collector {
	return &Collector{
		env:     env,
		dbnames: dbnames,
		smaps:   withSmaps,
	}
}

// Describe is part of the prometheus.Collect interface
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Describe is implemented with DescribeByCollect. That's possible because the
	// Collect method will always return the same metrics with the same descriptors.
	prometheus.DescribeByCollect(c, ch)
}

// Collect is part of the prometheus.Collect interface. It fetches statistics
// from LMDB.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	do := func() error {
		// TODO: smaps (optional?)

		// Collect env info
		info, err := c.env.Info()
		if err != nil {
			return errors.Wrap(err, "env info")
		}
		ch <- prometheus.MustNewConstMetric(
			envMapSizeDesc,
			prometheus.GaugeValue,
			float64(info.MapSize),
		)
		ch <- prometheus.MustNewConstMetric(
			envCurrentReadersDesc,
			prometheus.GaugeValue,
			float64(info.NumReaders),
		)
		ch <- prometheus.MustNewConstMetric(
			envMaxReadersDesc,
			prometheus.GaugeValue,
			float64(info.MaxReaders),
		)

		// Collect file size
		path, err := c.env.Path()
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
		)

		// Collect per database stat
		err = c.env.View(func(txn *lmdb.Txn) error {
			var totalUsedBytes uint64
			for _, dbname := range c.dbnames {
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
					dbname,
				)
				totalUsedBytes += usedBytes

				ch <- prometheus.MustNewConstMetric(
					statEntriesDesc,
					prometheus.GaugeValue,
					float64(stat.Entries),
					dbname,
				)
				ch <- prometheus.MustNewConstMetric(
					statDepthDesc,
					prometheus.GaugeValue,
					float64(stat.Depth),
					dbname,
				)
				ch <- prometheus.MustNewConstMetric(
					statPagesDesc,
					prometheus.GaugeValue,
					float64(stat.BranchPages),
					dbname, "branch",
				)
				ch <- prometheus.MustNewConstMetric(
					statPagesDesc,
					prometheus.GaugeValue,
					float64(stat.LeafPages),
					dbname, "leaf",
				)
				ch <- prometheus.MustNewConstMetric(
					statPagesDesc,
					prometheus.GaugeValue,
					float64(stat.OverflowPages),
					dbname, "overflow",
				)
			}
			ch <- prometheus.MustNewConstMetric(
				statTotalUsageBytesDesc,
				prometheus.GaugeValue,
				float64(totalUsedBytes),
			)
			// Should always be true
			if info.MapSize > 0 {
				ch <- prometheus.MustNewConstMetric(
					statTotalUsageFractionDesc,
					prometheus.GaugeValue,
					float64(totalUsedBytes)/float64(info.MapSize),
				)
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "open view")
		}

		// Collect memory info from smaps (only available on Linux)
		if c.smaps {
			fullpath, err := lmdbFullPath(path)
			if err != nil {
				return errors.Wrap(err, "full path")
			}

			data, err := ioutil.ReadFile("/proc/self/smaps")
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
		nil,
		nil,
	)
	envCurrentReadersDesc = prometheus.NewDesc(
		"lmdb_env_readers_current",
		"Number of current readers for LMDB database",
		nil,
		nil,
	)
	envMaxReadersDesc = prometheus.NewDesc(
		"lmdb_env_readers_max",
		"Maximum number of readers for LMDB database",
		nil,
		nil,
	)
	envFileSizeDesc = prometheus.NewDesc(
		"lmdb_filesize_bytes",
		"File size of LMDB database",
		nil,
		nil,
	)
	statUsageBytesDesc = prometheus.NewDesc(
		"lmdb_db_usage_bytes",
		"Bytes used in last version by data in databases",
		[]string{"db"},
		nil,
	)
	statTotalUsageBytesDesc = prometheus.NewDesc(
		"lmdb_total_usage_bytes",
		"Bytes used in last version by data in all databases",
		nil,
		nil,
	)
	statTotalUsageFractionDesc = prometheus.NewDesc(
		"lmdb_total_usage_fraction",
		"Bytes used in last version by data in all databases as fraction (0-1) of map size",
		nil,
		nil,
	)
	statEntriesDesc = prometheus.NewDesc(
		"lmdb_stat_entries",
		"Number of entries in named LMDB database",
		[]string{"db"},
		nil,
	)
	statPagesDesc = prometheus.NewDesc(
		"lmdb_stat_pages",
		"Number of pages (4kB) in named LMDB database per page type (branch, leaf and overflow)",
		[]string{"db", "pagetype"},
		nil,
	)
	statDepthDesc = prometheus.NewDesc(
		"lmdb_stat_depth",
		"Tree depth in named LMDB database",
		[]string{"db"},
		nil,
	)
	smapsDesc = prometheus.NewDesc(
		"lmdb_smaps_bytes",
		"Memory statistics forLMDB database from /proc/self/smaps (Linux only)",
		[]string{"database_path", "smap"},
		nil,
	)
)

func lmdbFullPath(path string) (string, error) {
	if !strings.HasSuffix(path, ".mdb") {
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
