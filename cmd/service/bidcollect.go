package service

/**
 * https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md
 */

import (
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/services/bidcollect"
	"github.com/flashbots/relayscan/vars"
	"github.com/spf13/cobra"
)

var (
	collectUltrasoundStream bool
	collectGetHeader        bool
	collectDataAPI          bool
	outFileCSV              string
)

func init() {
	bidCollectCmd.Flags().BoolVar(&collectUltrasoundStream, "ultrasound-stream", false, "use ultrasound top-bid stream")
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
		// Prepare relays
		relay, err := common.NewRelayEntry(vars.RelayUltrasound, false)
		relays := []common.RelayEntry{relay}
		// relays, err = common.GetRelays()
		if err != nil {
			log.WithError(err).Fatal("failed to get relays")
		}

		log.Infof("Using %d relays", len(relays))
		for index, relay := range relays {
			log.Infof("- relay #%d: %s", index+1, relay.Hostname())
		}

		opts := bidcollect.BidCollectorOpts{
			Log:                     log,
			Relays:                  relays,
			CollectUltrasoundStream: collectUltrasoundStream,
			CollectGetHeader:        collectGetHeader,
			CollectDataAPI:          collectDataAPI,
			BeaconNodeURI:           beaconNodeURI,
			OutFile:                 outFileCSV,
		}

		bidCollector := bidcollect.NewBidCollector(&opts)
		bidCollector.MustStart()
	},
}
