package runner

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/request"
)

// RedirectionMethod indicates how to redirect clients to an SSH-over-WebSocket Relay.
type RedirectionMethod int

const (
	// LegacyRedirect redirects clients via a plain HTTP redirect, used for v1 clients.
	LegacyRedirect RedirectionMethod = iota

	// Direct returns a JSON response with an XSSI header.  The client is responsible for parsing and handling the actual redirection.
	Direct

	// JSRedirect returns a base64-encoded JSON in a URI frament response in a JavaScript redirect.
	JSRedirect
)

// Map of redirection method name to enum.
var redirectionMethodMap = map[string]RedirectionMethod{
	"":            LegacyRedirect,
	"direct":      Direct,
	"js-redirect": JSRedirect,
}

// A CookieRequest is a normalized request to /cookie.
type CookieRequest struct {
	ext     string
	path    string
	version int
	method  RedirectionMethod
}

// handleCookie services /cookie requests.
// TODO: Implement actual client authnz.
func (r *Runner) handleCookie(w http.ResponseWriter, req *http.Request) {
	cr, err := NewRequest(w, req, r.cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redir := func() error {
		http.Error(w, request.ErrBadRequest.Error(), http.StatusBadRequest)
		return nil
	}
	switch cr.cr.method {
	case LegacyRedirect:
		redir = cr.RedirectURI
	case Direct:
		redir = cr.RedirectDirect
	case JSRedirect:
		redir = cr.RedirectJS
	}
	if err := redir(); err != nil {
		glog.Error(err)
	}
}
