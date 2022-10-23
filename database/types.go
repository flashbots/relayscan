package database

import (
	"database/sql"
	"time"
)

func NewNullBool(b bool) sql.NullBool {
	return sql.NullBool{
		Bool:  b,
		Valid: true,
	}
}

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

type DataAPIPayloadDeliveredEntry struct {
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
	ValueClaimedWei      string        `db:"value_claimed_wei"`
	ValueClaimedEth      string        `db:"value_claimed_eth"`
	NumTx                sql.NullInt64 `db:"num_tx"`
	BlockNumber          sql.NullInt64 `db:"block_number"`

	ValueCheckOk            sql.NullBool   `db:"value_check_ok"`
	ValueCheckMethod        sql.NullString `db:"value_check_method"`
	ValueDeliveredWei       sql.NullString `db:"value_delivered_wei"`
	ValueDeliveredEth       sql.NullString `db:"value_delivered_eth"`
	ValueDeliveredDiffWei   sql.NullString `db:"value_delivered_diff_wei"`
	ValueDeliveredDiffEth   sql.NullString `db:"value_delivered_diff_eth"`
	BlockCoinbaseAddress    sql.NullString `db:"block_coinbase_addr"`
	BlockCoinbaseIsProposer sql.NullBool   `db:"block_coinbase_is_proposer"`
	CoinbaseDiffWei         sql.NullString `db:"coinbase_diff_wei"`
	CoinbaseDiffEth         sql.NullString `db:"coinbase_diff_eth"`
}

type DataAPIBuilderBidEntry struct {
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
	Timestamp            time.Time     `db:"timestamp"`
}

type SignedBuilderBidEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	Relay       string    `db:"relay"`
	RequestedAt time.Time `db:"requested_at"`
	ReceivedAt  time.Time `db:"received_at"`
	LatencyMS   int64     `db:"duration_ms"`

	Slot           uint64 `db:"slot"`
	ParentHash     string `db:"parent_hash"`
	ProposerPubkey string `db:"proposer_pubkey"`

	Pubkey    string `db:"pubkey"`
	Signature string `db:"signature"`

	Value        string `db:"value"`
	FeeRecipient string `db:"fee_recipient"`
	BlockHash    string `db:"block_hash"`
	BlockNumber  uint64 `db:"block_number"`
	GasLimit     uint64 `db:"gas_limit"`
	GasUsed      uint64 `db:"gas_used"`
	ExtraData    string `db:"extra_data"`
	Epoch        uint64 `db:"epoch"`
}
