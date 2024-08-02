package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourceTypes(t *testing.T) {
	require.Equal(t, 0, SourceTypeGetHeader)
	require.Equal(t, 1, SourceTypeDataAPI)
	require.Equal(t, 2, SourceTypeUltrasoundStream)
}

func TestCSVHasNotChanged(t *testing.T) {
	// The specific field ordering is used in many places throughout the ecosystem and must not be changed.
	expectedResult := "source_type,received_at_ms,timestamp_ms,slot,slot_t_ms,value,block_hash,parent_hash,builder_pubkey,block_number,block_fee_recipient,relay,proposer_pubkey,proposer_fee_recipient,optimistic_submission"
	currentResult := strings.Join(CommonBidCSVFields, ",")
	require.Equal(t, expectedResult, currentResult)

	bid := CommonBid{
		SourceType:           SourceTypeGetHeader,
		ReceivedAtMs:         1,
		TimestampMs:          2,
		Slot:                 3,
		BlockNumber:          4,
		BlockHash:            "5",
		ParentHash:           "6",
		BuilderPubkey:        "7",
		Value:                "8",
		BlockFeeRecipient:    "9",
		Relay:                "10",
		ProposerPubkey:       "11",
		ProposerFeeRecipient: "12",
		OptimisticSubmission: true,
	}
	asCSV := bid.ToCSVLine(",")
	expected := "0,1,2,3,-1606824058998,8,5,6,7,4,9,10,11,12,"
	require.Equal(t, expected, asCSV)

	// When source type is data-api, then optimistic field is included
	bid.SourceType = SourceTypeDataAPI
	asCSV = bid.ToCSVLine(",")
	expected = "1,1,2,3,-1606824058998,8,5,6,7,4,9,10,11,12,true"
	require.Equal(t, expected, asCSV)
}
