package main

import (
	"github.com/metachris/relayscan/cmd"
)

var Version = "dev" // is set during build process

func main() {
	cmd.Version = Version
	cmd.Execute()
}
