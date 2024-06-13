package bidcollect

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
	"github.com/sirupsen/logrus"
)

type GetHeaderPollerBidsMsg struct {
	Slot       uint64
	Bid        types.GetHeaderResponse
	Relay      common.RelayEntry
	ReceivedAt time.Time
}

type GetHeaderPollerOpts struct {
	Log       *logrus.Entry
	BidC      chan GetHeaderPollerBidsMsg
	BeaconURI string
	Relays    []common.RelayEntry
}

type GetHeaderPoller struct {
	log    *logrus.Entry
	bidC   chan GetHeaderPollerBidsMsg
	relays []common.RelayEntry
	bn     *beaconclient.ProdBeaconInstance
}

func NewGetHeaderPoller(opts *GetHeaderPollerOpts) *GetHeaderPoller {
	return &GetHeaderPoller{
		log:    opts.Log,
		bidC:   opts.BidC,
		relays: opts.Relays,
		bn:     beaconclient.NewProdBeaconInstance(opts.Log, opts.BeaconURI),
	}
}

func (poller *GetHeaderPoller) Start() {
	poller.log.WithField("relays", common.RelayEntriesToHostnameStrings(poller.relays)).Info("Starting GetHeaderPoller ...")

	//  Check beacon-node sync status, process current slot and start slot updates
	syncStatus, err := poller.bn.SyncStatus()
	if err != nil {
		poller.log.WithError(err).Fatal("couldn't get BN sync status")
	} else if syncStatus.IsSyncing {
		poller.log.Fatal("beacon node is syncing")
	}

	// var headSlot uint64
	var headSlot, nextSlot, currentEpoch, lastDutyUpdateEpoch uint64
	var duties map[uint64]string

	// subscribe to head events (because then, the BN will know the block + proposer details for the next slot)
	c := make(chan beaconclient.HeadEventData)
	go poller.bn.SubscribeToHeadEvents(c)

	// then run polling loop
	for {
		headEvent := <-c
		if headEvent.Slot <= headSlot {
			continue
		}

		headSlot = headEvent.Slot
		nextSlot = headSlot + 1
		tNextSlot := common.SlotToTime(nextSlot)
		untilNextSlot := tNextSlot.Sub(time.Now().UTC())

		currentEpoch = headSlot / relaycommon.SlotsPerEpoch
		poller.log.Infof("[getHeader poller] headSlot slot: %d / next slot: %d (%s), waitTime: %s", headSlot, nextSlot, tNextSlot.String(), untilNextSlot.String())

		// On every new epoch, get proposer duties for current and next epoch (to avoid boundary problems)
		if len(duties) == 0 || currentEpoch > lastDutyUpdateEpoch {
			dutiesResp, err := poller.bn.GetProposerDuties(currentEpoch)
			if err != nil {
				poller.log.WithError(err).Error("couldn't get proposer duties")
				continue
			}

			duties = make(map[uint64]string)
			for _, d := range dutiesResp.Data {
				duties[d.Slot] = d.Pubkey
			}

			dutiesResp, err = poller.bn.GetProposerDuties(currentEpoch + 1)
			if err != nil {
				poller.log.WithError(err).Error("failed get proposer duties")
			} else {
				for _, d := range dutiesResp.Data {
					duties[d.Slot] = d.Pubkey
				}
			}
			poller.log.Debugf("[getHeader poller] duties updated: %d entries", len(duties))
			lastDutyUpdateEpoch = currentEpoch
		}

		// Now get the latest block, for the execution payload
		block, err := poller.bn.GetBlock("head")
		if err != nil {
			poller.log.WithError(err).Error("failed get latest block from BN")
			continue
		}

		if block.Data.Message.Slot != headSlot {
			poller.log.WithField("slot", headSlot).WithField("bnSlot", block.Data.Message.Slot).Error("latest block slot is not current slot")
			continue
		}

		nextProposerPubkey := duties[nextSlot]
		poller.log.Debugf("[getHeader poller] next slot: %d / block: %s / parent: %s / proposerPubkey: %s", nextSlot, block.Data.Message.Body.ExecutionPayload.BlockHash.String(), block.Data.Message.Body.ExecutionPayload.ParentHash, nextProposerPubkey)

		if nextProposerPubkey == "" {
			poller.log.WithField("duties", duties).Error("no proposerPubkey for next slot")
		} else {
			// go poller.pollRelaysForBids(0*time.Second, nextSlot, block.Data.Message.Body.ExecutionPayload.BlockHash.String(), duties[nextSlot])
			go poller.pollRelaysForBids(1000*time.Millisecond, nextSlot, block.Data.Message.Body.ExecutionPayload.BlockHash.String(), duties[nextSlot])
		}
	}
}

// pollRelaysForBids will poll data api for given slot with t seconds offset
func (poller *GetHeaderPoller) pollRelaysForBids(tOffset time.Duration, slot uint64, parentHash, proposerPubkey string) {
	tSlotStart := common.SlotToTime(slot)
	tStart := tSlotStart.Add(tOffset)
	waitTime := tStart.Sub(time.Now().UTC())

	// poller.Log.Debugf("[getHeader poller] - prepare polling for slot %d t %d (tSlotStart: %s, tStart: %s, waitTime: %s)", slot, t, tSlotStart.String(), tStart.String(), waitTime.String())
	if waitTime < 0 {
		poller.log.Debugf("[getHeader poller] waitTime is negative: %s", waitTime.String())
		return
	}

	// Wait until expected time
	time.Sleep(waitTime)

	// Poll for bids now
	untilSlot := tSlotStart.Sub(time.Now().UTC())
	poller.log.Debugf("[getHeader poller] polling for slot %d at t=%s (tNow=%s)", slot, tOffset.String(), (untilSlot * -1).String())

	for _, relay := range poller.relays {
		go poller._pollRelayForBids(relay, tOffset, slot, parentHash, proposerPubkey)
	}
}

func (poller *GetHeaderPoller) _pollRelayForBids(relay common.RelayEntry, t time.Duration, slot uint64, parentHash, proposerPubkey string) {
	// log := poller.Log.WithField("relay", relay.Hostname()).WithField("slot", slot)
	log := poller.log.WithFields(logrus.Fields{
		"relay": relay.Hostname(),
		"slot":  slot,
		"t":     t.String(),
	})
	log.Debugf("[getHeader poller] polling relay %s for slot %d", relay.Hostname(), slot)

	path := fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHash, proposerPubkey)
	url := relay.GetURI(path)
	// log.Debugf("Querying %s", url)

	var bid types.GetHeaderResponse
	timeRequestStart := time.Now().UTC()
	code, err := common.SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, url, nil, &bid)
	timeRequestEnd := time.Now().UTC()
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "no builder bid") {
			return
		} else if strings.Contains(msg, "Too many getHeader requests! Use relay-analytics.ultrasound.money or the Websocket API") {
			return
		} else if code == 429 {
			log.Warn("[getHeader poller] 429 received")
			return
		}
		log.WithFields(logrus.Fields{
			"code": code,
			"url":  url,
		}).WithError(err).Error("[getHeader poller] error on getHeader request")
		return
	}
	if code != 200 {
		log.WithField("code", code).Debug("[getHeader poller] no bid received")
		return
	}
	log.WithField("durationMs", timeRequestEnd.Sub(timeRequestStart).Milliseconds()).Infof("[getHeader poller] bid received! slot: %d - value: %s - block_hash: %s -", slot, bid.Data.Message.Value.String(), bid.Data.Message.Header.BlockHash.String())

	// send data to channel
	poller.bidC <- GetHeaderPollerBidsMsg{Slot: slot, Bid: bid, Relay: relay, ReceivedAt: time.Now().UTC()}
}
