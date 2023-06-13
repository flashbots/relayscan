package common

import (
	"os"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
)

var (
	Version  = "dev" // is set during build process
	LogDebug = os.Getenv("DEBUG") != ""
	LogJSON  = os.Getenv("LOG_JSON") != ""
	Genesis  = 1_606_824_023

	DefaultBeaconURI        = relaycommon.GetEnv("BEACON_URI", "http://localhost:3500")
	DefaultPostgresDSN      = relaycommon.GetEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	DefaultLogLevel         = relaycommon.GetEnv("LOG_LEVEL", "info")
	DefaultEthNodeURI       = relaycommon.GetEnv("ETH_NODE_URI", "")
	DefaultEthBackupNodeURI = relaycommon.GetEnv("ETH_NODE_BACKUP_URI", "")
)
