package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/syncer"
	"powerdns.com/platform/lightningstream/utils"
)

var (
	dumpName string
	dumpHide bool
)

func init() {
	rootCmd.AddCommand(dumpCmd)
	dumpCmd.Flags().BoolVarP(&dumpHide, "hide", "H", false, "Hide private lightningstream databases")
	dumpCmd.Flags().StringVarP(&dumpName, "name", "n", "", "Only dump given database name")
}

func dumpLMDB(name string, lc config.LMDB) error {
	env, err := lmdbenv.NewWithOptions(lc.Path, lc.Options)
	if err != nil {
		return err
	}
	defer env.Close()

	err = env.View(func(txn *lmdb.Txn) error {
		names, err := lmdbenv.ReadDBINames(txn)
		if err != nil {
			return err
		}

		for _, dbiName := range names {
			if dumpHide && strings.HasPrefix(dbiName, syncer.SyncDBIPrefix) {
				continue
			}

			fmt.Printf("\n### %s :: %s\n\n", name, dbiName)

			dbi, err := txn.OpenDBI(dbiName, 0)
			if err != nil {
				return errors.Wrap(err, "dbi "+dbiName)
			}

			items, err := lmdbenv.ReadDBI(txn, dbi)
			if err != nil {
				return errors.Wrap(err, "read dbi "+dbiName)
			}

			for _, item := range items {
				fmt.Printf("%s  =  %s\n",
					utils.DisplayASCII(item.Key),
					utils.DisplayASCII(item.Val),
				)
			}
		}

		return nil
	})
	return err
}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump LMDB contents",
	Run: func(cmd *cobra.Command, args []string) {
		var names []string
		for name := range conf.LMDBs {
			if dumpName != "" && dumpName != name {
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			lc := conf.LMDBs[name]
			if err := dumpLMDB(name, lc); err != nil {
				logrus.WithError(err).WithField("db", name).Error("LMDB dump error")
			}
		}
	},
}
