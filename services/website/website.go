// Package website contains the service delivering the website
package website

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"text/template"
	"time"

	_ "net/http/pprof"

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

	indexTemplate    *template.Template
	HTMLData         HTMLData
	rootResponseLock sync.RWMutex

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

	server.indexTemplate, err = ParseIndexTemplate()
	if err != nil {
		return nil, err
	}

	server.HTMLData = HTMLData{}

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

	since := time.Now().UTC().Add(-24 * time.Hour)
	topRelays, err := srv.db.GetTopRelays(since)
	if err != nil {
		srv.log.WithError(err).Error("failed getting top relays from database")
		return
	}

	topBuilders, err := srv.db.GetTopBuilders(since)
	if err != nil {
		srv.log.WithError(err).Error("failed getting top builders from database")
		return
	}

	htmlData := HTMLData{}
	htmlData.TopRelays = topRelays
	topRelaysNumPayloads := uint64(0)
	for _, entry := range topRelays {
		topRelaysNumPayloads += entry.NumPayloads
	}
	for i, entry := range topRelays {
		p := float64(entry.NumPayloads) / float64(topRelaysNumPayloads) * 100
		htmlData.TopRelays[i].Percent = fmt.Sprintf("%.2f", p)
	}

	htmlData.TopBuilders = topBuilders
	topBuildersNumPayloads := uint64(0)
	for _, entry := range topBuilders {
		topBuildersNumPayloads += entry.NumBlocks
	}
	for i, entry := range topBuilders {
		p := float64(entry.NumBlocks) / float64(topBuildersNumPayloads) * 100
		htmlData.TopBuilders[i].Percent = fmt.Sprintf("%.2f", p)
	}

	htmlData.GeneratedAt = time.Now().UTC()
	htmlData.LastUpdateTime = htmlData.GeneratedAt.Format("2006-01-02 15:04")

	// default view
	if err := srv.indexTemplate.Execute(&htmlDefault, htmlData); err != nil {
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

func (srv *Webserver) handleRoot(w http.ResponseWriter, req *http.Request) {
	var err error

	srv.rootResponseLock.RLock()
	defer srv.rootResponseLock.RUnlock()

	if srv.opts.Dev {
		// tpl :=
		tpl, err := template.New("website.html").Funcs(funcMap).ParseFiles("services/website/website.html")
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
	w.Write(*srv.statsAPIResp)
}
