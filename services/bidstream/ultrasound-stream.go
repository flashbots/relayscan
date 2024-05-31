// Package bidstream contains code for the ultrasound stream
package bidstream

import (
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/relayscan/common"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	ultrasoundStreamDefaultURL = "ws://relay-builders-eu.ultrasound.money/ws/v1/top_bid"
	initialBackoffSec          = 5
	maxBackoffSec              = 120
)

type UltrasoundStreamOpts struct {
	BidC chan common.UltrasoundStreamBid
	Log  *logrus.Entry
	URL  string // optional override, default: ultrasoundStreamDefaultURL
}

// StartUltrasoundStreamConnection starts a Websocket or gRPC subscription (depending on URL) in the background
func StartUltrasoundStreamConnection(opts UltrasoundStreamOpts) {
	ultrasoundStream := NewUltrasoundStreamConnection(opts)
	go ultrasoundStream.Start()
}

type UltrasoundStreamConnection struct {
	log        *logrus.Entry
	url        string
	bidC       chan common.UltrasoundStreamBid
	backoffSec int
}

func NewUltrasoundStreamConnection(opts UltrasoundStreamOpts) *UltrasoundStreamConnection {
	url := opts.URL
	if url == "" {
		url = ultrasoundStreamDefaultURL
	}

	return &UltrasoundStreamConnection{
		log:        opts.Log,
		url:        url,
		bidC:       opts.BidC,
		backoffSec: initialBackoffSec,
	}
}

func (nc *UltrasoundStreamConnection) Start() {
	nc.connect()
}

func (nc *UltrasoundStreamConnection) reconnect() {
	backoffDuration := time.Duration(nc.backoffSec) * time.Second
	nc.log.Infof("reconnecting to ultrasound stream in %s sec ...", backoffDuration.String())
	time.Sleep(backoffDuration)

	// increase backoff timeout for next try
	nc.backoffSec *= 2
	if nc.backoffSec > maxBackoffSec {
		nc.backoffSec = maxBackoffSec
	}

	nc.connect()
}

func (nc *UltrasoundStreamConnection) connect() {
	nc.log.WithField("uri", nc.url).Info("connecting...")

	dialer := websocket.DefaultDialer
	wsSubscriber, resp, err := dialer.Dial(nc.url, nil)
	if err != nil {
		nc.log.WithError(err).Error("failed to connect to bloxroute, reconnecting in a bit...")
		go nc.reconnect()
		return
	}
	defer wsSubscriber.Close()
	defer resp.Body.Close()

	nc.log.Info("ultrasound stream connection successful")
	nc.backoffSec = initialBackoffSec // reset backoff timeout

	bid := new(common.UltrasoundStreamBid)

	for {
		_, nextNotification, err := wsSubscriber.ReadMessage()
		if err != nil {
			// Handle websocket errors, by closing and reconnecting. Errors seen previously:
			nc.log.WithError(err).Error("ultrasound stream websocket error")
			go nc.reconnect()
			return
		}

		// nc.log.WithField("msg", hexutil.Encode(nextNotification)).Info("got message from ultrasound stream")

		// Unmarshal SSZ
		err = bid.UnmarshalSSZ(nextNotification)
		if err != nil {
			nc.log.WithError(err).WithField("msg", hexutil.Encode(nextNotification)).Error("failed to unmarshal ultrasound stream message")
			continue
		}

		nc.bidC <- *bid
	}
}
