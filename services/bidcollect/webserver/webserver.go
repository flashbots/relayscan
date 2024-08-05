// Package webserver provides a SSE stream of new bids (via Redis subscription)
package webserver

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/flashbots/relayscan/services/bidcollect/types"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type HTTPServerConfig struct {
	ListenAddr string
	RedisAddr  string
	Log        *logrus.Entry
}

type Server struct {
	cfg *HTTPServerConfig
	log *logrus.Entry
	srv *http.Server

	redisClient *redis.Client

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

	router := chi.NewRouter()
	router.Get("/v1/sse/bids", srv.handleSSESubscription)

	srv.srv = &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 1 * time.Second,
	}
	return srv
}

func (srv *Server) MustStart() {
	go srv.MustSubscribeToRedis()

	srv.log.WithField("listenAddress", srv.cfg.ListenAddr).Info("Starting HTTP server")
	if err := srv.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		srv.log.WithField("err", err).Error("HTTP server failed")
	}
}

func (srv *Server) MustSubscribeToRedis() {
	if srv.cfg.RedisAddr == "" {
		srv.log.Fatal("Redis address is required")
	}

	srv.log.Info("Subscribing to Redis...")
	srv.redisClient = redis.NewClient(&redis.Options{
		Addr:     srv.cfg.RedisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Make sure we can connect to redis to connect to redis
	if _, err := srv.redisClient.Ping(context.Background()).Result(); err != nil {
		srv.log.WithError(err).Fatal("failed to ping redis")
	}

	pubsub := srv.redisClient.Subscribe(context.Background(), types.RedisChannel)
	ch := pubsub.Channel()
	srv.log.Info("Subscribed to Redis")

	for msg := range ch {
		srv.SendToSubscribers(msg.Payload)
	}
}

func (srv *Server) addSubscriber(sub *SSESubscription) {
	srv.sseConnectionLock.Lock()
	defer srv.sseConnectionLock.Unlock()
	srv.sseConnectionMap[sub.uid] = sub
	srv.log.WithField("subscribers", len(srv.sseConnectionMap)).Info("subscriber added")
}

func (srv *Server) removeSubscriber(sub *SSESubscription) {
	srv.sseConnectionLock.Lock()
	defer srv.sseConnectionLock.Unlock()
	delete(srv.sseConnectionMap, sub.uid)
	srv.log.WithField("subscribers", len(srv.sseConnectionMap)).Info("subscriber removed")
}

func (srv *Server) SendToSubscribers(msg string) {
	srv.sseConnectionLock.RLock()
	defer srv.sseConnectionLock.RUnlock()
	if len(srv.sseConnectionMap) == 0 {
		return
	}

	// Send tx to all subscribers (only if channel is not full)
	for _, sub := range srv.sseConnectionMap {
		select {
		case sub.msgC <- msg:
		default:
		}
	}
}
