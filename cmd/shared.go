package cmd

import (
	"os"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/metachris/relayscan/common"
)

var (
	Version = "dev" // is set during build process
	log     = common.LogSetup(logJSON, logLevel)

	defaultBeaconURI = relaycommon.GetEnv("BEACON_URI", "http://localhost:3500")
	postgresDSN      = relaycommon.GetEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	logJSON          = os.Getenv("LOG_JSON") != ""
	logLevel         = relaycommon.GetEnv("LOG_LEVEL", "info")
)
