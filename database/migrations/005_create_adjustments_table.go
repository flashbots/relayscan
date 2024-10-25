package migrations

import (
	"github.com/flashbots/relayscan/database/vars"
	migrate "github.com/rubenv/sql-migrate"
)

var migration005SQL = `CREATE TABLE IF NOT EXISTS ` + vars.TableAdjustments + ` (
                id SERIAL PRIMARY KEY,
                inserted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                slot BIGINT NOT NULL,
                adjusted_block_hash TEXT NOT NULL,
                adjusted_value TEXT NOT NULL,
                block_number BIGINT NOT NULL,
                builder_pubkey TEXT NOT NULL,
                delta TEXT NOT NULL,
                submitted_block_hash TEXT NOT NULL,
                submitted_received_at TIMESTAMP WITH TIME ZONE NOT NULL,
                submitted_value TEXT NOT NULL,
                UNIQUE(slot, adjusted_block_hash)
            );`

var Migration005CreateAdjustmentsTable = &migrate.Migration{
	Id: "005-create-adjustments-table",
	Up: []string{migration005SQL},

	DisableTransactionUp:   false,
	DisableTransactionDown: true,
}
