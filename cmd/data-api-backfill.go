package cmd

import (
	"github.com/metachris/relayscan/common"
	"github.com/spf13/cobra"
)

var ()

func init() {
	rootCmd.AddCommand(apiCmd)
	// apiCmd.Flags().BoolVar(&logJSON, "json", defaultLogJSON, "log in JSON format instead of text")
	// apiCmd.Flags().StringVar(&logLevel, "loglevel", defaultLogLevel, "log-level: trace, debug, info, warn/warning, error, fatal, panic")
	// apiCmd.Flags().StringVar(&apiLogTag, "log-tag", apiDefaultLogTag, "if set, a 'tag' field will be added to all log entries")
	// apiCmd.Flags().BoolVar(&apiLogVersion, "log-version", apiDefaultLogVersion, "if set, a 'version' field will be added to all log entries")
	// apiCmd.Flags().BoolVar(&apiDebug, "debug", false, "debug logging")

	// apiCmd.Flags().StringVar(&apiListenAddr, "listen-addr", apiDefaultListenAddr, "listen address for webserver")
	// apiCmd.Flags().StringSliceVar(&beaconNodeURIs, "beacon-uris", defaultBeaconURIs, "beacon endpoints")
	// apiCmd.Flags().StringVar(&redisURI, "redis-uri", defaultRedisURI, "redis uri")
	// apiCmd.Flags().StringVar(&postgresDSN, "db", defaultPostgresDSN, "PostgreSQL DSN")
	// apiCmd.Flags().StringVar(&apiSecretKey, "secret-key", apiDefaultSecretKey, "secret key for signing bids")
	// apiCmd.Flags().StringVar(&apiBlockSimURL, "blocksim", apiDefaultBlockSim, "URL for block simulator")
	// apiCmd.Flags().StringVar(&network, "network", defaultNetwork, "Which network to use")

	// apiCmd.Flags().BoolVar(&apiPprofEnabled, "pprof", apiDefaultPprofEnabled, "enable pprof API")
	// apiCmd.Flags().BoolVar(&apiInternalAPI, "internal-api", apiDefaultInternalAPIEnabled, "enable internal API (/internal/...)")
}

var apiCmd = &cobra.Command{
	Use:   "data-api-backfill",
	Short: "Backfill all relays data API",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("hello")
		relays, err := common.GetRelays()
		if err != nil {
			log.WithError(err).Fatal("failed to get relays")
		}

		log.Infof("Using %d relays", len(relays))
		for index, relay := range relays {
			log.Infof("relay #%d: %s", index+1, relay.Hostname())
		}

		for _, relay := range relays {
			backfillRelayPayloadsDelivered(relay)
			return
		}
	},
}

func backfillRelayPayloadsDelivered(relay common.RelayEntry) error {
	log.Info("backfilling relay: ", relay.Hostname())
	baseURL := relay.GetURI("/relay/v1/data/bidtraces/proposer_payload_delivered")
	for {

	}
	return nil
}
