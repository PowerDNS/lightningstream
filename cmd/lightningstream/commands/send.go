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
	rootCmd.AddCommand(sendCmd)
}

func runSend() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := storage.GetBackend(conf.Storage)
	if err != nil {
		return err
	}
	logrus.WithField("storage_type", conf.Storage.Type).Info("Storage backend initialised")

	var wg sync.WaitGroup
	for name, lc := range conf.LMDBs {
		s := syncer.New(name, st, conf, lc)
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			err := s.Send(ctx)
			if err != nil {
				if err == context.Canceled {
					logrus.WithField("db", name).Error("Send cancelled")
					return
				}
				logrus.WithError(err).WithField("db", name).Error("Send failed")
			}
		}(name)
	}

	logrus.Info("All senders running")
	wg.Wait()
	return nil
}

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Continuously send data to storage",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSend(); err != nil {
			logrus.WithError(err).Error("Error")
		}
	},
}
