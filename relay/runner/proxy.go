package runner

import (
	"fmt"
	"net"
	"net/http"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/relay/request/proxy"
	"github.com/hazaelsan/ssh-relay/request"
)

// proxyHandle handles /proxy requests.
// Sets up the SSH connection and returns the SID to the client.
func (r *Runner) proxyHandle(w http.ResponseWriter, req *http.Request) {
	pr, err := proxy.New(req)
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
	s, err := r.mgr.New(ssh)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	origin, err := r.origin(req)
	if err != nil {
		if glog.V(1) {
			glog.Errorf("%v: Invalid origin for: %v", s, origin)
		}
		http.Error(w, request.ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Add("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	fmt.Fprint(w, s.SID)
}
