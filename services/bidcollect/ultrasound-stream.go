package bidcollect

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

type UltrasoundStreamBidsMsg struct {
	Bid        common.UltrasoundStreamBid
	Relay      string
	ReceivedAt time.Time
}

type UltrasoundStreamOpts struct {
	Log  *logrus.Entry
	BidC chan UltrasoundStreamBidsMsg
}

type UltrasoundStreamConnection struct {
	log        *logrus.Entry
	url        string
	bidC       chan UltrasoundStreamBidsMsg
	backoffSec int
}

func NewUltrasoundStreamConnection(opts UltrasoundStreamOpts) *UltrasoundStreamConnection {
	return &UltrasoundStreamConnection{
		log:        opts.Log,
		url:        ultrasoundStreamDefaultURL,
		bidC:       opts.BidC,
		backoffSec: initialBackoffSec,
	}
}

func (ustream *UltrasoundStreamConnection) Start() {
	ustream.connect()
}

func (ustream *UltrasoundStreamConnection) reconnect() {
	backoffDuration := time.Duration(ustream.backoffSec) * time.Second
	ustream.log.Infof("reconnecting to ultrasound stream in %s sec ...", backoffDuration.String())
	time.Sleep(backoffDuration)

	// increase backoff timeout for next try
	ustream.backoffSec *= 2
	if ustream.backoffSec > maxBackoffSec {
		ustream.backoffSec = maxBackoffSec
	}

	ustream.connect()
}

func (ustream *UltrasoundStreamConnection) connect() {
	ustream.log.WithField("uri", ustream.url).Info("Starting Ultrasound bid stream...")

	dialer := websocket.DefaultDialer
	wsSubscriber, resp, err := dialer.Dial(ustream.url, nil)
	if err != nil {
		ustream.log.WithError(err).Error("failed to connect to bloxroute, reconnecting in a bit...")
		go ustream.reconnect()
		return
	}
	defer wsSubscriber.Close()
	defer resp.Body.Close()

	ustream.log.Info("ultrasound stream connection successful")
	ustream.backoffSec = initialBackoffSec // reset backoff timeout

	bid := new(common.UltrasoundStreamBid)

	for {
		_, nextNotification, err := wsSubscriber.ReadMessage()
		if err != nil {
			// Handle websocket errors, by closing and reconnecting. Errors seen previously:
			ustream.log.WithError(err).Error("ultrasound stream websocket error")
			go ustream.reconnect()
			return
		}

		// nc.log.WithField("msg", hexutil.Encode(nextNotification)).Info("got message from ultrasound stream")

		// Unmarshal SSZ
		err = bid.UnmarshalSSZ(nextNotification)
		if err != nil {
			ustream.log.WithError(err).WithField("msg", hexutil.Encode(nextNotification)).Error("failed to unmarshal ultrasound stream message")
			continue
		}

		ustream.bidC <- UltrasoundStreamBidsMsg{
			Bid:        *bid,
			Relay:      "relay.ultrasound.money",
			ReceivedAt: time.Now().UTC(),
		}
	}
}
