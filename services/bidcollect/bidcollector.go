// Package bidcollect contains code for bid collection from various sources.
package bidcollect

import (
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/services/bidcollect/types"
	"github.com/sirupsen/logrus"
)

type BidCollectorOpts struct {
	Log *logrus.Entry
	UID string

	CollectUltrasoundStream bool
	CollectGetHeader        bool
	CollectDataAPI          bool

	Relays        []common.RelayEntry
	BeaconNodeURI string // for getHeader

	OutDir    string
	OutputTSV bool

	RedisAddr string
	UseRedis  bool
}

type BidCollector struct {
	opts *BidCollectorOpts
	log  *logrus.Entry

	ultrasoundBidC chan UltrasoundStreamBidsMsg
	dataAPIBidC    chan DataAPIPollerBidsMsg
	getHeaderBidC  chan GetHeaderPollerBidsMsg

	processor *BidProcessor
}

func NewBidCollector(opts *BidCollectorOpts) (c *BidCollector, err error) {
	c = &BidCollector{
		log:  opts.Log,
		opts: opts,
	}

	if c.opts.OutDir == "" {
		opts.Log.Fatal("outDir is required")
	}

	// inputs
	c.dataAPIBidC = make(chan DataAPIPollerBidsMsg, types.BidCollectorInputChannelSize)
	c.ultrasoundBidC = make(chan UltrasoundStreamBidsMsg, types.BidCollectorInputChannelSize)
	c.getHeaderBidC = make(chan GetHeaderPollerBidsMsg, types.BidCollectorInputChannelSize)

	// output
	c.processor, err = NewBidProcessor(&BidProcessorOpts{
		Log:       opts.Log,
		UID:       opts.UID,
		OutDir:    opts.OutDir,
		OutputTSV: opts.OutputTSV,
		RedisAddr: opts.RedisAddr,
		UseRedis:  opts.UseRedis,
	})
	return c, err
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
			c.processor.processBids([]*types.CommonBid{commonBid})
		case bids := <-c.dataAPIBidC:
			commonBids := DataAPIToCommonBids(bids)
			c.processor.processBids(commonBids)
		case bid := <-c.getHeaderBidC:
			commonBid := GetHeaderToCommonBid(bid)
			c.processor.processBids([]*types.CommonBid{commonBid})
		}
	}
}
