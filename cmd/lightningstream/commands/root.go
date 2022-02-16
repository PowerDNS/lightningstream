package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/config/logger"
)

const (
	MaximumMinPID   = 200
	SkipPIDCheckEnv = "LIGHTNINGSTREAM_NO_PID_CHECK"
)

var (
	configFile   string
	instanceName string
	debug        bool
	minimumPID   int
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
		ensureMinimumPID()
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
	rootCmd.PersistentFlags().IntVar(&minimumPID, "minimum-pid", 0, fmt.Sprintf(
		"Try to fork processes until we reach a minimum PID to avoid LMDB lock PID clashes when running in a container. The maximum allowed value is %d", MaximumMinPID))
	logger.RegisterFlagsWith(rootCmd.PersistentFlags().StringVar)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Error("Error")
		os.Exit(1)
	}
}

// ensureMinimumPID ensures that we are running with a certain minimum PID to
// avoid LMDB PID locking issues when running in a container.
func ensureMinimumPID() {
	if minimumPID <= 0 {
		return
	}
	pid := os.Getpid()
	l := logrus.WithField("pid", pid)
	l.Debug("Checking PID")
	if minimumPID > MaximumMinPID {
		minimumPID = MaximumMinPID
		l.WithField("minimum_pid", minimumPID).Warn("Adjusted minimum PID to limit")
	}
	if pid >= minimumPID {
		l.WithField("minimum_pid", minimumPID).Info("PID satisfies minimum")
		return
	}
	if os.Getenv(SkipPIDCheckEnv) != "" {
		l.WithField("minimum_pid", minimumPID).Warn(
			"PID does NOT satisfy minimum, but requested to skip check")
		return
	}
	// Spawn processes to increase the last PID before we restart
	n := minimumPID - pid
	l.WithField("n", n).Info("Spawning processes to increase PID")
	for i := 0; i < n; i++ {
		cmd := exec.Command("/non-existent")
		_ = cmd.Run()
	}
	// Restart this process ins a subprocess
	l.Info("Starting new instance")
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Env = append(os.Environ(), SkipPIDCheckEnv+"=1")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			l.WithField("exitcode", exiterr.ExitCode()).Warn("Exiting with exit code")
			os.Exit(exiterr.ExitCode())
		}
		l.WithError(err).Fatal("Error running as subcommand")
	}
	l.Debug("Exiting with success")
	os.Exit(0)
}
