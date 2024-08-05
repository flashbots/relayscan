package types

const (
	SourceTypeGetHeader        = 0
	SourceTypeDataAPI          = 1
	SourceTypeUltrasoundStream = 2

	UltrasoundStreamDefaultURL = "ws://relay-builders-eu.ultrasound.money/ws/v1/top_bid"
	InitialBackoffSec          = 5
	MaxBackoffSec              = 120

	// bucketMinutes is the number of minutes to write into each CSV file (i.e. new file created for every X minutes bucket)
	BucketMinutes = 60

	// channel size for bid collector inputs
	BidCollectorInputChannelSize = 1000

	RedisChannel = "bidcollect/bids"
)

var (
// csvFileEnding = relaycommon.GetEnv("CSV_FILE_END", "tsv")
// csvSeparator  = relaycommon.GetEnv("CSV_SEP", "\t")
)
