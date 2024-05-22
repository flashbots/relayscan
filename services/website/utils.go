package website

import (
	_ "embed"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/flashbots/relayscan/database"
	"github.com/flashbots/relayscan/vars"
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
			builder.ProfitTotal,
			builder.SubsidiesTotal,
		})
	}
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Builder extra_data", "Blocks", "Blocks profit", "Blocks subsidy", "Profit total", "Subsidies total"})
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
		updated := false
		for k, v := range vars.BuilderAliases {
			// Check if this is one of the known aliases.
			if v(entry.ExtraData) {
				updated = true
				topBuilderEntry, isKnown := buildersMap[k]
				if isKnown {
					topBuilderEntry.NumBlocks += entry.NumBlocks
					topBuilderEntry.Aliases = append(topBuilderEntry.Aliases, entry.ExtraData)
				} else {
					buildersMap[k] = &database.TopBuilderEntry{
						ExtraData: k,
						NumBlocks: entry.NumBlocks,
						Aliases:   []string{entry.ExtraData},
					}
				}
				break
			}
		}
		if !updated {
			buildersMap[entry.ExtraData] = entry
		}
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
		updated := false
		for k, v := range vars.BuilderAliases {
			// Check if this is one of the known aliases.
			if v(entry.ExtraData) {
				updated = true
				entryConsolidated, isKnown := buildersMap[k]
				if isKnown {
					entryConsolidated.Aliases = append(entryConsolidated.Aliases, entry.ExtraData)
					entryConsolidated.NumBlocks += entry.NumBlocks
					entryConsolidated.NumBlocksProfit += entry.NumBlocksProfit
					entryConsolidated.NumBlocksSubsidised += entry.NumBlocksSubsidised
					entryConsolidated.ProfitTotal = addFloatStrings(entryConsolidated.ProfitTotal, entry.ProfitTotal, 4)
					entryConsolidated.SubsidiesTotal = addFloatStrings(entryConsolidated.SubsidiesTotal, entry.SubsidiesTotal, 4)
					entryConsolidated.ProfitPerBlockAvg = divFloatStrings(entryConsolidated.ProfitTotal, fmt.Sprint(entryConsolidated.NumBlocks), 4)
				} else {
					buildersMap[k] = &database.BuilderProfitEntry{
						ExtraData:           k,
						NumBlocks:           entry.NumBlocks,
						NumBlocksProfit:     entry.NumBlocksProfit,
						NumBlocksSubsidised: entry.NumBlocksSubsidised,
						ProfitTotal:         entry.ProfitTotal,
						SubsidiesTotal:      entry.SubsidiesTotal,
						ProfitPerBlockAvg:   entry.ProfitPerBlockAvg,
						Aliases:             []string{entry.ExtraData},
					}
				}
				break
			}
		}
		if !updated {
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
	numPayloads := uint64(0)
	resp := []*database.TopRelayEntry{}
	for _, entry := range relays {
		if entry.NumPayloads == 0 {
			continue
		}
		resp = append(resp, entry)
		numPayloads += entry.NumPayloads
	}
	for i, entry := range resp {
		p := float64(entry.NumPayloads) / float64(numPayloads) * 100
		resp[i].Percent = fmt.Sprintf("%.2f", p)
	}
	return resp
}

func getLastWednesday() time.Time {
	now := time.Now().UTC()
	dayOffset := now.Weekday() - time.Wednesday
	targetDate := now.AddDate(0, 0, -int(dayOffset))
	targetDate = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location()).UTC()
	if targetDate.After(now) {
		targetDate = targetDate.AddDate(0, 0, -7)
	}
	return targetDate
}
