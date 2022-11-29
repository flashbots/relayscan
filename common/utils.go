// Package common includes common utilities
package common

import (
	"fmt"
	"math/big"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/params"
	"github.com/flashbots/go-boost-utils/types"
)

var ErrMissingRelayPubkey = fmt.Errorf("missing relay public key")

// RelayEntry represents a relay that mev-boost connects to.
type RelayEntry struct {
	PublicKey types.PublicKey
	URL       *url.URL
}

func (r *RelayEntry) String() string {
	return r.URL.String()
}

func (r *RelayEntry) Hostname() string {
	return r.URL.Hostname()
}

// GetURI returns the full request URI with scheme, host, path and args for the relay.
func (r *RelayEntry) GetURI(path string) string {
	return GetURI(r.URL, path)
}

// NewRelayEntry creates a new instance based on an input string
// relayURL can be IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
func NewRelayEntry(relayURL string, requireUser bool) (entry RelayEntry, err error) {
	// Add protocol scheme prefix if it does not exist.
	if !strings.HasPrefix(relayURL, "http") {
		relayURL = "https://" + relayURL
	}

	// Parse the provided relay's URL and save the parsed URL in the RelayEntry.
	entry.URL, err = url.ParseRequestURI(relayURL)
	if err != nil {
		return entry, err
	}

	// Extract the relay's public key from the parsed URL.
	if requireUser && entry.URL.User.Username() == "" {
		return entry, ErrMissingRelayPubkey
	}

	if entry.URL.User.Username() != "" {
		err = entry.PublicKey.UnmarshalText([]byte(entry.URL.User.Username()))
	}
	return entry, err
}

// RelayEntriesToStrings returns the string representation of a list of relay entries
func RelayEntriesToStrings(relays []RelayEntry) []string {
	ret := make([]string, len(relays))
	for i, entry := range relays {
		ret[i] = entry.String()
	}
	return ret
}

// GetURI returns the full request URI with scheme, host, path and args.
func GetURI(url *url.URL, path string) string {
	u2 := *url
	u2.User = nil
	u2.Path = path
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
