package website

import (
	_ "embed"
	"math/big"
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
