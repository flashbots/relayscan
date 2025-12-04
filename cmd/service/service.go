// Package service contains code for the service subcommands
package service

import (
	"github.com/flashbots/relayscan/common"
	"github.com/spf13/cobra"
)

var (
	log           = common.Logger
	beaconNodeURI string
)

var ServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "service subcommand",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	ServiceCmd.AddCommand(websiteCmd)
	ServiceCmd.AddCommand(bidCollectCmd)
	ServiceCmd.AddCommand(backfillRunnerCmd)
}
