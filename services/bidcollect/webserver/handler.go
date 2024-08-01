package webserver

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type SSESubscription struct {
	uid  string
	msgC chan string
}

func (srv *Server) handleSSESubscription(w http.ResponseWriter, r *http.Request) {
	// SSE server for transactions
	srv.log.Info("SSE connection opened for transactions")

	// Set CORS headers to allow all origins. You may want to restrict this to specific origins in a production environment.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	subscriber := SSESubscription{
		uid:  uuid.New().String(),
		msgC: make(chan string, 100),
	}
	srv.addSubscriber(&subscriber)

	// pingTicker := time.NewTicker(5 * time.Second)

	// Wait for txs or end of request...
	for {
		select {
		case <-r.Context().Done():
			srv.log.Info("SSE closed, removing subscriber")
			srv.removeSubscriber(&subscriber)
			return

		case msg := <-subscriber.msgC:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush() //nolint:forcetypeassert

			// case <-pingTicker.C:
			// 	fmt.Fprintf(w, ": ping\n\n")
			// 	w.(http.Flusher).Flush() //nolint:forcetypeassert
		}
	}
}
