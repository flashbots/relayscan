package cmd

import (
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/services/collector"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(liveBidsCmd)
	liveBidsCmd.Flags().StringVar(&beaconNodeURI, "beacon-uri", defaultBeaconURI, "beacon endpoint")
}

var liveBidsCmd = &cobra.Command{
	Use:   "collect-live-bids",
	Short: "On every slot, ask for live bids",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to Postgres
		db := mustConnectPostgres(defaultPostgresDSN)

		// _relay, err := common.RelayURLToEntry(common.RelayURLs[0])
		// if err != nil {
		// 	log.WithError(err).Fatal("failed to get relays")
		// }
		// relays := []common.RelayEntry{_relay}
		relays, err := common.GetRelays()
		if err != nil {
			log.WithError(err).Fatal("failed to get relays")
		}

		log.Infof("Using %d relays", len(relays))
		for index, relay := range relays {
			log.Infof("relay #%d: %s", index+1, relay.Hostname())
		}

		relayCollector := collector.NewRelayCollector(log, relays, beaconNodeURI, db)
		relayCollector.Start()
	},
}
