// Package stats implements a Prometheus Collector for LMDBs
package stats

import (
	"fmt"
	"os"

	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/sirupsen/logrus"
)

// Log logs all LMDB statistics once using logrus
// If dbnames is nil, all databases are logged.
func Log(env *lmdb.Env, dbnames []string, withSmaps bool, log logrus.FieldLogger) {
	if log == nil {
		log = logrus.New()
	}

	do := func() error {
		// TODO: smaps (optional?)

		// Collect env info
		info, err := env.Info()
		if err != nil {
			return fmt.Errorf("env info: %w", err)
		}

		// Collect file size
		path, err := env.Path()
		if err != nil {
			return fmt.Errorf("env path: %w", err)
		}
		filesize, err := lmdbFileSize(path)
		if err != nil {
			return fmt.Errorf("file size: %w", err)
		}

		log.WithFields(logrus.Fields{
			"map_size":    info.MapSize,
			"num_readers": info.NumReaders,
			"max_readers": info.MaxReaders,
			"file_size":   filesize,
		}).Info("LMDB info")

		// Collect per database stat
		err = env.View(func(txn *lmdb.Txn) error {
			var err error
			if dbnames == nil {
				dbnames, err = lmdbenv.ReadDBINames(txn)
				if err != nil {
					return err
				}
			}

			for _, dbname := range dbnames {
				dbi, err := txn.OpenDBI(dbname, 0)
				if err != nil {
					return fmt.Errorf("opendbi %s: %w", dbname, err)
				}

				stat, err := txn.Stat(dbi)
				if err != nil {
					return fmt.Errorf("stat %s: %w", dbname, err)
				}

				log.WithFields(logrus.Fields{
					"DB":             dbname,
					"entries":        stat.Entries,
					"depth":          stat.Depth,
					"branch_pages":   stat.BranchPages,
					"overflow_pages": stat.OverflowPages,
					"psize":          stat.PSize,
				}).Info("LMDB db stat")
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("open view: %w", err)
		}

		// Collect memory info from smaps (only available on Linux)
		if withSmaps {
			fullpath, err := lmdbFullPath(path)
			if err != nil {
				return fmt.Errorf("full path: %w", err)
			}

			data, err := os.ReadFile("/proc/self/smaps")
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("read smaps: %w", err)
				}
			} else {
				m := getMemoryStats(string(data), fullpath)
				fields := make(logrus.Fields)
				for k, v := range m {
					fields[k] = v
				}
				log.WithFields(fields).Info("LMDB memory")
			}
		}

		return nil
	}
	if err := do(); err != nil {
		logrus.WithError(err).Error("LMDB stats logger")
	}
}
