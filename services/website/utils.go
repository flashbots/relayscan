package website

import (
	_ "embed"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/metachris/relayscan/database"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	// Printer for pretty printing numbers
	printer = message.NewPrinter(language.English)

	// Caser is used for casing strings
	caser = cases.Title(language.English)
)

func weiToEth(wei string) string {
	weiBigInt := new(big.Int)
	weiBigInt.SetString(wei, 10)
	ethValue := weiBigIntToEthBigFloat(weiBigInt)
	return ethValue.String()
}

func weiBigIntToEthBigFloat(wei *big.Int) (ethValue *big.Float) {
	// wei / 10^18
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return
}

func prettyInt(i uint64) string {
	return printer.Sprintf("%d", i)
}

func caseIt(s string) string {
	return caser.String(s)
}

func percent(cnt, total uint64) string {
	p := float64(cnt) / float64(total) * 100
	return printer.Sprintf("%.2f", p)
}

func builderTable(builders []*database.TopBuilderEntry) string {
	buildersEntries := [][]string{}
	for _, builder := range builders {
		buildersEntries = append(buildersEntries, []string{
			builder.ExtraData,
			printer.Sprintf("%d", builder.NumBlocks),
			builder.Percent,
		})
	}
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Builder extra_data", "Blocks", "%"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetAutoWrapText(false)
	table.SetCenterSeparator("|")
	table.AppendBulk(buildersEntries)
	table.Render()
	// fmt.Println(tableString.String())
	return tableString.String()
}

func builderProfitTable(entries []*database.BuilderProfitEntry) string {
	tableEntries := [][]string{}
	for _, builder := range entries {
		tableEntries = append(tableEntries, []string{
			builder.ExtraData,
			printer.Sprintf("%d", builder.NumBlocks),
			printer.Sprintf("%d", builder.NumBlocksProfit),
			printer.Sprintf("%d", builder.NumBlocksSubsidised),
			builder.ProfitPerBlockAvg,
			builder.ProfitTotal,
			builder.SubsidiesTotal,
		})
	}
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Builder extra_data", "Blocks", "Blocks with profit", "Blocks with subsidy", "Avg. profit / block", "Profit total", "Subsidies total"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetAutoWrapText(false)
	table.SetCenterSeparator("|")
	table.AppendBulk(tableEntries)
	table.Render()
	// fmt.Println(tableString.String())
	return tableString.String()
}

func relayTable(relays []*database.TopRelayEntry) string {
	relayEntries := [][]string{}
	for _, relay := range relays {
		relayEntries = append(relayEntries, []string{
			relay.Relay,
			printer.Sprintf("%d", relay.NumPayloads),
			relay.Percent,
		})
	}
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Relay", "Payloads", "%"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetAutoWrapText(false)
	table.SetCenterSeparator("|")
	table.AppendBulk(relayEntries)
	table.Render()
	// fmt.Println(tableString.String())
	return tableString.String()
}

func strToBigFloat(f string) *big.Float {
	bf := new(big.Float)
	bf.SetString(f)
	return bf
}

func addFloatStrings(f1, f2 string, decimals int) string {
	bf1 := strToBigFloat(f1)
	bf2 := strToBigFloat(f2)
	return new(big.Float).Add(bf1, bf2).Text('f', decimals)
}

func divFloatStrings(f1, f2 string, decimals int) string {
	bf1 := strToBigFloat(f1)
	bf2 := strToBigFloat(f2)
	return new(big.Float).Quo(bf1, bf2).Text('f', decimals)
}

func consolidateBuilderEntries(builders []*database.TopBuilderEntry) []*database.TopBuilderEntry {
	// Get total builder payloads, and build consolidated builder list
	buildersMap := make(map[string]*database.TopBuilderEntry)
	buildersNumPayloads := uint64(0)
	for _, entry := range builders {
		buildersNumPayloads += entry.NumBlocks
		if strings.Contains(entry.ExtraData, "builder0x69") {
			topBuilderEntry, isKnown := buildersMap["builder0x69"]
			if isKnown {
				topBuilderEntry.NumBlocks += entry.NumBlocks
				topBuilderEntry.Aliases = append(topBuilderEntry.Aliases, entry.ExtraData)
			} else {
				buildersMap["builder0x69"] = &database.TopBuilderEntry{
					ExtraData: "builder0x69",
					NumBlocks: entry.NumBlocks,
					Aliases:   []string{entry.ExtraData},
				}
			}
		} else {
			buildersMap[entry.ExtraData] = entry
		}
	}

	// Prepare top builders by extra stats
	for _, entry := range builders {
		p := float64(entry.NumBlocks) / float64(buildersNumPayloads) * 100
		entry.Percent = fmt.Sprintf("%.2f", p)
	}

	// Prepare top builders by summary stats
	resp := []*database.TopBuilderEntry{}
	for _, entry := range buildersMap {
		p := float64(entry.NumBlocks) / float64(buildersNumPayloads) * 100
		entry.Percent = fmt.Sprintf("%.2f", p)
		resp = append(resp, entry)
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].NumBlocks > resp[j].NumBlocks
	})
	return resp
}

func consolidateBuilderProfitEntries(entries []*database.BuilderProfitEntry) []*database.BuilderProfitEntry {
	buildersMap := make(map[string]*database.BuilderProfitEntry)
	buildersNumPayloads := uint64(0)
	for _, entry := range entries {
		buildersNumPayloads += entry.NumBlocks
		if strings.Contains(entry.ExtraData, "builder0x69") {
			entryConsolidated, isKnown := buildersMap["builder0x69"]
			if isKnown {
				entryConsolidated.Aliases = append(entryConsolidated.Aliases, entry.ExtraData)
				entryConsolidated.NumBlocks += entry.NumBlocks
				entryConsolidated.NumBlocksProfit += entry.NumBlocksProfit
				entryConsolidated.NumBlocksSubsidised += entry.NumBlocksSubsidised
				entryConsolidated.ProfitTotal = addFloatStrings(entryConsolidated.ProfitTotal, entry.ProfitTotal, 6)
				entryConsolidated.SubsidiesTotal = addFloatStrings(entryConsolidated.SubsidiesTotal, entry.SubsidiesTotal, 6)
				entryConsolidated.ProfitPerBlockAvg = divFloatStrings(entryConsolidated.ProfitTotal, fmt.Sprint(entryConsolidated.NumBlocks), 6)

			} else {
				buildersMap["builder0x69"] = &database.BuilderProfitEntry{
					ExtraData:           "builder0x69",
					NumBlocks:           entry.NumBlocks,
					NumBlocksProfit:     entry.NumBlocksProfit,
					NumBlocksSubsidised: entry.NumBlocksSubsidised,
					ProfitTotal:         entry.ProfitTotal,
					SubsidiesTotal:      entry.SubsidiesTotal,
					ProfitPerBlockAvg:   entry.ProfitPerBlockAvg,
					Aliases:             []string{entry.ExtraData},
				}
			}
		} else {
			buildersMap[entry.ExtraData] = entry
		}
	}

	resp := []*database.BuilderProfitEntry{}
	for _, entry := range buildersMap {
		resp = append(resp, entry)
	}

	sort.Slice(resp, func(i, j int) bool {
		return strToBigFloat(resp[i].ProfitTotal).Cmp(strToBigFloat(resp[j].ProfitTotal)) > 0
	})
	return resp
}

func prepareRelaysEntries(relays []*database.TopRelayEntry) []*database.TopRelayEntry {
	topRelaysNumPayloads := uint64(0)
	for _, entry := range relays {
		topRelaysNumPayloads += entry.NumPayloads
	}
	for i, entry := range relays {
		p := float64(entry.NumPayloads) / float64(topRelaysNumPayloads) * 100
		relays[i].Percent = fmt.Sprintf("%.2f", p)
	}
	return relays
}
