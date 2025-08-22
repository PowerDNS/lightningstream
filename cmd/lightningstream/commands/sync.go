package commands

import (
	"context"
	"errors"
	"os"
	"slices"

	"github.com/PowerDNS/lightningstream/snapshot/storage"
	"github.com/PowerDNS/lightningstream/status"
	"github.com/PowerDNS/lightningstream/syncer"
	"github.com/PowerDNS/lightningstream/utils"
	"github.com/PowerDNS/simpleblob"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wojas/go-healthz"
	"golang.org/x/sync/errgroup"
)

var (
	onlyOnce   bool
	markerFile string
	onlyDBs    []string
)

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVar(&onlyOnce, "only-once", false, "Only do a single run and exit")
	syncCmd.Flags().StringVar(&markerFile, "wait-for-marker-file", "", "Marker file to wait for in storage before starting syncers")
	syncCmd.Flags().StringArrayVar(&onlyDBs, "only-db", nil, "Only sync this named db (can be repeated)")
}

func runSync(receiveOnly bool) error {
	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	if onlyOnce {
		conf.OnlyOnce = true
	}

	// If it has not been logged yet in PersistentPreRun, log it now for
	// the sync command to ensure we always have a configuration dump
	// when debugging.
	if !logConfig {
		logrus.Infof("Effective configuration:\n%s\n", conf.String())
	}

	st, err := simpleblob.GetBackend(ctx, conf.Storage.Type, conf.Storage.Options)
	if err != nil {
		return err
	}
	logrus.WithField("storage_type", conf.Storage.Type).Info("Storage backend initialised")
	storage.SetGlobal(st)
	status.SetStorage(st)

	// If enabled, wait for marker file to be present in storage before starting syncers
	if markerFile != "" {
		logrus.Infof("waiting for marker file '%s' to be present in storage", markerFile)
		for {
			if _, err := st.Load(ctx, markerFile); err == nil {
				logrus.Infof("marker file '%s' found, proceeding", markerFile)
				break
			} else {
				if !os.IsNotExist(err) {
					logrus.WithError(err).Errorf("unable to check storage for marker file '%s'", markerFile)
				}
			}

			logrus.Debugf("waiting for marker file '%s'", markerFile)

			if err := utils.SleepContext(ctx, conf.StoragePollInterval); err != nil {
				return err
			}
		}
	}

	eg, ctx := errgroup.WithContext(ctx)
	for name, lc := range conf.LMDBs {
		l := logrus.WithField("db", name)

		if len(onlyDBs) > 0 {
			if !slices.Contains(onlyDBs, name) {
				l.Warn("Skipping because of --only-db flag")
				continue
			}
		}

		env, err := syncer.OpenEnv(l, lc)
		if err != nil {
			return err
		}

		opt := syncer.Options{
			ReceiveOnly: receiveOnly,
		}
		if SyncerOptionsCallback != nil {
			opt = SyncerOptionsCallback(opt, l)
		}

		s, err := syncer.New(name, env, st, conf, lc, opt)
		if err != nil {
			return err
		}

		eg.Go(func() error {
			defer func() {
				if err := env.Close(); err != nil {
					l.WithError(err).Error("Env close failed")
				}
			}()
			err := s.Sync(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					l.Error("Sync cancelled")
					return err
				}
				l.WithError(err).Error("Sync failed")
			}
			return err
		})
	}

	healthz.AddBuildInfo()
	if hostname, err := os.Hostname(); err == nil {
		healthz.SetMeta("hostname", hostname)
	}
	healthz.SetMeta("version", version)

	if !conf.OnlyOnce {
		status.StartHTTPServer(conf)
	} else {
		logrus.Info("Not starting the HTTP server, because OnlyOnce is set")
	}

	logrus.Info("All syncers running")
	return eg.Wait()
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Continuous bidirectional syncing",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSync(false); err != nil {
			logrus.WithError(err).Fatal("Error")
		}
	},
}
