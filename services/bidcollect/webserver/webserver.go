// Package webserver provides a webserver for SSE stream of transactions
package webserver

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/flashbots/relayscan/services/bidcollect/types"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
)

type HTTPServerConfig struct {
	ListenAddr string
	Log        *logrus.Entry

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Server struct {
	cfg     *HTTPServerConfig
	isReady atomic.Bool
	log     *logrus.Entry

	srv               *http.Server
	sseConnectionMap  map[string]*SSESubscription
	sseConnectionLock sync.RWMutex
}

func New(cfg *HTTPServerConfig) (srv *Server) {
	srv = &Server{
		cfg:              cfg,
		log:              cfg.Log,
		srv:              nil,
		sseConnectionMap: make(map[string]*SSESubscription),
	}
	srv.isReady.Swap(true)

	router := chi.NewRouter()
	router.Get("/v1/sse/bids", srv.handleSSESubscription)

	srv.srv = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return srv
}

func (srv *Server) RunInBackground() {
	go func() {
		srv.log.WithField("listenAddress", srv.cfg.ListenAddr).Info("Starting HTTP server")
		if err := srv.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srv.log.WithField("err", err).Error("HTTP server failed")
		}
	}()
}

func (srv *Server) Shutdown() {
	// api
	if err := srv.srv.Shutdown(context.Background()); err != nil {
		srv.log.WithField("err", err).Error("Graceful HTTP server shutdown failed")
	} else {
		srv.log.Info("HTTP server gracefully stopped")
	}
}

func (srv *Server) addSubscriber(sub *SSESubscription) {
	srv.sseConnectionLock.Lock()
	defer srv.sseConnectionLock.Unlock()
	srv.sseConnectionMap[sub.uid] = sub
}

func (srv *Server) removeSubscriber(sub *SSESubscription) {
	srv.sseConnectionLock.Lock()
	defer srv.sseConnectionLock.Unlock()
	delete(srv.sseConnectionMap, sub.uid)
	srv.log.WithField("subscribers", len(srv.sseConnectionMap)).Info("removed subscriber")
}

func (srv *Server) SendBid(bid *types.CommonBid) {
	srv.sseConnectionLock.RLock()
	defer srv.sseConnectionLock.RUnlock()
	if len(srv.sseConnectionMap) == 0 {
		return
	}

	msg := bid.ToCSVLine("\t")

	// Send tx to all subscribers (only if channel is not full)
	for _, sub := range srv.sseConnectionMap {
		select {
		case sub.msgC <- msg:
		default:
		}
	}
}
