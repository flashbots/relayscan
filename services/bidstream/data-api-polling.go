package bidstream

import (
	"time"

	"github.com/flashbots/relayscan/common"
	"github.com/sirupsen/logrus"
)

type DataAPIPollerOpts struct {
	Log  *logrus.Entry
	BidC chan common.UltrasoundStreamBid
}

type DataAPIPoller struct {
	Log  *logrus.Entry
	BidC chan common.UltrasoundStreamBid
}

func NewDataAPIPoller(opts *DataAPIPollerOpts) *DataAPIPoller {
	return &DataAPIPoller{
		Log:  opts.Log,
		BidC: opts.BidC,
	}
}

func (poller *DataAPIPoller) Start() {
	poller.Log.Info("Starting DataAPIPoller ...")

	for {
		t := time.Now().UTC()
		slot := common.TimeToSlot(t)

		nextSlot := slot + 1
		tNextSlot := common.SlotToTime(nextSlot)

		untilNextSlot := tNextSlot.Sub(t)
		poller.Log.Infof("[data-api poller] current slot: %d / next slot: %d (%s) / time until: %s", slot, nextSlot, tNextSlot.String(), untilNextSlot.String())

		// poll at t-4, t-2, t=0, t=2
		go poller.pollRelaysForBids(nextSlot, -4)
		go poller.pollRelaysForBids(nextSlot, -2)
		go poller.pollRelaysForBids(nextSlot, 0)
		go poller.pollRelaysForBids(nextSlot, 2)

		// wait until next slot
		time.Sleep(untilNextSlot)
	}
}

func (poller *DataAPIPoller) pollRelaysForBids(slot uint64, t int64) {
	tSlotStart := common.SlotToTime(slot)
	tStart := tSlotStart.Add(time.Duration(t) * time.Second)
	waitTime := tStart.Sub(time.Now().UTC())

	poller.Log.Infof("[data-api poller] - prepare polling for slot %d t %d (tSlotStart: %s, tStart: %s, waitTime: %s)", slot, t, tSlotStart.String(), tStart.String(), waitTime.String())
	if waitTime < 0 {
		poller.Log.Warnf("[data-api poller] - waitTime is negative: %s", waitTime.String())
		return
	}

	// Wait until expected time
	time.Sleep(waitTime)

	// Poll for bids now
	untilSlot := tSlotStart.Sub(time.Now().UTC())
	poller.Log.Infof("[data-api poller] - polling for slot %d at %d (tNow=%s)", slot, t, untilSlot.String())

	poller.Log.Warn("not yet implemented: actually polling the relays for bids")
}
