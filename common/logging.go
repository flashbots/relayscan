package common

import (
	"os"

	"github.com/sirupsen/logrus"
)

func LogSetup(json bool, logLevel string, logDebug bool) *logrus.Entry {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetOutput(os.Stdout)

	if json {
		log.Logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	if logDebug {
		logLevel = "debug"
	}
	if logLevel != "" {
		lvl, err := logrus.ParseLevel(logLevel)
		if err != nil {
			log.Fatalf("Invalid loglevel: %s", logLevel)
		}
		log.Logger.SetLevel(lvl)
	}
	return log
}
