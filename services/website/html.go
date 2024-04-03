package website

import (
	_ "embed"
	"text/template"
	"time"

	"github.com/flashbots/relayscan/database"
)

type Stats struct {
	Since time.Time
	Until time.Time

	TopRelays          []*database.TopRelayEntry
	TopBuilders        []*database.TopBuilderEntry
	BuilderProfits     []*database.BuilderProfitEntry
	TopBuildersByRelay map[string][]*database.TopBuilderEntry
}

func NewStats() *Stats {
	return &Stats{
		TopRelays:          make([]*database.TopRelayEntry, 0),
		TopBuilders:        make([]*database.TopBuilderEntry, 0),
		BuilderProfits:     make([]*database.BuilderProfitEntry, 0),
		TopBuildersByRelay: make(map[string][]*database.TopBuilderEntry),
	}
}

type HTMLData struct {
	Title string

	GeneratedAt    time.Time
	LastUpdateTime string
	LastUpdateSlot uint64
	TimeSpans      []string

	TimeSpan string
	View     string
	Stats    *Stats
}

type HTMLDataDailyStats struct {
	Title string

	Day       string
	DayPrev   string
	DayNext   string
	TimeSince string
	TimeUntil string

	TopRelays            []*database.TopRelayEntry
	TopBuildersBySummary []*database.TopBuilderEntry
}

var funcMap = template.FuncMap{
	"weiToEth":           weiToEth,
	"prettyInt":          prettyInt,
	"caseIt":             caseIt,
	"percent":            percent,
	"relayTable":         relayTable,
	"builderTable":       builderTable,
	"builderProfitTable": builderProfitTable,
}

func ParseIndexTemplate() (*template.Template, error) {
	return template.New("index.html").Funcs(funcMap).ParseFiles("services/website/templates/index.html", "services/website/templates/base.html")
}

func ParseDailyStatsTemplate() (*template.Template, error) {
	return template.New("daily-stats.html").Funcs(funcMap).ParseFiles("services/website/templates/daily-stats.html", "services/website/templates/base.html")
}
