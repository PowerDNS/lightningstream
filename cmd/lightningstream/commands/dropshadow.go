package commands

import (
	"sort"
	"strings"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/syncer"
)

func init() {
	rootCmd.AddCommand(dropshadowCmd)
}

func dropshadowForLMDB(name string, lc config.LMDB) error {
	env, err := lmdbenv.NewWithOptions(lc.Path, lc.Options)
	if err != nil {
		return err
	}
	defer env.Close()

	err = env.Update(func(txn *lmdb.Txn) error {
		names, err := lmdbenv.ReadDBINames(txn)
		if err != nil {
			return err
		}

		for _, dbiName := range names {
			if !strings.HasPrefix(dbiName, syncer.SyncDBIPrefix) {
				continue
			}

			dbi, err := txn.OpenDBI(dbiName, 0)
			if err != nil {
				return errors.Wrap(err, "dbi "+dbiName)
			}

			logrus.WithField("db", dbiName).Info("Dropping")
			err = txn.Drop(dbi, true)
			if err != nil {
				return errors.Wrap(err, "dbi drop "+dbiName)
			}
		}
		return nil
	})
	return err
}

var dropshadowCmd = &cobra.Command{
	Use:   "drop-shadow",
	Short: "Drop all shadow dbs and other sync metadata. This can potentially cause data loss.",
	Run: func(cmd *cobra.Command, args []string) {
		var names []string
		for name := range conf.LMDBs {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			lc := conf.LMDBs[name]
			if err := dropshadowForLMDB(name, lc); err != nil {
				logrus.WithError(err).WithField("db", name).Error("LMDB drop-shadow error")
			}
		}
	},
}
