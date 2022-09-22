// Package service provides the relay check service
package service

import (
	"fmt"
	"time"

	"github.com/flashbots/go-template/database"
	"github.com/flashbots/mev-boost-relay/beaconclient"
	"github.com/flashbots/mev-boost-relay/common"
	"github.com/sirupsen/logrus"
)

type RelayCheckerService struct {
	log    *logrus.Entry
	relays []RelayEntry
	bn     *beaconclient.ProdBeaconInstance
	db     database.IDatabaseService
}

func NewRelayCheckerService(log *logrus.Entry, relays []RelayEntry, beaconURL string, db database.IDatabaseService) *RelayCheckerService {
	srv := &RelayCheckerService{
		log:    log,
		relays: relays,
		db:     db,
		bn:     beaconclient.NewProdBeaconInstance(log, beaconURL),
	}

	return srv
}

func (s *RelayCheckerService) Start() {
	s.log.Info("Starting relay checker service")

	//  Check beacon-node sync status, process current slot and start slot updates
	syncStatus, err := s.bn.SyncStatus()
	if err != nil {
		s.log.WithError(err).Fatal("couldn't get BN sync status")
	}
	if syncStatus.IsSyncing {
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
		currentEpoch := latestSlot / uint64(common.SlotsPerEpoch)

		// On every new epoch, get proposer duties for current and next epoch (to skip boundary problems)
		if currentEpoch > latestEpoch {
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
				s.log.WithError(err).Error("couldn't get proposer duties")
			} else {
				for _, d := range dutiesResp.Data {
					duties[d.Slot] = d.Pubkey
				}
			}
			s.log.Infof("Got %d duties", len(duties))
		}
		latestEpoch = currentEpoch

		// Now get the latest block, for the execution payload
		block, err := s.bn.GetBlock()
		if err != nil {
			s.log.WithError(err).Error("couldn't get proposer duties")
			continue
		}
		s.log.Infof("slot: %d / block: %s / parent: %s", block.Data.Message.Slot, block.Data.Message.Body.ExecutionPayload.BlockHash, block.Data.Message.Body.ExecutionPayload.ParentHash)

		// Prepare URL to request the head
		nextSlot := block.Data.Message.Slot + 1
		s.CallGetHeader(nextSlot, block.Data.Message.Body.ExecutionPayload.BlockHash.String(), duties[nextSlot])
		fmt.Println("")
	}
}

func (s *RelayCheckerService) CallGetHeader(slot uint64, parentHash, proposerPubkey string) {
	url := fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHash, proposerPubkey)
	s.log.Infof("querying in 12 sec: %s", url)

	// Wait 12 seconds, allowing the builder to prepare bids
	time.Sleep(12 * time.Second)

	// // Query the relay for bid
	// url = *relay + url
	// s.log.Infof("Querying %s", url)
	// res, err := http.Get(url)
	// if err != nil {
	// 	s.log.WithError(err).Error("error on getHeader request")
	// 	continue
	// }
	// s.log.Infof("relay status returned: %d", res.StatusCode)

	// if res.StatusCode == 200 {
	// 	bodyBytes, err := io.ReadAll(res.Body)
	// 	res.Body.Close()

	// 	if err != nil {
	// 		s.log.WithError(err).Error("Couldn't read response body")
	// 		continue
	// 	}

	// 	var dst types.GetHeaderResponse
	// 	if err := json.Unmarshal(bodyBytes, &dst); err != nil {
	// 		s.log.WithError(err).Errorf("Couldn't unmarshal response body: %s", string(bodyBytes))
	// 		continue
	// 	}

	// 	s.log.Infof("bid received! slot: %d / value: %s / hash: %s / parentHash: %s", nextSlot, dst.Data.Message.Value.String(), dst.Data.Message.Header.BlockHash, block.Data.Message.Body.ExecutionPayload.BlockHash)
	// }
}
