package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/metachris/relayscan/common"
	"github.com/metachris/relayscan/database"
	"github.com/spf13/cobra"
)

var (
	cliRelay   string
	initCursor uint64
)

func init() {
	rootCmd.AddCommand(backfillDataAPICmd)
	backfillDataAPICmd.Flags().StringVar(&cliRelay, "relay", "", "specific relay only")
	backfillDataAPICmd.Flags().Uint64Var(&initCursor, "cursor", 0, "initial cursor")
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

var backfillDataAPICmd = &cobra.Command{
	Use:   "data-api-backfill",
	Short: "Backfill all relays data API",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var relays []common.RelayEntry

		if cliRelay != "" {
			var relayEntry common.RelayEntry
			if cliRelay == "fb" {
				relayEntry, err = common.NewRelayEntry(common.RelayURLs[0], false)
			} else {
				relayEntry, err = common.NewRelayEntry(cliRelay, false)
			}
			if err != nil {
				log.WithField("relay", cliRelay).WithError(err).Fatal("failed to decode relay")
			}
			relays = []common.RelayEntry{relayEntry}
		} else {
			relays, err = common.GetRelays()
			if err != nil {
				log.WithError(err).Fatal("failed to get relays")
			}
		}

		log.Infof("Using %d relays", len(relays))
		for index, relay := range relays {
			log.Infof("relay #%d: %s", index+1, relay.Hostname())
		}

		// Connect to Postgres
		dbURL, err := url.Parse(postgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("couldn't read db URL")
		}
		log.Infof("Connecting to Postgres database at %s%s ...", dbURL.Host, dbURL.Path)
		db, err := database.NewDatabaseService(postgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("Failed to connect to Postgres database at %s%s", dbURL.Host, dbURL.Path)
		}

		for _, relay := range relays {
			backfiller := newBackfiller(db, relay, initCursor)
			backfiller.backfillPayloadsDelivered()
		}
	},
}

type backfiller struct {
	relay      common.RelayEntry
	db         database.IDatabaseService
	cursorSlot uint64
}

func newBackfiller(db database.IDatabaseService, relay common.RelayEntry, cursorSlot uint64) *backfiller {
	return &backfiller{
		relay:      relay,
		db:         db,
		cursorSlot: cursorSlot,
	}
}

func (bf *backfiller) backfillPayloadsDelivered() error {
	log.Infof("backfilling relay %s ...", bf.relay.Hostname())

	// 1. get latest entry from DB
	latestEntry, err := bf.db.GetDataAPILatestPayloadDelivered(bf.relay.Hostname())
	latestSlotInDB := uint64(0)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.WithError(err).Fatal("failed to get latest entry")
		return err
	} else {
		latestSlotInDB = latestEntry.Slot
	}
	log.Infof("last known slot: %d", latestSlotInDB)

	// 2. backfill until latest DB entry is reached
	baseURL := bf.relay.GetURI("/relay/v1/data/bidtraces/proposer_payload_delivered")
	cursorSlot := bf.cursorSlot
	slotsReceived := make(map[uint64]bool)

	for {
		payloadsNew := 0
		url := baseURL
		if cursorSlot > 0 {
			url = fmt.Sprintf("%s?cursor=%d", baseURL, cursorSlot)
		}
		log.Info("url: ", url)
		var data []relaycommon.BidTraceV2JSON
		common.SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, url, nil, &data)

		log.Infof("got %d entries", len(data))
		entries := make([]*database.PayloadDeliveredEntry, len(data))

		for index, dataEntry := range data {
			log.Debugf("saving entry for slot %d", dataEntry.Slot)
			dbEntry := database.BidTraceV2JSONToPayloadDeliveredEntry(bf.relay.Hostname(), dataEntry)
			entries[index] = &dbEntry

			if !slotsReceived[dataEntry.Slot] {
				slotsReceived[dataEntry.Slot] = true
				payloadsNew += 1
			}

			if cursorSlot == 0 || cursorSlot > dataEntry.Slot {
				cursorSlot = dataEntry.Slot
			}
		}

		err := bf.db.SaveDataAPIPayloadDeliveredBatch(entries)
		if err != nil {
			log.WithError(err).Fatal("failed to save entries")
			return err
		}

		if payloadsNew == 0 {
			log.Info("No new payloads, all done")
			return nil
		}

		if cursorSlot < latestSlotInDB {
			log.Infof("Payloads backfilled until last in DB (%d)", latestSlotInDB)
			return nil
		}
		// time.Sleep(1 * time.Second)
	}
}
