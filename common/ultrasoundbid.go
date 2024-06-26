package common

import "math/big"

// https://github.com/ultrasoundmoney/docs/blob/main/top-bid-websocket.md

type (
	U64       [8]byte
	Hash      [32]byte
	PublicKey [48]byte
	Address   [20]byte
	U256      [32]byte
)

func (n *U256) String() string {
	return new(big.Int).SetBytes(ReverseBytes(n[:])).String()
}

type UltrasoundStreamBid struct {
	Timestamp     uint64    `json:"timestamp"`
	Slot          uint64    `json:"slot"`
	BlockNumber   uint64    `json:"block_number"`
	BlockHash     Hash      `json:"block_hash" ssz-size:"32"`
	ParentHash    Hash      `json:"parent_hash" ssz-size:"32"`
	BuilderPubkey PublicKey `json:"builder_pubkey" ssz-size:"48"`
	FeeRecipient  Address   `json:"fee_recipient" ssz-size:"20"`
	Value         U256      `json:"value" ssz-size:"32"`
}
