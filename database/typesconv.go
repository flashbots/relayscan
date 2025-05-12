package database

import (
	"math/big"
	"time"
	"unicode/utf8"

	"github.com/flashbots/go-boost-utils/types"
	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/flashbots/relayscan/common"
)

func BidTraceV2JSONToPayloadDeliveredEntry(relay string, entry relaycommon.BidTraceV2JSON) DataAPIPayloadDeliveredEntry {
	wei, ok := new(big.Int).SetString(entry.Value, 10)
	if !ok {
		wei = big.NewInt(0)
	}
	eth := common.WeiToEth(wei)
	ret := DataAPIPayloadDeliveredEntry{
		Relay:                relay,
		Epoch:                entry.Slot / 32,
		Slot:                 entry.Slot,
		ParentHash:           entry.ParentHash,
		BlockHash:            entry.BlockHash,
		BuilderPubkey:        entry.BuilderPubkey,
		ProposerPubkey:       entry.ProposerPubkey,
		ProposerFeeRecipient: entry.ProposerFeeRecipient,
		GasLimit:             entry.GasLimit,
		GasUsed:              entry.GasUsed,
		ValueClaimedWei:      entry.Value,
		ValueClaimedEth:      eth.String(),
	}

	if entry.NumTx > 0 {
		ret.NumTx = NewNullInt64(int64(entry.NumTx)) //nolint:gosec
	}

	if entry.BlockNumber > 0 {
		ret.BlockNumber = NewNullInt64(int64(entry.BlockNumber)) //nolint:gosec
	}
	return ret
}

func BidTraceV2WithTimestampJSONToBuilderBidEntry(relay string, entry relaycommon.BidTraceV2WithTimestampJSON) DataAPIBuilderBidEntry {
	ret := DataAPIBuilderBidEntry{
		Relay:                relay,
		Epoch:                entry.Slot / 32,
		Slot:                 entry.Slot,
		ParentHash:           entry.ParentHash,
		BlockHash:            entry.BlockHash,
		BuilderPubkey:        entry.BuilderPubkey,
		ProposerPubkey:       entry.ProposerPubkey,
		ProposerFeeRecipient: entry.ProposerFeeRecipient,
		GasLimit:             entry.GasLimit,
		GasUsed:              entry.GasUsed,
		Value:                entry.Value,
		Timestamp:            time.Unix(entry.Timestamp, 0).UTC(),
	}

	if entry.NumTx > 0 {
		ret.NumTx = NewNullInt64(int64(entry.NumTx)) //nolint:gosec
	}

	if entry.BlockNumber > 0 {
		ret.BlockNumber = NewNullInt64(int64(entry.BlockNumber)) //nolint:gosec
	}
	return ret
}

func ExtraDataToUtf8Str(extraData types.ExtraData) string {
	// replace non-ascii bytes
	for i, b := range extraData {
		if b < 32 || b > 126 {
			extraData[i] = 32
		}
	}

	// convert to str
	if !utf8.Valid(extraData) {
		return ""
	}

	return string(extraData)
}

func SignedBuilderBidToEntry(relay string, slot uint64, parentHash, proposerPubkey string, timeRequestStart, timeRequestEnd time.Time, bid *types.SignedBuilderBid) SignedBuilderBidEntry {
	return SignedBuilderBidEntry{
		Relay:       relay,
		RequestedAt: timeRequestStart,
		ReceivedAt:  timeRequestEnd,
		LatencyMS:   timeRequestEnd.Sub(timeRequestStart).Milliseconds(),

		Slot:           slot,
		ParentHash:     parentHash,
		ProposerPubkey: proposerPubkey,

		Pubkey:    bid.Message.Pubkey.String(),
		Signature: bid.Signature.String(),

		Value:        bid.Message.Value.String(),
		FeeRecipient: bid.Message.Header.FeeRecipient.String(),
		BlockHash:    bid.Message.Header.BlockHash.String(),
		BlockNumber:  bid.Message.Header.BlockNumber,
		GasLimit:     bid.Message.Header.GasLimit,
		GasUsed:      bid.Message.Header.GasUsed,
		ExtraData:    ExtraDataToUtf8Str(bid.Message.Header.ExtraData),
		Epoch:        slot / 32,
		Timestamp:    bid.Message.Header.Timestamp,
		PrevRandao:   bid.Message.Header.Random.String(),
	}
}
