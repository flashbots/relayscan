// Package database exposes the postgres database
package database

import (
	"os"
	"time"

	"github.com/flashbots/relayscan/database/migrations"
	"github.com/flashbots/relayscan/database/vars"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
)

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

	if os.Getenv("DB_DONT_APPLY_SCHEMA") == "" {
		migrate.SetTable(vars.TableMigrations)
		_, err := migrate.Exec(db.DB, "postgres", migrations.Migrations, migrate.Up)
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
	query := `INSERT INTO ` + vars.TableSignedBuilderBid + `
		(relay, requested_at, received_at, duration_ms, slot, parent_hash, proposer_pubkey, pubkey, signature, value, fee_recipient, block_hash, block_number, gas_limit, gas_used, extra_data, epoch, timestamp, prev_randao) VALUES
		(:relay, :requested_at, :received_at, :duration_ms, :slot, :parent_hash, :proposer_pubkey, :pubkey, :signature, :value, :fee_recipient, :block_hash, :block_number, :gas_limit, :gas_used, :extra_data, :epoch, :timestamp, :prev_randao)
		ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveBuilder(entry *BlockBuilderEntry) error {
	query := `INSERT INTO ` + vars.TableBlockBuilder + ` (builder_pubkey, description) VALUES (:builder_pubkey, :description) ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveDataAPIPayloadDelivered(entry *DataAPIPayloadDeliveredEntry) error {
	query := `INSERT INTO ` + vars.TableDataAPIPayloadDelivered + `
		(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number, extra_data) VALUES
		(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value_claimed_wei, :value_claimed_eth, :num_tx, :block_number, :extra_data)
		ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entry)
	return err
}

func (s *DatabaseService) SaveDataAPIPayloadDeliveredBatch(entries []*DataAPIPayloadDeliveredEntry) (rowsAffected int64, err error) {
	if len(entries) == 0 {
		return 0, nil
	}

	query := `INSERT INTO ` + vars.TableDataAPIPayloadDelivered + `
	(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number, extra_data) VALUES
	(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value_claimed_wei, :value_claimed_eth, :num_tx, :block_number, :extra_data)
	ON CONFLICT DO NOTHING`

	// Postgres can do max 65535 parameters at a time (otherwise error: "pq: got ... parameters but PostgreSQL only supports 65535 parameters")
	for i := 0; i < len(entries); i += 3000 {
		end := i + 3000
		if end > len(entries) {
			end = len(entries)
		}

		r, err := s.DB.NamedExec(query, entries[i:end])
		if err != nil {
			return 0, err
		}

		_rowsAffected, err := r.RowsAffected()
		if err != nil {
			return 0, err
		}

		rowsAffected += _rowsAffected
	}
	return rowsAffected, nil
}

func (s *DatabaseService) GetDataAPILatestPayloadDelivered(relay string) (*DataAPIPayloadDeliveredEntry, error) {
	entry := new(DataAPIPayloadDeliveredEntry)
	query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number, extra_data, slot_missed, value_check_ok, value_check_method, value_delivered_wei, value_delivered_eth, value_delivered_diff_wei, value_delivered_diff_eth, block_coinbase_addr, block_coinbase_is_proposer, coinbase_diff_wei, coinbase_diff_eth, found_onchain, notes FROM ` + vars.TableDataAPIPayloadDelivered + ` WHERE relay=$1 ORDER BY slot DESC LIMIT 1`
	err := s.DB.Get(entry, query, relay)
	return entry, err
}

func (s *DatabaseService) SaveDataAPIBid(entry *DataAPIBuilderBidEntry) error {
	query := `INSERT INTO ` + vars.TableDataAPIBuilderBid + `
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
	query := `INSERT INTO ` + vars.TableDataAPIBuilderBid + `
	(relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value, num_tx, block_number, timestamp) VALUES
	(:relay, :epoch, :slot, :parent_hash, :block_hash, :builder_pubkey, :proposer_pubkey, :proposer_fee_recipient, :gas_limit, :gas_used, :value, :num_tx, :block_number, :timestamp)
	ON CONFLICT DO NOTHING`
	_, err := s.DB.NamedExec(query, entries)
	return err
}

func (s *DatabaseService) GetDataAPILatestBid(relay string) (*DataAPIBuilderBidEntry, error) {
	entry := new(DataAPIBuilderBidEntry)
	query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value, num_tx, block_number, timestamp FROM ` + vars.TableDataAPIBuilderBid + ` WHERE relay=$1 ORDER BY slot DESC, timestamp DESC LIMIT 1`
	err := s.DB.Get(entry, query, relay)
	return entry, err
}

func (s *DatabaseService) GetTopRelays(since, until time.Time) (res []*TopRelayEntry, err error) {
	startSlot := timeToSlot(since)
	endSlot := timeToSlot(until)

	// query := `SELECT relay, count(relay) as payloads FROM ` + vars.TableDataAPIPayloadDelivered + ` WHERE inserted_at > $1 AND inserted_at < $2 GROUP BY relay ORDER BY payloads DESC;`
	query := `SELECT relay, count(relay) as payloads FROM ` + vars.TableDataAPIPayloadDelivered + ` WHERE value_check_ok IS NOT NULL AND slot >= $1 AND slot <= $2 GROUP BY relay ORDER BY payloads DESC;`
	err = s.DB.Select(&res, query, startSlot, endSlot)
	return res, err
}

func (s *DatabaseService) GetTopBuilders(since, until time.Time, relay string) (res []*TopBuilderEntry, err error) {
	startSlot := timeToSlot(since)
	endSlot := timeToSlot(until)

	query := `SELECT extra_data, count(extra_data) as blocks FROM (
		SELECT distinct(slot), extra_data FROM ` + vars.TableDataAPIPayloadDelivered + ` WHERE value_check_ok IS NOT NULL AND slot >= $1 AND slot <= $2`
	if relay != "" {
		query += ` AND relay = '` + relay + `'`
	}
	query += ` GROUP BY slot, extra_data
	) as x GROUP BY extra_data ORDER BY blocks DESC;`
	err = s.DB.Select(&res, query, startSlot, endSlot)
	return res, err
}

func (s *DatabaseService) GetBuilderProfits(since, until time.Time) (res []*BuilderProfitEntry, err error) {
	startSlot := timeToSlot(since)
	endSlot := timeToSlot(until)

	query := `SELECT
		extra_data,
		count(extra_data) as blocks,
		count(extra_data) filter (where coinbase_diff_eth > 0) as blocks_profit,
		count(extra_data) filter (where coinbase_diff_eth < 0) as blocks_sub,
		round(avg(CASE WHEN coinbase_diff_eth IS NOT NULL THEN coinbase_diff_eth ELSE 0 END), 4) as avg_profit_per_block,
		round(PERCENTILE_DISC(0.5) WITHIN GROUP(ORDER BY CASE WHEN coinbase_diff_eth IS NOT NULL THEN coinbase_diff_eth ELSE 0 END), 4) as median_profit_per_block,
		round(sum(CASE WHEN coinbase_diff_eth IS NOT NULL THEN coinbase_diff_eth ELSE 0 END), 4) as total_profit,
		round(abs(sum(CASE WHEN coinbase_diff_eth < 0 THEN coinbase_diff_eth ELSE 0 END)), 4) as total_subsidies
	FROM (
		SELECT distinct(slot), extra_data, coinbase_diff_eth FROM ` + vars.TableDataAPIPayloadDelivered + ` WHERE value_check_ok IS NOT NULL AND slot >= $1 AND slot <= $2
	) AS x
	GROUP BY extra_data
	ORDER BY total_profit DESC;`
	err = s.DB.Select(&res, query, startSlot, endSlot)
	return res, err
}

func (s *DatabaseService) GetStatsForTimerange(since, until time.Time, relay string) (relays []*TopRelayEntry, builders []*TopBuilderEntry, builderProfits []*BuilderProfitEntry, err error) {
	relays, err = s.GetTopRelays(since, until)
	if err != nil {
		return nil, nil, nil, err
	}
	builders, err = s.GetTopBuilders(since, until, relay)
	if err != nil {
		return nil, nil, nil, err
	}
	builderProfits, err = s.GetBuilderProfits(since, until)
	if err != nil {
		return nil, nil, nil, err
	}

	return relays, builders, builderProfits, nil
}

func (s *DatabaseService) GetDeliveredPayloadsForSlot(slot uint64) (res []*DataAPIPayloadDeliveredEntry, err error) {
	query := `SELECT
		id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number
	FROM ` + vars.TableDataAPIPayloadDelivered + ` WHERE slot=$1;`
	err = s.DB.Select(&res, query, slot)
	return res, err
}

func (s *DatabaseService) GetLatestDeliveredPayload() (*DataAPIPayloadDeliveredEntry, error) {
	query := `SELECT
		id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number
	FROM ` + vars.TableDataAPIPayloadDelivered + ` WHERE value_check_ok IS NOT NULL ORDER BY slot DESC LIMIT 1;`
	entry := new(DataAPIPayloadDeliveredEntry)
	err := s.DB.Get(entry, query)
	return entry, err
}

func (s *DatabaseService) GetDeliveredPayloadsForSlots(slotStart, slotEnd uint64) (res []*DataAPIPayloadDeliveredEntry, err error) {
	query := `SELECT
		id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number, extra_data
	FROM ` + vars.TableDataAPIPayloadDelivered + ` WHERE slot>=$1 AND slot<=$2 ORDER BY slot ASC;`
	err = s.DB.Select(&res, query, slotStart, slotEnd)
	return res, err
}

func (s *DatabaseService) GetSignedBuilderBidsForSlot(slot uint64) (res []*SignedBuilderBidEntry, err error) {
	query := `SELECT
		id, relay, requested_at, received_at, duration_ms, slot, parent_hash, proposer_pubkey, pubkey, signature, value, fee_recipient, block_hash, block_number, gas_limit, gas_used, extra_data, epoch, timestamp, prev_randao
	FROM ` + vars.TableSignedBuilderBid + ` WHERE slot=$1;`
	err = s.DB.Select(&res, query, slot)
	return res, err
}

func (s *DatabaseService) SaveBuilderStats(entries []*BuilderStatsEntry) error {
	if len(entries) == 0 {
		return nil
	}
	query := `INSERT INTO ` + vars.TableBlockBuilderInclusionStats + `
	(type, hours, time_start, time_end, builder_name, extra_data, builder_pubkeys, blocks_included) VALUES
	(:type, :hours, :time_start, :time_end, :builder_name, :extra_data, :builder_pubkeys, :blocks_included)
		ON CONFLICT (type, hours, time_start, time_end, builder_name) DO UPDATE SET
		builder_pubkeys = EXCLUDED.builder_pubkeys,
		extra_data = EXCLUDED.extra_data,
		blocks_included = EXCLUDED.blocks_included;`
	_, err := s.DB.NamedExec(query, entries)
	return err
}

func (s *DatabaseService) GetLastDailyBuilderStatsEntry(filterType string) (*BuilderStatsEntry, error) {
	query := `SELECT type, hours, time_start, time_end, builder_name, extra_data, builder_pubkeys, blocks_included FROM ` + vars.TableBlockBuilderInclusionStats + ` WHERE hours=24 AND type=$1 ORDER BY time_end DESC LIMIT 1;`
	entry := new(BuilderStatsEntry)
	err := s.DB.Get(entry, query, filterType)
	return entry, err
}

func (s *DatabaseService) GetRecentPayloadsForExtraData(extraData []string, limit int) (resp []*TmpPayloadsForExtraDataEntry, err error) {
	query := `
		SELECT
			DISTINCT ON (slot) slot, extra_data, inserted_at, block_timestamp
		FROM ` + vars.TableDataAPIPayloadDelivered + `
		wHERE extra_data IN (?)
		ORDER BY slot DESC
		LIMIT ?;`

	// See also https://jmoiron.github.io/sqlx/#inQueries
	query, args, err := sqlx.In(query, extraData, limit)
	if err != nil {
		return nil, err
	}
	query = s.DB.Rebind(query)
	err = s.DB.Select(&resp, query, args...)
	return resp, err
}

func (s *DatabaseService) SaveAdjustments(entries []*AdjustmentEntry) error {
    if len(entries) == 0 {
        return nil
    }
    query := `INSERT INTO ` + vars.TableAdjustments + `
    (slot, adjusted_block_hash, adjusted_value, block_number, builder_pubkey, delta, submitted_block_hash, submitted_received_at, submitted_value) VALUES
    (:slot, :adjusted_block_hash, :adjusted_value, :block_number, :builder_pubkey, :delta, :submitted_block_hash, :submitted_received_at, :submitted_value)
    ON CONFLICT (slot, adjusted_block_hash) DO NOTHING`
    _, err := s.DB.NamedExec(query, entries)
    return err
}

func (s *DatabaseService) GetLatestAdjustmentSlot() (uint64, error) {
    var slot uint64
    query := `SELECT COALESCE(MAX(slot), 0) FROM ` + vars.TableAdjustments
    err := s.DB.Get(&slot, query)
    return slot, err
}
