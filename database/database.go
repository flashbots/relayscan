// Package database exposes the postgres database
package database

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type IDatabaseService interface {
	Close() error
}

type DatabaseService struct {
	DB *sqlx.DB
}

func NewDatabaseService(dsn string) (*DatabaseService, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.DB.SetMaxOpenConns(50)
	db.DB.SetMaxIdleConns(10)
	db.DB.SetConnMaxIdleTime(0)

	if os.Getenv("PRINT_SCHEMA") == "1" {
		fmt.Println(schema)
	}

	if os.Getenv("DB_DONT_APPLY_SCHEMA") == "" {
		_, err = db.Exec(schema)
		if err != nil {
			return nil, err
		}
	}

	return &DatabaseService{
		DB: db,
	}, nil
}

func (s *DatabaseService) Close() error {
	return s.DB.Close()
}

// func (s *DatabaseService) SaveValidatorRegistration(registration types.SignedValidatorRegistration) error {
// 	entry := ValidatorRegistrationEntry{
// 		Pubkey:       registration.Message.Pubkey.String(),
// 		FeeRecipient: registration.Message.FeeRecipient.String(),
// 		Timestamp:    registration.Message.Timestamp,
// 		GasLimit:     registration.Message.GasLimit,
// 		Signature:    registration.Signature.String(),
// 	}

// 	query := `INSERT INTO ` + TableValidatorRegistration + `
// 		(pubkey, fee_recipient, timestamp, gas_limit, signature) VALUES
// 		(:pubkey, :fee_recipient, :timestamp, :gas_limit, :signature)
// 		ON CONFLICT (pubkey, fee_recipient) DO NOTHING;`
// 	_, err := s.DB.NamedExec(query, entry)
// 	return err
// }
