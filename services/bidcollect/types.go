package bidcollect

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/relayscan/common"
)

var CommonBidCSVFields = []string{
	"source_type", "received_at_ms",
	"timestamp_ms",
	"slot", "slot_t_ms", "value",
	"block_hash", "parent_hash", "builder_pubkey", "block_number",
	"block_fee_recipient",
	"relay", "proposer_pubkey", "proposer_fee_recipient", "optimistic_submission",
}

type CommonBid struct {
	// Collector-internal fields
	SourceType   int   `json:"source_type"`
	ReceivedAtMs int64 `json:"received_at"`

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

func (bid *CommonBid) UniqueKey() string {
	return fmt.Sprintf("%d-%s-%s-%s-%s", bid.Slot, bid.BlockHash, bid.ParentHash, bid.BuilderPubkey, bid.Value)
}

func (bid *CommonBid) ValueAsBigInt() *big.Int {
	value := new(big.Int)
	value.SetString(bid.Value, 10)
	return value
}

func (bid *CommonBid) ToCSVFields() []string {
	tsMs := bid.TimestampMs
	if tsMs == 0 {
		if bid.Timestamp > 0 {
			tsMs = bid.Timestamp * 1000
		} else {
			tsMs = bid.ReceivedAtMs // fallback for getHeader bids (which don't include the bid timestamp)
		}
	}

	slotTms := tsMs - common.SlotToTime(bid.Slot).UnixMilli()

	return []string{
		// Collector-internal fields
		fmt.Sprint(bid.SourceType), fmt.Sprint(bid.ReceivedAtMs),

		// Common fields
		fmt.Sprint(tsMs),
		fmt.Sprint(bid.Slot), fmt.Sprint(slotTms), bid.Value,
		bid.BlockHash, bid.ParentHash, bid.BuilderPubkey, fmt.Sprint(bid.BlockNumber),

		// Ultrasound top-bid stream
		bid.BlockFeeRecipient,

		// Data API
		bid.Relay, bid.ProposerPubkey, bid.ProposerFeeRecipient, boolToString(bid.OptimisticSubmission),
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

func UltrasoundStreamToCommonBid(bid *UltrasoundStreamBidsMsg) *CommonBid {
	blockHash := hexutil.Encode(bid.Bid.BlockHash[:])
	parentHash := hexutil.Encode(bid.Bid.ParentHash[:])
	builderPubkey := hexutil.Encode(bid.Bid.BuilderPubkey[:])
	blockFeeRecipient := hexutil.Encode(bid.Bid.FeeRecipient[:])

	return &CommonBid{
		SourceType:   SourceTypeUltrasoundStream,
		ReceivedAtMs: bid.ReceivedAt.UnixMilli(),

		Timestamp:         int64(bid.Bid.Timestamp) / 1000,
		TimestampMs:       int64(bid.Bid.Timestamp),
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

func DataAPIToCommonBids(bids DataAPIPollerBidsMsg) []*CommonBid {
	commonBids := make([]*CommonBid, 0, len(bids.Bids))
	for _, bid := range bids.Bids {
		commonBids = append(commonBids, &CommonBid{
			SourceType:   SourceTypeDataAPI,
			ReceivedAtMs: bids.ReceivedAt.UnixMilli(),

			Timestamp:            bid.Timestamp,
			TimestampMs:          bid.TimestampMs,
			Slot:                 bid.Slot,
			BlockNumber:          bid.BlockNumber,
			BlockHash:            strings.ToLower(bid.BlockHash),
			ParentHash:           strings.ToLower(bid.ParentHash),
			BuilderPubkey:        strings.ToLower(bid.BuilderPubkey),
			Value:                bid.Value,
			Relay:                bids.Relay.Hostname(),
			ProposerPubkey:       strings.ToLower(bid.ProposerPubkey),
			ProposerFeeRecipient: strings.ToLower(bid.ProposerFeeRecipient),
			OptimisticSubmission: bid.OptimisticSubmission,
		})
	}
	return commonBids
}

func GetHeaderToCommonBid(bid GetHeaderPollerBidsMsg) *CommonBid {
	return &CommonBid{
		SourceType:   SourceTypeGetHeader,
		ReceivedAtMs: bid.ReceivedAt.UnixMilli(),
		Relay:        bid.Relay.Hostname(),
		Slot:         bid.Slot,

		BlockNumber: bid.Bid.Data.Message.Header.BlockNumber,
		BlockHash:   strings.ToLower(bid.Bid.Data.Message.Header.BlockHash.String()),
		ParentHash:  strings.ToLower(bid.Bid.Data.Message.Header.ParentHash.String()),
		Value:       bid.Bid.Data.Message.Value.String(),
	}
}
