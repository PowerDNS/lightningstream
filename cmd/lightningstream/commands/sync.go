package commands

import (
	"context"

	"github.com/PowerDNS/simpleblob"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"powerdns.com/platform/lightningstream/status"

	"powerdns.com/platform/lightningstream/syncer"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := simpleblob.GetBackend(ctx, conf.Storage.Type, conf.Storage.Options)
	if err != nil {
		return err
	}
	logrus.WithField("storage_type", conf.Storage.Type).Info("Storage backend initialised")

	eg, ctx := errgroup.WithContext(ctx)
	for name, lc := range conf.LMDBs {
		s, err := syncer.New(name, st, conf, lc)
		if err != nil {
			return err
		}

		name := name
		eg.Go(func() error {
			err := s.Sync(ctx)
			if err != nil {
				if err == context.Canceled {
					logrus.WithField("db", name).Error("Sync cancelled")
					return err
				}
				logrus.WithError(err).WithField("db", name).Error("Sync failed")
			}
			return err
		})
	}

	status.StartHTTPServer(conf)

	logrus.Info("All syncers running")
	return eg.Wait()
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Continuous bidirectional syncing",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSync(); err != nil {
			logrus.WithError(err).Error("Error")
		}
	},
}
