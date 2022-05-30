package main

import (
	"powerdns.com/platform/lightningstream/cmd/lightningstream/commands"

	// Register storage backends
	_ "github.com/PowerDNS/simpleblob/backends/fs"
	_ "github.com/PowerDNS/simpleblob/backends/memory"
	_ "github.com/PowerDNS/simpleblob/backends/s3"
)

// version is overridden during the build with the go linker
var version = "dev"

func main() {
	commands.SetVersion(version)
	commands.Execute()
}
