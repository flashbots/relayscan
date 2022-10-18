package database

import (
	"os"

	"github.com/flashbots/mev-boost-relay/common"
)

var (
	runDBTests = os.Getenv("RUN_DB_TESTS") == "1"
	testDBDSN  = common.GetEnv("TEST_DB_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
)

// func TestPostgresStuff(t *testing.T) {
// 	if !runDBTests {
// 		t.Skip("Skipping database tests")
// 	}

// 	db, err := NewDatabaseService(testDBDSN)
// 	require.NoError(t, err)

// }
