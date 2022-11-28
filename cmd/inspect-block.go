package cmd

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/metachris/go-ethutils/smartcontracts"
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
)

func init() {
	rootCmd.AddCommand(inspectBlockCmd)
	inspectBlockCmd.Flags().StringVar(&ethNodeURI, "eth-node", defaultEthNodeURI, "eth node URI (i.e. Infura)")
	inspectBlockCmd.Flags().StringVar(&ethNodeBackupURI, "eth-node-backup", defaultEthBackupNodeURI, "eth node backup URI (i.e. Infura)")
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

		log.Infof("Connecting to eth nodes %v ...", ethUris)
		node, err := NewEthNode(ethUris...)
		if err != nil {
			log.WithError(err).Fatalf("failed connecting to eth nodes")
		}

		db, err := connectPostgres(defaultPostgresDSN)
		if err != nil {
			log.WithError(err).Fatalf("failed connecting to postgres")
		}

		inspectBlockBySlot(slot, node, db)
	},
}

func inspectBlockBySlot(slot uint64, node *EthNode, db *database.DatabaseService) {
	log.Info("Getting bids...")
	bids, err := db.GetSignedBuilderBidsForSlot(slot)
	if err != nil {
		log.WithError(err).Fatalf("couldn't get bids")
	}
	if len(bids) == 0 {
		log.Infof("no bids found for slot %d", slot)
		return
	}
	for _, bid := range bids {
		log.Infof("- %40s: %12s / %s", bid.Relay, common.WeiStrToEthStr(bid.Value, 6), bid.BlockHash)
	}

	log.Info("Getting delivered payload entries...")
	payloadDeliveredEntries, err := db.GetDeliveredPayloadsForSlot(slot)
	if err != nil {
		log.WithError(err).Fatalf("couldn't get payloads")
	}
	if len(payloadDeliveredEntries) == 0 {
		log.Infof("no delivered payload entries found for slot %d", slot)
		return
	}
	payload := payloadDeliveredEntries[0]
	for _, entry := range payloadDeliveredEntries {
		log.Infof("- %40s: %12s / %s", entry.Relay, common.WeiStrToEthStr(entry.ValueClaimedWei, 6), entry.BlockHash)
		if entry.BlockHash != payload.BlockHash {
			log.Fatalf("error: different blockhash: %s", entry.BlockHash)
		}
		if entry.ValueClaimedWei != payload.ValueClaimedWei {
			log.Fatalf("error: different value claimed: %s", entry.ValueClaimedWei)
		}
	}

	fmt.Println("")
	feeRec := payload.ProposerFeeRecipient
	log.Infof("Proposer")
	log.Infof("- pubkey: %s", payload.ProposerPubkey)
	log.Infof("- feeRecipient: %s", feeRec)
	balanceDiff, err := node.GetBalanceDiff(feeRec, int(payload.BlockNumber.Int64))
	if err != nil {
		log.WithError(err).Fatalf("couldn't get balance diff")
	}
	log.Infof("- balance diff: %s ETH", common.WeiToEth(balanceDiff).Text('f', 6))
	if balanceDiff.String() == payload.ValueClaimedWei {
		log.Infof("- balance diff ✅")
	} else {
		log.Fatalf("error: balance diff does not match value claimed")
	}

	inspectBlockByHash(payload.BlockHash, feeRec, node, db)
}

func inspectBlockByHash(blockHash string, proposerFeeRecipient string, node *EthNode, db *database.DatabaseService) {
	fmt.Println("")
	log.Info("Getting block...")
	block, err := node.GetBlockByHash(blockHash, true)
	if err != nil {
		log.WithError(err).Fatalf("couldn't get block")
	}

	log.Infof("- Block: %d %s", block.Number, block.Hash)
	log.Infof("- Coinbase: %s", block.Miner)
	balanceDiff, err := node.GetBalanceDiff(block.Miner, block.Number)
	if err != nil {
		log.WithError(err).Fatalf("couldn't get balance diff")
	}
	if proposerFeeRecipient == block.Miner {
		log.Infof("- Coinbase balance diff (proposer feeRec): %s ETH", common.WeiToEth(balanceDiff).Text('f', 6))
	} else {
		log.Infof("- Coinbase balance diff (builder): %s ETH", common.WeiToEth(balanceDiff).Text('f', 6))
	}
	log.Infof("- Gas used: %s / %s", printer.Sprint(block.GasUsed), printer.Sprint(block.GasLimit))

	totalTxValue := big.NewInt(0)
	totalTxValueToCoinbase := big.NewInt(0)
	totalTxValueToProposer := big.NewInt(0)
	numTxToCoinbase := 0
	numTxToProposer := 0
	gasFee := big.NewInt(0)
	toAddresses := make(map[string]int)
	topToAddress := ""
	topToAddressCount := 0
	for _, tx := range block.Transactions {
		totalTxValue.Add(totalTxValue, &tx.Value)
		gasFee = new(big.Int).Add(gasFee, new(big.Int).Mul(&tx.GasPrice, big.NewInt(int64(tx.Gas))))
		if tx.To == block.Miner {
			numTxToCoinbase += 1
			totalTxValueToCoinbase.Add(totalTxValueToCoinbase, &tx.Value)
			log.Infof("- tx to coinbase: %s / %s ETH, from %s", tx.Hash, common.WeiToEth(&tx.Value).Text('f', 6), tx.From)
		}
		if tx.To == proposerFeeRecipient {
			numTxToProposer += 1
			totalTxValueToProposer.Add(totalTxValueToProposer, &tx.Value)
			log.Infof("- tx to proposer: %s / %s ETH, from %s", tx.Hash, common.WeiToEth(&tx.Value).Text('f', 6), tx.From)
		}
		toAddresses[tx.To] += 1
		if toAddresses[tx.To] > topToAddressCount {
			topToAddress = tx.To
			topToAddressCount = toAddresses[tx.To]
		}
	}

	log.Infof("- Total tx gas: %s", common.WeiToEth(gasFee).Text('f', 6))
	log.Infof("- Total tx value: %s ETH", common.WeiToEth(totalTxValue).Text('f', 6))
	log.Infof("- %d tx to coinbase - value: %s ETH", numTxToCoinbase, common.WeiToEth(totalTxValueToCoinbase).Text('f', 6))
	log.Infof("- %d tx to proposer - value: %s ETH", numTxToProposer, common.WeiToEth(totalTxValueToProposer).Text('f', 6))
	log.Infof("- Transactions: %d - to %d addresses", len(block.Transactions), len(toAddresses))

	a, _ := smartcontracts.GetAddressDetailFromBlockchain(topToAddress, node.gethClients[0])
	log.Infof("- Top address with %d tx: %s (%s [%s])", topToAddressCount, topToAddress, a.Name, a.Type)
}
