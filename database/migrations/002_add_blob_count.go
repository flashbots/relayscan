package migrations

import (
	"github.com/flashbots/relayscan/database/vars"
	migrate "github.com/rubenv/sql-migrate"
)

var migration002SQL = `
	ALTER TABLE ` + vars.TableDataAPIPayloadDelivered + ` ADD num_blob_txs int DEFAULT NULL;
	ALTER TABLE ` + vars.TableDataAPIPayloadDelivered + ` ADD num_blobs int DEFAULT NULL;
`

var Migration002AddBlobCount = &migrate.Migration{
	Id: "002-add-blob-count",
	Up: []string{migration002SQL},

	DisableTransactionUp:   false,
	DisableTransactionDown: true,
}
