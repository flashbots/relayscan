package main

import (
	"github.com/metachris/relayscan/cmd"
)

var Version = "dev" // is set during build process

func main() {
	cmd.Version = Version
	cmd.Execute()
}

// package main

// import (
// 	"flag"
// 	"net/url"
// 	"os"

// 	"github.com/metachris/relayscan/common"
// 	"github.com/metachris/relayscan/database"
// 	"github.com/metachris/relayscan/services/collector"
// 	"github.com/sirupsen/logrus"
// )

// var (
// 	// version = "dev" // is set during build process

// 	// Default values
// 	defaultDebug       = os.Getenv("DEBUG") == "1"
// 	defaultLogProd     = os.Getenv("LOG_PROD") == "1"
// 	defaultLogService  = os.Getenv("LOG_SERVICE")
// 	defaultPostgresDSN = os.Getenv("POSTGRES_DSN")

// 	// Flags
// 	debugPtr      = flag.Bool("debug", defaultDebug, "print debug output")
// 	logProdPtr    = flag.Bool("log-prod", defaultLogProd, "log in production mode (json)")
// 	logServicePtr = flag.String("log-service", defaultLogService, "'service' tag to logs")
// 	beaconNodeURI = flag.String("beacon-uri", "http://localhost:3500", "beacon node URI")
// 	postgresDSN   = flag.String("db", defaultPostgresDSN, "postgres DSN")
// )

// func main() {
// 	flag.Parse()

// 	logrus.SetOutput(os.Stdout)

// 	if *logProdPtr {
// 		logrus.SetFormatter(&logrus.JSONFormatter{})
// 	} else {
// 		logrus.SetFormatter(&logrus.TextFormatter{
// 			FullTimestamp: true,
// 		})
// 	}

// 	if *debugPtr {
// 		logrus.SetLevel(logrus.DebugLevel)
// 	}

// 	log := logrus.WithField("service", *logServicePtr)

// 	var err error
// 	relayEntries := make([]common.RelayEntry, len(relays))
// 	for i, relayStr := range relays {
// 		relayEntries[i], err = common.NewRelayEntry(relayStr)
// 		if err != nil {
// 			log.WithError(err).Fatal("failed to parse relay entry")
// 		}
// 	}

// 	// Connect to Postgres
// 	dbURL, err := url.Parse(*postgresDSN)
// 	if err != nil {
// 		log.WithError(err).Fatalf("couldn't read db URL")
// 	}
// 	log.Infof("Connecting to Postgres database at %s%s ...", dbURL.Host, dbURL.Path)
// 	db, err := database.NewDatabaseService(*postgresDSN)
// 	if err != nil {
// 		log.WithError(err).Fatalf("Failed to connect to Postgres database at %s%s", dbURL.Host, dbURL.Path)
// 	}

// 	srv := collector.NewRelayCheckerService(log, relayEntries, *beaconNodeURI, db)
// 	log.Info("Starting server...")
// 	srv.Start()
// }
