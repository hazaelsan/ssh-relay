// Package handler implements an HTTP handler for corp-relay@google.com /connect requests.
package handler

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/hazaelsan/ssh-relay/relay/request"
	"github.com/hazaelsan/ssh-relay/relay/request/corprelay/connect"
	"github.com/hazaelsan/ssh-relay/session"

	configpb "github.com/hazaelsan/ssh-relay/relay/proto/v1/config"
)

// New creates a *Handler for an HTTP request.
func New(cfg *configpb.Config, s session.Session, cr *connect.Request, w http.ResponseWriter, r *http.Request) (*Handler, error) {
	h := &Handler{
		s:  s,
		cr: cr,
		w:  w,
		r:  r,
	}
	var err error
	h.origin, err = request.Origin(r, cfg.OriginCookieName)
	return h, err
}

// A Handler is an HTTP handler for /connect requests, handles bidirectional SSH traffic.
type Handler struct {
	origin string
	s      session.Session
	cr     *connect.Request
	w      http.ResponseWriter
	r      *http.Request
}

// Handle processes the /connect HTTP request, WebSocket session.
func (h *Handler) Handle() error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Origin") == h.origin
		},
	}
	ws, err := upgrader.Upgrade(h.w, h.r, nil)
	if err != nil {
		return err
	}
	defer ws.Close()
	return h.s.Run(ws)
}
