package core

import (
	"database/sql"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/flashbots/relayscan/vars"
	"github.com/metachris/flashbotsrpc"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	limit              uint64
	slotMax            uint64
	ethNodeURI         string
	ethNodeBackupURI   string
	checkIncorrectOnly bool
	checkMissedOnly    bool
	checkTx            bool
	checkAll           bool
	beaconNodeURI      string
)

func init() {
	checkPayloadValueCmd.Flags().Uint64Var(&slot, "slot", 0, "a specific slot")
	checkPayloadValueCmd.Flags().Uint64Var(&slotMax, "slot-max", 0, "a specific max slot, only check slots below this")
	checkPayloadValueCmd.Flags().Uint64Var(&limit, "limit", 1000, "how many payloads")
	checkPayloadValueCmd.Flags().Uint64Var(&numThreads, "threads", 10, "how many threads")
	checkPayloadValueCmd.Flags().StringVar(&ethNodeURI, "eth-node", vars.DefaultEthNodeURI, "eth node URI (i.e. Infura)")
	checkPayloadValueCmd.Flags().StringVar(&ethNodeBackupURI, "eth-node-backup", vars.DefaultEthBackupNodeURI, "eth node backup URI (i.e. Infura)")
	checkPayloadValueCmd.Flags().StringVar(&beaconNodeURI, "beacon-uri", vars.DefaultBeaconURI, "beacon endpoint")
	checkPayloadValueCmd.Flags().BoolVar(&checkIncorrectOnly, "check-incorrect", false, "whether to double-check incorrect values only")
	checkPayloadValueCmd.Flags().BoolVar(&checkMissedOnly, "check-missed", false, "whether to double-check missed slots only")
	checkPayloadValueCmd.Flags().BoolVar(&checkTx, "check-tx", false, "whether to check for tx from/to proposer feeRecipient")
	checkPayloadValueCmd.Flags().BoolVar(&checkAll, "check-all", false, "whether to check all entries")
}

var checkPayloadValueCmd = &cobra.Command{
	Use:   "check-payload-value",
	Short: "Check payload value for delivered payloads",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		log.Infof("Using eth node: %s", ethNodeURI)
		client := flashbotsrpc.New(ethNodeURI)
		var client2 *flashbotsrpc.FlashbotsRPC
		if ethNodeBackupURI != "" {
			log.Infof("Using eth backup node: %s", ethNodeBackupURI)
			client2 = flashbotsrpc.New(ethNodeBackupURI)
		}

		// Connect to Postgres
		db := database.MustConnectPostgres(log, vars.DefaultPostgresDSN)

		// Connect to BN
		// bn, headSlot := common.MustConnectBeaconNode(log, beaconNodeURI, false)
		// log.Infof("beacon node connected. headslot: %d", headSlot)

		entries := []database.DataAPIPayloadDeliveredEntry{}
		query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number FROM ` + database.TableDataAPIPayloadDelivered
		if checkIncorrectOnly {
			query += ` WHERE value_check_ok=false ORDER BY slot DESC`
			if limit > 0 {
				query += fmt.Sprintf(" limit %d", limit)
			}
			err = db.DB.Select(&entries, query)
		} else if checkMissedOnly {
			query += ` WHERE slot_missed=true ORDER BY slot DESC`
			if limit > 0 {
				query += fmt.Sprintf(" limit %d", limit)
			}
			err = db.DB.Select(&entries, query)
		} else if checkAll {
			if slotMax > 0 {
				query += fmt.Sprintf(" WHERE slot<=%d", slotMax)
			}
			query += ` ORDER BY slot DESC`
			if limit > 0 {
				query += fmt.Sprintf(" limit %d", limit)
			}
			err = db.DB.Select(&entries, query)
		} else if slot != 0 {
			query += ` WHERE slot=$1`
			err = db.DB.Select(&entries, query, slot)
		} else {
			// query += ` WHERE value_check_ok IS NULL AND slot_missed IS NULL ORDER BY slot DESC LIMIT $1`
			query += ` WHERE value_check_ok IS NULL ORDER BY slot DESC LIMIT $1`
			err = db.DB.Select(&entries, query, limit)
		}
		if err != nil {
			log.WithError(err).Fatalf("couldn't get entries")
		}

		log.Infof("query: %s", query)
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

func _getBalanceDiff(ethClient *flashbotsrpc.FlashbotsRPC, address string, blockNumber int) (*big.Int, error) {
	balanceBefore, err := ethClient.EthGetBalance(address, fmt.Sprintf("0x%x", blockNumber-1))
	if err != nil {
		return nil, fmt.Errorf("couldn't get balance for %s @ %d", address, blockNumber-1) //nolint
	}
	balanceAfter, err := ethClient.EthGetBalance(address, fmt.Sprintf("0x%x", blockNumber))
	if err != nil {
		return nil, fmt.Errorf("couldn't get balance for %s @ %d", address, blockNumber-1) //nolint
	}
	balanceDiff := new(big.Int).Sub(&balanceAfter, &balanceBefore)
	return balanceDiff, nil
}

// func startUpdateWorker(wg *sync.WaitGroup, db *database.DatabaseService, client, client2 *flashbotsrpc.FlashbotsRPC, entryC chan database.DataAPIPayloadDeliveredEntry, bn *beaconclient.ProdBeaconInstance) {
func startUpdateWorker(wg *sync.WaitGroup, db *database.DatabaseService, client, client2 *flashbotsrpc.FlashbotsRPC, entryC chan database.DataAPIPayloadDeliveredEntry) {
	defer wg.Done()

	getBalanceDiff := func(address string, blockNumber int) (*big.Int, error) {
		r, err := _getBalanceDiff(client, address, blockNumber)
		if err != nil {
			r, err = _getBalanceDiff(client2, address, blockNumber)
		}
		return r, err
	}

	getBlockByHash := func(blockHash string, withTransactions bool) (*flashbotsrpc.Block, error) {
		block, err := client.EthGetBlockByHash(blockHash, withTransactions)
		if err != nil || block == nil {
			block, err = client2.EthGetBlockByHash(blockHash, withTransactions)
		}
		return block, err
	}

	getBlockByNumber := func(blockNumber int, withTransactions bool) (*flashbotsrpc.Block, error) {
		block, err := client.EthGetBlockByNumber(blockNumber, withTransactions)
		if err != nil || block == nil {
			block, err = client2.EthGetBlockByNumber(blockNumber, withTransactions)
		}
		return block, err
	}

	saveEntry := func(_log *logrus.Entry, entry database.DataAPIPayloadDeliveredEntry) {
		query := `UPDATE ` + database.TableDataAPIPayloadDelivered + ` SET
				block_number=:block_number,
				extra_data=:extra_data,
				slot_missed=:slot_missed,
				value_check_ok=:value_check_ok,
				value_check_method=:value_check_method,
				value_delivered_wei=:value_delivered_wei,
				value_delivered_eth=:value_delivered_eth,
				value_delivered_diff_wei=:value_delivered_diff_wei,
				value_delivered_diff_eth=:value_delivered_diff_eth,
				block_coinbase_addr=:block_coinbase_addr,
				block_coinbase_is_proposer=:block_coinbase_is_proposer,
				coinbase_diff_wei=:coinbase_diff_wei,
				coinbase_diff_eth=:coinbase_diff_eth,
				found_onchain=:found_onchain -- should rename field, because getBlockByHash might succeed even though this slot was missed
				WHERE slot=:slot`
		_, err := db.DB.NamedExec(query, entry)
		if err != nil {
			_log.WithError(err).Fatalf("failed to save entry")
		}
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
		claimedProposerValue, ok := new(big.Int).SetString(entry.ValueClaimedWei, 10)
		if !ok {
			_log.Fatalf("couldn't convert claimed value to big.Int: %s", entry.ValueClaimedWei)
		}

		// // Check if slot was delivered
		// _log.Infof("%d - %d = %d", headSlot, entry.Slot, headSlot-entry.Slot)
		// if headSlot-entry.Slot < 30_000 { // before, my BN always returns the error
		// 	_, err := bn.GetHeaderForSlot(entry.Slot)
		// 	entry.SlotWasMissed = database.NewNullBool(false)
		// 	if err != nil {
		// 		if strings.Contains(err.Error(), "Could not find requested block") {
		// 			entry.SlotWasMissed = database.NewNullBool(true)
		// 			_log.Warn("couldn't find block in beacon node, probably missed the proposal!")
		// 		} else {
		// 			_log.WithError(err).Fatalf("couldn't get slot from BN")
		// 		}
		// 	}
		// }

		// query block by hash
		block, err = getBlockByHash(entry.BlockHash, true)
		if err != nil {
			_log.WithError(err).Fatalf("couldn't get block %s", entry.BlockHash)
		} else if block == nil {
			_log.WithError(err).Warnf("block not found: %s", entry.BlockHash)
			entry.FoundOnChain = database.NewNullBool(false)
			saveEntry(_log, entry)
			continue
		}

		entry.FoundOnChain = sql.NullBool{} //nolint:exhaustruct
		if !entry.BlockNumber.Valid {
			entry.BlockNumber = database.NewNullInt64(int64(block.Number))
		}

		entry.BlockCoinbaseAddress = database.NewNullString(block.Miner)
		coinbaseIsProposer := strings.EqualFold(block.Miner, entry.ProposerFeeRecipient)
		entry.BlockCoinbaseIsProposer = database.NewNullBool(coinbaseIsProposer)

		// query block by number to ensure that's what landed on-chain
		blockByNum, err := getBlockByNumber(block.Number, false)
		if err != nil {
			_log.WithError(err).Fatalf("couldn't get block by number %d", block.Number)
		} else if blockByNum == nil {
			_log.WithError(err).Warnf("block by number not found: %d", block.Number)
			continue
		} else if blockByNum.Hash != entry.BlockHash {
			_log.Warnf("block hash mismatch when checking by number. probably missed slot! entry hash: %s / by number: %s", entry.BlockHash, blockByNum.Hash)
			entry.SlotWasMissed = database.NewNullBool(true)
			saveEntry(_log, entry)
			continue
		}

		// Block was found on chain and is same for this blocknumber. Now check the payment!
		checkMethod := "balanceDiff"
		proposerBalanceDiffWei, err := getBalanceDiff(entry.ProposerFeeRecipient, block.Number)
		if err != nil {
			_log.WithError(err).Fatalf("couldn't get balance diff")
		}

		proposerValueDiffFromClaim := new(big.Int).Sub(claimedProposerValue, proposerBalanceDiffWei)
		if proposerValueDiffFromClaim.String() != "0" {
			// Value delivered is off. Might be due to a forwarder contract... Checking payment tx...
			checkMethod = "balanceDiff+txValue"
			isDeliveredValueIncorrect := true
			if len(block.Transactions) > 0 {
				paymentTx := block.Transactions[len(block.Transactions)-1]
				if paymentTx.To == entry.ProposerFeeRecipient {
					proposerValueDiffFromClaim = new(big.Int).Sub(claimedProposerValue, &paymentTx.Value)
					if proposerValueDiffFromClaim.String() == "0" {
						_log.Debug("all good, payment is in last tx but was probably forwarded through smart contract")
						isDeliveredValueIncorrect = false
					}
				}
			}

			if isDeliveredValueIncorrect {
				_log.Warnf("Value delivered to %s diffs by %s from claim. delivered: %s - claim: %s - relay: %s - slot: %d / block: %d", entry.ProposerFeeRecipient, proposerValueDiffFromClaim.String(), proposerBalanceDiffWei, entry.ValueClaimedWei, entry.Relay, entry.Slot, block.Number)
			}
		}

		// check for transactions to/from proposer feeRecipient
		if checkTx {
			log.Infof("checking %d tx...", len(block.Transactions))
			for i, tx := range block.Transactions {
				if tx.From == entry.ProposerFeeRecipient {
					_log.Infof("- tx %d from feeRecipient with value %s", i, tx.Value.String())
					proposerValueDiffFromClaim = new(big.Int).Add(proposerValueDiffFromClaim, &tx.Value)
				} else if tx.To == entry.ProposerFeeRecipient {
					_log.Infof("- tx %d to feeRecipient with value %s", i, tx.Value.String())
				}
			}
		}

		extraDataBytes, err := hexutil.Decode(block.ExtraData)
		if err != nil {
			log.WithError(err).Errorf("failed to decode extradata %s", block.ExtraData)
		} else {
			entry.ExtraData = database.ExtraDataToUtf8Str(extraDataBytes)
		}
		entry.ValueCheckOk = database.NewNullBool(proposerValueDiffFromClaim.String() == "0")
		entry.ValueCheckMethod = database.NewNullString(checkMethod)
		entry.ValueDeliveredWei = database.NewNullString(proposerBalanceDiffWei.String())
		entry.ValueDeliveredEth = database.NewNullString(common.WeiToEth(proposerBalanceDiffWei).String())
		entry.ValueDeliveredDiffWei = database.NewNullString(proposerValueDiffFromClaim.String())
		entry.ValueDeliveredDiffEth = database.NewNullString(common.WeiToEth(proposerValueDiffFromClaim).String())

		log.WithFields(logrus.Fields{
			"coinbaseIsProposer": coinbaseIsProposer,
			// "coinbase":                 block.Miner,
			// "proposerFeeRec(reported)": entry.ProposerFeeRecipient,
			"valueCheckOk":     entry.ValueCheckOk.Bool,
			"valueCheckMethod": entry.ValueCheckMethod.String,
			// "valueDeliveredWei":       entry.ValueDeliveredWei,
			"valueDeliveredEth": entry.ValueDeliveredEth.String,
			// "valueDeliveredDiffWei":   entry.ValueDeliveredDiffWei,
			"valueDeliveredDiffEth": entry.ValueDeliveredDiffEth.String,
		}).Info("value check done")

		if !coinbaseIsProposer {
			// Get builder balance diff
			builderBalanceDiffWei, err := getBalanceDiff(block.Miner, block.Number)
			if err != nil {
				_log.WithError(err).Fatalf("couldn't get balance diff")
			}
			// fmt.Println("builder diff", block.Miner, builderBalanceDiffWei)
			entry.CoinbaseDiffWei = database.NewNullString(builderBalanceDiffWei.String())
			entry.CoinbaseDiffEth = database.NewNullString(common.WeiToEth(builderBalanceDiffWei).String())
		}
		saveEntry(_log, entry)
	}
}
