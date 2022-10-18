package database

import (
	"database/sql"
	"time"
)

func NewNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: i,
		Valid: true,
	}
}

func NewNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

type PayloadDeliveredEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`
	Relay      string    `db:"relay"`

	Epoch uint64 `db:"epoch"`
	Slot  uint64 `db:"slot"`

	ParentHash           string        `db:"parent_hash"`
	BlockHash            string        `db:"block_hash"`
	BuilderPubkey        string        `db:"builder_pubkey"`
	ProposerPubkey       string        `db:"proposer_pubkey"`
	ProposerFeeRecipient string        `db:"proposer_fee_recipient"`
	GasLimit             uint64        `db:"gas_limit"`
	GasUsed              uint64        `db:"gas_used"`
	Value                string        `db:"value"`
	NumTx                sql.NullInt64 `db:"num_tx"`
	BlockNumber          sql.NullInt64 `db:"block_number"`
}

type SignedBuilderBidEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`
	ReceivedAt time.Time `db:"received_at"`
	Relay      string    `db:"relay"`

	Epoch uint64 `db:"epoch"`
	Slot  uint64 `db:"slot"`

	Signature    string `db:"signature"`
	Pubkey       string `db:"pubkey"`
	Value        string `db:"value"`
	ParentHash   string `db:"parent_hash"`
	FeeRecipient string `db:"fee_recipient"`
	BlockHash    string `db:"block_hash"`
	BlockNumber  uint64 `db:"block_number"`
	GasLimit     uint64 `db:"gas_limit"`
	GasUsed      uint64 `db:"gas_used"`
	ExtraData    string `db:"extra_data"`
}
