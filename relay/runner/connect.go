package runner

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"github.com/hazaelsan/ssh-relay/relay/request/connect"
	"github.com/hazaelsan/ssh-relay/request"
)

// connectHandle handles /connect requests.
// WebSocket session, handles bidirectional traffic.
func (r *Runner) connectHandle(w http.ResponseWriter, req *http.Request) {
	cr, err := connect.New(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s, err := r.mgr.Get(cr.SID)
	if err != nil {
		if glog.V(1) {
			glog.Errorf("session(%v) error: %v", s, err)
		}
		http.Error(w, request.ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.mgr.Delete(s); err != nil && glog.V(1) {
			glog.Errorf("Delete(%v) error: %v", s, err)
		}
	}()
	origin, err := r.origin(req)
	if err != nil {
		if glog.V(1) {
			glog.Errorf("%v: Invalid origin: %v", s, origin)
		}
		http.Error(w, request.ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(req *http.Request) bool {
			return req.Header.Get("Origin") == origin
		},
	}
	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		if glog.V(1) {
			glog.Errorf("%v: Upgrade() error: %v", cr, err)
		}
		return
	}
	defer ws.Close()

	notify, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}
	go func() {
		select {
		case <-notify.CloseNotify():
			glog.Infof("%v: Closing connection from %v", s, req.RemoteAddr)
			s.Close()
		}
	}()

	if err := s.Run(ws); err != nil && glog.V(2) {
		glog.Errorf("%v: s.Run() error: %v", s, err)
	}
}
