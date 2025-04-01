package runner

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"github.com/hazaelsan/ssh-relay/relay/request"
	"github.com/hazaelsan/ssh-relay/session"
)

// TODO: Implement /v4/reconnect logic.

// connectHandleV4 handles /v4/connect requests.
func (r *Runner) connectHandleV4(w http.ResponseWriter, req *http.Request) {
	var s session.Session
	var ws *websocket.Conn
	var addr string
	code, err := func() (code int, err error) {
		host := req.URL.Query().Get("host")
		port := req.URL.Query().Get("port")
		origin, err := request.Origin(req, r.cfg.OriginCookieName)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("request.Origin(%v) error: %w", r.cfg.OriginCookieName, err)
		}
		addr = net.JoinHostPort(host, port)
		ssh, err := net.Dial("tcp", addr)
		if err != nil {
			return http.StatusBadGateway, fmt.Errorf("net.Dial(%v) error: %w", addr, err)
		}
		s, err = r.mgr.New(ssh, session.CorpRelayV4)
		if err != nil {
			return http.StatusServiceUnavailable, fmt.Errorf("mgr.New(%v) error: %w", addr, err)
		}
		glog.V(4).Infof("%v: Connected to %v", s, addr)

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return r.Header.Get("Origin") == origin
			},
			Subprotocols: []string{"ssh"},
		}
		ws, err = upgrader.Upgrade(w, req, nil)
		if err != nil {
			return http.StatusBadGateway, fmt.Errorf("upgrader.Upgrade(%v) error: %w", origin, err)
		}
		return 0, nil
	}()
	if err != nil {
		http.Error(w, errors.Unwrap(err).Error(), code)
		if glog.V(5) {
			glog.Error(err)
		}
		return
	}
	defer ws.Close()
	if err := s.Run(ws); err != nil {
		if errors.Is(err, io.EOF) {
			glog.V(1).Infof("%v: Connection to %v closed", s, addr)
			if err := r.mgr.Delete(s.SID()); err != nil {
				glog.Errorf("mgr.Delete(%v) error: %v", s, err)
			}
			return
		}
		glog.Error(err)
	}
}
