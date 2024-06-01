// Package bidcollect contains code for bid collection from various sources.
package bidcollect

import (
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/vars"
	"github.com/sirupsen/logrus"
)

type BidCollectorOpts struct {
	Log *logrus.Entry

	CollectUltrasoundStream bool
	CollectGetHeader        bool
	CollectDataAPI          bool

	Relays        []common.RelayEntry
	BeaconNodeURI string // for getHeader
	OutDir        string
}

type BidCollector struct {
	opts *BidCollectorOpts
	log  *logrus.Entry

	ultrasoundBidC chan UltrasoundStreamBidsMsg
	dataAPIBidC    chan DataAPIPollerBidsMsg
	// getHeaderBidC  chan DataAPIPollerBidsMsg

	processor *BidProcessor
}

func NewBidCollector(opts *BidCollectorOpts) *BidCollector {
	c := &BidCollector{
		log:  opts.Log,
		opts: opts,
	}

	if c.opts.OutDir == "" {
		c.opts.OutDir = "csv"
	}

	// inputs
	c.dataAPIBidC = make(chan DataAPIPollerBidsMsg, 1000)
	c.ultrasoundBidC = make(chan UltrasoundStreamBidsMsg, 1-00)

	// output
	c.processor = NewBidProcessor(&BidProcessorOpts{
		Log:    opts.Log,
		OutDir: "csv",
	})
	return c
}

func (c *BidCollector) MustStart() {
	c.log.WithField("version", vars.Version).Info("Starting BidCollector ...")
	go c.processor.Start()

	if c.opts.CollectGetHeader {
		c.log.Fatal("not yet implemented")
	}

	if c.opts.CollectDataAPI {
		poller := NewDataAPIPoller(&DataAPIPollerOpts{
			Log:    c.log,
			BidC:   c.dataAPIBidC,
			Relays: c.opts.Relays,
		})
		go poller.Start()
	}

	if c.opts.CollectUltrasoundStream {
		ultrasoundStream := NewUltrasoundStreamConnection(UltrasoundStreamOpts{
			Log:  c.log,
			BidC: c.ultrasoundBidC,
		})
		go ultrasoundStream.Start()
	}

	for {
		select {
		case bid := <-c.ultrasoundBidC:
			commonBid := UltrasoundStreamToCommonBid(&bid)
			c.processor.processBids([]*CommonBid{commonBid})
		case bids := <-c.dataAPIBidC:
			commonBids := DataAPIToCommonBids(bids)
			c.processor.processBids(commonBids)
		}
	}
}

// func (c *BidCollector) processBid(bid *CommonBid) {
// 	if c.outF != nil {
// 		_, err := fmt.Fprint(c.outF, bid.ToCSVLine("\t")+"\n")
// 		if err != nil {
// 			c.log.WithError(err).Error("couldn't write bid to file")
// 		}
// 	}
// }
