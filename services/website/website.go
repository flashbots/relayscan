package website

import (
	_ "embed"
	"math/big"
	"text/template"
	"time"

	"github.com/metachris/relayscan/database"
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

type HTMLData struct {
	GeneratedAt    time.Time
	LastUpdateTime string

	TopRelays []*database.TopRelayEntry

	TopBuildersByExtraData []*database.TopBuilderEntry
	TopBuildersBySummary   []*database.TopBuilderEntry
}

type HTMLDataDailyStats struct {
	Day       string
	DayPrev   string
	DayNext   string
	TimeSince string
	TimeUntil string

	TopRelays            []*database.TopRelayEntry
	TopBuildersBySummary []*database.TopBuilderEntry
}

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

var funcMap = template.FuncMap{
	"weiToEth":  weiToEth,
	"prettyInt": prettyInt,
	"caseIt":    caseIt,
	"percent":   percent,
}

//go:embed website.html
var htmlContentIndex string

//go:embed website-daily-stats.html
var htmlContentDailyStats string

func ParseIndexTemplate() (*template.Template, error) {
	return template.New("index").Funcs(funcMap).Parse(htmlContentIndex)
}
