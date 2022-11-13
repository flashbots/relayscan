package website

import (
	_ "embed"
	"math/big"
	"text/template"

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
	TopRelays        []*database.TopRelayEntry
	NumPayloadsTotal uint64

	TopBuilders            []*database.TopBuilderEntry
	TopBuildersNumPayloads uint64

	LastUpdateTime string
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
var htmlContent string

func ParseIndexTemplate() (*template.Template, error) {
	return template.New("index").Funcs(funcMap).Parse(htmlContent)
}
