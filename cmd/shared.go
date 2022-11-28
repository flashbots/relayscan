package cmd

import (
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	relaycommon "github.com/flashbots/mev-boost-relay/common"
	"github.com/metachris/flashbotsrpc"
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

func connectEthNodes(uri1, uri2 string) (client1, client2 *flashbotsrpc.FlashbotsRPC) {
	client1 = flashbotsrpc.New(uri1)
	if uri2 != "" {
		client2 = flashbotsrpc.New(uri2)
	}
	return client1, client2
}

func connectPostgres(dsn string) (*database.DatabaseService, error) {
	dbURL, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	log.Infof("Connecting to Postgres database at %s%s ...", dbURL.Host, dbURL.Path)
	return database.NewDatabaseService(defaultPostgresDSN)
}

type EthNode struct {
	clients     []*flashbotsrpc.FlashbotsRPC
	gethClients []*ethclient.Client
}

func NewEthNode(uris ...string) (*EthNode, error) {
	if len(uris) == 0 {
		return nil, errors.New("uri1 is empty")
	}
	node := &EthNode{}
	for _, uri := range uris {
		node.clients = append(node.clients, flashbotsrpc.New(uri))
		gethClient, err := ethclient.Dial(uri)
		if err != nil {
			fmt.Println("Error connecting to eth node", uri, err)
			return nil, err
		}
		node.gethClients = append(node.gethClients, gethClient)
	}
	return node, nil
}

func (n *EthNode) GetBlockByNumber(blockNumber int, withTransactions bool) (block *flashbotsrpc.Block, err error) {
	for _, client := range n.clients {
		block, err = client.EthGetBlockByNumber(blockNumber, withTransactions)
		if err == nil {
			return block, nil
		}
	}
	return nil, err
}

func (n *EthNode) GetBlockByHash(blockHash string, withTransactions bool) (block *flashbotsrpc.Block, err error) {
	for _, client := range n.clients {
		block, err = client.EthGetBlockByHash(blockHash, withTransactions)
		if err == nil {
			return block, nil
		}
	}
	return nil, err
}

func (n *EthNode) GetBalanceDiff(address string, blockNumber int) (diff *big.Int, err error) {
	balanceBefore := *big.NewInt(0)
	balanceAfter := *big.NewInt(0)
	for _, client := range n.clients {
		balanceBefore, err = client.EthGetBalance(address, fmt.Sprintf("0x%x", blockNumber-1))
		if err != nil {
			continue
		}

		balanceAfter, err = client.EthGetBalance(address, fmt.Sprintf("0x%x", blockNumber))
		if err != nil {
			continue
		}

		balanceDiff := new(big.Int).Sub(&balanceAfter, &balanceBefore)
		return balanceDiff, nil
	}
	return nil, err
}
