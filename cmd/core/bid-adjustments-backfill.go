package core

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/flashbots/relayscan/vars"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var bidAdjustmentRelay string

func init() {
	bidAdjustmentsBackfillCmd.Flags().StringVar(&bidAdjustmentRelay, "relay", "relay.ultrasound.money", "relay to fetch bid adjustments from")
	bidAdjustmentsBackfillCmd.Flags().Int64Var(&minSlot, "min-slot", 0, "minimum slot (if unset, backfill until the merge, negative number for that number of slots before latest)")
}

var bidAdjustmentsBackfillCmd = &cobra.Command{
	Use:   "bid-adjustments-backfill",
	Short: "Backfill bid adjustments data",
	Run: func(cmd *cobra.Command, args []string) {
		db := database.MustConnectPostgres(log, vars.DefaultPostgresDSN)
		defer db.Close()

		relay, err := common.NewRelayEntry(bidAdjustmentRelay, false)
		if err != nil {
			log.WithError(err).Fatal("failed to create relay entry")
		}

		log.Infof("minSlot %d", minSlot)
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

		backfiller := newBidAdjustmentsBackfiller(db, relay, uint64(minSlot))
		err = backfiller.backfillAdjustments()
		if err != nil {
			log.WithError(err).Fatal("failed to backfill adjustments")
		}
	},
}

type bidAdjustmentsBackfiller struct {
	db      *database.DatabaseService
	relay   common.RelayEntry
	minSlot uint64
}

func newBidAdjustmentsBackfiller(db *database.DatabaseService, relay common.RelayEntry, minSlot uint64) *bidAdjustmentsBackfiller {
	return &bidAdjustmentsBackfiller{
		db:      db,
		relay:   relay,
		minSlot: minSlot,
	}
}

func (bf *bidAdjustmentsBackfiller) backfillAdjustments() error {
	_log := log.WithField("relay", bf.relay.Hostname())
	_log.Info("Backfilling adjustments...")

	baseURL := bf.relay.GetURI("/ultrasound/v1/data/adjustments")
	latestSlot, err := bf.db.GetLatestAdjustmentSlot()
	if err != nil {
		return fmt.Errorf("failed to get latest adjustment slot: %w", err)
	}

	if bf.minSlot < latestSlot {
		bf.minSlot = latestSlot
	}

	// Hardcoded ultrasoiund first slot with data see https://github.com/ultrasoundmoney/docs/blob/main/bid_adjustment.md#data-api
	const ultrasoundFirstBidAdjustmentSlot = 7869470
	if bf.minSlot < ultrasoundFirstBidAdjustmentSlot {
		bf.minSlot = ultrasoundFirstBidAdjustmentSlot
	}

	const ultrasoundEndStatusCode = 403
	for slot := bf.minSlot; ; slot++ {
		_log.WithField("slot", slot).Info("Fetching adjustments...")
		url := fmt.Sprintf("%s?slot=%d", baseURL, slot)

		var response common.UltrasoundAdjustmentResponse
		statusCode, err := common.SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, url, nil, &response)
		_log.WithField("status code", statusCode).Info("Response")
		if statusCode == ultrasoundEndStatusCode {
			_log.WithField("Status Code", statusCode).Info("Stopping backfill due to 403")
			break
		}
		if err != nil {
			_log.WithError(err).Error("Failed to fetch adjustments")
			return nil
		}

		if len(response.Data) > 0 {
			adjustments := make([]*database.AdjustmentEntry, len(response.Data))
			for i, adjustment := range response.Data {
				submittedReceivedAt, err := time.Parse(time.RFC3339, adjustment.SubmittedReceivedAt)
				if err != nil {
					_log.WithError(err).Error("Failed to parse SubmittedReceivedAt")
					continue
				}
				adjustments[i] = &database.AdjustmentEntry{
					Slot:                slot,
					AdjustedBlockHash:   adjustment.AdjustedBlockHash,
					AdjustedValue:       adjustment.AdjustedValue,
					BlockNumber:         adjustment.BlockNumber,
					BuilderPubkey:       adjustment.BuilderPubkey,
					Delta:               adjustment.Delta,
					SubmittedBlockHash:  adjustment.SubmittedBlockHash,
					SubmittedReceivedAt: submittedReceivedAt,
					SubmittedValue:      adjustment.SubmittedValue,
				}
			}

			err = bf.db.SaveAdjustments(adjustments)
			if err != nil {
				_log.WithError(err).Error("Failed to save adjustments")
			} else {
				for _, entry := range adjustments {
					_log.WithFields(logrus.Fields{
						"Slot":           entry.Slot,
						"SubmittedValue": entry.SubmittedValue,
						"AdjustedValue":  entry.AdjustedValue,
						"Delta":          entry.Delta,
					}).Info("Adjustment data")
				}
				_log.WithField("count", len(adjustments)).Info("Saved adjustments")
			}
		} else {
			_log.Info("No adjustments found for this slot")
			// break
		}

		time.Sleep(time.Duration(50) * time.Microsecond) // Rate limiting
	}

	return nil
}
