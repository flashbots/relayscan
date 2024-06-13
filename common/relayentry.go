package common

import (
	"net/url"
	"strings"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/relayscan/vars"
)

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

func MustNewRelayEntry(relayURL string, requireUser bool) (entry RelayEntry) {
	entry, err := NewRelayEntry(relayURL, requireUser)
	Check(err)
	return entry
}

// RelayEntriesToStrings returns the string representation of a list of relay entries
func RelayEntriesToStrings(relays []RelayEntry) []string {
	ret := make([]string, len(relays))
	for i, entry := range relays {
		ret[i] = entry.String()
	}
	return ret
}

// RelayEntriesToHostnameStrings returns the hostnames of a list of relay entries
func RelayEntriesToHostnameStrings(relays []RelayEntry) []string {
	ret := make([]string, len(relays))
	for i, entry := range relays {
		ret[i] = entry.Hostname()
	}
	return ret
}

func GetRelays() ([]RelayEntry, error) {
	var err error
	relays := make([]RelayEntry, len(vars.RelayURLs))
	for i, relayStr := range vars.RelayURLs {
		relays[i], err = NewRelayEntry(relayStr, true)
		if err != nil {
			return relays, err
		}
	}
	return relays, nil
}

func MustGetRelays() []RelayEntry {
	relays, err := GetRelays()
	Check(err)
	return relays
}
