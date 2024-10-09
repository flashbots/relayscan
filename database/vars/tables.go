// Package vars contains the database variables such as dynamic table names
package vars

import (
	relaycommon "github.com/flashbots/mev-boost-relay/common"
)

var (
	tableBase = relaycommon.GetEnv("DB_TABLE_PREFIX", "rsdev")

	TableMigrations                 = tableBase + "_migrations"
	TableSignedBuilderBid           = tableBase + "_signed_builder_bid"
	TableDataAPIPayloadDelivered    = tableBase + "_data_api_payload_delivered"
	TableDataAPIBuilderBid          = tableBase + "_data_api_builder_bid"
	TableError                      = tableBase + "_error"
	TableBlockBuilder               = tableBase + "_blockbuilder"
	TableBlockBuilderInclusionStats = tableBase + "_blockbuilder_stats_inclusion"
	TableAdjustments                = tableBase + "_adjustments"
)
