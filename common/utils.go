// Package common includes common utilities
package common

import (
	"math/big"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/params"
	"github.com/flashbots/mev-boost-relay/beaconclient"
	"github.com/flashbots/relayscan/vars"
	"github.com/sirupsen/logrus"
)

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

// GetURI returns the full request URI with scheme, host, path and args.
func GetURI(url *url.URL, path string) string {
	u2 := *url
	u2.User = nil
	u2.Path = path
	return u2.String()
}

func GetURIWithQuery(url *url.URL, path string, queryArgs map[string]string) string {
	u2 := *url
	u2.User = nil
	u2.Path = path
	q := u2.Query()
	for key, value := range queryArgs {
		q.Add(key, value)
	}
	u2.RawQuery = q.Encode()
	return u2.String()
}

func EthToWei(eth *big.Int) *big.Float {
	if eth == nil {
		return big.NewFloat(0)
	}
	return new(big.Float).Quo(new(big.Float).SetInt(eth), new(big.Float).SetInt(big.NewInt(params.Ether)))
}

func PercentDiff(x, y *big.Int) *big.Float {
	fx := new(big.Float).SetInt(x)
	fy := new(big.Float).SetInt(y)
	r := new(big.Float).Quo(fy, fx)
	return new(big.Float).Sub(r, big.NewFloat(1))
}

func WeiToEth(wei *big.Int) (ethValue *big.Float) {
	// wei / 10^18
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return
}

func WeiStrToEthStr(wei string, decimals int) string {
	weiBigInt := new(big.Int)
	weiBigInt.SetString(wei, 10)
	ethValue := WeiToEth(weiBigInt)
	return ethValue.Text('f', decimals)
}

func WeiToEthStr(wei *big.Int) string {
	return WeiToEth(wei).Text('f', 6)
}

func StrToBigInt(s string) *big.Int {
	i := new(big.Int)
	i.SetString(s, 10)
	return i
}

func StringSliceContains(haystack []string, needle string) bool {
	for _, entry := range haystack {
		if entry == needle {
			return true
		}
	}
	return false
}

func TimeToSlot(t time.Time) uint64 {
	return uint64((t.Unix() - int64(vars.Genesis)) / 12)
}

func SlotToTime(slot uint64) time.Time {
	timestamp := (slot * 12) + uint64(vars.Genesis)
	return time.Unix(int64(timestamp), 0).UTC()
}

func MustParseDateTimeStr(s string) time.Time {
	layout1 := "2006-01-02"
	layout2 := "2006-01-02 15:04"
	t, err := time.Parse(layout1, s)
	if err != nil {
		t, err = time.Parse(layout2, s)
		Check(err)
	}
	return t
}

func BeginningOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func MustConnectBeaconNode(log *logrus.Entry, beaconNodeURI string, allowSyncing bool) (bn *beaconclient.ProdBeaconInstance, headSlot uint64) {
	bn = beaconclient.NewProdBeaconInstance(log, beaconNodeURI)
	syncStatus, err := bn.SyncStatus()
	Check(err)
	if syncStatus.IsSyncing && !allowSyncing {
		panic("beacon node is syncing")
	}
	return bn, syncStatus.HeadSlot
}

func ReverseBytes(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	for i := len(dst)/2 - 1; i >= 0; i-- {
		opp := len(dst) - 1 - i
		dst[i], dst[opp] = dst[opp], dst[i]
	}
	return dst
}
