package main

import (
	"flag"
	"os"

	"github.com/metachris/relayscan/common"
	"github.com/metachris/relayscan/database"
	"github.com/metachris/relayscan/services/collector"
	"github.com/sirupsen/logrus"
)

var (
	// version = "dev" // is set during build process

	// Default values
	defaultDebug      = os.Getenv("DEBUG") == "1"
	defaultLogProd    = os.Getenv("LOG_PROD") == "1"
	defaultLogService = os.Getenv("LOG_SERVICE")

	// Flags
	debugPtr      = flag.Bool("debug", defaultDebug, "print debug output")
	logProdPtr    = flag.Bool("log-prod", defaultLogProd, "log in production mode (json)")
	logServicePtr = flag.String("log-service", defaultLogService, "'service' tag to logs")
	beaconNodeURI = flag.String("beacon-uri", "http://localhost:3500", "beacon node URI")
)

func main() {
	flag.Parse()

	logrus.SetOutput(os.Stdout)

	if *logProdPtr {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	if *debugPtr {
		logrus.SetLevel(logrus.DebugLevel)
	}

	log := logrus.WithField("service", *logServicePtr)

	relays := []string{
		"https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net",
		"https://0x8b5d2e73e2a3a55c6c87b8b6eb92e0149a125c852751db1422fa951e42a09b82c142c3ea98d0d9930b056a3bc9896b8f@bloxroute.max-profit.blxrbdn.com",
	}

	var err error
	relayEntries := make([]common.RelayEntry, len(relays))
	for i, relayStr := range relays {
		relayEntries[i], err = common.NewRelayEntry(relayStr)
		if err != nil {
			log.WithError(err).Fatal("failed to parse relay entry")
		}
	}

	srv := collector.NewRelayCheckerService(log, relayEntries, *beaconNodeURI, &database.MockDB{})
	log.Info("Starting server...")
	srv.Start()
}
