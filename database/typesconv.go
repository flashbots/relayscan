package database

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/flashbots/go-boost-utils/types"
	relaycommon "github.com/flashbots/mev-boost-relay/common"
)

func BidTraceV2JSONToPayloadDeliveredEntry(relay string, entry relaycommon.BidTraceV2JSON) PayloadDeliveredEntry {
	ret := PayloadDeliveredEntry{
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
	}

	if entry.NumTx > 0 {
		ret.NumTx = NewNullInt64(int64(entry.NumTx))
	}

	if entry.BlockNumber > 0 {
		ret.BlockNumber = NewNullInt64(int64(entry.BlockNumber))
	}
	return ret
}

func SignedBuilderBidToEntry(relay string, slot uint64, receivedAt time.Time, bid *types.SignedBuilderBid) SignedBuilderBidEntry {
	extraDataBytes := bid.Message.Header.ExtraData
	for i, b := range extraDataBytes {
		if b > 127 {
			extraDataBytes[i] = 32
		}
	}

	extraData := string(extraDataBytes)
	if !utf8.Valid(bid.Message.Header.ExtraData) {
		extraData = ""
		fmt.Printf("invalid extradata utf8: %s bytes: %s \n", extraData, extraDataBytes)
	}

	return SignedBuilderBidEntry{
		Relay:      relay,
		ReceivedAt: receivedAt,
		Epoch:      slot / 32,
		Slot:       slot,

		Signature:    bid.Signature.String(),
		Pubkey:       bid.Message.Pubkey.String(),
		Value:        bid.Message.Value.String(),
		ParentHash:   bid.Message.Header.ParentHash.String(),
		FeeRecipient: bid.Message.Header.FeeRecipient.String(),
		BlockHash:    bid.Message.Header.BlockHash.String(),
		BlockNumber:  bid.Message.Header.BlockNumber,
		GasLimit:     bid.Message.Header.GasLimit,
		GasUsed:      bid.Message.Header.GasUsed,
		ExtraData:    extraData,
	}
}
