package service

/**
 * https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md
 */

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/services/bidstream"
	"github.com/flashbots/relayscan/vars"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var bidStreamCmd = &cobra.Command{
	Use:   "bidstream",
	Short: "Stream bids",
	Run: func(cmd *cobra.Command, args []string) {
		log.WithField("version", vars.Version).Info("starting bidstream ...")
		bidC := make(chan common.UltrasoundStreamBid, 100)
		opts := bidstream.UltrasoundStreamOpts{
			Log:  log,
			BidC: bidC,
		}
		bidstream.StartUltrasoundStreamConnection(opts)

		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

		log.Info("Waiting...")

		for {
			select {
			case bid := <-bidC:
				log.WithFields(logrus.Fields{
					"slot":       bid.Slot,
					"block_hash": hexutil.Encode(bid.BlockHash[:]),
				}).Info("received bid")
			case <-done:
				log.Info("bye ...")
				return
			}
		}
	},
}
