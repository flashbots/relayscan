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
	ExtraData            string        `db:"extra_data"`

	FoundOnChain  sql.NullBool `db:"found_onchain"`
	SlotWasMissed sql.NullBool `db:"slot_missed"`

	ValueCheckOk     sql.NullBool   `db:"value_check_ok"`
	ValueCheckMethod sql.NullString `db:"value_check_method"`

	ValueDeliveredWei       sql.NullString `db:"value_delivered_wei"`
	ValueDeliveredEth       sql.NullString `db:"value_delivered_eth"`
	ValueDeliveredDiffWei   sql.NullString `db:"value_delivered_diff_wei"`
	ValueDeliveredDiffEth   sql.NullString `db:"value_delivered_diff_eth"`
	BlockCoinbaseAddress    sql.NullString `db:"block_coinbase_addr"`
	BlockCoinbaseIsProposer sql.NullBool   `db:"block_coinbase_is_proposer"`
	CoinbaseDiffWei         sql.NullString `db:"coinbase_diff_wei"`
	CoinbaseDiffEth         sql.NullString `db:"coinbase_diff_eth"`
	Notes                   sql.NullString `db:"notes"`
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
	PrevRandao   string `db:"prev_randao"`
	Timestamp    uint64 `db:"timestamp"`
	Epoch        uint64 `db:"epoch"`
}

type BlockBuilderEntry struct {
	ID            int64     `db:"id"`
	InsertedAt    time.Time `db:"inserted_at"`
	BuilderPubkey string    `db:"builder_pubkey"`
	Description   string    `db:"description"`
}

type TopRelayEntry struct {
	Relay       string `db:"relay" json:"relay"`
	NumPayloads uint64 `db:"payloads" json:"num_payloads"`
	Percent     string `json:"percent"`
}

type TopBuilderEntry struct {
	ExtraData string   `db:"extra_data" json:"extra_data"`
	NumBlocks uint64   `db:"blocks" json:"num_blocks"`
	Percent   string   `json:"percent"`
	Aliases   []string `json:"aliases,omitempty"`
}

// type RelayProfitability struct {
// 	Relay       string `db:"relay" json:"relay"`
// 	TimeSince   time.Time
// 	TimeUntil   time.Time
// 	NumPayloads uint64 `db:"payloads" json:"num_payloads"`
// }

type BuilderProfitEntry struct {
	ExtraData string   `db:"extra_data" json:"extra_data"`
	Aliases   []string `json:"aliases,omitempty"`

	NumBlocks           uint64 `db:"blocks" json:"num_blocks"`
	NumBlocksProfit     uint64 `db:"blocks_profit" json:"num_blocks_profit"`
	NumBlocksSubsidised uint64 `db:"blocks_sub" json:"num_blocks_sub"`

	ProfitPerBlockAvg    string `db:"avg_profit_per_block" json:"avg_profit_per_block"`
	ProfitPerBlockMedian string `db:"median_profit_per_block" json:"median_profit_per_block"`

	ProfitTotal    string `db:"total_profit" json:"profit_total"`
	SubsidiesTotal string `db:"total_subsidies" json:"subsidies_total"`
}

type BuilderStatsEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	Hours int `db:"hours" json:"hours"`

	TimeStart time.Time `db:"time_start" json:"time_start"`
	TimeEnd   time.Time `db:"time_end" json:"time_end"`

	BuilderName string   `db:"builder_name" json:"builder_name"`
	ExtraData   []string `db:"extra_data" json:"extra_data"`

	BuilderPubkeys []string `db:"builder_pubkeys" json:"builder_pubkeys"`
	BlocksIncluded int      `db:"blocks_included" json:"blocks_included"`
}
