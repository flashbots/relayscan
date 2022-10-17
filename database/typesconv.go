package database

import (
	relaycommon "github.com/flashbots/mev-boost-relay/common"
)

func BidTraceV2JSONToPayloadDeliveredEntry(relay string, entry relaycommon.BidTraceV2JSON) PayloadDeliveredEntry {
	return PayloadDeliveredEntry{
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
		NumTx:                entry.NumTx,
		BlockNumber:          entry.BlockNumber,
	}
}
