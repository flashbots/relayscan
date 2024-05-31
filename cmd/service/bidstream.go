package service

/**
 * https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md
 */

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/flashbots/relayscan/services/bidstream"
	"github.com/flashbots/relayscan/vars"
	"github.com/spf13/cobra"
)

var bidStreamCmd = &cobra.Command{
	Use:   "bidstream",
	Short: "Stream bids",
	Run: func(cmd *cobra.Command, args []string) {
		log.WithField("version", vars.Version).Info("starting bidstream ...")
		opts := bidstream.UltrasoundStreamOpts{
			Log: log,
		}
		bidstream.StartUltrasoundStreamConnection(opts)

		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
		<-done
		log.Info("bye!")
	},
}
