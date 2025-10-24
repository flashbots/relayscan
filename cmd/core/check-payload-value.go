package core

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	dbvars "github.com/flashbots/relayscan/database/vars"
	"github.com/flashbots/relayscan/vars"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	limit              uint64
	slotMax            uint64
	slotMin            uint64
	ethNodeURI         string
	ethNodeBackupURI   string
	checkIncorrectOnly bool
	checkMissedOnly    bool
	checkTx            bool
	checkAll           bool
)

func init() {
	checkPayloadValueCmd.Flags().Uint64Var(&slot, "slot", 0, "a specific slot")
	checkPayloadValueCmd.Flags().Uint64Var(&slotMax, "slot-max", 0, "a specific max slot, only check slots before (only works with --check-all)")
	checkPayloadValueCmd.Flags().Uint64Var(&slotMin, "slot-min", 0, "only check slots after this one")
	checkPayloadValueCmd.Flags().Uint64Var(&limit, "limit", 1000, "how many payloads")
	checkPayloadValueCmd.Flags().Uint64Var(&numThreads, "threads", 10, "how many threads")
	checkPayloadValueCmd.Flags().StringVar(&ethNodeURI, "eth-node", vars.DefaultEthNodeURI, "eth node URI (i.e. Infura)")
	checkPayloadValueCmd.Flags().StringVar(&ethNodeBackupURI, "eth-node-backup", vars.DefaultEthBackupNodeURI, "eth node backup URI (i.e. Infura)")
	checkPayloadValueCmd.Flags().BoolVar(&checkIncorrectOnly, "check-incorrect", false, "whether to double-check incorrect values only")
	checkPayloadValueCmd.Flags().BoolVar(&checkMissedOnly, "check-missed", false, "whether to double-check missed slots only")
	checkPayloadValueCmd.Flags().BoolVar(&checkTx, "check-tx", false, "whether to check for tx from/to proposer feeRecipient")
	checkPayloadValueCmd.Flags().BoolVar(&checkAll, "check-all", false, "whether to check all entries")
}

var checkPayloadValueCmd = &cobra.Command{
	Use:   "check-payload-value",
	Short: "Check payload value for delivered payloads",
	Run:   checkPayloadValue,
}

func checkPayloadValue(cmd *cobra.Command, args []string) {
	var err error
	startTime := time.Now().UTC()

	client, err := ethclient.Dial(ethNodeURI)
	if err != nil {
		log.Fatalf("Failed to create RPC client for '%s'", ethNodeURI)
	}
	log.Infof("Using eth node: %s", ethNodeURI)

	client2 := client
	if ethNodeBackupURI != "" {
		client2, err = ethclient.Dial(ethNodeBackupURI)
		if err != nil {
			log.Fatalf("Failed to create backup RPC client for '%s'", ethNodeBackupURI)
		}
		log.Infof("Using eth backup node: %s", ethNodeBackupURI)
	}

	// Connect to Postgres
	db := database.MustConnectPostgres(log, vars.DefaultPostgresDSN)

	entries := []database.DataAPIPayloadDeliveredEntry{}
	query := `SELECT id, inserted_at, relay, epoch, slot, parent_hash, block_hash, builder_pubkey, proposer_pubkey, proposer_fee_recipient, gas_limit, gas_used, value_claimed_wei, value_claimed_eth, num_tx, block_number FROM ` + dbvars.TableDataAPIPayloadDelivered
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
	for i := 0; i < int(numThreads); i++ { //nolint:gosec,intrange
		log.Infof("starting worker %d", i+1)
		wg.Add(1)
		go startUpdateWorker(wg, db, client, client2, entryC)
	}

	for _, entry := range entries {
		// possibly skip
		if slotMin != 0 && entry.Slot < slotMin {
			continue
		}

		entryC <- entry
	}
	close(entryC)
	wg.Wait()

	timeNeeded := time.Since(startTime)
	log.WithField("timeNeeded", timeNeeded).Info("All done!")
}

func _getBalanceDiff(ethClient *ethclient.Client, address ethcommon.Address, blockNumber *big.Int) (*big.Int, error) {
	blockNumberMinusOne := new(big.Int).Sub(blockNumber, big.NewInt(1))

	balanceBefore, err := ethClient.BalanceAt(context.TODO(), address, blockNumberMinusOne)
	if err != nil {
		return nil, fmt.Errorf("couldn't get balance for %s @ %d", address, blockNumberMinusOne) //nolint
	}

	balanceAfter, err := ethClient.BalanceAt(context.TODO(), address, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("couldn't get balance for %s @ %d", address, blockNumber) //nolint
	}
	balanceDiff := new(big.Int).Sub(balanceAfter, balanceBefore)
	return balanceDiff, nil
}

// func startUpdateWorker(wg *sync.WaitGroup, db *database.DatabaseService, client, client2 *flashbotsrpc.FlashbotsRPC, entryC chan database.DataAPIPayloadDeliveredEntry, bn *beaconclient.ProdBeaconInstance) {
func startUpdateWorker(wg *sync.WaitGroup, db *database.DatabaseService, client, client2 *ethclient.Client, entryC chan database.DataAPIPayloadDeliveredEntry) {
	defer wg.Done()

	getBalanceDiff := func(address ethcommon.Address, blockNumber *big.Int) (*big.Int, error) {
		r, err := _getBalanceDiff(client, address, blockNumber)
		if err != nil {
			r, err = _getBalanceDiff(client2, address, blockNumber)
		}
		return r, err
	}

	getBlockByHash := func(blockHashHex string) (*types.Block, error) {
		blockHash := ethcommon.HexToHash(blockHashHex)
		block, err := client.BlockByHash(context.Background(), blockHash)
		if err != nil || block == nil {
			block, err = client2.BlockByHash(context.Background(), blockHash)
		}
		return block, err
	}

	getHeaderByNumber := func(blockNumber *big.Int) (*types.Header, error) {
		block, err := client.HeaderByNumber(context.Background(), blockNumber)
		if err != nil || block == nil {
			block, err = client2.HeaderByNumber(context.Background(), blockNumber)
		}
		return block, err
	}

	saveEntry := func(_log *logrus.Entry, entry database.DataAPIPayloadDeliveredEntry) {
		query := `UPDATE ` + dbvars.TableDataAPIPayloadDelivered + ` SET
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
				found_onchain=:found_onchain, -- should rename field, because getBlockByHash might succeed even though this slot was missed
				num_blob_txs=:num_blob_txs,
				num_blobs=:num_blobs,
				block_timestamp=:block_timestamp
				WHERE slot=:slot`
		_, err := db.DB.NamedExec(query, entry)
		if err != nil {
			_log.WithError(err).Fatalf("failed to save entry")
		}
	}

	var err error
	var block *types.Block
	for entry := range entryC {
		_log := log.WithFields(logrus.Fields{
			"slot":        entry.Slot,
			"blockNumber": entry.BlockNumber.Int64,
			"blockHash":   entry.BlockHash,
			"relay":       entry.Relay,
		})
		_log.Infof("checking slot %d ...", entry.Slot)
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
		block, err = getBlockByHash(entry.BlockHash)
		if err != nil {
			if err.Error() == "not found" {
				_log.WithError(err).Warnf("block by hash not found: %s", entry.BlockHash)
				_log.WithError(err).Warnf("block not found: %s", entry.BlockHash)
				entry.FoundOnChain = database.NewNullBool(false)
				saveEntry(_log, entry)
				continue
			} else {
				_log.WithError(err).Fatalf("error querying block by hash: %s", entry.BlockHash)
			}
		}

		// We found this block by hash, it's on chain
		entry.FoundOnChain = database.NewNullBool(true)

		if !entry.BlockNumber.Valid {
			entry.BlockNumber = database.NewNullInt64(block.Number().Int64())
		}

		entry.BlockCoinbaseAddress = database.NewNullString(block.Coinbase().Hex())
		coinbaseIsProposer := strings.EqualFold(block.Coinbase().Hex(), entry.ProposerFeeRecipient)
		entry.BlockCoinbaseIsProposer = database.NewNullBool(coinbaseIsProposer)

		entryBlockHash := ethcommon.HexToHash(entry.BlockHash)

		// query block by number to ensure that's what landed on-chain
		//
		// TODO: This reports "slot is missed" when actually an EL block with that number is there, but the hash is different.
		//       Should refactor this to instead say elBlockHashMismatch (and save both hashes)
		blockByNum, err := getHeaderByNumber(block.Number())
		if err != nil {
			_log.WithError(err).Fatalf("couldn't get block by number %d", block.NumberU64())
		} else if blockByNum == nil {
			_log.WithError(err).Warnf("block by number not found: %d", block.NumberU64())
			continue
		} else if blockByNum.Hash() != entryBlockHash {
			_log.Warnf("block hash mismatch when checking by number. probably missed slot! entry hash: %s / by number: %s", entry.BlockHash, blockByNum.Hash().Hex())
			entry.SlotWasMissed = database.NewNullBool(true)
			saveEntry(_log, entry)
			continue
		}

		// Block was found on chain and is same for this blocknumber. Now check the payment!
		checkMethod := "balanceDiff"
		proposerFeeRecipientAddr := ethcommon.HexToAddress(entry.ProposerFeeRecipient)
		proposerBalanceDiffWei, err := getBalanceDiff(proposerFeeRecipientAddr, block.Number())
		if err != nil {
			_log.WithError(err).Fatalf("couldn't get balance diff")
		}

		txs := block.Transactions()

		proposerValueDiffFromClaim := new(big.Int).Sub(claimedProposerValue, proposerBalanceDiffWei)
		if proposerValueDiffFromClaim.String() != "0" {
			// Value delivered is off. Might be due to a forwarder contract... Checking payment tx...
			checkMethod = "balanceDiff+txValue"
			isDeliveredValueIncorrect := true
			if len(txs) > 0 {
				paymentTx := txs[len(txs)-1]
				if paymentTx.To().Hex() == entry.ProposerFeeRecipient {
					proposerValueDiffFromClaim = new(big.Int).Sub(claimedProposerValue, paymentTx.Value())
					if proposerValueDiffFromClaim.String() == "0" {
						_log.Debug("all good, payment is in last tx but was probably forwarded through smart contract")
						isDeliveredValueIncorrect = false
					}
				}
			}

			if isDeliveredValueIncorrect {
				_log.Warnf("Value delivered to %s diffs by %s from claim. delivered: %s - claim: %s - relay: %s - slot: %d / block: %d", entry.ProposerFeeRecipient, proposerValueDiffFromClaim.String(), proposerBalanceDiffWei, entry.ValueClaimedWei, entry.Relay, entry.Slot, block.NumberU64())
			}
		}

		// check for transactions to/from proposer feeRecipient
		if checkTx {
			log.Infof("checking %d tx...", len(txs))

			for i, tx := range txs {
				if tx.ChainId().Uint64() == 0 {
					continue
				}
				txFrom, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
				if txFrom.Hex() == entry.ProposerFeeRecipient {
					_log.Infof("- tx %d from feeRecipient with value %s", i, tx.Value().String())
					proposerValueDiffFromClaim = new(big.Int).Add(proposerValueDiffFromClaim, tx.Value())
				} else if tx.To().Hex() == entry.ProposerFeeRecipient {
					_log.Infof("- tx %d to feeRecipient with value %s", i, tx.Value().String())
				}
			}
		}

		// find number of blob transactions
		numBlobTxs := 0
		numBlobs := 0
		for _, tx := range txs {
			if tx.Type() == types.BlobTxType {
				numBlobTxs++
				numBlobs += len(tx.BlobHashes())
			}
		}
		entry.NumBlobTxs = database.NewNullInt64(int64(numBlobTxs))
		entry.NumBlobs = database.NewNullInt64(int64(numBlobs))

		entry.ExtraData = database.ExtraDataToUtf8Str(block.Extra())
		entry.ValueCheckOk = database.NewNullBool(proposerValueDiffFromClaim.String() == "0")
		entry.ValueCheckMethod = database.NewNullString(checkMethod)
		entry.ValueDeliveredWei = database.NewNullString(proposerBalanceDiffWei.String())
		entry.ValueDeliveredEth = database.NewNullString(common.WeiToEth(proposerBalanceDiffWei).String())
		entry.ValueDeliveredDiffWei = database.NewNullString(proposerValueDiffFromClaim.String())
		entry.ValueDeliveredDiffEth = database.NewNullString(common.WeiToEth(proposerValueDiffFromClaim).String())

		// set block timestamp
		blockTime := time.Unix(int64(block.Time()), 0).UTC() //nolint:gosec
		entry.BlockTimestamp = database.NewNullTime(blockTime)

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
			"numBlobTxs":            numBlobTxs,
			"numBlobs":              numBlobs,
		}).Info("value check done")

		if !coinbaseIsProposer {
			// Get builder profit/subsidy, taking into account possible tx from coinbase to builder-owned address
			// First, get the overall balance diff
			builderBalanceDiffWei, err := getBalanceDiff(block.Coinbase(), block.Number())
			if err != nil {
				_log.WithError(err).Fatalf("couldn't get balance diff")
			}

			// Second, adjust for any tx from coinbase to builder-owned address.
			builderOwnedAddresses := vars.BuilderAddresses[strings.ToLower(block.Coinbase().Hex())]
			for _, tx := range txs {
				if tx.ChainId().Uint64() == 0 {
					continue
				}

				txFrom, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
				isFromBuilderCoinbase := txFrom.Hex() == block.Coinbase().Hex()
				isToBuilderOwnedAddress := false
				if builderOwnedAddresses != nil && builderOwnedAddresses[strings.ToLower(tx.To().Hex())] {
					isToBuilderOwnedAddress = true
				}

				if isFromBuilderCoinbase && isToBuilderOwnedAddress {
					_log.Infof("adjusting builder profit for tx from coinbase to builder address: %s", tx.Hash().Hex())
					builderBalanceDiffWei = new(big.Int).Add(builderBalanceDiffWei, tx.Value())
				}
			}

			// save
			entry.CoinbaseDiffWei = database.NewNullString(builderBalanceDiffWei.String())
			entry.CoinbaseDiffEth = database.NewNullString(common.WeiToEth(builderBalanceDiffWei).String())
		}
		saveEntry(_log, entry)
	}
}
