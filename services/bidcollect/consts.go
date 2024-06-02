package bidcollect

import (
	relaycommon "github.com/flashbots/mev-boost-relay/common"
)

const (
	ultrasoundStreamDefaultURL = "ws://relay-builders-eu.ultrasound.money/ws/v1/top_bid"
	initialBackoffSec          = 5
	maxBackoffSec              = 120

	// bucketMinutes is the number of minutes to write into each CSV file (i.e. new file created for every X minutes bucket)
	bucketMinutes = 60

	// channel size for bid collector inputs
	bidCollectorInputChannelSize = 1000
)

var csvSeparator = relaycommon.GetEnv("CSV_SEP", "\t")
