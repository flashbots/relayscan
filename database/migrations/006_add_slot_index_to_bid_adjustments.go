package migrations

import (
	"github.com/flashbots/relayscan/database/vars"
	migrate "github.com/rubenv/sql-migrate"
)

var migration006SQL = `CREATE INDEX IF NOT EXISTS idx_` + vars.TableAdjustments + `_slot ON ` + vars.TableAdjustments + ` (slot);`

var Migration006AddSlotIndexToAdjustments = &migrate.Migration{
	Id: "006-add-slot-index-to-adjustments",
	Up: []string{migration006SQL},

	DisableTransactionUp:   false,
	DisableTransactionDown: true,
}
