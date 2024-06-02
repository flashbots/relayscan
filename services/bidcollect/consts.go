package bidcollect

const (
	SourceTypeGetHeader        = 0
	SourceTypeDataAPI          = 1
	SourceTypeUltrasoundStream = 2

	ultrasoundStreamDefaultURL = "ws://relay-builders-eu.ultrasound.money/ws/v1/top_bid"
	initialBackoffSec          = 5
	maxBackoffSec              = 120

	// bucketMinutes is the number of minutes to write into each CSV file (i.e. new file created for every X minutes bucket)
	bucketMinutes = 60

	// channel size for bid collector inputs
	bidCollectorInputChannelSize = 1000
)

var (
// csvFileEnding = relaycommon.GetEnv("CSV_FILE_END", "tsv")
// csvSeparator  = relaycommon.GetEnv("CSV_SEP", "\t")
)
