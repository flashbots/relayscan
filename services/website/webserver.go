// Package website contains the service delivering the website
package website

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"text/template"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/flashbots/go-utils/httplogger"
	"github.com/gorilla/mux"
	"github.com/metachris/relayscan/database"
	"github.com/sirupsen/logrus"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
	uberatomic "go.uber.org/atomic"
)

var ErrServerAlreadyStarted = errors.New("server was already started")

type WebserverOpts struct {
	ListenAddress string
	DB            *database.DatabaseService
	Log           *logrus.Entry
	EnablePprof   bool
	Dev           bool // reloads template on every request
}

type Webserver struct {
	opts *WebserverOpts
	log  *logrus.Entry

	db *database.DatabaseService

	srv        *http.Server
	srvStarted uberatomic.Bool

	HTMLData         HTMLData
	rootResponseLock sync.RWMutex

	templateIndex      *template.Template
	templateDailyStats *template.Template

	statsAPIResp     *[]byte
	statsAPIRespLock sync.RWMutex

	htmlDefault *[]byte
	minifier    *minify.M
}

func NewWebserver(opts *WebserverOpts) (*Webserver, error) {
	var err error

	minifier := minify.New()
	minifier.AddFunc("text/css", html.Minify)
	minifier.AddFunc("text/html", html.Minify)

	server := &Webserver{
		opts: opts,
		log:  opts.Log,
		db:   opts.DB,

		htmlDefault: &[]byte{},
		minifier:    minifier,
	}

	server.templateDailyStats, err = template.New("index").Funcs(funcMap).Parse(htmlContentDailyStats)
	if err != nil {
		return nil, err
	}

	server.templateIndex, err = ParseIndexTemplate()
	if err != nil {
		return nil, err
	}

	server.HTMLData = HTMLData{} //nolint:exhaustruct

	return server, nil
}

func (srv *Webserver) StartServer() (err error) {
	if srv.srvStarted.Swap(true) {
		return ErrServerAlreadyStarted
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
		WriteTimeout:      3 * time.Second,
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
	r.HandleFunc("/", srv.handleRoot).Methods(http.MethodGet)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.HandleFunc("/stats/day/{day:[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}}", srv.handleDailyStats).Methods(http.MethodGet)
	r.HandleFunc("/api/stats", srv.handleStatsAPI).Methods(http.MethodGet)

	if srv.opts.EnablePprof {
		srv.log.Info("pprof API enabled")
		r.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	}

	loggedRouter := httplogger.LoggingMiddlewareLogrus(srv.log, r)
	withGz := gziphandler.GzipHandler(loggedRouter)
	return withGz
}

func (srv *Webserver) updateHTML() {
	// Now generate the HTML
	htmlDefault := bytes.Buffer{}

	now := time.Now().UTC()
	since := now.Add(-24 * time.Hour)
	topRelays, err := srv.db.GetTopRelays(since, now)
	if err != nil {
		srv.log.WithError(err).Error("failed getting top relays from database")
		return
	}

	topBuilders, err := srv.db.GetTopBuilders(since, now)
	if err != nil {
		srv.log.WithError(err).Error("failed getting top builders from database")
		return
	}

	htmlData := HTMLData{} //nolint:exhaustruct
	htmlData.GeneratedAt = time.Now().UTC()
	htmlData.LastUpdateTime = htmlData.GeneratedAt.Format("2006-01-02 15:04")

	// Prepare top relay stats
	htmlData.TopRelays = prepareRelaysEntries(topRelays)
	htmlData.TopBuilders = consolidateBuilderEntries(topBuilders)

	// Render template
	if err := srv.templateIndex.Execute(&htmlDefault, htmlData); err != nil {
		srv.log.WithError(err).Error("error rendering template")
	}

	// Minify
	htmlDefaultBytes, err := srv.minifier.Bytes("text/html", htmlDefault.Bytes())
	if err != nil {
		srv.log.WithError(err).Error("error minifying htmlDefault")
	}

	// Swap the html pointers
	srv.rootResponseLock.Lock()
	srv.HTMLData = htmlData
	srv.htmlDefault = &htmlDefaultBytes
	srv.rootResponseLock.Unlock()

	srv.statsAPIRespLock.Lock()
	resp := statsResp{
		GeneratedAt: uint64(srv.HTMLData.GeneratedAt.Unix()),
		DataStartAt: uint64(since.Unix()),
		TopRelays:   srv.HTMLData.TopRelays,
		TopBuilders: srv.HTMLData.TopBuilders,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		srv.log.WithError(err).Error("error marshalling statsAPIResp")
	} else {
		srv.statsAPIResp = &respBytes
	}
	srv.statsAPIRespLock.Unlock()
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

func (srv *Webserver) RespondOK(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		srv.log.WithField("response", response).WithError(err).Error("Couldn't write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (srv *Webserver) handleRoot(w http.ResponseWriter, req *http.Request) {
	var err error

	srv.rootResponseLock.RLock()
	defer srv.rootResponseLock.RUnlock()

	if srv.opts.Dev {
		tpl, err := template.New("index.html").Funcs(funcMap).ParseFiles("services/website/templates/index.html")
		if err != nil {
			srv.log.WithError(err).Error("error parsing template")
			return
		}
		err = tpl.Execute(w, srv.HTMLData)
		if err != nil {
			srv.log.WithError(err).Error("error executing template")
			return
		}

		srv.log.Info("rendered template")
	} else {
		_, err = w.Write(*srv.htmlDefault)
	}
	if err != nil {
		srv.log.WithError(err).Error("error writing template")
	}
}

func (srv *Webserver) handleStatsAPI(w http.ResponseWriter, req *http.Request) {
	srv.statsAPIRespLock.RLock()
	defer srv.statsAPIRespLock.RUnlock()
	_, _ = w.Write(*srv.statsAPIResp)
}

func (srv *Webserver) handleDailyStats(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	// 1. ensure date is before today
	layout := "2006-01-02"
	t, err := time.Parse(layout, vars["day"])
	if err != nil {
		srv.RespondError(w, http.StatusBadRequest, "invalid date")
		return
	}
	now := time.Now().UTC()
	minDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour).UTC()
	if t.UTC().After(minDate.UTC()) {
		srv.RespondError(w, http.StatusBadRequest, "date is too recent")
		return
	}

	// 2. lookup daily stats from DB?
	// 3. query stats from DB
	since := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	until := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
	relays, builders, err := srv.db.GetStatsForTimerange(since, until)
	if err != nil {
		srv.RespondError(w, http.StatusBadRequest, "invalid date")
		return
	}

	dateNext := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour).UTC()
	dayNext := dateNext.Format("2006-01-02")
	if dateNext.After(minDate.UTC()) {
		dayNext = ""
	}

	htmlData := &HTMLDataDailyStats{
		Day:                  t.Format("2006-01-02"),
		DayPrev:              t.Add(-24 * time.Hour).Format("2006-01-02"),
		DayNext:              dayNext,
		TimeSince:            since.Format("2006-01-02 15:04:05 UTC"),
		TimeUntil:            until.Format("2006-01-02 15:04:05 UTC"),
		TopRelays:            prepareRelaysEntries(relays),
		TopBuildersBySummary: consolidateBuilderEntries(builders),
	}

	if srv.opts.Dev {
		tpl, err := template.New("daily-stats.html").Funcs(funcMap).ParseFiles("services/website/templates/daily-stats.html")
		if err != nil {
			srv.log.WithError(err).Error("error parsing template")
			srv.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		err = tpl.Execute(w, htmlData)
		if err != nil {
			srv.log.WithError(err).Error("error executing template")
			srv.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		err = srv.templateDailyStats.Execute(w, htmlData)
		if err != nil {
			srv.log.WithError(err).Error("error executing template")
			srv.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
}
