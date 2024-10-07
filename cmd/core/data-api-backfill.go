package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/flashbots/relayscan/vars"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cliRelay   string
	minSlot    int64
	initCursor uint64
	pageLimit  = 100 // 100 is max on bloxroute
)

func init() {
	backfillDataAPICmd.Flags().StringVar(&cliRelay, "relay", "", "specific relay only")
	backfillDataAPICmd.Flags().Uint64Var(&initCursor, "cursor", 0, "initial cursor")
	backfillDataAPICmd.Flags().Int64Var(&minSlot, "min-slot", 0, "minimum slot (if unset, backfill until the merge, negative number for that number of slots before latest)")
}

var backfillDataAPICmd = &cobra.Command{
	Use:   "data-api-backfill",
	Short: "Backfill all relays data API",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var relays []common.RelayEntry
		startTime := time.Now().UTC()

		if cliRelay != "" {
			var relayEntry common.RelayEntry
			if cliRelay == "fb" {
				relayEntry, err = common.NewRelayEntry(vars.RelayURLs[0], false)
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

		log.Infof("Relayscan %s", vars.Version)
		log.Infof("Using %d relays", len(relays))
		for index, relay := range relays {
			log.Infof("- relay #%d: %s", index+1, relay.Hostname())
		}

		// Connect to Postgres
		db := database.MustConnectPostgres(log, vars.DefaultPostgresDSN)

		// If needed, get latest slot (i.e. if min-slot is negative)
		if minSlot < 0 {
			log.Infof("Getting latest slot from beaconcha.in for offset %d", minSlot)
			latestSlotOnBeaconChain := common.MustGetLatestSlot()
			log.Infof("Latest slot from beaconcha.in: %d", latestSlotOnBeaconChain)
			minSlot = int64(latestSlotOnBeaconChain) + minSlot
		}

		if minSlot != 0 {
			log.Infof("Using min slot: %d", minSlot)
		}

		for _, relay := range relays {
			log.Infof("Starting backfilling for relay %s ...", relay.Hostname())
			backfiller := newBackfiller(db, relay, initCursor, uint64(minSlot))
			err = backfiller.backfillPayloadsDelivered()
			if err != nil {
				log.WithError(err).WithField("relay", relay).Error("backfill payloads failed")
			}
			
			// Add this new call
			err = backfiller.backfillAdjustments()
			if err != nil {
				log.WithError(err).WithField("relay", relay).Error("backfill adjustments failed")
			}
		}

		timeNeeded := time.Since(startTime)
		log.WithField("timeNeeded", timeNeeded).Info("All done!")
	},
}

type backfiller struct {
	relay      common.RelayEntry
	db         *database.DatabaseService
	cursorSlot uint64
	minSlot    uint64
	withAdjustments bool
}

func newBackfiller(db *database.DatabaseService, relay common.RelayEntry, cursorSlot, minSlot uint64, withAdjustments bool) *backfiller {
	return &backfiller{
		relay:      relay,
		db:         db,
		cursorSlot: cursorSlot,
		minSlot:    minSlot,
		withAdjustments: withAdjustments,
	}
}

func (bf *backfiller) backfillPayloadsDelivered() error {
	_log := log.WithField("relay", bf.relay.Hostname())
	// _log.Info("backfilling payloads from relay data-api ...")

	// 1. get latest entry from DB
	latestEntry, err := bf.db.GetDataAPILatestPayloadDelivered(bf.relay.Hostname())
	latestSlotInDB := uint64(0)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		_log.WithError(err).Fatal("failed to get latest entry")
		return err
	} else {
		latestSlotInDB = latestEntry.Slot
	}
	_log.Infof("Latest payload in DB for slot: %d", latestSlotInDB)

	// 2. backfill until latest DB entry is reached
	baseURL := bf.relay.GetURI("/relay/v1/data/bidtraces/proposer_payload_delivered")
	if bf.withAdjustments {
		baseURL += "?adjustments=1"
	}
	cursorSlot := bf.cursorSlot
	slotsReceived := make(map[uint64]bool)
	builders := make(map[string]bool)

	for {
		payloadsNew := 0
		url := fmt.Sprintf("%s&limit=%d", baseURL, pageLimit)
		if cursorSlot > 0 {
			url = fmt.Sprintf("%s&cursor=%d", url, cursorSlot)
		}
		_log.WithField("url: ", url).Info("Fetching payloads...")
		var data []relaycommon.BidTraceV2JSON
		_, err = common.SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, url, nil, &data)
		if err != nil {
			return err
		}

		_log.Infof("Response contains %d delivered payloads", len(data))

		// build a list of entries for batch DB update
		entries := make([]*database.DataAPIPayloadDeliveredEntry, len(data))
		slotFirst := uint64(0)
		slotLast := uint64(0)
		for index, payload := range data {
			_log.Debugf("saving entry for slot %d", payload.Slot)
			dbEntry := database.BidTraceV2JSONToPayloadDeliveredEntry(bf.relay.Hostname(), payload)
			
			if bf.withAdjustments && payload.AdjustmentData != nil {
				dbEntry.AdjustmentData = &database.AdjustmentData{
					StateRoot:           payload.AdjustmentData.StateRoot,
					TransactionsRoot:    payload.AdjustmentData.TransactionsRoot,
					ReceiptsRoot:        payload.AdjustmentData.ReceiptsRoot,
					BuilderAddress:      payload.AdjustmentData.BuilderAddress,
					FeeRecipientAddress: payload.AdjustmentData.FeeRecipientAddress,
					FeePayerAddress:     payload.AdjustmentData.FeePayerAddress,
				}
			}
			
			entries[index] = &dbEntry

			// Set first and last slot
			if slotFirst == 0 || payload.Slot < slotFirst {
				slotFirst = payload.Slot
			}
			if slotLast == 0 || payload.Slot > slotLast {
				slotLast = payload.Slot
			}

			// Count number of slots with payloads
			if !slotsReceived[payload.Slot] {
				slotsReceived[payload.Slot] = true
				payloadsNew += 1
			}

			// Set cursor for next request
			if cursorSlot == 0 || cursorSlot > payload.Slot {
				cursorSlot = payload.Slot
			}

			// Remember the builder
			builders[payload.BuilderPubkey] = true
		}

		// Save entries
		newEntries, err := bf.db.SaveDataAPIPayloadDeliveredBatch(entries)
		if err != nil {
			_log.WithError(err).Fatal("failed to save entries")
			return err
		}

		_log.WithFields(logrus.Fields{
			"newEntries": newEntries,
			"slotFirst":  slotFirst,
			"slotLast":   slotLast,
		}).Info("Batch of payloads saved to database")

		// Save builders
		for builderPubkey := range builders {
			err = bf.db.SaveBuilder(&database.BlockBuilderEntry{BuilderPubkey: builderPubkey})
			if err != nil {
				_log.WithError(err).Error("failed to save builder")
			}
		}

		// Stop as soon as no new payloads are received
		if payloadsNew == 0 {
			_log.Infof("No new payloads, all done. Earliest payload for slot: %d", cursorSlot)
			return nil
		}

		// Stop if at the latest slot in DB
		if cursorSlot < latestSlotInDB {
			_log.Infof("Payloads backfilled until latest slot in DB: %d", latestSlotInDB)
			return nil
		}

		// Stop if at min slot
		if cursorSlot < bf.minSlot {
			_log.Infof("Payloads backfilled until min slot: %d", bf.minSlot)
			return nil
		}
	}
}
