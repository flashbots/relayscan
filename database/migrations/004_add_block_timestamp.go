package migrations

import (
	"github.com/flashbots/relayscan/database/vars"
	migrate "github.com/rubenv/sql-migrate"
)

var migration004SQL = `
	ALTER TABLE ` + vars.TableDataAPIPayloadDelivered + ` ADD block_timestamp timestamp DEFAULT NULL;
`

var Migration004AddBlockTimestamp = &migrate.Migration{
	Id: "004-add-block-timestamp",
	Up: []string{migration004SQL},

	DisableTransactionUp:   false,
	DisableTransactionDown: true,
}
