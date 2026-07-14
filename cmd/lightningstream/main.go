package main

import (
	// Expose pprof in the webserver
	_ "net/http/pprof"

	"github.com/PowerDNS/lightningstream/cmd/lightningstream/commands"

	// Register storage backends
	_ "github.com/PowerDNS/simpleblob/backends/azure"
	_ "github.com/PowerDNS/simpleblob/backends/fs"
	_ "github.com/PowerDNS/simpleblob/backends/memory"
	_ "github.com/PowerDNS/simpleblob/backends/s3"
)

func main() {
	commands.SetBuildInfo("Lightning Stream")
	commands.Execute()
}
