package database

import (
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
