package migrations

import (
	"github.com/flashbots/relayscan/database/vars"
	migrate "github.com/rubenv/sql-migrate"
)

var migration003SQL = `
	CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_num_blob_txs_idx ON ` + vars.TableDataAPIPayloadDelivered + `("num_blob_txs");
	CREATE INDEX IF NOT EXISTS ` + vars.TableDataAPIPayloadDelivered + `_num_blobs_idx ON ` + vars.TableDataAPIPayloadDelivered + `("num_blobs");
`

var Migration003AddBlobIndexes = &migrate.Migration{
	Id: "003-add-blob-indexes",
	Up: []string{migration003SQL},

	DisableTransactionUp:   false,
	DisableTransactionDown: true,
}
