package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlotTimeConversion(t *testing.T) {
	slot := 8901362
	slotTime := slotToTime(uint64(slot))
	require.Equal(t, 1713640367, int(slotTime.Unix()))
	convertedSlot := timeToSlot(slotTime)
	require.Equal(t, uint64(slot), convertedSlot)
}
