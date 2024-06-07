// Package website contains the service delivering the website
package website

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
	uberatomic "go.uber.org/atomic"
)

var ErrServerAlreadyStarted = errors.New("server was already started")

type DevWebserverOpts struct {
	ListenAddress string
	Log           *logrus.Entry
}

type DevWebserver struct {
	opts *DevWebserverOpts
	log  *logrus.Entry

	srv        *http.Server
	srvStarted uberatomic.Bool
	minifier   *minify.M
}

func NewDevWebserver(opts *DevWebserverOpts) (server *DevWebserver, err error) {
	minifier := minify.New()
	minifier.AddFunc("text/css", html.Minify)
	minifier.AddFunc("text/html", html.Minify)
	minifier.AddFunc("application/javascript", html.Minify)

	server = &DevWebserver{ //nolint:exhaustruct
		opts:     opts,
		log:      opts.Log,
		minifier: minifier,
	}

	return server, nil
}

func (srv *DevWebserver) StartServer() (err error) {
	if srv.srvStarted.Swap(true) {
		return ErrServerAlreadyStarted
	}

	srv.srv = &http.Server{ //nolint:exhaustruct
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

func (srv *DevWebserver) getRouter() http.Handler {
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./website/static"))))

	r.HandleFunc("/", srv.handleRoot).Methods(http.MethodGet)
	r.HandleFunc("/index.html", srv.handleRoot).Methods(http.MethodGet)
	r.HandleFunc("/ethereum/mainnet/{month}/index.html", srv.handleMonth).Methods(http.MethodGet)

	return r
}

func (srv *DevWebserver) RespondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := HTTPErrorResp{code, message}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		srv.log.WithError(err).Error("Couldn't write error response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (srv *DevWebserver) RespondOK(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		srv.log.WithError(err).Error("Couldn't write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (srv *DevWebserver) handleRoot(w http.ResponseWriter, req *http.Request) {
	tpl, err := ParseIndexTemplate()
	if err != nil {
		srv.log.WithError(err).Error("wroot: error parsing template")
		return
	}
	w.WriteHeader(http.StatusOK)

	data := *DummyHTMLData
	data.Path = "/"
	err = tpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		srv.log.WithError(err).Error("wroot: error executing template")
		return
	}
}

func (srv *DevWebserver) handleMonth(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	layout := "2006-01"
	_, err := time.Parse(layout, vars["month"])
	if err != nil {
		srv.RespondError(w, http.StatusBadRequest, "invalid date")
		return
	}

	tpl, err := ParseFilesTemplate()
	if err != nil {
		srv.log.WithError(err).Error("wroot: error parsing template")
		return
	}
	w.WriteHeader(http.StatusOK)

	data := *DummyHTMLData
	data.Title = vars["month"]
	data.Path = fmt.Sprintf("ethereum/mainnet/%s/index.html", vars["month"])

	err = tpl.ExecuteTemplate(w, "base", &data)
	if err != nil {
		srv.log.WithError(err).Error("wroot: error executing template")
		return
	}
}
