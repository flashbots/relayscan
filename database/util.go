package database

import (
	"net/url"

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
