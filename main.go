package main

import (
	"github.com/flashbots/relayscan/cmd"
	"github.com/flashbots/relayscan/vars"
)

var Version = "dev" // is set during build process

func main() {
	vars.Version = Version
	cmd.Execute()
}
