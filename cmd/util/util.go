// Package util contains code for the util subcommands
package util

import (
	"github.com/flashbots/relayscan/common"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	log               = common.Logger
	numThreads uint64 = 10

	// Printer for pretty printing numbers
	printer = message.NewPrinter(language.English)

	ethNodeURI       string
	ethNodeBackupURI string
	slotStr          string
	blockHash        string
	mevGethURI       string
	loadAddresses    bool
	scLookup         bool // whether to lookup smart contract details
	printAllSimTx    bool
)

var UtilCmd = &cobra.Command{
	Use:   "util",
	Short: "util subcommand",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	UtilCmd.AddCommand(backfillExtradataCmd)
}
