package common

import (
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/require"
)

func TestSlotToTime(t *testing.T) {
	require.Equal(t, int64(1685923199), SlotToTime(6591598).Unix())
}

func TestTimeToSlot(t *testing.T) {
	require.Equal(t, uint64(6591598), TimeToSlot(SlotToTime(6591598)))
}

func TestBytesFormat(t *testing.T) {
	n := uint64(795025173)

	s := humanize.Bytes(n)
	require.Equal(t, "795 MB", s)

	s = humanize.IBytes(n)
	require.Equal(t, "758 MiB", s)

	s = HumanBytes(n)
	require.Equal(t, "758 MB", s)

	s = HumanBytes(n * 10)
	require.Equal(t, "7.4 GB", s)

	s = HumanBytes(n / 1000)
	require.Equal(t, "776 KB", s)
}
