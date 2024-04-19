package database

import (
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

func MustConnectPostgres(log *logrus.Entry, dsn string) *DatabaseService {
	dbURL, err := url.Parse(dsn)
	if err != nil {
		return nil
	}
	log.Infof("Connecting to Postgres database at %s%s ...", dbURL.Host, dbURL.Path)
	db, err := NewDatabaseService(dsn)
	if err != nil {
		log.WithError(err).Fatalf("Failed to connect to Postgres database at %s%s", dbURL.Host, dbURL.Path)
	}
	log.Infof("Connected to Postgres database at %s%s ✅", dbURL.Host, dbURL.Path)
	return db
}

func slotToTime(slot uint64) time.Time {
	timestamp := (slot * 12) + 1606824023 // mainnet
	return time.Unix(int64(timestamp), 0).UTC()
}

func timeToSlot(t time.Time) uint64 {
	return uint64(t.UTC().Unix()-1606824023) / 12 // mainnet
}
