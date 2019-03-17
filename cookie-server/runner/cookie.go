package runner

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/cookie-server/request/cookie"
	"github.com/hazaelsan/ssh-relay/cookie-server/request/cookie/handler"
)

// handleCookie services /cookie requests.
// TODO: Implement actual client authnz.
func (r *Runner) handleCookie(w http.ResponseWriter, req *http.Request) {
	cr, err := cookie.New(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h, err := handler.New(r.cfg, cr, w, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Handle(); err != nil {
		glog.Error(err)
	}
}
