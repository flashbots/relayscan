package migrations

import (
	"github.com/flashbots/relayscan/database/vars"
	migrate "github.com/rubenv/sql-migrate"
)

var initialSchema = `
CREATE TABLE IF NOT EXISTS ` + vars.TableSignedBuilderBid + ` (
	id          bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,

	relay        text NOT NULL,
	requested_at timestamp NOT NULL,
	received_at  timestamp NOT NULL,
	duration_ms	 bigint NOT NULL,

	slot            bigint NOT NULL,
	parent_hash     varchar(66) NOT NULL,
	proposer_pubkey	varchar(98) NOT NULL,

	pubkey 		varchar(98) NOT NULL,
	signature   text NOT NULL,

	value         NUMERIC(48, 0) NOT NULL,
	fee_recipient varchar(42) NOT NULL,
	block_hash    varchar(66) NOT NULL,
	block_number  bigint NOT NULL,
	gas_limit     bigint NOT NULL,
	gas_used      bigint NOT NULL,
	extra_data    text NOT NULL,
	timestamp     bigint NOT NULL,
	prev_randao   text NOT NULL,

	epoch bigint NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS ` + vars.TableSignedBuilderBid + `_u_relay_slot_n_hashes_idx ON ` + vars.TableSignedBuilderBid + `("relay", "slot", "parent_hash", "block_hash");
CREATE INDEX IF NOT EXISTS ` + vars.TableSignedBuilderBid + `_insertedat_idx ON ` + vars.TableSignedBuilderBid + `("inserted_at");
CREATE INDEX IF NOT EXISTS ` + vars.TableSignedBuilderBid + `_slot_idx ON ` + vars.TableSignedBuilderBid + `("slot");
CREATE INDEX IF NOT EXISTS ` + vars.TableSignedBuilderBid + `_block_number_idx ON ` + vars.TableSignedBuilderBid + `("block_number");
CREATE INDEX IF NOT EXISTS ` + vars.TableSignedBuilderBid + `_value_idx ON ` + vars.TableSignedBuilderBid + `("value");


CREATE TABLE IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + ` (
	id          bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,
	relay       text NOT NULL,

	epoch bigint NOT NULL,
	slot  bigint NOT NULL,

	parent_hash            varchar(66) NOT NULL,
	block_hash             varchar(66) NOT NULL,
	builder_pubkey         varchar(98) NOT NULL,
	proposer_pubkey        varchar(98) NOT NULL,
	proposer_fee_recipient varchar(42) NOT NULL,
	gas_limit              bigint NOT NULL,
	gas_used               bigint NOT NULL,
	value_claimed_wei      NUMERIC(48, 0) NOT NULL,
	value_claimed_eth      NUMERIC(16, 8) NOT NULL,
	num_tx                 int,
	block_number           bigint,
	extra_data    		   text NOT NULL,

	slot_missed	                boolean, 		-- null means not yet checked
	value_check_ok              boolean, 		-- null means not yet checked
	value_check_method          text,  		    -- how value was checked (i.e. blockBalanceDiff)
	value_delivered_wei         NUMERIC(48, 0), -- actually delivered value
	value_delivered_eth         NUMERIC(16, 8), -- actually delivered value
	value_delivered_diff_wei    NUMERIC(48, 0), -- value_delivered - value_claimed
	value_delivered_diff_eth    NUMERIC(16, 8), -- value_delivered - value_claimed
	block_coinbase_addr		    varchar(42),    -- block coinbase address
	block_coinbase_is_proposer  boolean,        -- true if coinbase == proposerFeeRecipient
	coinbase_diff_wei           NUMERIC(48, 0), -- builder value difference
	coinbase_diff_eth           NUMERIC(16, 8), -- builder value difference
	found_onchain               boolean,        -- whether the payload blockhash can be found on chain (at all)
	notes						text
);

CREATE UNIQUE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_u_relay_slot_blockhash_idx ON ` + vars.TableDataAPIPayloadDelivered + `("relay", "slot", "parent_hash", "block_hash");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_insertedat_idx ON ` + vars.TableDataAPIPayloadDelivered + `("inserted_at");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_slot_idx ON ` + vars.TableDataAPIPayloadDelivered + `("slot");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_builder_pubkey_idx ON ` + vars.TableDataAPIPayloadDelivered + `("builder_pubkey");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_block_number_idx ON ` + vars.TableDataAPIPayloadDelivered + `("block_number");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_value_wei_idx ON ` + vars.TableDataAPIPayloadDelivered + `("value_claimed_wei");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_valuecheck_ok_idx ON ` + vars.TableDataAPIPayloadDelivered + `("value_check_ok");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_slotmissed_idx ON ` + vars.TableDataAPIPayloadDelivered + `("slot_missed");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_cb_diff_eth_idx ON ` + vars.TableDataAPIPayloadDelivered + `("coinbase_diff_eth");


CREATE TABLE IF NOT EXISTS ` + vars.TableDataAPIBuilderBid + ` (
	id          bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,
	relay       text NOT NULL,

	epoch bigint NOT NULL,
	slot  bigint NOT NULL,

	parent_hash            varchar(66) NOT NULL,
	block_hash             varchar(66) NOT NULL,
	builder_pubkey         varchar(98) NOT NULL,
	proposer_pubkey        varchar(98) NOT NULL,
	proposer_fee_recipient varchar(42) NOT NULL,
	gas_limit              bigint NOT NULL,
	gas_used               bigint NOT NULL,
	value                  NUMERIC(48, 0) NOT NULL,
	num_tx                 int,
	block_number           bigint,
	timestamp			   timestamp NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS ` + vars.TableDataAPIBuilderBid + `_unique_idx ON ` + vars.TableDataAPIBuilderBid + `("relay", "slot", "builder_pubkey", "parent_hash", "block_hash");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIBuilderBid + `_insertedat_idx ON ` + vars.TableDataAPIBuilderBid + `("inserted_at");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIBuilderBid + `_slot_idx ON ` + vars.TableDataAPIBuilderBid + `("slot");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIBuilderBid + `_builder_pubkey_idx ON ` + vars.TableDataAPIBuilderBid + `("builder_pubkey");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIBuilderBid + `_block_number_idx ON ` + vars.TableDataAPIBuilderBid + `("block_number");
CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIBuilderBid + `_value_idx ON ` + vars.TableDataAPIBuilderBid + `("value");


CREATE TABLE IF NOT EXISTS ` + vars.TableBlockBuilder + ` (
	id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,

	builder_pubkey  varchar(98) NOT NULL,
	description    	text NOT NULL,

	UNIQUE (builder_pubkey)
);

CREATE TABLE IF NOT EXISTS ` + vars.TableBlockBuilderInclusionStats + ` (
	id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,

	type 	text NOT NULL, -- "extra_data" or "builder_pubkey"
	hours 	int NOT NULL,  -- the amount of hours aggregated over (i.e. 24 for daily)

	time_start    timestamp NOT NULL,
	time_end      timestamp NOT NULL,
	builder_name  text NOT NULL,

	extra_data    	text NOT NULL,
	builder_pubkeys text NOT NULL,
	blocks_included int NOT NULL,

	UNIQUE (type, hours, time_start, time_end, builder_name)
);

CREATE INDEX IF NOT EXISTS ` + vars.TableBlockBuilderInclusionStats + `_type_hours_idx ON ` + vars.TableBlockBuilderInclusionStats + `("type", "hours");
CREATE INDEX IF NOT EXISTS ` + vars.TableBlockBuilderInclusionStats + `_time_start_idx ON ` + vars.TableBlockBuilderInclusionStats + `("time_start");
CREATE INDEX IF NOT EXISTS ` + vars.TableBlockBuilderInclusionStats + `_time_end_idx ON ` + vars.TableBlockBuilderInclusionStats + `("time_end");
CREATE INDEX IF NOT EXISTS ` + vars.TableBlockBuilderInclusionStats + `_builder_name_idx ON ` + vars.TableBlockBuilderInclusionStats + `("builder_name");
CREATE INDEX IF NOT EXISTS ` + vars.TableBlockBuilderInclusionStats + `_extra_data_idx ON ` + vars.TableBlockBuilderInclusionStats + `("extra_data");
`

var Migration001InitDatabase = &migrate.Migration{
	Id: "001-init-database",
	Up: []string{initialSchema},

	DisableTransactionUp:   false,
	DisableTransactionDown: true,
}
