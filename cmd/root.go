// Package cmd contains the cobra command line setup
package cmd

import (
	"fmt"
	"os"

	"github.com/flashbots/relayscan/cmd/core"
	"github.com/flashbots/relayscan/cmd/service"
	"github.com/flashbots/relayscan/cmd/util"
	"github.com/flashbots/relayscan/vars"
	"github.com/spf13/cobra"
)

var configFile string

var rootCmd = &cobra.Command{
	Short: "relayscan",
	Long:  `https://github.com/flashbots/relayscan`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if configFile == "" {
			configFile = "config-mainnet.yaml"
		}
		return vars.LoadConfig(configFile)
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("relayscan %s\n", vars.Version)
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", os.Getenv("CONFIG_FILE"), "path to config file (default: config-mainnet.yaml)")
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
