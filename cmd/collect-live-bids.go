package cmd

import (
	"net/url"

	"github.com/metachris/relayscan/common"
	"github.com/metachris/relayscan/database"
	"github.com/metachris/relayscan/services/collector"
	"github.com/spf13/cobra"
)

var (
	beaconNodeURI string
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
		dbURL, err := url.Parse(defaultPostgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("couldn't read db URL")
		}
		log.Infof("Connecting to Postgres database at %s%s ...", dbURL.Host, dbURL.Path)
		db, err := database.NewDatabaseService(defaultPostgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("Failed to connect to Postgres database at %s%s", dbURL.Host, dbURL.Path)
		}

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
