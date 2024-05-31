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

	t := time.Now().UTC()
	slot := common.TimeToSlot(t)

	nextSlot := slot + 1
	tNextSlot := common.SlotToTime(nextSlot)

	untilNextSlot := tNextSlot.Sub(t)
	poller.Log.Infof("current slot: %d / next slot: %d (%s) / time until: %s", slot, nextSlot, tNextSlot.String(), untilNextSlot.String())
}
