package main

import (
	"github.com/flashbots/relayscan/cmd"
	"github.com/flashbots/relayscan/common"
)

var Version = "dev" // is set during build process

func main() {
	common.Version = Version
	cmd.Execute()
}
