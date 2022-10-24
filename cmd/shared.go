package cmd

import (
	"os"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/metachris/relayscan/common"
)

var (
	Version  = "dev" // is set during build process
	log      = common.LogSetup(logJSON, defaultLogLevel, logDebug)
	logDebug = os.Getenv("DEBUG") != ""
	logJSON  = os.Getenv("LOG_JSON") != ""

	defaultBeaconURI        = relaycommon.GetEnv("BEACON_URI", "http://localhost:3500")
	defaultPostgresDSN      = relaycommon.GetEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	defaultLogLevel         = relaycommon.GetEnv("LOG_LEVEL", "info")
	defaultEthNodeURI       = relaycommon.GetEnv("ETH_NODE_URI", "")
	defaultEthBackupNodeURI = relaycommon.GetEnv("ETH_NODE_BACKUP_URI", "")
)
