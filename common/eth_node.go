package common

import (
	"context"
	"fmt"
	"math/big"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthNode struct {
	Clients []*ethclient.Client
}

func NewEthNode(uris ...string) (*EthNode, error) {
	if len(uris) == 0 {
		return nil, ErrURLEmpty
	}
	node := &EthNode{} //nolint:exhaustruct
	for _, uri := range uris {
		client, err := ethclient.Dial(uri)
		if err != nil {
			fmt.Println("Error connecting to eth node", uri, err)
			return nil, err
		}
		node.Clients = append(node.Clients, client)
	}
	return node, nil
}

func (n *EthNode) BlockByNumber(blockNumber int64) (block *types.Block, err error) {
	for _, client := range n.Clients {
		block, err = client.BlockByNumber(context.Background(), big.NewInt(blockNumber))
		if err == nil {
			return block, nil
		}
	}
	return nil, err
}

func (n *EthNode) BlockByHash(blockHash string) (block *types.Block, err error) {
	for _, client := range n.Clients {
		block, err = client.BlockByHash(context.Background(), ethcommon.HexToHash(blockHash))
		if err == nil {
			return block, nil
		}
	}
	return nil, err
}

func (n *EthNode) GetBalanceDiff(address string, blockNumber int64) (diff *big.Int, err error) {
	for _, client := range n.Clients {
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
