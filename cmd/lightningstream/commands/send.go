package commands

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"powerdns.com/platform/lightningstream/status"

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

	eg, ctx := errgroup.WithContext(ctx)
	for name, lc := range conf.LMDBs {
		s, err := syncer.New(name, st, conf, lc)
		if err != nil {
			return err
		}

		name := name
		eg.Go(func() error {
			err := s.Send(ctx)
			if err != nil {
				if err == context.Canceled {
					logrus.WithField("db", name).Error("Send cancelled")
					return err
				}
				logrus.WithError(err).WithField("db", name).Error("Send failed")
			}
			return err
		})
	}

	status.StartHTTPServer(conf)

	logrus.Info("All senders running")
	return eg.Wait()
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
