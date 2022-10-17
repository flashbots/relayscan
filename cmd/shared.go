package cmd

import (
	"os"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/metachris/relayscan/common"
)

var (
	Version = "dev" // is set during build process
	log     = common.LogSetup(logJSON, logLevel)

	// defaultNetwork     = common.GetEnv("NETWORK", "")
	// defaultBeaconURIs  = common.GetSliceEnv("BEACON_URIS", []string{"http://localhost:3500"})
	// defaultRedisURI    = common.GetEnv("REDIS_URI", "localhost:6379")
	postgresDSN = relaycommon.GetEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	logJSON     = os.Getenv("LOG_JSON") != ""
	logLevel    = relaycommon.GetEnv("LOG_LEVEL", "info")

	// beaconNodeURIs []string
	// redisURI       string
	// postgresDSN    string

	// network string
)
