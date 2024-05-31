package bidcollect

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/relayscan/common"
)

// iota
const (
	CollectGetHeader = iota
	CollectDataAPI
	CollectUltrasoundStream
)

var CommonBidCSVFields = []string{
	"source", "received_at",
	"timestamp", "slot", "block_number", "block_hash", "parent_hash", "builder_pubkey", "value",
	"block_fee_recipient",
	"relay", "timestamp_ms", "proposer_pubkey", "proposer_fee_recipient", "optimistic_submission",
}

type CommonBid struct {
	// Collector-internal fields
	Source     int   `json:"source"`
	ReceivedAt int64 `json:"received_at"`

	// Common fields
	Timestamp     int64  `json:"timestamp"`
	Slot          uint64 `json:"slot"`
	BlockNumber   uint64 `json:"block_number"`
	BlockHash     string `json:"block_hash"`
	ParentHash    string `json:"parent_hash"`
	BuilderPubkey string `json:"builder_pubkey"`
	Value         string `json:"value"`

	// Ultrasound top-bid stream - https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md
	BlockFeeRecipient string `json:"block_fee_recipient"`

	// Data API
	// - Ultrasound: https://relay-analytics.ultrasound.money/relay/v1/data/bidtraces/builder_blocks_received?slot=9194844
	// - Flashbots: https://boost-relay.flashbots.net/relay/v1/data/bidtraces/builder_blocks_received?slot=8969837
	Relay                string `json:"relay"`
	TimestampMs          int64  `json:"timestamp_ms"`
	ProposerPubkey       string `json:"proposer_pubkey"`
	ProposerFeeRecipient string `json:"proposer_fee_recipient"`
	OptimisticSubmission bool   `json:"optimistic_submission"`

	// getHeader
}

func (bid *CommonBid) ToCSVFields() []string {
	return []string{
		// Collector-internal fields
		fmt.Sprint(bid.Source), fmt.Sprint(bid.ReceivedAt),

		// Common fields
		fmt.Sprint(bid.Timestamp), fmt.Sprint(bid.Slot), fmt.Sprint(bid.BlockNumber), bid.BlockHash, bid.ParentHash, bid.BuilderPubkey, bid.Value,

		// Ultrasound top-bid stream
		bid.BlockFeeRecipient,

		// Data API
		bid.Relay, fmt.Sprint(bid.TimestampMs), bid.ProposerPubkey, bid.ProposerFeeRecipient, boolToString(bid.OptimisticSubmission),
	}
}

func (bid *CommonBid) ToCSVLine(separator string) string {
	return strings.Join(bid.ToCSVFields(), separator)
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func UltrasoundStreamToCommonBid(bid *common.UltrasoundStreamBid) *CommonBid {
	blockHash := hexutil.Encode(bid.BlockHash[:])
	parentHash := hexutil.Encode(bid.ParentHash[:])
	builderPubkey := hexutil.Encode(bid.BuilderPubkey[:])
	blockFeeRecipient := hexutil.Encode(bid.FeeRecipient[:])

	return &CommonBid{
		Source:     CollectUltrasoundStream,
		ReceivedAt: time.Now().UTC().Unix(),

		Timestamp:         int64(bid.Timestamp),
		Slot:              bid.Slot,
		BlockNumber:       bid.BlockNumber,
		BlockHash:         blockHash,
		ParentHash:        parentHash,
		BuilderPubkey:     builderPubkey,
		Value:             bid.Value.String(),
		BlockFeeRecipient: blockFeeRecipient,
	}
}

func DataAPIToCommonBids(bids DataAPIPollerBidsMsg) []*CommonBid {
	commonBids := make([]*CommonBid, 0, len(bids.Bids))
	for _, bid := range bids.Bids {
		commonBids = append(commonBids, &CommonBid{
			Source:     CollectDataAPI,
			ReceivedAt: bids.ReceivedAt.Unix(),

			Timestamp:            bid.Timestamp,
			Slot:                 bid.Slot,
			BlockNumber:          bid.BlockNumber,
			BlockHash:            bid.BlockHash,
			ParentHash:           bid.ParentHash,
			BuilderPubkey:        bid.BuilderPubkey,
			Value:                bid.Value,
			Relay:                bids.Relay.Hostname(),
			TimestampMs:          bid.TimestampMs,
			ProposerPubkey:       bid.ProposerPubkey,
			ProposerFeeRecipient: bid.ProposerFeeRecipient,
			OptimisticSubmission: bid.OptimisticSubmission,
		})
	}
	return commonBids
}
