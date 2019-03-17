// Package cookie represents a /cookie request to the Cookie Server.
package cookie

import (
	"net/http"
	"strconv"

	"github.com/hazaelsan/ssh-relay/request"
)

// RedirectionMethod indicates how to redirect clients to an SSH Relay.
type RedirectionMethod int

const (
	// HTTPRedirect redirects clients via a plain HTTP redirect, used for v1 clients.
	HTTPRedirect RedirectionMethod = iota

	// Direct returns a JSON response with an XSSI header.  The client is responsible for parsing and handling the actual redirection.
	Direct

	// JSRedirect returns a base64-encoded JSON in a URI frament response in a JavaScript redirect.
	JSRedirect
)

// Map of redirection method name to enum.
var redirectionMethodMap = map[string]RedirectionMethod{
	"direct":      Direct,
	"js-redirect": JSRedirect,
}

// New creates a *Request from an *http.Request.
// There's some special logic regarding query params:
//   * ext/path are required
//   * If version is unspecified, it defaults to 1
//   * Version 1 clients only support HTTP redirection
//   * Version 2+ clients must specify a redirection method (which must not be HTTP redirection)
func New(req *http.Request) (*Request, error) {
	r := &Request{
		Ext:     req.URL.Query().Get("ext"),
		Path:    req.URL.Query().Get("path"),
		Version: 1,
		Method:  HTTPRedirect,
	}
	if r.Ext == "" || r.Path == "" {
		return nil, request.ErrBadRequest
	}

	method := req.URL.Query().Get("method")
	if version := req.URL.Query().Get("version"); version != "" {
		v, err := strconv.Atoi(req.URL.Query().Get("version"))
		if err != nil || v < 1 {
			return nil, request.ErrBadRequest
		}
		r.Version = v

		// V2+ clients must specify version.
		if v >= 2 && method == "" {
			return nil, request.ErrBadRequest
		}
	}

	if method != "" {
		// V1 clients must not specify version.
		if r.Version == 1 {
			return nil, request.ErrBadRequest
		}

		m, ok := redirectionMethodMap[req.URL.Query().Get("method")]
		if !ok {
			return nil, request.ErrBadRequest
		}
		r.Method = m
	}

	return r, nil
}

// A Request is a normalized request to /cookie.
type Request struct {
	Ext     string
	Path    string
	Version int
	Method  RedirectionMethod
}
