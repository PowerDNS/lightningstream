package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	rtdebug "runtime/debug"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	buildInfo   *rtdebug.BuildInfo
	mainName    string
	mainVersion string

	versionJSON  bool
	versionExtra bool
)

// Deprecated: use SetBuildInfo
func SetVersion(v string) {
	mainName = "Lightning Stream"
	mainVersion = v
	rootCmd.Version = strings.TrimPrefix(v, "v")
}

func SetBuildInfo(name string) {
	mainName = name
	if bi, ok := rtdebug.ReadBuildInfo(); ok {
		mainVersion = bi.Main.Version
		rootCmd.Version = strings.TrimPrefix(bi.Main.Version, "v")
		buildInfo = bi
	} else {
		mainVersion = "dev"
		rootCmd.Version = "dev"
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVarP(&versionJSON, "json", "j", false, "Output full build information as a json object")
	versionCmd.Flags().BoolVarP(&versionExtra, "verbose", "v", false, "Output more information than just the version")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Just override the root one for this command and do nothing
		// (no config loading)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if versionJSON {
			e := json.NewEncoder(os.Stdout)
			e.SetIndent("", "  ")
			err := e.Encode(buildInfo)
			if err != nil {
				logrus.WithError(err).Error("failed to marshal BuildInfo")
			}
		} else {
			if !versionExtra {
				fmt.Println(mainVersion)
				return
			}
			fmt.Printf("%s %s\n", mainName, mainVersion)
			if buildInfo != nil {
				for _, dep := range buildInfo.Deps {
					switch dep.Path {
					case "github.com/PowerDNS/lightningstream":
						fmt.Printf("\tbased on Lightning Stream %s\n", dep.Version)
					case "github.com/PowerDNS/simpleblob":
						fmt.Printf("\tusing Simpleblob %s\n", dep.Version)
					}
				}
				fmt.Printf("\tbuilt with %s\n", buildInfo.GoVersion)
			} else {
				fmt.Printf("\tbuilt with %s\n", runtime.Version())
			}
		}
	},
}
