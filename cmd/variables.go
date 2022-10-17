package cmd

import (
	"os"

	"github.com/flashbots/mev-boost-relay/common"
)

var (
	Version = "dev" // is set during build process

	// defaultNetwork     = common.GetEnv("NETWORK", "")
	// defaultBeaconURIs  = common.GetSliceEnv("BEACON_URIS", []string{"http://localhost:3500"})
	// defaultRedisURI    = common.GetEnv("REDIS_URI", "localhost:6379")
	// defaultPostgresDSN = common.GetEnv("POSTGRES_DSN", "")
	logJSON  = os.Getenv("LOG_JSON") != ""
	logLevel = common.GetEnv("LOG_LEVEL", "info")

	// beaconNodeURIs []string
	// redisURI       string
	// postgresDSN    string

	// network string
)
