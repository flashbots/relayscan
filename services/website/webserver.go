// Package website contains the service delivering the website
package website

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/flashbots/go-utils/httplogger"
	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/database"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
	uberatomic "go.uber.org/atomic"
)

var (
	ErrServerAlreadyStarted = errors.New("server was already started")
	envSkip7dStats          = os.Getenv("SKIP_7D_STATS") != ""
)

type WebserverOpts struct {
	ListenAddress string
	DB            *database.DatabaseService
	Log           *logrus.Entry
	EnablePprof   bool
	Dev           bool // reloads template on every request
	Only24h       bool
}

type Webserver struct {
	opts *WebserverOpts
	log  *logrus.Entry

	db *database.DatabaseService

	srv        *http.Server
	srvStarted uberatomic.Bool
	minifier   *minify.M

	templateIndex      *template.Template
	templateDailyStats *template.Template

	statsLock sync.RWMutex
	stats     map[string]*Stats
	htmlData  *HTMLData

	markdownSummaryRespLock sync.RWMutex
	markdownOverview        *[]byte
	markdownBuilderProfit   *[]byte

	latestSlot uint64
}

func NewWebserver(opts *WebserverOpts) (*Webserver, error) {
	var err error

	minifier := minify.New()
	minifier.AddFunc("text/css", html.Minify)
	minifier.AddFunc("text/html", html.Minify)
	minifier.AddFunc("application/javascript", html.Minify)

	opts.Only24h = opts.Dev

	server := &Webserver{
		opts: opts,
		log:  opts.Log,
		db:   opts.DB,

		htmlData:              &HTMLData{}, //nolint:exhaustruct
		stats:                 make(map[string]*Stats),
		minifier:              minifier,
		markdownOverview:      &[]byte{},
		markdownBuilderProfit: &[]byte{},
	}

	server.templateDailyStats, err = ParseDailyStatsTemplate()
	if err != nil {
		return nil, err
	}

	server.templateIndex, err = ParseIndexTemplate()
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (srv *Webserver) StartServer() (err error) {
	if srv.srvStarted.Swap(true) {
		return ErrServerAlreadyStarted
	}

	if envSkip7dStats {
		srv.log.Warn("SKIP_7D_STATS - Skipping 7d stats")
	}

	// Start background task to regularly update status HTML data
	srv.updateHTML()
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			srv.updateHTML()
		}
	}()

	srv.srv = &http.Server{
		Addr:    srv.opts.ListenAddress,
		Handler: srv.getRouter(),

		ReadTimeout:       600 * time.Millisecond,
		ReadHeaderTimeout: 400 * time.Millisecond,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       3 * time.Second,
	}

	err = srv.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (srv *Webserver) getRouter() http.Handler {
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	r.HandleFunc("/", srv.handleRoot).Methods(http.MethodGet)
	r.HandleFunc("/overview", srv.handleRoot).Methods(http.MethodGet)
	r.HandleFunc("/builder-profit", srv.handleRoot).Methods(http.MethodGet)

	r.HandleFunc("/overview/md", srv.handleOverviewMarkdown).Methods(http.MethodGet)
	r.HandleFunc("/builder-profit/md", srv.handleBuilderProfitMarkdown).Methods(http.MethodGet)

	r.HandleFunc("/stats/cowstats", srv.handleCowstatsJSON).Methods(http.MethodGet)
	r.HandleFunc("/stats/day/{day:[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}}", srv.handleDailyStats).Methods(http.MethodGet)
	r.HandleFunc("/stats/day/{day:[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}}/json", srv.handleDailyStatsJSON).Methods(http.MethodGet)

	r.HandleFunc("/livez", srv.handleLivenessCheck)
	r.HandleFunc("/healthz", srv.handleHealthCheck)

	if srv.opts.EnablePprof {
		srv.log.Info("pprof API enabled")
		r.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	}

	loggedRouter := httplogger.LoggingMiddlewareLogrus(srv.log, r)
	withGz := gziphandler.GzipHandler(loggedRouter)
	return withGz
}

func (srv *Webserver) getStatsForHours(duration time.Duration) (stats *Stats, err error) {
	now := time.Now().UTC()
	since := now.Add(-1 * duration.Abs())

	srv.log.Debug("- loading top relays...")
	startTime := time.Now()
	topRelays, err := srv.db.GetTopRelays(since, now)
	if err != nil {
		return nil, err
	}
	srv.log.WithField("duration", time.Since(startTime).String()).Debug("- got top relays")

	srv.log.Debug("- loading top builders...")
	startTime = time.Now()
	topBuilders, err := srv.db.GetTopBuilders(since, now, "")
	if err != nil {
		return nil, err
	}
	srv.log.WithField("duration", time.Since(startTime).String()).Debug("- got top builders")

	srv.log.Debug("- loading builder profits...")
	startTime = time.Now()
	builderProfits, err := srv.db.GetBuilderProfits(since, now)
	if err != nil {
		return nil, err
	}
	srv.log.WithField("duration", time.Since(startTime).String()).Debug("- got builder profits")

	stats = &Stats{
		Since: since,
		Until: now,

		TopRelays:          prepareRelaysEntries(topRelays),
		TopBuilders:        consolidateBuilderEntries(topBuilders),
		BuilderProfits:     consolidateBuilderProfitEntries(builderProfits),
		TopBuildersByRelay: make(map[string][]*database.TopBuilderEntry),
	}

	// Query builders for each relay
	srv.log.Debug("- loading builders per relay...")
	startTime = time.Now()
	for _, relay := range topRelays {
		topBuildersForRelay, err := srv.db.GetTopBuilders(since, now, relay.Relay)
		if err != nil {
			return nil, err
		}
		stats.TopBuildersByRelay[relay.Relay] = consolidateBuilderEntries(topBuildersForRelay)
	}
	srv.log.WithField("duration", time.Since(startTime).String()).Debug("- got builders per relay")

	return stats, nil
}

func (srv *Webserver) updateHTML() {
	var err error
	srv.log.Info("Updating HTML data...")

	// Now generate the HTML
	// htmlDefault := bytes.Buffer{}

	startTime := time.Now().UTC()
	htmlData := HTMLData{} //nolint:exhaustruct
	htmlData.GeneratedAt = startTime
	htmlData.LastUpdateTime = startTime.Format("2006-01-02 15:04")
	htmlData.TimeSpans = []string{"7d", "24h", "12h", "1h"}
	// htmlData.TimeSpans = []string{"24h", "12h"}

	stats := make(map[string]*Stats)

	srv.log.Info("getting last delivered entry...")
	entry, err := srv.db.GetLatestDeliveredPayload()
	if errors.Is(err, sql.ErrNoRows) {
		srv.log.Info("No last delivered payload entry found")
	} else if err != nil {
		srv.log.WithError(err).Error("Failed to get last delivered payload entry")
		return
	} else {
		htmlData.LastUpdateTime = entry.InsertedAt.Format("2006-01-02 15:04")
		htmlData.LastUpdateSlot = entry.Slot
		srv.latestSlot = entry.Slot
	}

	startUpdate := time.Now()
	srv.log.Info("updating 24h stats...")
	stats["24h"], err = srv.getStatsForHours(24 * time.Hour)
	if err != nil {
		srv.log.WithError(err).Error("Failed to get stats for 24h")
		return
	}
	srv.log.WithField("duration", time.Since(startUpdate).String()).Info("updated 24h stats")

	if srv.opts.Only24h {
		stats["1h"] = NewStats()
		stats["12h"] = NewStats()
		stats["7d"] = NewStats()
	} else {
		startUpdate = time.Now()
		srv.log.Info("updating 12h stats...")
		stats["12h"], err = srv.getStatsForHours(12 * time.Hour)
		if err != nil {
			srv.log.WithError(err).Error("Failed to get stats for 12h")
			return
		}
		srv.log.WithField("duration", time.Since(startUpdate).String()).Info("updated 12h stats")

		startUpdate = time.Now()
		srv.log.Info("updating 1h stats...")
		stats["1h"], err = srv.getStatsForHours(1 * time.Hour)
		if err != nil {
			srv.log.WithError(err).Error("Failed to get stats for 1h")
			return
		}
		srv.log.WithField("duration", time.Since(startUpdate).String()).Info("updated 1h stats")

		if envSkip7dStats {
			stats["7d"] = NewStats()
		} else {
			startUpdate = time.Now()
			srv.log.Info("updating 7d stats...")
			stats["7d"], err = srv.getStatsForHours(7 * 24 * time.Hour)
			if err != nil {
				srv.log.WithError(err).Error("Failed to get stats for 24h")
				return
			}
			srv.log.WithField("duration", time.Since(startUpdate).String()).Info("updated 7d stats")
		}
	}

	// Save the html data
	srv.statsLock.Lock()
	srv.stats = stats
	srv.htmlData = &htmlData
	srv.statsLock.Unlock()

	// helper
	stats24h := stats["24h"]

	// create overviewMd markdown
	overviewMd := fmt.Sprintf("Top relays - 24h, %s UTC, via relayscan.io \n\n```\n", startTime.Format("2006-01-02 15:04"))
	overviewMd += relayTable(stats24h.TopRelays)
	overviewMd += fmt.Sprintf("```\n\nTop builders - 24h, %s UTC, via relayscan.io \n\n```\n", startTime.Format("2006-01-02 15:04"))
	overviewMd += builderTable(stats24h.TopBuilders)
	overviewMd += "```"
	overviewMdBytes := []byte(overviewMd)

	builderProfitMd := fmt.Sprintf("Builder profits - 24h, %s UTC, via relayscan.io/builder-profit \n\n```\n", startTime.Format("2006-01-02 15:04"))
	builderProfitMd += builderProfitTable(stats24h.BuilderProfits)
	builderProfitMd += "```"
	builderProfitMdBytes := []byte(builderProfitMd)

	// prepare commonly used views
	srv.markdownSummaryRespLock.Lock()
	srv.markdownOverview = &overviewMdBytes
	srv.markdownBuilderProfit = &builderProfitMdBytes
	srv.markdownSummaryRespLock.Unlock()

	// srv.statsAPIRespLock.Lock()
	// resp := statsResp{
	// 	GeneratedAt: uint64(srv.HTMLData.GeneratedAt.Unix()),
	// 	DataStartAt: uint64(stats24h.Since.Unix()),
	// 	TopRelays:   stats24h.TopRelays,
	// 	TopBuilders: stats24h.TopBuilders,
	// }
	// respBytes, err := json.Marshal(resp)
	// if err != nil {
	// 	srv.log.WithError(err).Error("error marshalling statsAPIResp")
	// } else {
	// 	srv.statsAPIResp = &respBytes
	// }
	// srv.statsAPIRespLock.Unlock()
	duration := time.Since(startTime)
	srv.log.WithField("duration", duration.String()).Info("Updating HTML data complete.")
}

func (srv *Webserver) RespondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := HTTPErrorResp{code, message}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		srv.log.WithField("response", resp).WithError(err).Error("Couldn't write error response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (srv *Webserver) RespondErrorJSON(w http.ResponseWriter, code int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		srv.log.WithField("response", response).WithError(err).Error("Couldn't write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (srv *Webserver) RespondOK(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		srv.log.WithField("response", response).WithError(err).Error("Couldn't write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (srv *Webserver) handleRoot(w http.ResponseWriter, req *http.Request) {
	timespan := req.URL.Query().Get("t")
	if timespan == "" {
		timespan = "24h"
	}

	view := "overview"
	title := "MEV-Boost Relay & Builder Stats"
	if strings.HasSuffix(req.URL.Path, "builder-profit") {
		view = "builder-profit"
		title = "MEV-Boost Builder Profitability"
	}

	srv.statsLock.RLock()
	htmlData := srv.htmlData
	htmlData.Stats = srv.stats[timespan]
	srv.statsLock.RUnlock()

	htmlData.Title = title
	htmlData.TimeSpan = timespan
	htmlData.View = view

	if srv.opts.Dev {
		tpl, err := ParseIndexTemplate()
		if err != nil {
			srv.log.WithError(err).Error("root: error parsing template")
			return
		}
		w.WriteHeader(http.StatusOK)
		err = tpl.ExecuteTemplate(w, "base", htmlData)
		if err != nil {
			srv.log.WithError(err).Error("root: error executing template")
			return
		}
		return
	}

	// production flow...
	htmlBuf := bytes.Buffer{}

	// Render template
	if err := srv.templateIndex.ExecuteTemplate(&htmlBuf, "base", htmlData); err != nil {
		srv.log.WithError(err).Error("error rendering template")
		srv.RespondError(w, http.StatusInternalServerError, "error rendering template")
		return
	}

	// Minify
	htmlBytes, err := srv.minifier.Bytes("text/html", htmlBuf.Bytes())
	if err != nil {
		srv.log.WithError(err).Error("error minifying html")
		srv.RespondError(w, http.StatusInternalServerError, "error minifying html")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(htmlBytes)
}

// func (srv *Webserver) handleStatsAPI(w http.ResponseWriter, req *http.Request) {
// 	srv.statsAPIRespLock.RLock()
// 	defer srv.statsAPIRespLock.RUnlock()
// 	_, _ = w.Write(*srv.statsAPIResp)
// }

func (srv *Webserver) handleOverviewMarkdown(w http.ResponseWriter, req *http.Request) {
	srv.markdownSummaryRespLock.RLock()
	defer srv.markdownSummaryRespLock.RUnlock()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(*srv.markdownOverview)
}

func (srv *Webserver) handleBuilderProfitMarkdown(w http.ResponseWriter, req *http.Request) {
	srv.markdownSummaryRespLock.RLock()
	defer srv.markdownSummaryRespLock.RUnlock()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(*srv.markdownBuilderProfit)
}

func (srv *Webserver) _getDailyStats(t time.Time) (since, until, minDate time.Time, relays []*database.TopRelayEntry, builders []*database.TopBuilderEntry, builderProfits []*database.BuilderProfitEntry, err error) {
	now := time.Now().UTC()
	minDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour).UTC()
	if t.UTC().After(minDate.UTC()) {
		return now, now, minDate, nil, nil, nil, fmt.Errorf("date is too recent") //nolint:goerr113
	}

	since = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	until = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
	relays, builders, builderProfits, err = srv.db.GetStatsForTimerange(since, until, "")
	return since, until, minDate, relays, builders, builderProfits, err
}

func (srv *Webserver) handleDailyStats(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	layout := "2006-01-02"
	t, err := time.Parse(layout, vars["day"])
	if err != nil {
		srv.RespondError(w, http.StatusBadRequest, "invalid date")
		return
	}

	srv.log.Infof("Loading daily stats for %s ...", t.Format("2006-01-02"))
	since, until, minDate, relays, builders, builderProfits, err := srv._getDailyStats(t)
	if err != nil {
		srv.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	srv.log.Infof("Loading daily stats for %s completed. builderProfits: %d", t.Format("2006-01-02"), len(builderProfits))

	dateNext := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour).UTC()
	dayNext := dateNext.Format("2006-01-02")
	if dateNext.After(minDate.UTC()) {
		dayNext = ""
	}

	htmlData := &HTMLDataDailyStats{
		Title:                "MEV-Boost Stats for " + t.Format("2006-01-02"),
		Day:                  t.Format("2006-01-02"),
		DayPrev:              t.Add(-24 * time.Hour).Format("2006-01-02"),
		DayNext:              dayNext,
		TimeSince:            since.Format("2006-01-02 15:04"),
		TimeUntil:            until.Format("2006-01-02 15:04"),
		TopRelays:            prepareRelaysEntries(relays),
		TopBuildersBySummary: consolidateBuilderEntries(builders),
		BuilderProfits:       consolidateBuilderProfitEntries(builderProfits),
	}

	if srv.opts.Dev {
		tpl, err := ParseDailyStatsTemplate()
		if err != nil {
			srv.log.WithError(err).Error("root: error parsing template")
			return
		}
		w.WriteHeader(http.StatusOK)
		err = tpl.ExecuteTemplate(w, "base", htmlData)
		if err != nil {
			srv.log.WithError(err).Error("root: error executing template")
			return
		}
		return
	} else {
		w.WriteHeader(http.StatusOK)
		err = srv.templateDailyStats.ExecuteTemplate(w, "base", htmlData)
		if err != nil {
			srv.log.WithError(err).Error("error executing template")
			srv.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
}

func (srv *Webserver) handleDailyStatsJSON(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	layout := "2006-01-02"
	t, err := time.Parse(layout, vars["day"])
	if err != nil {
		srv.RespondError(w, http.StatusBadRequest, "invalid date")
		return
	}

	_, _, _, relays, builders, _, err := srv._getDailyStats(t) //nolint:dogsled
	if err != nil {
		srv.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	type apiResp struct {
		Date     string                      `json:"date"`
		Relays   []*database.TopRelayEntry   `json:"relays"`
		Builders []*database.TopBuilderEntry `json:"builders"`
	}

	resp := apiResp{
		Date:     t.Format("2006-01-02"),
		Relays:   prepareRelaysEntries(relays),
		Builders: consolidateBuilderEntries(builders),
	}

	srv.RespondOK(w, resp)
}

func (srv *Webserver) handleCowstatsJSON(w http.ResponseWriter, req *http.Request) {
	// builder stats for wednesday utc 00:00 to next wednesday 00:00
	type apiResp struct {
		DateFrom    string                      `json:"date_from"`
		DateTo      string                      `json:"date_to"`
		TopBuilders []*database.TopBuilderEntry `json:"top_builders"`
	}

	wednesday1 := getLastWednesday()
	wednesday2 := wednesday1.AddDate(0, 0, -7)

	startTime := time.Now()
	srv.log.WithField("from", wednesday2).WithField("to", wednesday1).Info("[cowstats] getting top builders...")
	topBuilders, err := srv.db.GetTopBuilders(wednesday2, wednesday1, "")
	if err != nil {
		srv.log.WithError(err).Error("Failed to get top builders")
		return
	}
	srv.log.WithField("duration", time.Since(startTime).String()).WithField("nBuilders", len(topBuilders)).Info("[cowstats] got top builders")

	resp := apiResp{
		DateFrom:    wednesday2.String(),
		DateTo:      wednesday1.String(),
		TopBuilders: consolidateBuilderEntries(topBuilders),
	}
	srv.RespondOK(w, resp)
}

func (srv *Webserver) handleLivenessCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (srv *Webserver) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	currentSlot := common.TimeToSlot(time.Now().UTC())
	slotsPerMinute := 5
	maxMinutesSinceLastUpdate := 10
	maxSlotsSinceLastUpdate := maxMinutesSinceLastUpdate * slotsPerMinute

	type apiResp struct {
		IsHealthy        bool   `json:"is_healthy"`
		CurrentSlot      uint64 `json:"current_slot"`
		LatestUpdateSlot uint64 `json:"latest_update_slot"`
		SlotsSinceUpdate uint64 `json:"slots_since_update"`
		Message          string `json:"message"`
	}

	resp := apiResp{
		IsHealthy:        true,
		CurrentSlot:      currentSlot,
		LatestUpdateSlot: srv.latestSlot,
		SlotsSinceUpdate: currentSlot - srv.latestSlot,
	}

	if currentSlot-srv.latestSlot > uint64(maxSlotsSinceLastUpdate) {
		resp.IsHealthy = false
		resp.Message = "No updates for too long"
		srv.RespondErrorJSON(w, http.StatusInternalServerError, resp)
	}

	srv.RespondOK(w, resp)
}
