package bidcollect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourceTypes(t *testing.T) {
	require.Equal(t, 0, CollectGetHeader)
	require.Equal(t, 1, CollectDataAPI)
	require.Equal(t, 2, CollectUltrasoundStream)
}
