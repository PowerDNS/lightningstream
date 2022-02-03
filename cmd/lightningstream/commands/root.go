package commands

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/config/logger"
)

var (
	configFile   string
	instanceName string
	debug        bool
	conf         config.Config
)

var rootHelp = `This tool syncs one or more LMDB databases with an S3 bucket
`

var rootCmd = &cobra.Command{
	Use:   "lightningstream",
	Short: "This tool syncs one or more LMDB databases with an S3 bucket",
	Long:  rootHelp,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		conf = config.Default()
		conf.Version = version
		err := conf.LoadYAMLFile(configFile, true)
		if err != nil {
			logrus.Fatalf("Load config file %q: %v", configFile, err)
		}
		// Also check at this stage. A config must always be valid, even if you
		// later override some items.
		if err := conf.Check(); err != nil {
			logrus.Fatalf("Config file error: %v", err)
		}

		conf.Log = conf.Log.Merge(logger.FlagConfig)
		if debug {
			conf.Log.Level = "debug"
		}
		if instanceName != "" {
			conf.Instance = instanceName
		}
		logger.Configure(conf.Log)
		logrus.WithField("version", version).Debug("Running")
	},
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "lightningstream.yaml", "Config file")
	rootCmd.PersistentFlags().StringVarP(&instanceName, "instance", "i", "", "Instance name, defaults to hostname. MUST be unique for each instance")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	logger.RegisterFlagsWith(rootCmd.PersistentFlags().StringVar)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Error("Error")
		os.Exit(1)
	}
}
