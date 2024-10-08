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

var (
	bidAdjustmentMinSlot int64
	bidAdjustmentRelay   string
)

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

		backfiller := newBidAdjustmentsBackfiller(db, relay, uint64(bidAdjustmentMinSlot))
		err = backfiller.backfillAdjustments()
		if err != nil {
			log.WithError(err).Fatal("failed to backfill adjustments")
		}
	},
}

func init() {
	bidAdjustmentsBackfillCmd.Flags().StringVar(&bidAdjustmentRelay, "relay", "relay.ultrasound.money", "relay to fetch bid adjustments from")
	bidAdjustmentsBackfillCmd.Flags().Int64Var(&bidAdjustmentMinSlot, "min-slot", 0, "minimum slot (if unset, backfill until the merge, negative number for that number of slots before latest)")
	CoreCmd.AddCommand(bidAdjustmentsBackfillCmd)
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

	for slot := bf.minSlot; ; slot++ {
		_log.WithField("slot", slot).Info("Fetching adjustments...")
		url := fmt.Sprintf("%s?slot=%d", baseURL, slot)

		var response common.UltrasoundAdjustmentResponse
		_, err := common.SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, url, nil, &response)
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
					Slot:                 slot,
					AdjustedBlockHash:    adjustment.AdjustedBlockHash,
					AdjustedValue:        adjustment.AdjustedValue,
					BlockNumber:          adjustment.BlockNumber,
					BuilderPubkey:        adjustment.BuilderPubkey,
					Delta:                adjustment.Delta,
					SubmittedBlockHash:   adjustment.SubmittedBlockHash,
					SubmittedReceivedAt:  submittedReceivedAt,
					SubmittedValue:       adjustment.SubmittedValue,
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
			break
		}

		time.Sleep(time.Second) // Rate limiting
	}

	return nil
}
