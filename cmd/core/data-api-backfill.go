package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/spf13/cobra"
)

var (
	cliRelay   string
	initCursor uint64
	minSlot    uint64
	// bidsOnly   bool
)

func init() {
	backfillDataAPICmd.Flags().StringVar(&cliRelay, "relay", "", "specific relay only")
	backfillDataAPICmd.Flags().Uint64Var(&initCursor, "cursor", 0, "initial cursor")
	backfillDataAPICmd.Flags().Uint64Var(&minSlot, "min-slot", 0, "minimum slot (if unset, backfill until the merge)")
	// backfillDataAPICmd.Flags().BoolVar(&bidsOnly, "bids", false, "only bids")
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
		db := database.MustConnectPostgres(log, common.DefaultPostgresDSN)

		for _, relay := range relays {
			backfiller := newBackfiller(db, relay, initCursor, minSlot)
			// backfiller.backfillDataAPIBids()
			err = backfiller.backfillPayloadsDelivered()
			if err != nil {
				log.WithError(err).WithField("relay", relay).Error("backfill failed")
			}
		}
	},
}

type backfiller struct {
	relay      common.RelayEntry
	db         *database.DatabaseService
	cursorSlot uint64
	minSlot    uint64
}

func newBackfiller(db *database.DatabaseService, relay common.RelayEntry, cursorSlot, minSlot uint64) *backfiller {
	return &backfiller{
		relay:      relay,
		db:         db,
		cursorSlot: cursorSlot,
		minSlot:    minSlot,
	}
}

func (bf *backfiller) backfillPayloadsDelivered() error {
	log.Infof("backfilling payloads data-api for relay %s ...", bf.relay.Hostname())

	// 1. get latest entry from DB
	latestEntry, err := bf.db.GetDataAPILatestPayloadDelivered(bf.relay.Hostname())
	latestSlotInDB := uint64(0)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.WithError(err).Fatal("failed to get latest entry")
		return err
	} else {
		latestSlotInDB = latestEntry.Slot
	}
	log.Infof("last payload in db at slot: %d", latestSlotInDB)

	// 2. backfill until latest DB entry is reached
	baseURL := bf.relay.GetURI("/relay/v1/data/bidtraces/proposer_payload_delivered")
	cursorSlot := bf.cursorSlot
	slotsReceived := make(map[uint64]bool)
	builders := make(map[string]bool)

	for {
		payloadsNew := 0
		url := baseURL
		if cursorSlot > 0 {
			url = fmt.Sprintf("%s?cursor=%d", baseURL, cursorSlot)
		}
		log.Info("url: ", url)
		var data []relaycommon.BidTraceV2JSON
		_, err = common.SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, url, nil, &data)
		if err != nil {
			return err
		}

		log.Infof("got %d entries", len(data))
		entries := make([]*database.DataAPIPayloadDeliveredEntry, len(data))

		for index, dataEntry := range data {
			log.Debugf("saving entry for slot %d", dataEntry.Slot)
			dbEntry := database.BidTraceV2JSONToPayloadDeliveredEntry(bf.relay.Hostname(), dataEntry)
			entries[index] = &dbEntry

			if !slotsReceived[dataEntry.Slot] {
				slotsReceived[dataEntry.Slot] = true
				payloadsNew += 1
			}

			if cursorSlot == 0 {
				log.Infof("latest received payload at slot %d", dataEntry.Slot)
				cursorSlot = dataEntry.Slot
			} else if cursorSlot > dataEntry.Slot {
				cursorSlot = dataEntry.Slot
			}

			builders[dataEntry.BuilderPubkey] = true
		}

		err := bf.db.SaveDataAPIPayloadDeliveredBatch(entries)
		if err != nil {
			log.WithError(err).Fatal("failed to save entries")
			return err
		}

		// save builders
		for builderPubkey := range builders {
			err = bf.db.SaveBuilder(&database.BlockBuilderEntry{BuilderPubkey: builderPubkey})
			if err != nil {
				log.WithError(err).Error("failed to save builder")
			}
		}

		if payloadsNew == 0 {
			log.Infof("No new payloads, all done. Earliest payload for slot: %d", cursorSlot)
			return nil
		}

		if cursorSlot < latestSlotInDB {
			log.Infof("Payloads backfilled until last in DB - at slot %d", latestSlotInDB)
			return nil
		}

		if cursorSlot < bf.minSlot {
			log.Infof("Payloads backfilled until min slot %d", bf.minSlot)
			return nil
		}
		// time.Sleep(1 * time.Second)
	}
}

// func (bf *backfiller) backfillDataAPIBids() error {
// 	log.Infof("backfilling bids from relay %s ...", bf.relay.Hostname())

// 	// 1. get latest entry from DB
// 	latestEntry, err := bf.db.GetDataAPILatestBid(bf.relay.Hostname())
// 	latestSlotInDB := uint64(0)
// 	if err != nil && !errors.Is(err, sql.ErrNoRows) {
// 		log.WithError(err).Fatal("failed to get latest entry")
// 		return err
// 	} else {
// 		latestSlotInDB = latestEntry.Slot
// 	}
// 	log.Infof("last known slot: %d", latestSlotInDB)

// 	// 2. backfill until latest DB entry is reached
// 	baseURL := bf.relay.GetURI("/relay/v1/data/bidtraces/builder_blocks_received")
// 	cursorSlot := bf.cursorSlot
// 	slotsReceived := make(map[uint64]bool)

// 	for {
// 		entriesNew := 0
// 		url := baseURL
// 		if cursorSlot > 0 {
// 			url = fmt.Sprintf("%s?slot=%d", baseURL, cursorSlot)
// 		}
// 		log.Info("url: ", url)
// 		var data []relaycommon.BidTraceV2WithTimestampJSON
// 		common.SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, url, nil, &data)

// 		log.Infof("got %d entries", len(data))
// 		entries := make([]*database.DataAPIBuilderBidEntry, len(data))

// 		for index, dataEntry := range data {
// 			log.Debugf("saving entry for slot %d", dataEntry.Slot)
// 			dbEntry := database.BidTraceV2WithTimestampJSONToBuilderBidEntry(bf.relay.Hostname(), dataEntry)
// 			entries[index] = &dbEntry

// 			if !slotsReceived[dataEntry.Slot] {
// 				slotsReceived[dataEntry.Slot] = true
// 				entriesNew += 1
// 			}

// 			if cursorSlot == 0 {
// 				cursorSlot = dataEntry.Slot
// 			}
// 		}

// 		err := bf.db.SaveDataAPIBids(entries)
// 		if err != nil {
// 			log.WithError(err).Fatal("failed to save bids")
// 			return err
// 		}

// 		if entriesNew == 0 {
// 			log.Info("No new bids, all done")
// 			return nil
// 		}

// 		if cursorSlot < latestSlotInDB {
// 			log.Infof("Bids backfilled until last in DB (%d)", latestSlotInDB)
// 			return nil
// 		}
// 		cursorSlot -= 1
// 		// time.Sleep(1 * time.Second)
// 	}
// }
