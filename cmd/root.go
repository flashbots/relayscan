// Package cmd contains the cobra command line setup
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "relay",
	Short: "relayscan",
	Long:  `https://github.com/metachris/relayscan`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("relayscan %s\n", Version)
		_ = cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
