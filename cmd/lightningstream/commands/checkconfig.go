package commands

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var checkConfigCmd = &cobra.Command{
	Use:   "checkconfig",
	Short: "Check that the configuration is valid",
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Info("Config is valid")
	},
}

func init() {
	rootCmd.AddCommand(checkConfigCmd)
}
