package util

import (
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/relayscan/database"
	dbvars "github.com/flashbots/relayscan/database/vars"
	"github.com/flashbots/relayscan/vars"
	"github.com/metachris/flashbotsrpc"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var slot uint64

func init() {
	// rootCmd.AddCommand(backfillExtradataCmd)
	backfillExtradataCmd.Flags().StringVar(&ethNodeURI, "eth-node", vars.DefaultEthNodeURI, "eth node URI (i.e. Infura)")
	backfillExtradataCmd.Flags().StringVar(&ethNodeBackupURI, "eth-node-backup", vars.DefaultEthBackupNodeURI, "eth node backup URI (i.e. Infura)")
}

var backfillExtradataCmd = &cobra.Command{
	Use:   "backfill-extradata",
	Short: "Backfill extra_data",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		log.Infof("Using eth node: %s", ethNodeURI)
		client := flashbotsrpc.New(ethNodeURI)
		var client2 *flashbotsrpc.FlashbotsRPC
		if ethNodeBackupURI != "" {
			log.Infof("Using eth backup node: %s", ethNodeBackupURI)
			client2 = flashbotsrpc.New(ethNodeBackupURI)
		}
		_, _ = client, client2

		// Connect to Postgres
		db := database.MustConnectPostgres(log, vars.DefaultPostgresDSN)

		entries := []database.DataAPIPayloadDeliveredEntry{}
		query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number FROM ` + dbvars.TableDataAPIPayloadDelivered + ` WHERE slot < 4823872 AND extra_data = ''`
		// query += ` LIMIT 1000`
		err = db.DB.Select(&entries, query)
		if err != nil {
			log.WithError(err).Fatalf("couldn't get entries")
		}

		log.Infof("got %d entries", len(entries))
		if len(entries) == 0 {
			return
		}

		wg := new(sync.WaitGroup)
		entryC := make(chan database.DataAPIPayloadDeliveredEntry)
		if slot != 0 {
			numThreads = 1
		}
		for i := 0; i < int(numThreads); i++ {
			log.Infof("starting worker %d", i+1)
			wg.Add(1)
			go startBackfillWorker(wg, db, client, client2, entryC)
		}

		for _, entry := range entries {
			entryC <- entry
		}
		close(entryC)
		wg.Wait()
	},
}

func startBackfillWorker(wg *sync.WaitGroup, db *database.DatabaseService, client, client2 *flashbotsrpc.FlashbotsRPC, entryC chan database.DataAPIPayloadDeliveredEntry) {
	defer wg.Done()

	getBlockByHash := func(blockHash string, withTransactions bool) (*flashbotsrpc.Block, error) {
		block, err := client.EthGetBlockByHash(blockHash, withTransactions)
		if err != nil || block == nil {
			block, err = client2.EthGetBlockByHash(blockHash, withTransactions)
		}
		return block, err
	}

	var err error
	var block *flashbotsrpc.Block
	for entry := range entryC {
		_log := log.WithFields(logrus.Fields{
			"slot":        entry.Slot,
			"blockNumber": entry.BlockNumber.Int64,
			"blockHash":   entry.BlockHash,
			"relay":       entry.Relay,
		})
		_log.Infof("checking slot...")

		block, err = getBlockByHash(entry.BlockHash, true)
		if err != nil {
			_log.WithError(err).Fatalf("couldn't get block %s", entry.BlockHash)
		} else if block == nil {
			_log.WithError(err).Warnf("block not found: %s", entry.BlockHash)
			continue
		}

		extraDataBytes, err := hexutil.Decode(block.ExtraData)
		if err != nil {
			log.WithError(err).Errorf("failed to decode extradata %s", block.ExtraData)
		} else {
			entry.ExtraData = database.ExtraDataToUtf8Str(extraDataBytes)
			_log.Infof("id: %d, extradata: %s", entry.ID, entry.ExtraData)
			if entry.ExtraData == "" {
				continue
			}

			query := `UPDATE ` + dbvars.TableDataAPIPayloadDelivered + ` SET extra_data=$1 WHERE id=$2`
			_, err := db.DB.Exec(query, entry.ExtraData, entry.ID)
			if err != nil {
				_log.WithError(err).Fatalf("failed to save entry")
			}
		}
	}
}
