// Package cmd contains the cobra command line setup
package cmd

import (
	"fmt"
	"os"

	"github.com/flashbots/relayscan/cmd/core"
	"github.com/flashbots/relayscan/cmd/service"
	"github.com/flashbots/relayscan/cmd/util"
	"github.com/flashbots/relayscan/common"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Short: "relayscan",
	Long:  `https://github.com/flashbots/relayscan`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("relayscan %s\n", common.Version)
		_ = cmd.Help()
	},
}

func Execute() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(core.CoreCmd)
	rootCmd.AddCommand(util.UtilCmd)
	rootCmd.AddCommand(service.ServiceCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
