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
			ExtraData: "foo-builder",
			NumBlocks: 1,
		},
	}
	expected := []*TopBuilderDisplayEntry{
		{
			Info: &database.TopBuilderEntry{
				ExtraData: "builder0x69",
				NumBlocks: 2,
				Percent:   "66.67",
			},
			Children: []*database.TopBuilderEntry{
				{
					ExtraData: "made by builder0x69",
					NumBlocks: 1,
					Percent:   "33.33",
					Aliases:   []string{"builder0x69"},
				},
				{
					ExtraData: "builder0x69",
					NumBlocks: 1,
					Percent:   "33.33",
					Aliases:   []string{"builder0x69"},
				},
			},
		},
		{
			Info: &database.TopBuilderEntry{
				ExtraData: "foo-builder",
				NumBlocks: 1,
				Percent:   "33.33",
			},
			Children: []*database.TopBuilderEntry{},
		},
	}

	out := consolidateBuilderEntries(in)
	for i, o := range out {
		require.Equal(t, expected[i], o)
	}
}

func TestConsolidateBuilderProfitEntries(t *testing.T) {
	in := []*database.BuilderProfitEntry{
		{
			ExtraData:       "made by builder0x69",
			NumBlocks:       1,
			NumBlocksProfit: 1,
			ProfitTotal:     "1",
			Aliases:         []string{"builder0x69"},
		},
		{
			ExtraData:       "builder0x69",
			NumBlocks:       1,
			NumBlocksProfit: 1,
			ProfitTotal:     "1",
			Aliases:         []string{"builder0x69"},
		},
		{
			ExtraData:           "s3e6f",
			NumBlocks:           1,
			NumBlocksSubsidised: 1,
			SubsidiesTotal:      "1",
			Aliases:             []string{"bob the builder"},
		},
		{
			ExtraData:           "s0e3f",
			NumBlocks:           1,
			NumBlocksSubsidised: 1,
			SubsidiesTotal:      "1",
			Aliases:             []string{"bob the builder"},
		},
		{
			ExtraData:           "s12e14t",
			NumBlocks:           1,
			NumBlocksSubsidised: 1,
			SubsidiesTotal:      "1",
			Aliases:             []string{"bob the builder"},
		},
		{
			ExtraData:           "s0e2ts10e11t",
			NumBlocks:           1,
			NumBlocksSubsidised: 1,
			SubsidiesTotal:      "1",
			Aliases:             []string{"bob the builder"},
		},
		{
			ExtraData:       "manta-builder",
			NumBlocks:       1,
			NumBlocksProfit: 1,
			ProfitTotal:     "3",
		},
	}
	expected := []*database.BuilderProfitEntry{
		{
			ExtraData:       "manta-builder",
			NumBlocks:       1,
			NumBlocksProfit: 1,
			ProfitTotal:     "3",
		},
		{
			ExtraData:         "builder0x69",
			NumBlocks:         2,
			NumBlocksProfit:   2,
			ProfitTotal:       "2.0000",
			ProfitPerBlockAvg: "1.0000",
			SubsidiesTotal:    "0.0000",
			Aliases:           []string{"made by builder0x69", "builder0x69"},
		},
		{
			ExtraData:           "bob the builder",
			NumBlocks:           4,
			NumBlocksSubsidised: 4,
			ProfitTotal:         "0.0000",
			ProfitPerBlockAvg:   "0.0000",
			SubsidiesTotal:      "4.0000",
			Aliases:             []string{"s3e6f", "s0e3f", "s12e14t", "s0e2ts10e11t"},
		},
	}

	out := consolidateBuilderProfitEntries(in)
	for i, o := range out {
		require.Equal(t, expected[i], o)
	}
}

func TestLowercaseNoWhitespace(t *testing.T) {
	c1 := lowercaseNoWhitespace("abCD 123!@#")
	require.Equal(t, "abcd123!@#", c1)
}
