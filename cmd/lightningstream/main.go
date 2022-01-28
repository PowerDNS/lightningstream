package main

import (
	"powerdns.com/platform/lightningstream/cmd/lightningstream/commands"

	// Register storage backends
	_ "powerdns.com/platform/lightningstream/storage/memory"
)

// version is overridden during the build with the go linker
var version = "dev"

func main() {
	commands.SetVersion(version)
	commands.Execute()
}
