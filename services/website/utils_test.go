package website

import (
	"testing"

	"github.com/flashbots/relayscan/database"
	"github.com/stretchr/testify/require"
)

func TestConsolidateBuilderEntries(t *testing.T) {
	in := []*database.TopBuilderEntry{
		{
			ExtraData: "made by builder0x69",
			NumBlocks: 1,
			Aliases:   []string{"builder0x69"},
		},
		{
			ExtraData: "builder0x69",
			NumBlocks: 1,
			Aliases:   []string{"builder0x69"},
		},
		{
			ExtraData: "s3e6f",
			NumBlocks: 1,
			Aliases:   []string{"bob the builder"},
		},
		{
			ExtraData: "s0e3f",
			NumBlocks: 1,
			Aliases:   []string{"bob the builder"},
		},
		{
			ExtraData: "s0e2ts10e11t",
			NumBlocks: 1,
			Aliases:   []string{"bob the builder"},
		},
		{
			ExtraData: "manta-builder",
			NumBlocks: 1,
		},
	}
	expected := []*database.TopBuilderEntry{
		{
			ExtraData: "bob the builder",
			NumBlocks: 3,
			Percent:   "50.00",
			Aliases:   []string{"s3e6f", "s0e3f", "s0e2ts10e11t"},
		},
		{
			ExtraData: "builder0x69",
			NumBlocks: 2,
			Percent:   "33.33",
			Aliases:   []string{"made by builder0x69", "builder0x69"},
		},
		{
			ExtraData: "manta-builder",
			Percent:   "16.67",
			NumBlocks: 1,
		},
	}

	out := consolidateBuilderEntries(in)
	for i, o := range out {
		require.Equal(t, expected[i], o)
	}
}
