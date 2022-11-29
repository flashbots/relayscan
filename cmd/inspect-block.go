package cmd

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/metachris/flashbotsrpc"
	"github.com/metachris/go-ethutils/addresslookup"
	"github.com/metachris/go-ethutils/utils"
	"github.com/metachris/relayscan/common"
	"github.com/metachris/relayscan/database"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	slotStr string
	// Printer for pretty printing numbers
	printer = message.NewPrinter(language.English)

	mevGethURI string
)

func init() {
	rootCmd.AddCommand(inspectBlockCmd)
	inspectBlockCmd.Flags().StringVar(&ethNodeURI, "eth-node", defaultEthNodeURI, "eth node URI (i.e. Infura)")
	inspectBlockCmd.Flags().StringVar(&ethNodeBackupURI, "eth-node-backup", defaultEthBackupNodeURI, "eth node backup URI (i.e. Infura)")
	inspectBlockCmd.Flags().StringVar(&mevGethURI, "mev-geth", "", "mev-geth node URI (to find coinbase payments via block simulation)")
	inspectBlockCmd.Flags().StringVar(&slotStr, "slot", "", "a specific slot")
}

var inspectBlockCmd = &cobra.Command{
	Use:   "inspect-block",
	Short: "Inspect a block",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		if slotStr == "" {
			log.Fatalf("Please provide a slot")
		}

		slotStr = strings.ReplaceAll(slotStr, ",", "")
		slot, err = strconv.ParseUint(slotStr, 10, 64)
		if err != nil {
			log.WithError(err).Fatalf("failed converting slot to uint")
		}

		if ethNodeURI == "" {
			log.Fatalf("Please provide an eth node URI")
		}
		ethUris := []string{ethNodeURI}
		if ethNodeBackupURI != "" {
			ethUris = append(ethUris, ethNodeBackupURI)
		}

		fmt.Printf("Connecting to eth nodes %v ... \n", ethUris)
		node, err := NewEthNode(ethUris...)
		if err != nil {
			log.WithError(err).Fatalf("failed connecting to eth nodes")
		}

		db, err := connectPostgres(defaultPostgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("failed connecting to postgres")
		}

		var mevGethRPC *flashbotsrpc.FlashbotsRPC
		if mevGethURI == "" {
			log.Warn("No mev-geth uri provided, cannot simulate block to find coinbase payments")
		} else {
			mevGethRPC = flashbotsrpc.New(mevGethURI)
		}

		inspector := NewBlockInspector(node, mevGethRPC, db)
		inspector.inspectSlot(slot)
	},
}

type BlockInspector struct {
	ethNode  *EthNode
	mevGeth  *flashbotsrpc.FlashbotsRPC
	db       *database.DatabaseService
	addrLkup *addresslookup.AddressLookupService
}

func NewBlockInspector(ethNode *EthNode, mevGeth *flashbotsrpc.FlashbotsRPC, db *database.DatabaseService) *BlockInspector {
	return &BlockInspector{
		ethNode:  ethNode,
		mevGeth:  mevGeth,
		db:       db,
		addrLkup: addresslookup.NewAddressLookupService(ethNode.clients[0]),
	}
}

func (b *BlockInspector) inspectSlot(slot uint64) {
	fmt.Println("Getting bids...")
	bids, err := b.db.GetSignedBuilderBidsForSlot(slot)
	if err != nil {
		log.WithError(err).Fatalf("couldn't get bids")
	}
	if len(bids) == 0 {
		fmt.Printf("no bids found for slot %d\n", slot)
		return
	}
	for _, bid := range bids {
		fmt.Printf("- %40s: %12s / %s\n", bid.Relay, common.WeiStrToEthStr(bid.Value, 6), bid.BlockHash)
	}

	fmt.Println("Getting delivered payload entries...")
	payloadDeliveredEntries, err := b.db.GetDeliveredPayloadsForSlot(slot)
	if err != nil {
		log.WithError(err).Fatalf("couldn't get payloads")
	}
	if len(payloadDeliveredEntries) == 0 {
		fmt.Printf("no delivered payload entries found for slot %d\n", slot)
		return
	}
	payload := payloadDeliveredEntries[0]
	for _, entry := range payloadDeliveredEntries {
		fmt.Printf("- %40s: %12s / %s\n", entry.Relay, common.WeiStrToEthStr(entry.ValueClaimedWei, 6), entry.BlockHash)
		if entry.BlockHash != payload.BlockHash {
			log.Fatalf("error: different blockhash: %s\n", entry.BlockHash)
		}
		if entry.ValueClaimedWei != payload.ValueClaimedWei {
			log.Fatalf("error: different value claimed: %s\n", entry.ValueClaimedWei)
		}
	}

	fmt.Println("")
	feeRec := payload.ProposerFeeRecipient
	fmt.Println("Proposer")
	fmt.Printf("- pubkey: %s\n", payload.ProposerPubkey)
	fmt.Printf("- feeRecipient: %s\n", feeRec)
	balanceDiff, err := b.ethNode.GetBalanceDiff(feeRec, payload.BlockNumber.Int64)
	if err != nil {
		log.WithError(err).Fatalf("couldn't get balance diff")
	}
	fmt.Printf("- balance diff: %s ETH\n", common.WeiToEthStr(balanceDiff))
	if balanceDiff.String() == payload.ValueClaimedWei {
		fmt.Println("- balance diff ✅")
	} else {
		log.Fatalf("error: balance diff does not match value claimed")
	}

	b.inspectBlockByHash(payload.BlockHash, feeRec)
}

func (b *BlockInspector) inspectBlockByHash(blockHash string, proposerFeeRecipient string) {
	fmt.Println("")
	fmt.Println("Getting block...")
	block, err := b.ethNode.BlockByHash(blockHash)
	if err != nil {
		log.WithError(err).Fatalf("couldn't get block")
	}

	fmt.Printf("- Block: %d %s \n", block.Number().Int64(), block.Hash().String())
	fmt.Printf("- Coinbase: %s \n", block.Coinbase().Hex())
	balanceDiff, err := b.ethNode.GetBalanceDiff(block.Coinbase().Hex(), block.Number().Int64())
	if err != nil {
		log.WithError(err).Fatalf("couldn't get balance diff")
	}
	if proposerFeeRecipient == block.Coinbase().Hex() {
		fmt.Printf("- Coinbase balance diff (proposer feeRec): %s ETH \n", common.WeiToEthStr(balanceDiff))
	} else {
		fmt.Printf("- Coinbase balance diff (builder): %s ETH \n", common.WeiToEthStr(balanceDiff))
	}
	fmt.Printf("- Gas used: %s / %s \n", printer.Sprint(block.GasUsed()), printer.Sprint(block.GasLimit()))

	totalTxValue := big.NewInt(0)
	totalTxValueToCoinbase := big.NewInt(0)
	totalTxValueToProposer := big.NewInt(0)
	numTxToCoinbase := 0
	numTxToProposer := 0
	// gasFee := big.NewInt(0)
	toAddresses := make(map[string]int)
	topToAddress := ""
	topToAddressCount := 0

	for _, tx := range block.Transactions() {
		txFrom, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
		totalTxValue.Add(totalTxValue, tx.Value())
		// gasFee = new(big.Int).Add(gasFee, new(big.Int).Mul(&tx.GasPrice, big.NewInt(int64(tx.Gas))))
		if tx.To() != nil && *tx.To() == block.Coinbase() {
			numTxToCoinbase += 1
			totalTxValueToCoinbase.Add(totalTxValueToCoinbase, tx.Value())
			fmt.Printf("- tx to coinbase: %s / %s ETH, from %s\n", tx.Hash(), common.WeiToEthStr(tx.Value()), txFrom)
		}
		if tx.To() != nil && tx.To().Hex() == proposerFeeRecipient {
			numTxToProposer += 1
			totalTxValueToProposer.Add(totalTxValueToProposer, tx.Value())
			fmt.Printf("- tx to proposer: %s / %s ETH, from %s\n", tx.Hash(), common.WeiToEth(tx.Value()).Text('f', 6), txFrom)
		}

		toAddresses[tx.To().Hex()] += 1
		if toAddresses[tx.To().Hex()] > topToAddressCount {
			topToAddress = tx.To().Hex()
			topToAddressCount = toAddresses[tx.To().Hex()]
		}
	}

	topToAddressWeiReceived := big.NewInt(0)
	for _, tx := range block.Transactions() {
		if tx.To().Hex() != topToAddress {
			continue
		}
		topToAddressWeiReceived.Add(topToAddressWeiReceived, tx.Value())
	}

	// fmt.Printf("- Total tx gas: %s", common.WeiToEthStr(gasFee))
	fmt.Printf("- Total tx value: %s ETH \n", common.WeiToEthStr(totalTxValue))
	fmt.Printf("- %d tx to coinbase - value: %s ETH \n", numTxToCoinbase, common.WeiToEthStr(totalTxValueToCoinbase))
	fmt.Printf("- %d tx to proposer - value: %s ETH \n", numTxToProposer, common.WeiToEthStr(totalTxValueToProposer))
	fmt.Printf("- Transactions: %d (to %d addresses) \n", len(block.Transactions()), len(toAddresses))

	a, _ := b.addrLkup.GetAddressDetail(topToAddress)
	fmt.Printf("- Top address (%d tx, %s ETH): %s (%s [%s]) \n", topToAddressCount, common.WeiToEthStr(topToAddressWeiReceived), topToAddress, a.Name, a.Type)

	if mevGethURI != "" {
		fmt.Println("")
		fmt.Println("Simulating block to find coinbase payments...")
		b.simBlock(block)
	}
}

func (b *BlockInspector) simBlock(block *types.Block) {
	txs := make([]string, 0)
	for _, tx := range block.Transactions() {
		// fmt.Println("tx", i, tx.Hash(), "type", tx.Type())
		// from, fromErr := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
		// txIsFromCoinbase := fromErr == nil && from == block.Coinbase()
		// if txIsFromCoinbase {
		// 	if rpc.Debug {
		// 		fmt.Printf("- skip tx from coinbase: %s\n", tx.Hash())
		// 	}
		// 	continue
		// }

		// to := tx.To()
		// txIsToCoinbase := to != nil && *to == block.Coinbase()
		// if txIsToCoinbase {
		// 	if rpc.Debug {
		// 		fmt.Printf("- skip tx to coinbase: %s\n", tx.Hash())
		// 	}
		// 	continue
		// }

		rlp := flashbotsrpc.TxToRlp(tx)

		// Might need to strip beginning bytes
		if rlp[:2] == "b9" {
			rlp = rlp[6:]
		} else if rlp[:2] == "b8" {
			rlp = rlp[4:]
		}

		// callBundle expects a 0x prefix
		rlp = "0x" + rlp
		txs = append(txs, rlp)

		// if maxTx > 0 && len(txs) == maxTx {
		// 	break
		// }
	}

	params := flashbotsrpc.FlashbotsCallBundleParam{
		Txs:              txs,
		BlockNumber:      fmt.Sprintf("0x%x", block.Number()),
		StateBlockNumber: block.ParentHash().Hex(),
		GasLimit:         block.GasLimit(),
		Difficulty:       block.Difficulty().Uint64(),
		BaseFee:          block.BaseFee().Uint64(),
	}

	privateKey, _ := crypto.GenerateKey()
	result, err := b.mevGeth.FlashbotsCallBundle(privateKey, params)
	utils.Perror(err)

	fmt.Println("Simulation result:")
	fmt.Printf("- CoinbaseDiff:      %10s ETH\n", common.WeiStrToEthStr(result.CoinbaseDiff, 4))
	fmt.Printf("- GasFees:           %10s ETH\n", common.WeiStrToEthStr(result.GasFees, 4))
	fmt.Printf("- EthSentToCoinbase: %10s ETH\n", common.WeiStrToEthStr(result.EthSentToCoinbase, 4))

	blockCbDiffWei := big.NewFloat(0)
	blockCbDiffWei, _ = blockCbDiffWei.SetString(result.CoinbaseDiff)

	// sort transactions by coinbasediff
	sort.Slice(result.Results, func(i, j int) bool {
		a := common.StrToBigInt(result.Results[i].CoinbaseDiff)
		b := common.StrToBigInt(result.Results[j].CoinbaseDiff)
		return a.Cmp(b) == 1
	})

	numTxNeededFor80PercentValue := 0
	currentValue := big.NewFloat(0)

	fmt.Println("\nTransactions by value, accounting for 80%% of coinbase value:")
	for i, entry := range result.Results {
		_to := entry.ToAddress
		detail, found := b.addrLkup.GetAddressDetail(entry.ToAddress)
		if found {
			_to = fmt.Sprintf("%s (%s [%s])", _to, detail.Name, detail.Type)
		}
		fmt.Printf("%4d %s - cbD=%8s, gasFee=%8s \t to=%-64s \n", i+1, entry.TxHash, common.WeiStrToEthStr(entry.CoinbaseDiff, 4), common.WeiStrToEthStr(entry.GasFees, 4), _to)

		cbDiffWei := new(big.Float)
		cbDiffWei, _ = cbDiffWei.SetString(entry.CoinbaseDiff)
		currentValue = new(big.Float).Add(currentValue, cbDiffWei)
		percentValueReached := new(big.Float).Quo(currentValue, blockCbDiffWei)
		if numTxNeededFor80PercentValue == 0 && percentValueReached.Cmp(big.NewFloat(0.8)) > -1 {
			numTxNeededFor80PercentValue = i + 1
			break
		}
	}

	fmt.Printf("\n%d/%d tx needed for 80%% of miner value.\n", numTxNeededFor80PercentValue, len(result.Results))
}
