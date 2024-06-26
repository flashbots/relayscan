package bidcollect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourceTypes(t *testing.T) {
	require.Equal(t, 0, SourceTypeGetHeader)
	require.Equal(t, 1, SourceTypeDataAPI)
	require.Equal(t, 2, SourceTypeTopBidWSStream)
}
