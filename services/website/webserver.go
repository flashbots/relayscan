// Package website contains the service delivering the website
package website

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
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
	timespans               = []string{"7d", "24h", "12h", "1h"}
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

	// data
	stats    map[string]*Stats
	html     map[string]*[]byte // HTML for common views
	dataLock sync.RWMutex

	latestSlot uberatomic.Uint64

	markdownSummaryRespLock sync.RWMutex
	markdownOverview        *[]byte
	markdownBuilderProfit   *[]byte
}

func NewWebserver(opts *WebserverOpts) (*Webserver, error) {
	var err error

	minifier := minify.New()
	minifier.AddFunc("text/css", html.Minify)
	minifier.AddFunc("text/html", html.Minify)
	minifier.AddFunc("application/javascript", html.Minify)

	opts.Only24h = opts.Dev

	server := &Webserver{
		opts:                  opts,
		log:                   opts.Log,
		db:                    opts.DB,
		stats:                 make(map[string]*Stats),
		html:                  make(map[string]*[]byte),
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
	go srv.startRootHTMLUpdateLoops()

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
	r.HandleFunc("/overview/json", srv.handleOverviewJSON).Methods(http.MethodGet)
	r.HandleFunc("/builder-profit/md", srv.handleBuilderProfitMarkdown).Methods(http.MethodGet)
	r.HandleFunc("/builder-profit/json", srv.handleBuilderProfitJSON).Methods(http.MethodGet)

	r.HandleFunc("/stats/cowstats", srv.handleCowstatsJSON).Methods(http.MethodGet)
	r.HandleFunc("/stats/day/{day:[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}}", srv.handleDailyStats).Methods(http.MethodGet)
	r.HandleFunc("/stats/day/{day:[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}}/json", srv.handleDailyStatsJSON).Methods(http.MethodGet)
	r.HandleFunc("/stats/_test/extradata-payloads", srv.handleExtraDataPayloads).Methods(http.MethodGet)

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
	if strings.HasSuffix(req.URL.Path, "builder-profit") {
		view = "builder-profit"
	}

	// Re-render in dev mode
	if srv.opts.Dev {
		srv.dataLock.RLock()
		stats, dataFound := srv.stats[timespan]
		srv.dataLock.RUnlock()
		if !dataFound {
			srv.RespondError(w, http.StatusInternalServerError, "no data for timespan")
			return
		}
		overviewBytes, profitBytes, err := srv._renderRootHTML(stats)
		if err != nil {
			srv.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if view == "builder-profit" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(profitBytes)
			return
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(overviewBytes)
			return
		}
	}

	// In production mode, just return pre-rendered HTML bytes
	htmlKey := fmt.Sprintf("%s-%s", timespan, view)
	srv.dataLock.RLock()
	htmlBytes, htmlFound := srv.html[htmlKey]
	srv.dataLock.RUnlock()
	if !htmlFound {
		srv.log.WithFields(logrus.Fields{
			"timespan": timespan,
			"view":     view,
		}).Warn("No data for timespan")
		if timespan == "24h" && view == "overview" {
			srv.RespondError(w, http.StatusInternalServerError, "server starting, waiting for initial data...")
		} else {
			srv.RespondError(w, http.StatusInternalServerError, "no data for timespan")
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(*htmlBytes)
}

func (srv *Webserver) handleOverviewMarkdown(w http.ResponseWriter, req *http.Request) {
	srv.markdownSummaryRespLock.RLock()
	defer srv.markdownSummaryRespLock.RUnlock()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(*srv.markdownOverview)
}

func (srv *Webserver) handleOverviewJSON(w http.ResponseWriter, req *http.Request) {
	timespan := req.URL.Query().Get("t")
	if timespan == "" {
		timespan = "24h"
	}

	srv.dataLock.RLock()
	stats, dataFound := srv.stats[timespan]
	srv.dataLock.RUnlock()

	if !dataFound {
		srv.RespondError(w, http.StatusInternalServerError, "no data for timespan")
		return
	}

	type apiResp struct {
		Timespan string                    `json:"timespan"`
		Since    string                    `json:"since"`
		Until    string                    `json:"until"`
		Relays   []*database.TopRelayEntry `json:"relays"`
		Builders []*TopBuilderDisplayEntry `json:"builders"`
	}

	resp := apiResp{
		Timespan: stats.TimeStr,
		Since:    stats.Since.Format("2006-01-02 15:04:05"),
		Until:    stats.Until.Format("2006-01-02 15:04:05"),
		Relays:   stats.TopRelays,
		Builders: stats.TopBuilders,
	}

	srv.RespondOK(w, resp)
}

func (srv *Webserver) handleBuilderProfitMarkdown(w http.ResponseWriter, req *http.Request) {
	srv.markdownSummaryRespLock.RLock()
	defer srv.markdownSummaryRespLock.RUnlock()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(*srv.markdownBuilderProfit)
}

func (srv *Webserver) handleBuilderProfitJSON(w http.ResponseWriter, req *http.Request) {
	timespan := req.URL.Query().Get("t")
	if timespan == "" {
		timespan = "24h"
	}

	srv.dataLock.RLock()
	stats, dataFound := srv.stats[timespan]
	srv.dataLock.RUnlock()

	if !dataFound {
		srv.RespondError(w, http.StatusInternalServerError, "no data for timespan")
		return
	}

	type apiResp struct {
		Timespan       string                         `json:"timespan"`
		Since          string                         `json:"since"`
		Until          string                         `json:"until"`
		BuilderProfits []*database.BuilderProfitEntry `json:"builder_profits"`
	}

	resp := apiResp{
		Timespan:       stats.TimeStr,
		Since:          stats.Since.Format("2006-01-02 15:04:05"),
		Until:          stats.Until.Format("2006-01-02 15:04:05"),
		BuilderProfits: stats.BuilderProfits,
	}

	srv.RespondOK(w, resp)
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
		Date     string                    `json:"date"`
		Relays   []*database.TopRelayEntry `json:"relays"`
		Builders []*TopBuilderDisplayEntry `json:"builders"`
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
		DateFrom    string                    `json:"date_from"`
		DateTo      string                    `json:"date_to"`
		TopBuilders []*TopBuilderDisplayEntry `json:"top_builders"`
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

	latestSlotInDB := srv.latestSlot.Load()
	resp := apiResp{
		IsHealthy:        true,
		CurrentSlot:      currentSlot,
		LatestUpdateSlot: latestSlotInDB,
		SlotsSinceUpdate: currentSlot - latestSlotInDB,
	}

	if currentSlot-latestSlotInDB > uint64(maxSlotsSinceLastUpdate) { //nolint:gosec
		resp.IsHealthy = false
		resp.Message = "No updates for too long"
		srv.RespondErrorJSON(w, http.StatusInternalServerError, resp)
	}

	srv.RespondOK(w, resp)
}

// Return recent payloads for a given extra data string
func (srv *Webserver) handleExtraDataPayloads(w http.ResponseWriter, r *http.Request) {
	type apiRespEntry struct {
		Slot             uint64 `json:"slot"`
		SlotTime         string `json:"slot_time"`
		ExtraData        string `json:"extra_data"`
		PrevSlot         uint64 `json:"prev_slot"`
		MinSincePrevSlot uint64 `json:"min_since_prev_slot"`
	}

	extraData := r.URL.Query().Get("extra_data")
	extraDataSlice := []string{extraData}
	if extraData == "fb" {
		extraDataSlice = []string{"Illuminate Dmocratize Dstribute", "Illuminate Dmocrtz Dstrib Prtct"}
	}

	limit := 20
	limitArg := r.URL.Query().Get("limit")
	if limitArg != "" {
		l, err := strconv.Atoi(limitArg)
		if err != nil {
			srv.RespondError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = l
	}

	payloads, err := srv.db.GetRecentPayloadsForExtraData(extraDataSlice, limit+1)
	if err != nil {
		srv.log.WithError(err).Error("error getting payloads")
		srv.RespondError(w, http.StatusInternalServerError, "error getting payloads")
		return
	}

	entries := make([]apiRespEntry, len(payloads))
	var prevSlot uint64
	var prevSlotTime time.Time

	// iterate over payloads in reverse
	for i := len(payloads) - 1; i >= 0; i-- {
		payload := payloads[i]
		slotStartTime := common.SlotToTime(payload.Slot)
		entry := apiRespEntry{
			Slot:      payload.Slot,
			SlotTime:  slotStartTime.Format("2006-01-02 15:04:05"),
			ExtraData: payload.ExtraData,
		}
		if prevSlot > 0 {
			entry.PrevSlot = prevSlot
			entry.MinSincePrevSlot = uint64(slotStartTime.Sub(prevSlotTime).Abs().Minutes())
		}
		entries[i] = entry
		prevSlotTime = slotStartTime
		prevSlot = payload.Slot
	}

	// trim last entry
	entries = entries[:len(entries)-1]
	srv.RespondOK(w, entries)
}

// func (srv *Webserver) updateHTML() {
// 	var err error
// 	srv.log.Info("Updating HTML data...")

// 	// helper
// 	stats24h := stats

// 	// create overviewMd markdown
// 	overviewMd := fmt.Sprintf("Top relays - 24h, %s UTC, via relayscan.io \n\n```\n", startTime.Format("2006-01-02 15:04"))
// 	overviewMd += relayTable(stats24h.TopRelays)
// 	overviewMd += fmt.Sprintf("```\n\nTop builders - 24h, %s UTC, via relayscan.io \n\n```\n", startTime.Format("2006-01-02 15:04"))
// 	overviewMd += builderTable(stats24h.TopBuilders)
// 	overviewMd += "```"
// 	overviewMdBytes := []byte(overviewMd)

// 	builderProfitMd := fmt.Sprintf("Builder profits - 24h, %s UTC, via relayscan.io/builder-profit \n\n```\n", startTime.Format("2006-01-02 15:04"))
// 	builderProfitMd += builderProfitTable(stats24h.BuilderProfits)
// 	builderProfitMd += "```"
// 	builderProfitMdBytes := []byte(builderProfitMd)

// 	// prepare commonly used views
// 	srv.markdownSummaryRespLock.Lock()
// 	srv.markdownOverview = &overviewMdBytes
// 	srv.markdownBuilderProfit = &builderProfitMdBytes
// 	srv.markdownSummaryRespLock.Unlock()

// 	duration := time.Since(startTime)
// 	srv.log.WithField("duration", duration.String()).Info("Updating HTML data complete.")
// }
