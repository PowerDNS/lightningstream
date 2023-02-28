package commands

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(receiveCmd)
	receiveCmd.Flags().BoolVar(&onlyOnce, "only-once", false, "Only do a single run and exit")
	receiveCmd.Flags().StringVar(&markerFile, "wait-for-marker-file", "", "Marker file to wait for in storage before starting syncers")
}

var receiveCmd = &cobra.Command{
	Use:   "receive",
	Short: "Like sync, but never write snapshots",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSync(true); err != nil {
			logrus.WithError(err).Fatal("Error")
		}
	},
}
