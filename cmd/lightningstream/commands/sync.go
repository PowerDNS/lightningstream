package commands

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"powerdns.com/platform/lightningstream/storage"
	"powerdns.com/platform/lightningstream/syncer"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := storage.GetBackend(conf.Storage)
	if err != nil {
		return err
	}
	logrus.WithField("storage_type", conf.Storage.Type).Info("Storage backend initialised")

	var wg sync.WaitGroup
	for name, lc := range conf.LMDBs {
		s, err := syncer.New(name, st, conf, lc)
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			err := s.Sync(ctx)
			if err != nil {
				if err == context.Canceled {
					logrus.WithField("db", name).Error("Sync cancelled")
					return
				}
				logrus.WithError(err).WithField("db", name).Error("Sync failed")
			}
		}(name)
	}

	logrus.Info("All syncers running")
	wg.Wait()
	return nil
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
