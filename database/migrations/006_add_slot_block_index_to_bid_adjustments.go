package migrations

import (
	"github.com/flashbots/relayscan/database/vars"
	migrate "github.com/rubenv/sql-migrate"
)

var migration006SQL = `CREATE INDEX IF NOT EXISTS idx_` + vars.TableAdjustments + `_slot_block ON ` + vars.TableAdjustments + ` (slot, adjusted_block_hash);`

var Migration006AddSlotBlockIndexToAdjustments = &migrate.Migration{
	Id: "006-add-slot-block-index-to-adjustments",
	Up: []string{migration006SQL},

	DisableTransactionUp:   false,
	DisableTransactionDown: true,
}
