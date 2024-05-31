// Package bidcollect contains code for bid collection from various sources.
package bidcollect

import (
	"fmt"
	"os"
	"strings"

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
	OutFile       string
}

type BidCollector struct {
	opts *BidCollectorOpts
	log  *logrus.Entry
	outF *os.File

	ultrasoundBidC chan UltrasoundStreamBidsMsg
	dataAPIBidC    chan DataAPIPollerBidsMsg
	// getHeaderBidC  chan DataAPIPollerBidsMsg
}

func NewBidCollector(opts *BidCollectorOpts) *BidCollector {
	c := &BidCollector{
		log:  opts.Log,
		opts: opts,
	}

	c.dataAPIBidC = make(chan DataAPIPollerBidsMsg, 100)
	c.ultrasoundBidC = make(chan UltrasoundStreamBidsMsg, 100)
	return c
}

func (c *BidCollector) MustStart() {
	var err error
	c.log.WithField("version", vars.Version).Info("Starting BidCollector ...")

	// Setup output file
	if c.opts.OutFile != "" {
		c.log.Infof("writing to %s", c.opts.OutFile)
		c.outF, err = os.OpenFile(c.opts.OutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			c.log.WithError(err).WithField("filename", c.opts.OutFile).Fatal("failed to open output file")
		}

		fi, err := c.outF.Stat()
		if err != nil {
			c.log.WithError(err).Fatal("failed stat on output file")
		}

		if fi.Size() == 0 {
			_, err = fmt.Fprint(c.outF, strings.Join(CommonBidCSVFields, "\t")+"\n")
			if err != nil {
				c.log.WithError(err).Fatal("failed to write header to output file")
			}
		}
	}

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
			c.processBid(commonBid)
		case bids := <-c.dataAPIBidC:
			commonBids := DataAPIToCommonBids(bids)
			for _, commonBid := range commonBids {
				c.processBid(commonBid)
			}
		}
	}
}

func (c *BidCollector) processBid(bid *CommonBid) {
	if c.outF != nil {
		_, err := fmt.Fprint(c.outF, bid.ToCSVLine("\t")+"\n")
		if err != nil {
			c.log.WithError(err).Error("couldn't write bid to file")
		}
	}
}
