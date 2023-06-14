// Package core contains code for the core subcommands
package core

import (
	"github.com/flashbots/relayscan/common"
	"github.com/spf13/cobra"
)

var (
	log               = common.Logger
	check             = common.Check
	numThreads uint64 = 10
	slot       uint64
)

var CoreCmd = &cobra.Command{
	Use:   "core",
	Short: "core subcommand",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	CoreCmd.AddCommand(checkPayloadValueCmd)
	CoreCmd.AddCommand(backfillDataAPICmd)
	CoreCmd.AddCommand(updateBuilderStatsCmd)
}
