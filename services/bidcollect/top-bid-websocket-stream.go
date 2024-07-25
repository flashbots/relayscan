package bidcollect

import (
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/relayscan/common"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type TopBidWebsocketStreamBidsMsg struct {
	Bid        common.TopBidWebsocketStreamBid
	Relay      string
	ReceivedAt time.Time
}

type TopBidWebsocketStreamOpts struct {
	Log   *logrus.Entry
	Relay common.RelayEntry
	BidC  chan TopBidWebsocketStreamBidsMsg
}

type TopBidWebsocketStreamConnection struct {
	log        *logrus.Entry
	relay      common.RelayEntry
	bidC       chan TopBidWebsocketStreamBidsMsg
	backoffSec int
}

func NewTopBidWebsocketStreamConnection(opts TopBidWebsocketStreamOpts) *TopBidWebsocketStreamConnection {
	return &TopBidWebsocketStreamConnection{
		log:        opts.Log.WithField("uri", opts.Relay.String()),
		relay:      opts.Relay,
		bidC:       opts.BidC,
		backoffSec: initialBackoffSec,
	}
}

func (ustream *TopBidWebsocketStreamConnection) Start() {
	ustream.connect()
}

func (ustream *TopBidWebsocketStreamConnection) reconnect() {
	backoffDuration := time.Duration(ustream.backoffSec) * time.Second
	ustream.log.Infof("[websocket-stream] reconnecting to websocket stream in %s sec ...", backoffDuration.String())
	time.Sleep(backoffDuration)

	// increase backoff timeout for next try
	ustream.backoffSec *= 2
	if ustream.backoffSec > maxBackoffSec {
		ustream.backoffSec = maxBackoffSec
	}

	ustream.connect()
}

func (ustream *TopBidWebsocketStreamConnection) connect() {
	ustream.log.Info("[websocket-stream] Starting bid stream...")

	dialer := websocket.DefaultDialer
	wsSubscriber, resp, err := dialer.Dial(ustream.relay.String(), nil)
	if err != nil {
		ustream.log.WithError(err).Error("[websocket-stream] failed to connect to websocket stream, reconnecting in a bit...")
		go ustream.reconnect()
		return
	}
	defer wsSubscriber.Close()
	defer resp.Body.Close()

	ustream.log.Info("[websocket-stream] stream connection successful")
	ustream.backoffSec = initialBackoffSec // reset backoff timeout

	bid := new(common.TopBidWebsocketStreamBid)

	for {
		_, nextNotification, err := wsSubscriber.ReadMessage()
		if err != nil {
			// Handle websocket errors, by closing and reconnecting. Errors seen previously:
			ustream.log.WithError(err).Error("websocket stream websocket error")
			go ustream.reconnect()
			return
		}

		// Unmarshal SSZ
		err = bid.UnmarshalSSZ(nextNotification)
		if err != nil {
			ustream.log.WithError(err).WithField("msg", hexutil.Encode(nextNotification)).Error("[websocket-stream] failed to unmarshal websocket stream message")
			continue
		}

		ustream.bidC <- TopBidWebsocketStreamBidsMsg{
			Bid:        *bid,
			Relay:      ustream.relay.Hostname(),
			ReceivedAt: time.Now().UTC(),
		}
	}
}
