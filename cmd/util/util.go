// Package util contains code for the util subcommands
package util

import (
	"github.com/flashbots/relayscan/common"
	"github.com/spf13/cobra"
)

var (
	log               = common.Logger
	numThreads uint64 = 10
)

var UtilCmd = &cobra.Command{
	Use:   "util",
	Short: "util subcommand",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	UtilCmd.AddCommand(inspectBlockCmd)
	UtilCmd.AddCommand(backfillExtradataCmd)
}
