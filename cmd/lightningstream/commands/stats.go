package commands

import (
	"fmt"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/c2h5oh/datasize"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/stats"
)

func init() {
	rootCmd.AddCommand(statsCmd)
}

func statsForLMDB(name string, lc config.LMDB) error {
	env, err := lmdbenv.NewWithOptions(lc.Path, lc.Options)
	if err != nil {
		return err
	}
	defer env.Close()

	err = env.View(func(txn *lmdb.Txn) error {
		info, err := env.Info()
		if err != nil {
			return err
		}
		fmt.Printf("%s: Env info: %+v\n", name, *info)

		names, err := lmdbenv.ReadDBINames(txn)
		if err != nil {
			return err
		}

		var usedBytes uint64
		for _, dbiName := range names {
			dbi, err := txn.OpenDBI(dbiName, 0)
			if err != nil {
				return errors.Wrap(err, "dbi "+dbiName)
			}
			stat, err := txn.Stat(dbi)
			if err != nil {
				return errors.Wrap(err, "dbi "+dbiName)
			}
			used := stats.PageUsageBytes(stat)
			usedBytes += used
			usedHuman := datasize.ByteSize(used).HumanReadable()
			fmt.Printf("%s: dbi %s: %+v (%s)\n", name, dbiName, *stat, usedHuman)
		}

		var usedPct float64
		if info.MapSize > 0 {
			usedPct = 100 * float64(usedBytes) / float64(info.MapSize)
		}
		fmt.Printf("%s: Total Used: %s / %s (~ %.1f %%)\n",
			name,
			datasize.ByteSize(usedBytes).HumanReadable(),
			datasize.ByteSize(info.MapSize).HumanReadable(),
			usedPct)

		return nil
	})
	return err
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Print LMDB stats",
	Run: func(cmd *cobra.Command, args []string) {
		for name, lc := range conf.LMDBs {
			if err := statsForLMDB(name, lc); err != nil {
				logrus.WithError(err).WithField("db", name).Error("LMDB stats error")
			}
		}
	},
}
