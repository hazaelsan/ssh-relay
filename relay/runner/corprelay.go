package runner

import (
	"fmt"
	"net"
	"net/http"

	"github.com/golang/glog"
	rrequest "github.com/hazaelsan/ssh-relay/relay/request"
	"github.com/hazaelsan/ssh-relay/relay/request/corprelay/connect"
	"github.com/hazaelsan/ssh-relay/relay/request/corprelay/connect/handler"
	"github.com/hazaelsan/ssh-relay/relay/request/corprelay/proxy"
	"github.com/hazaelsan/ssh-relay/request"
	"github.com/hazaelsan/ssh-relay/session"
)

// connectHandle handles /connect requests.
// WebSocket session, handles bidirectional traffic.
func (r *Runner) connectHandle(w http.ResponseWriter, req *http.Request) {
	cr, err := connect.New(req)
	if err != nil {
		if glog.V(1) {
			glog.Errorf("connect.New(%v) error: %v", req, err)
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s, err := r.mgr.Get(cr.SID)
	if err != nil {
		if glog.V(1) {
			glog.Errorf("mgr.Get(%v) error: %v", cr.SID, err)
		}
		http.Error(w, request.ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.mgr.Delete(s.SID()); err != nil && glog.V(1) {
			glog.Errorf("mgr.Delete(%v) error: %v", s, err)
		}
	}()
	h, err := handler.New(r.cfg, s, cr, w, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Handle(); err != nil {
		glog.Error(err)
	}
}

// proxyHandle handles /proxy requests.
// Sets up the SSH connection and returns the SID to the client.
func (r *Runner) proxyHandle(w http.ResponseWriter, req *http.Request) {
	pr, err := proxy.New(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	origin, err := rrequest.Origin(req, r.cfg.OriginCookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	addr := net.JoinHostPort(pr.Host, pr.Port)
	ssh, err := net.Dial("tcp", addr)
	if err != nil {
		if glog.V(1) {
			glog.Errorf("Dial(%v) error: %v", addr, err)
		}
		http.Error(w, "connection error", http.StatusBadGateway)
		return
	}
	s, err := r.mgr.New(ssh, session.CorpRelay)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	glog.V(4).Infof("%v: Connected to %v", s, addr)
	w.Header().Add("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	fmt.Fprint(w, s.SID())
}
