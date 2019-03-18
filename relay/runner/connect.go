package runner

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/relay/request/connect"
	"github.com/hazaelsan/ssh-relay/relay/request/connect/handler"
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
