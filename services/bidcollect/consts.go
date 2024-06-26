package bidcollect

const (
	SourceTypeGetHeader      = 0
	SourceTypeDataAPI        = 1
	SourceTypeTopBidWSStream = 2

	initialBackoffSec = 5
	maxBackoffSec     = 120

	// bucketMinutes is the number of minutes to write into each CSV file (i.e. new file created for every X minutes bucket)
	bucketMinutes = 60

	// channel size for bid collector inputs
	bidCollectorInputChannelSize = 1000
)
