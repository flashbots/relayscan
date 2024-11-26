// Package vars contains global variables and configuration
package vars

import (
	"os"
	"strconv"

	relaycommon "github.com/flashbots/mev-boost-relay/common"
)

var (
	Version  = "dev" // is set during build process
	LogDebug = os.Getenv("DEBUG") != ""
	LogJSON  = os.Getenv("LOG_JSON") != ""
	Genesis  = getGenesis()

	DefaultBeaconURI        = relaycommon.GetEnv("BEACON_URI", "http://localhost:3500")
	DefaultPostgresDSN      = relaycommon.GetEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	DefaultLogLevel         = relaycommon.GetEnv("LOG_LEVEL", "info")
	DefaultEthNodeURI       = relaycommon.GetEnv("ETH_NODE_URI", "")
	DefaultEthBackupNodeURI = relaycommon.GetEnv("ETH_NODE_BACKUP_URI", "")
)

func getGenesis() int64 {
	genesis := int64(1_606_824_023)
	if envGenesis := os.Getenv("GENESIS"); envGenesis != "" {
		if parsedGenesis, err := strconv.ParseInt(envGenesis, 10, 64); err == nil {
			genesis = parsedGenesis
		}
	}
	return genesis
}
