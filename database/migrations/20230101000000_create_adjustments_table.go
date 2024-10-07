package migrations

import (
    "database/sql"
    migrate "github.com/rubenv/sql-migrate"
)

func init() {
    Migrations = append(Migrations, &migrate.Migration{
        Id: "20230101000000_create_adjustments_table",
        Up: []string{
            `CREATE TABLE IF NOT EXISTS adjustments (
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
            );`,
        },
        Down: []string{
            "DROP TABLE IF EXISTS adjustments;",
        },
    })
}
