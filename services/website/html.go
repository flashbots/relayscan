package website

import (
	_ "embed"
	"text/template"
	"time"

	"github.com/metachris/relayscan/database"
)

type Stats struct {
	Since time.Time
	Until time.Time

	TopRelays          []*database.TopRelayEntry
	TopBuilders        []*database.TopBuilderEntry
	BuilderProfits     []*database.BuilderProfitEntry
	TopBuildersByRelay map[string][]*database.TopBuilderEntry
}

type HTMLData struct {
	GeneratedAt    time.Time
	LastUpdateTime string

	Stats            map[string]*Stats
	StatsTimeSpans   []string
	StatsTimeInitial string
	InitialView      string
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

var funcMap = template.FuncMap{
	"weiToEth":     weiToEth,
	"prettyInt":    prettyInt,
	"caseIt":       caseIt,
	"percent":      percent,
	"builderTable": builderTable,
	"relayTable":   relayTable,
}

//go:embed templates/index.html
var htmlContentIndex string

//go:embed templates/daily-stats.html
var htmlContentDailyStats string

func ParseIndexTemplate() (*template.Template, error) {
	return template.New("index").Funcs(funcMap).Parse(htmlContentIndex)
}
