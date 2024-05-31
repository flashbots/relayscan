package service

/**
 * https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md
 */

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/services/bidstream"
	"github.com/flashbots/relayscan/vars"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	collectUltrasoundStream bool
	collectGetHeader        bool
	collectDataAPI          bool
	outFileCSV              string

	csvHeader = "local_timestamp\ttimestamp\tslot\tblock_number\tblock_hash\tparent_hash\tbuilder_pubkey\tfee_recipient\tvalue\n"
)

func init() {
	bidCollectCmd.Flags().BoolVar(&collectUltrasoundStream, "ultrasound-stream", true, "use ultrasound top-bid stream")
	bidCollectCmd.Flags().BoolVar(&collectGetHeader, "get-header", false, "use getHeader API")
	bidCollectCmd.Flags().BoolVar(&collectDataAPI, "data-api", false, "use data API")

	// for getHeader
	bidCollectCmd.Flags().StringVar(&beaconNodeURI, "beacon-uri", vars.DefaultBeaconURI, "beacon endpoint")

	// for saving to file
	bidCollectCmd.Flags().StringVar(&outFileCSV, "out", "", "output file for CSV")
}

var bidCollectCmd = &cobra.Command{
	Use:   "bidcollect",
	Short: "Collect bids",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var outF *os.File
		bidC := make(chan common.UltrasoundStreamBid, 100)

		log.WithField("version", vars.Version).Info("starting bidcollect ...")

		if outFileCSV != "" {
			log.Infof("writing to %s", outFileCSV)
			outF, err = os.OpenFile(outFileCSV, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				log.WithError(err).WithField("filename", outFileCSV).Fatal("failed to open output file")
			}

			fi, err := outF.Stat()
			if err != nil {
				log.WithError(err).Fatal("failed stat on output file")
			}

			if fi.Size() == 0 {
				_, err = fmt.Fprint(outF, csvHeader)
				if err != nil {
					log.WithError(err).Fatal("failed to write header to output file")
				}
			}
		}

		if collectGetHeader {
			log.Fatal("not yet implemented")
		}

		if collectDataAPI {
			log.Fatal("not yet implemented")
		}

		if collectUltrasoundStream {
			log.Info("using ultrasound stream")

			opts := bidstream.UltrasoundStreamOpts{
				Log:  log,
				BidC: bidC,
			}
			bidstream.StartUltrasoundStreamConnection(opts)
		}

		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

		log.Info("Waiting...")

		for {
			select {
			case bid := <-bidC:
				processBid(log, outF, bid)

			case <-done:
				log.Info("bye ...")
				return
			}
		}
	},
}

func processBid(log *logrus.Entry, outF *os.File, bid common.UltrasoundStreamBid) {
	blockHash := hexutil.Encode(bid.BlockHash[:])
	parentHash := hexutil.Encode(bid.ParentHash[:])
	builderPubkey := hexutil.Encode(bid.BuilderPubkey[:])
	feeRecipient := hexutil.Encode(bid.FeeRecipient[:])
	value := bid.Value.String()

	log.WithFields(logrus.Fields{
		"slot":       bid.Slot,
		"block_hash": blockHash,
		"value":      value,
	}).Info("received bid")

	if outF != nil {
		t := time.Now().UTC()
		_, err := fmt.Fprintf(outF, "%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\n", t.UnixMilli(), bid.Timestamp, bid.Slot, bid.BlockNumber, blockHash, parentHash, builderPubkey, feeRecipient, value)
		if err != nil {
			log.WithError(err).Error("couldn't write bid to file")
		}
	}
}
