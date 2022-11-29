package cmd

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/metachris/relayscan/common"
	"github.com/metachris/relayscan/database"
)

var (
	Version       = "dev" // is set during build process
	log           = common.LogSetup(logJSON, defaultLogLevel, logDebug)
	logDebug      = os.Getenv("DEBUG") != ""
	logJSON       = os.Getenv("LOG_JSON") != ""
	beaconNodeURI string
	slot          uint64

	defaultBeaconURI        = relaycommon.GetEnv("BEACON_URI", "http://localhost:3500")
	defaultPostgresDSN      = relaycommon.GetEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	defaultLogLevel         = relaycommon.GetEnv("LOG_LEVEL", "info")
	defaultEthNodeURI       = relaycommon.GetEnv("ETH_NODE_URI", "")
	defaultEthBackupNodeURI = relaycommon.GetEnv("ETH_NODE_BACKUP_URI", "")
)

func connectPostgres(dsn string) (*database.DatabaseService, error) {
	dbURL, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	log.Infof("Connecting to Postgres database at %s%s ...", dbURL.Host, dbURL.Path)
	return database.NewDatabaseService(defaultPostgresDSN)
}

type EthNode struct {
	clients []*ethclient.Client
}

func NewEthNode(uris ...string) (*EthNode, error) {
	if len(uris) == 0 {
		return nil, errors.New("uri1 is empty")
	}
	node := &EthNode{}
	for _, uri := range uris {
		client, err := ethclient.Dial(uri)
		if err != nil {
			fmt.Println("Error connecting to eth node", uri, err)
			return nil, err
		}
		node.clients = append(node.clients, client)
	}
	return node, nil
}

func (n *EthNode) BlockByNumber(blockNumber int64) (block *types.Block, err error) {
	for _, client := range n.clients {
		block, err = client.BlockByNumber(context.Background(), big.NewInt(blockNumber))
		if err == nil {
			return block, nil
		}
	}
	return nil, err
}

func (n *EthNode) BlockByHash(blockHash string) (block *types.Block, err error) {
	for _, client := range n.clients {
		block, err = client.BlockByHash(context.Background(), ethcommon.HexToHash(blockHash))
		if err == nil {
			return block, nil
		}
	}
	return nil, err
}

func (n *EthNode) GetBalanceDiff(address string, blockNumber int64) (diff *big.Int, err error) {
	for _, client := range n.clients {
		balanceBefore, err := client.BalanceAt(context.Background(), ethcommon.HexToAddress(address), big.NewInt(blockNumber-1))
		if err != nil {
			continue
		}

		balanceAfter, err := client.BalanceAt(context.Background(), ethcommon.HexToAddress(address), big.NewInt(blockNumber))
		if err != nil {
			continue
		}

		balanceDiff := new(big.Int).Sub(balanceAfter, balanceBefore)
		return balanceDiff, nil
	}
	return nil, err
}

// func (n *EthNode) GethGetBlockByNumber(blockNumber int, withTransactions bool) (block *types.Block, err error) {
// 	for _, client := range n.gethClients {
// 		block, err = client.BlockByNumber(context.Background(), big.NewInt(int64(blockNumber)))
// 		if err == nil {
// 			return block, nil
// 		}
// 	}
// 	return nil, err
// }
