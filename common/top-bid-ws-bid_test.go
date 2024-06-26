package common

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestValueDecoding(t *testing.T) {
	expected := "55539751698389157"
	hex := "0xa558e5221c51c500000000000000000000000000000000000000000000000000"
	hexBytes := hexutil.MustDecode(hex)
	value := new(big.Int).SetBytes(ReverseBytes(hexBytes[:])).String()
	require.Equal(t, expected, value)
}

func TestTopBidWSStreamBidSSZDecoding(t *testing.T) {
	hex := "0x704b87ce8f010000a94b8c0000000000b6043101000000002c02b28fd8fdb45fd6ac43dd04adad1449a35b64247b1ed23a723a1fcf6cac074d0668c9e0912134628c32a54854b952234ebb6c1fdd6b053566ac2d2a09498da03b00ddb78b2c111450a5417a8c368c40f1f140cdf97d95b7fa9565467e0bbbe27877d08e01c69b4e5b02b144e6a265df99a0839818b3f120ebac9b73f82b617dc6a5556c71794b1a9c5400000000000000000000000000000000000000000000000000"
	bytes := hexutil.MustDecode(hex)
	bid := new(TopBidWebsocketStreamBid)
	err := bid.UnmarshalSSZ(bytes)
	require.NoError(t, err)

	require.Equal(t, uint64(1717156924272), bid.Timestamp)
	require.Equal(t, uint64(9194409), bid.Slot)
	require.Equal(t, uint64(19989686), bid.BlockNumber)
	require.Equal(t, "0x2c02b28fd8fdb45fd6ac43dd04adad1449a35b64247b1ed23a723a1fcf6cac07", hexutil.Encode(bid.BlockHash[:]))
}
