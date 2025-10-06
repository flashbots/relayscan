package bidcollect

import (
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/services/bidcollect/types"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
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
		url:        types.UltrasoundStreamDefaultURL,
		bidC:       opts.BidC,
		backoffSec: types.InitialBackoffSec,
	}
}

func (ustream *UltrasoundStreamConnection) Start() {
	ustream.connect()
}

func (ustream *UltrasoundStreamConnection) reconnect() {
	backoffDuration := time.Duration(ustream.backoffSec) * time.Second
	ustream.log.Infof("[ultrasounds-stream] reconnecting to ultrasound stream in %s sec ...", backoffDuration.String())
	time.Sleep(backoffDuration)

	// increase backoff timeout for next try
	ustream.backoffSec *= 2
	if ustream.backoffSec > types.MaxBackoffSec {
		ustream.backoffSec = types.MaxBackoffSec
	}

	ustream.connect()
}

func (ustream *UltrasoundStreamConnection) connect() {
	ustream.log.WithField("uri", ustream.url).Info("[ultrasounds-stream] Starting bid stream...")

	dialer := websocket.DefaultDialer
	wsSubscriber, resp, err := dialer.Dial(ustream.url, nil)
	if err != nil {
		ustream.log.WithError(err).Error("[ultrasounds-stream] failed to connect to bloxroute, reconnecting in a bit...")
		go ustream.reconnect()
		return
	}
	defer wsSubscriber.Close() //nolint:errcheck
	defer resp.Body.Close()    //nolint:errcheck

	ustream.log.Info("[ultrasounds-stream] stream connection successful")
	ustream.backoffSec = types.InitialBackoffSec // reset backoff timeout

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
			ustream.log.WithError(err).WithField("msg", hexutil.Encode(nextNotification)).Error("[ultrasounds-stream] failed to unmarshal ultrasound stream message")
			continue
		}

		ustream.bidC <- UltrasoundStreamBidsMsg{
			Bid:        *bid,
			Relay:      "relay.ultrasound.money",
			ReceivedAt: time.Now().UTC(),
		}
	}
}

func UltrasoundStreamToCommonBid(bid *UltrasoundStreamBidsMsg) *types.CommonBid {
	blockHash := hexutil.Encode(bid.Bid.BlockHash[:])
	parentHash := hexutil.Encode(bid.Bid.ParentHash[:])
	builderPubkey := hexutil.Encode(bid.Bid.BuilderPubkey[:])
	blockFeeRecipient := hexutil.Encode(bid.Bid.FeeRecipient[:])

	return &types.CommonBid{
		SourceType:   types.SourceTypeUltrasoundStream,
		ReceivedAtMs: bid.ReceivedAt.UnixMilli(),

		TimestampMs:       int64(bid.Bid.Timestamp), //nolint:gosec
		Slot:              bid.Bid.Slot,
		BlockNumber:       bid.Bid.BlockNumber,
		BlockHash:         strings.ToLower(blockHash),
		ParentHash:        strings.ToLower(parentHash),
		BuilderPubkey:     strings.ToLower(builderPubkey),
		Value:             bid.Bid.Value.String(),
		BlockFeeRecipient: strings.ToLower(blockFeeRecipient),
		Relay:             bid.Relay,
	}
}
