package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlotToTime(t *testing.T) {
	require.Equal(t, int64(1685923199), SlotToTime(6591598).Unix())
}

func TestTimeToSlot(t *testing.T) {
	require.Equal(t, uint64(6591598), TimeToSlot(SlotToTime(6591598)))
}
