package bidcollect

import (
	"context"
	"fmt"
	"net/http"
	"time"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/flashbots/relayscan/common"
	"github.com/sirupsen/logrus"
)

type DataAPIPollerBidsMsg struct {
	Bids       []relaycommon.BidTraceV2WithTimestampJSON
	Relay      common.RelayEntry
	ReceivedAt time.Time
}

type DataAPIPollerOpts struct {
	Log    *logrus.Entry
	BidC   chan DataAPIPollerBidsMsg
	Relays []common.RelayEntry
}

type DataAPIPoller struct {
	Log    *logrus.Entry
	BidC   chan DataAPIPollerBidsMsg
	Relays []common.RelayEntry
}

func NewDataAPIPoller(opts *DataAPIPollerOpts) *DataAPIPoller {
	return &DataAPIPoller{
		Log:    opts.Log,
		BidC:   opts.BidC,
		Relays: opts.Relays,
	}
}

func (poller *DataAPIPoller) Start() {
	poller.Log.WithField("relays", common.RelayEntriesToHostnameStrings(poller.Relays)).Info("Starting DataAPIPoller ...")

	// initially, wait until start of next slot
	t := time.Now().UTC()
	slot := common.TimeToSlot(t)
	nextSlot := slot + 1
	tNextSlot := common.SlotToTime(nextSlot)
	untilNextSlot := tNextSlot.Sub(t)

	poller.Log.Infof("[data-api poller] waiting until start of next slot (%d - %s from now)", nextSlot, untilNextSlot.String())
	time.Sleep(untilNextSlot)

	// then run polling loop
	for {
		// calculate next slot details
		t := time.Now().UTC()
		slot := common.TimeToSlot(t)
		nextSlot := slot + 1
		tNextSlot := common.SlotToTime(nextSlot)
		untilNextSlot := tNextSlot.Sub(t)

		poller.Log.Infof("[data-api poller] scheduling polling for upcoming slot: %d (%s - in %s)", nextSlot, tNextSlot.String(), untilNextSlot.String())

		// Schedule polling at t-4, t-2, t=0, t=2
		go poller.pollRelaysForBids(nextSlot, -4*time.Second)
		go poller.pollRelaysForBids(nextSlot, -2*time.Second)
		go poller.pollRelaysForBids(nextSlot, -500*time.Millisecond)
		go poller.pollRelaysForBids(nextSlot, 500*time.Millisecond)
		go poller.pollRelaysForBids(nextSlot, 2*time.Second)

		// wait until next slot
		time.Sleep(untilNextSlot)
	}
}

// pollRelaysForBids will poll data api for given slot with t seconds offset
func (poller *DataAPIPoller) pollRelaysForBids(slot uint64, tOffset time.Duration) {
	tSlotStart := common.SlotToTime(slot)
	tStart := tSlotStart.Add(tOffset)
	waitTime := tStart.Sub(time.Now().UTC())

	// poller.Log.Debugf("[data-api poller] - prepare polling for slot %d t %d (tSlotStart: %s, tStart: %s, waitTime: %s)", slot, t, tSlotStart.String(), tStart.String(), waitTime.String())
	if waitTime < 0 {
		poller.Log.Debugf("[data-api poller] waitTime is negative: %s", waitTime.String())
		return
	}

	// Wait until expected time
	time.Sleep(waitTime)

	// Poll for bids now
	untilSlot := tSlotStart.Sub(time.Now().UTC())
	poller.Log.Debugf("[data-api poller] polling for slot %d at t=%s (tNow=%s)", slot, tOffset.String(), (untilSlot * -1).String())

	for _, relay := range poller.Relays {
		go poller._pollRelayForBids(slot, relay, tOffset)
	}
}

func (poller *DataAPIPoller) _pollRelayForBids(slot uint64, relay common.RelayEntry, t time.Duration) {
	// log := poller.Log.WithField("relay", relay.Hostname()).WithField("slot", slot)
	log := poller.Log.WithFields(logrus.Fields{
		"relay": relay.Hostname(),
		"slot":  slot,
		"t":     t.String(),
	})
	// log.Debugf("[data-api poller] polling relay %s for slot %d", relay.Hostname(), slot)

	// build query URL
	path := "/relay/v1/data/bidtraces/builder_blocks_received"
	url := common.GetURIWithQuery(relay.URL, path, map[string]string{"slot": fmt.Sprintf("%d", slot)})
	// log.Debugf("[data-api poller] Querying %s", url)

	// start query
	var data []relaycommon.BidTraceV2WithTimestampJSON
	timeRequestStart := time.Now().UTC()
	code, err := common.SendHTTPRequest(context.Background(), *http.DefaultClient, http.MethodGet, url, nil, &data)
	timeRequestEnd := time.Now().UTC()
	if err != nil {
		log.WithError(err).Error("[data-api poller] failed to get data")
		return
	}
	log = log.WithFields(logrus.Fields{"code": code, "entries": len(data), "durationMs": timeRequestEnd.Sub(timeRequestStart).Milliseconds()})
	log.Debug("[data-api poller] request complete")

	// send data to channel
	poller.BidC <- DataAPIPollerBidsMsg{Bids: data, Relay: relay, ReceivedAt: time.Now().UTC()}
}
