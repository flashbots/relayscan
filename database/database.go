// Package database exposes the postgres database
package database

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type IDatabaseService interface {
	SaveSignedBuilderBid(entry SignedBuilderBidEntry) error

	GetDataAPILatestPayloadDelivered(relay string) (*DataAPIPayloadDeliveredEntry, error)
	SaveDataAPIPayloadDelivered(entry *DataAPIPayloadDeliveredEntry) error
	SaveDataAPIPayloadDeliveredBatch(entries []*DataAPIPayloadDeliveredEntry) error

	GetDataAPILatestBid(relay string) (*DataAPIBuilderBidEntry, error)
	SaveDataAPIBid(entry *DataAPIBuilderBidEntry) error
	SaveDataAPIBids(entries []*DataAPIBuilderBidEntry) error
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

func (s *DatabaseService) SaveSignedBuilderBid(entry SignedBuilderBidEntry) error {
	query := `INSERT INTO ` + TableSignedBuilderBid + `
		(relay, requested_at, received_at, duration_ms, slot, parent_hash, proposer_pubkey, pubkey, signature, value, fee_recipient, block_hash, block_number, gas_limit, gas_used, extra_data, epoch) VALUES
		(:relay, :requested_at, :received_at, :duration_ms, :slot, :parent_hash, :proposer_pubkey, :pubkey, :signature, :value, :fee_recipient, :block_hash, :block_number, :gas_limit, :gas_used, :extra_data, :epoch)
		ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveDataAPIPayloadDelivered(entry *DataAPIPayloadDeliveredEntry) error {
	query := `INSERT INTO ` + TableDataAPIPayloadDelivered + `
		(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number) VALUES
		(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value_claimed_wei, :value_claimed_eth, :num_tx, :block_number)
		ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveDataAPIPayloadDeliveredBatch(entries []*DataAPIPayloadDeliveredEntry) error {
	if len(entries) == 0 {
		return nil
	}
	query := `INSERT INTO ` + TableDataAPIPayloadDelivered + `
	(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number) VALUES
	(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value_claimed_wei, :value_claimed_eth, :num_tx, :block_number)
	ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entries)
	return err
}

func (s *DatabaseService) GetDataAPILatestPayloadDelivered(relay string) (*DataAPIPayloadDeliveredEntry, error) {
	entry := new(DataAPIPayloadDeliveredEntry)
	query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number, slot_missed, value_check_ok, value_check_method, value_delivered_wei, value_delivered_eth, value_delivered_diff_wei, value_delivered_diff_eth, block_coinbase_addr, block_coinbase_is_proposer, coinbase_diff_wei, coinbase_diff_eth, found_onchain, was_uncled, block_hash_onchain, block_hash_onchain_diffs, notes FROM ` + TableDataAPIPayloadDelivered + ` WHERE relay=$1 ORDER BY slot DESC LIMIT 1`
	err := s.DB.Get(entry, query, relay)
	return entry, err
}

func (s *DatabaseService) SaveDataAPIBid(entry *DataAPIBuilderBidEntry) error {
	query := `INSERT INTO ` + TableDataAPIBuilderBid + `
		(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value, num_tx, block_number, timestamp) VALUES
		(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value, :num_tx, :block_number, :timestamp)
		ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveDataAPIBids(entries []*DataAPIBuilderBidEntry) error {
	if len(entries) == 0 {
		return nil
	}
	query := `INSERT INTO ` + TableDataAPIBuilderBid + `
	(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value, num_tx, block_number, timestamp) VALUES
	(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value, :num_tx, :block_number, :timestamp)
	ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entries)
	return err
}

func (s *DatabaseService) GetDataAPILatestBid(relay string) (*DataAPIBuilderBidEntry, error) {
	entry := new(DataAPIBuilderBidEntry)
	query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value, num_tx, block_number, timestamp FROM ` + TableDataAPIBuilderBid + ` WHERE relay=$1 ORDER BY slot DESC, timestamp DESC LIMIT 1`
	err := s.DB.Get(entry, query, relay)
	return entry, err
}
