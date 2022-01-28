package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Just override the root one for this command and do nothing
		// (no config loading)
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}
