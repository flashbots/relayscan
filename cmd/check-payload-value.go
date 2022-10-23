package cmd

import (
	"fmt"
	"math/big"
	"net/url"
	"sync"

	"github.com/metachris/relayscan/common"
	"github.com/metachris/relayscan/database"
	"github.com/onrik/ethrpc"
	"github.com/spf13/cobra"
)

var (
	slot             uint64
	numPayloads      uint64
	numThreads       uint64
	ethNodeURI       string
	ethNodeBackupURI string
)

func init() {
	rootCmd.AddCommand(checkPayloadValueCmd)
	checkPayloadValueCmd.Flags().Uint64Var(&slot, "slot", 0, "a specific slot")
	checkPayloadValueCmd.Flags().Uint64Var(&numPayloads, "num", 1000, "how many payloads")
	checkPayloadValueCmd.Flags().Uint64Var(&numThreads, "threads", 10, "how many threads")
	checkPayloadValueCmd.Flags().StringVar(&ethNodeURI, "eth-node", defaultEthNodeURI, "eth node URI (i.e. Infura)")
	checkPayloadValueCmd.Flags().StringVar(&ethNodeBackupURI, "eth-node-backup", defaultEthBackupNodeURI, "eth node backup URI (i.e. Infura)")
}

var checkPayloadValueCmd = &cobra.Command{
	Use:   "check-payload-value",
	Short: "Check payload value for delivered payloads",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		log.Infof("Using eth node: %s", ethNodeURI)
		client := ethrpc.New(ethNodeURI)
		var client2 *ethrpc.EthRPC
		if ethNodeBackupURI != "" {
			log.Infof("Using eth backup node: %s", ethNodeBackupURI)
			client2 = ethrpc.New(ethNodeBackupURI)
		}

		// Connect to Postgres
		dbURL, err := url.Parse(postgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("couldn't read db URL")
		}
		log.Infof("Connecting to Postgres database at %s%s ...", dbURL.Host, dbURL.Path)
		db, err := database.NewDatabaseService(postgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("Failed to connect to Postgres database at %s%s", dbURL.Host, dbURL.Path)
		}

		var entries = []database.DataAPIPayloadDeliveredEntry{}
		query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number FROM ` + database.TableDataAPIPayloadDelivered
		if slot != 0 {
			query += ` WHERE slot=$1`
			err = db.DB.Select(&entries, query, slot)
		} else {
			query += ` WHERE value_check_ok IS NULL ORDER BY slot DESC LIMIT $1`
			err = db.DB.Select(&entries, query, numPayloads)
		}
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
			go startUpdateWorker(wg, db, client, client2, entryC)
		}

		for _, entry := range entries {
			entryC <- entry
		}
		close(entryC)
		wg.Wait()
	},
}

func _getBalanceDiff(ethClient *ethrpc.EthRPC, address string, blockNumber int) (*big.Int, error) {
	balanceBefore, err := ethClient.EthGetBalance(address, fmt.Sprintf("0x%x", blockNumber-1))
	if err != nil {
		return nil, fmt.Errorf("couldn't get balance for %s @ %d", address, blockNumber-1)
	}
	balanceAfter, err := ethClient.EthGetBalance(address, fmt.Sprintf("0x%x", blockNumber))
	if err != nil {
		return nil, fmt.Errorf("couldn't get balance for %s @ %d", address, blockNumber-1)
	}
	balanceDiff := new(big.Int).Sub(&balanceAfter, &balanceBefore)
	return balanceDiff, nil
}

func startUpdateWorker(wg *sync.WaitGroup, db *database.DatabaseService, client, client2 *ethrpc.EthRPC, entryC chan database.DataAPIPayloadDeliveredEntry) {
	getBalanceDiff := func(address string, blockNumber int) (*big.Int, error) {
		r, err := _getBalanceDiff(client, address, blockNumber)
		if err != nil {
			r, err = _getBalanceDiff(client2, address, blockNumber)
		}
		return r, err
	}

	var err error
	var block *ethrpc.Block
	for entry := range entryC {
		log.Infof("checking slot: %d / relay: %s", entry.Slot, entry.Relay)
		claimedProposerValue, ok := new(big.Int).SetString(entry.ValueClaimedWei, 10)
		if !ok {
			log.Fatalf("couldn't convert claimed value to big.Int: %s", entry.ValueClaimedWei)
		}

		// log.Infof("querying block for %s", entry.BlockHash)
		block, err = client.EthGetBlockByHash(entry.BlockHash, true)
		if err != nil {
			log.WithError(err).Fatalf("couldn't get block %s", entry.BlockHash)
		}

		// Get proposer balance diff
		proposerBalanceDiffWei, err := getBalanceDiff(entry.ProposerFeeRecipient, block.Number)
		if err != nil {
			log.WithError(err).Fatalf("couldn't get balance diff")
		}

		proposerValueDiffFromClaim := new(big.Int).Sub(claimedProposerValue, proposerBalanceDiffWei)
		// fmt.Println("claim", entry.ValueClaimedWei)
		// fmt.Println("diff ", proposerBalanceDiffWei)
		// fmt.Println("rema ", proposerValueDiffFromClaim)
		if proposerValueDiffFromClaim.String() != "0" {
			log.Warnf("Value delivered to %s diffs by %s from claim. delivered: %s - claim: %s - relay: %s - slot: %d", entry.ProposerFeeRecipient, proposerValueDiffFromClaim.String(), proposerBalanceDiffWei, entry.ValueClaimedWei, entry.Relay, entry.Slot)
		}

		entry.ValueCheckOk = database.NewNullBool(proposerValueDiffFromClaim.String() == "0")
		entry.ValueCheckMethod = database.NewNullString("balanceDiffV1")
		entry.ValueDeliveredWei = database.NewNullString(proposerBalanceDiffWei.String())
		entry.ValueDeliveredEth = database.NewNullString(common.WeiToEth(proposerBalanceDiffWei).String())
		entry.ValueDeliveredDiffWei = database.NewNullString(proposerValueDiffFromClaim.String())
		entry.ValueDeliveredDiffEth = database.NewNullString(common.WeiToEth(proposerValueDiffFromClaim).String())
		entry.BlockCoinbaseAddress = database.NewNullString(block.Miner)

		coinbaseIsProposer := block.Miner == entry.ProposerFeeRecipient
		entry.BlockCoinbaseIsProposer = database.NewNullBool(coinbaseIsProposer)
		if !coinbaseIsProposer {
			// Get builder balance diff
			builderBalanceDiffWei, err := getBalanceDiff(block.Miner, block.Number)
			if err != nil {
				log.WithError(err).Fatalf("couldn't get balance diff")
			}
			// fmt.Println("builder diff", block.Miner, builderBalanceDiffWei)
			entry.CoinbaseDiffWei = database.NewNullString(builderBalanceDiffWei.String())
			entry.CoinbaseDiffEth = database.NewNullString(common.WeiToEth(builderBalanceDiffWei).String())
		}

		query := `UPDATE ` + database.TableDataAPIPayloadDelivered + ` SET
				value_check_ok=:value_check_ok,
				value_check_method=:value_check_method,
				value_delivered_wei=:value_delivered_wei,
				value_delivered_eth=:value_delivered_eth,
				value_delivered_diff_wei=:value_delivered_diff_wei,
				value_delivered_diff_eth=:value_delivered_diff_eth,
				block_coinbase_addr=:block_coinbase_addr,
				block_coinbase_is_proposer=:block_coinbase_is_proposer,
				coinbase_diff_wei=:coinbase_diff_wei,
				coinbase_diff_eth=:coinbase_diff_eth
			WHERE slot=:slot`
		_, err = db.DB.NamedExec(query, entry)
		if err != nil {
			log.WithError(err).Fatalf("failed to save entry")
		}
	}

	wg.Done()
}
