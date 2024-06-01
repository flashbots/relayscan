package bidcollect

const (
	ultrasoundStreamDefaultURL = "ws://relay-builders-eu.ultrasound.money/ws/v1/top_bid"
	initialBackoffSec          = 5
	maxBackoffSec              = 120

	// bucketMinutes is the number of minutes to write into each CSV file (i.e. new file created for every X minutes bucket)
	bucketMinutes = 60
)
