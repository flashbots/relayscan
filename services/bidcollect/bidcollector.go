// Package bidcollect contains code for bid collection from various sources.
package bidcollect

import (
	"github.com/flashbots/relayscan/common"
	"github.com/sirupsen/logrus"
)

type BidCollectorOpts struct {
	Log *logrus.Entry
	UID string

	CollectTopBidWebsocketStream bool
	CollectDataAPI               bool
	CollectGetHeader             bool
	BeaconNodeURI                string // for getHeader

	Relays                []common.RelayEntry
	TopBidWebsocketRelays []common.RelayEntry

	OutDir    string
	OutputTSV bool
}

type BidCollector struct {
	opts *BidCollectorOpts
	log  *logrus.Entry

	topBidWebsocketC chan TopBidWebsocketStreamBidsMsg
	dataAPIBidC      chan DataAPIPollerBidsMsg
	getHeaderBidC    chan GetHeaderPollerBidsMsg

	processor *BidProcessor
}

func NewBidCollector(opts *BidCollectorOpts) *BidCollector {
	c := &BidCollector{
		log:  opts.Log,
		opts: opts,
	}

	if c.opts.OutDir == "" {
		opts.Log.Fatal("outDir is required")
	}

	// inputs
	c.dataAPIBidC = make(chan DataAPIPollerBidsMsg, bidCollectorInputChannelSize)
	c.topBidWebsocketC = make(chan TopBidWebsocketStreamBidsMsg, bidCollectorInputChannelSize)
	c.getHeaderBidC = make(chan GetHeaderPollerBidsMsg, bidCollectorInputChannelSize)

	// output
	c.processor = NewBidProcessor(&BidProcessorOpts{
		Log:       opts.Log,
		UID:       opts.UID,
		OutDir:    opts.OutDir,
		OutputTSV: opts.OutputTSV,
	})
	return c
}

func (c *BidCollector) MustStart() {
	go c.processor.Start()

	if c.opts.CollectGetHeader {
		poller := NewGetHeaderPoller(&GetHeaderPollerOpts{
			Log:       c.log,
			BidC:      c.getHeaderBidC,
			BeaconURI: c.opts.BeaconNodeURI,
			Relays:    c.opts.Relays,
		})
		go poller.Start()
	}

	if c.opts.CollectDataAPI {
		poller := NewDataAPIPoller(&DataAPIPollerOpts{
			Log:    c.log,
			BidC:   c.dataAPIBidC,
			Relays: c.opts.Relays,
		})
		go poller.Start()
	}

	if c.opts.CollectTopBidWebsocketStream {
		for _, relay := range c.opts.TopBidWebsocketRelays {
			c.log.Infof("Starting top bid websocket stream for %s...", relay.String())
			topBidWebsocketStream := NewTopBidWebsocketStreamConnection(TopBidWebsocketStreamOpts{
				Log:   c.log,
				Relay: relay,
				BidC:  c.topBidWebsocketC,
			})
			go topBidWebsocketStream.Start()
		}
	}

	for {
		select {
		case bid := <-c.topBidWebsocketC:
			commonBid := TopBidWebsocketStreamToCommonBid(&bid)
			c.processor.processBids([]*CommonBid{commonBid})
		case bids := <-c.dataAPIBidC:
			commonBids := DataAPIToCommonBids(bids)
			c.processor.processBids(commonBids)
		case bid := <-c.getHeaderBidC:
			commonBid := GetHeaderToCommonBid(bid)
			c.processor.processBids([]*CommonBid{commonBid})
		}
	}
}
