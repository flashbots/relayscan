// Package collector collects data from the relays
package collector

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost-relay/beaconclient"
	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/sirupsen/logrus"
)

type RelayCollector struct {
	log    *logrus.Entry
	relays []common.RelayEntry
	bn     *beaconclient.ProdBeaconInstance
	db     *database.DatabaseService
}

func NewRelayCollector(log *logrus.Entry, relays []common.RelayEntry, beaconURL string, db *database.DatabaseService) *RelayCollector {
	srv := &RelayCollector{
		log:    log,
		relays: relays,
		db:     db,
		bn:     beaconclient.NewProdBeaconInstance(log, beaconURL),
	}

	return srv
}

func (s *RelayCollector) Start() {
	s.log.Info("Starting relay collector service")

	//  Check beacon-node sync status, process current slot and start slot updates
	syncStatus, err := s.bn.SyncStatus()
	if err != nil {
		s.log.WithError(err).Fatal("couldn't get BN sync status")
	} else if syncStatus.IsSyncing {
		s.log.Fatal("beacon node is syncing")
	}

	var latestSlot uint64
	var latestEpoch uint64
	var duties map[uint64]string

	// subscribe to head events
	c := make(chan beaconclient.HeadEventData)
	go s.bn.SubscribeToHeadEvents(c)
	for {
		headEvent := <-c
		if headEvent.Slot <= latestSlot {
			continue
		}

		latestSlot = headEvent.Slot
		currentEpoch := latestSlot / relaycommon.SlotsPerEpoch
		slotsUntilNextEpoch := relaycommon.SlotsPerEpoch - (latestSlot % relaycommon.SlotsPerEpoch)
		s.log.Infof("headSlot: %d / currentEpoch: %d / slotsUntilNextEpoch: %d", latestSlot, currentEpoch, slotsUntilNextEpoch)

		// On every new epoch, get proposer duties for current and next epoch (to avoid boundary problems)
		if len(duties) == 0 || currentEpoch > latestEpoch {
			dutiesResp, err := s.bn.GetProposerDuties(currentEpoch)
			if err != nil {
				s.log.WithError(err).Error("couldn't get proposer duties")
				continue
			}

			duties = make(map[uint64]string)
			for _, d := range dutiesResp.Data {
				duties[d.Slot] = d.Pubkey
			}

			dutiesResp, err = s.bn.GetProposerDuties(currentEpoch + 1)
			if err != nil {
				s.log.WithError(err).Error("failed get proposer duties")
			} else {
				for _, d := range dutiesResp.Data {
					duties[d.Slot] = d.Pubkey
				}
			}
			s.log.Infof("Got %d duties", len(duties))
		}
		latestEpoch = currentEpoch

		// Now get the latest block, for the execution payload
		block, err := s.bn.GetBlock("head")
		if err != nil {
			s.log.WithError(err).Error("failed get latest block from BN")
			continue
		}

		nextSlot := block.Data.Message.Slot + 1
		nextProposerPubkey := duties[nextSlot]
		s.log.Infof("next slot: %d / block: %s / parent: %s / proposerPubkey: %s", nextSlot, block.Data.Message.Body.ExecutionPayload.BlockHash.String(), block.Data.Message.Body.ExecutionPayload.ParentHash, nextProposerPubkey)

		if nextProposerPubkey == "" {
			s.log.WithField("duties", duties).Error("no proposerPubkey for next slot")
		} else {
			go s.CallGetHeader(10*time.Second, nextSlot, block.Data.Message.Body.ExecutionPayload.BlockHash.String(), duties[nextSlot])
			go s.CallGetHeader(12*time.Second, nextSlot, block.Data.Message.Body.ExecutionPayload.BlockHash.String(), duties[nextSlot])
		}
		fmt.Println("")
	}
}

func (s *RelayCollector) CallGetHeader(timeout time.Duration, slot uint64, parentHash, proposerPubkey string) {
	s.log.Infof("querying relays for bid in %.0f sec...", timeout.Seconds())

	// Wait 12 seconds, allowing the builder to prepare bids
	time.Sleep(timeout)

	for _, relay := range s.relays {
		go s.CallGetHeaderOnRelay(relay, slot, parentHash, proposerPubkey)
	}
}

func (s *RelayCollector) CallGetHeaderOnRelay(relay common.RelayEntry, slot uint64, parentHash, proposerPubkey string) {
	path := fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHash, proposerPubkey)
	url := relay.GetURI(path)
	log := s.log.WithField("relay", relay.Hostname())

	log.Debugf("Querying %s", url)
	var bid types.GetHeaderResponse
	timeRequestStart := time.Now().UTC()
	code, err := common.SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, url, nil, &bid)
	timeRequestEnd := time.Now().UTC()
	if err != nil {
		if strings.Contains(err.Error(), "no builder bid") {
			return
		}
		log.WithFields(logrus.Fields{
			"code": code,
			"url":  url,
		}).WithError(err).Error("error on getHeader request")
		return
	}
	if code != 200 {
		// log.WithField("code", code).Info("no bid received")
		return
	}
	entry := database.SignedBuilderBidToEntry(relay.Hostname(), slot, parentHash, proposerPubkey, timeRequestStart, timeRequestEnd, bid.Data)
	log.Infof("bid received! slot: %d \t value: %s \t block_hash: %s \t timestamp: %d / %d", slot, bid.Data.Message.Value.String(), bid.Data.Message.Header.BlockHash.String(), bid.Data.Message.Header.Timestamp, entry.Timestamp)
	err = s.db.SaveSignedBuilderBid(entry)
	if err != nil {
		log.WithFields(logrus.Fields{
			"extraData":          bid.Data.Message.Header.ExtraData.String(),
			"extraDataProcessed": entry.ExtraData,
		}).WithError(err).Error("failed saving bid to database")
		return
	}
}
