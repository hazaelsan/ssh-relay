// Package cookie represents a /cookie request to the Cookie Server.
package cookie

import (
	"net/http"

	"github.com/hazaelsan/ssh-relay/request"

	requestpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/request_go_proto"
)

// Map of redirection method name to enum.
var redirectionMethodMap = map[string]requestpb.RedirectionMethod{
	"direct":      requestpb.RedirectionMethod_DIRECT,
	"js-redirect": requestpb.RedirectionMethod_JS_REDIRECT,
}

// New creates a *Request from an *http.Request.
// There's some special logic regarding query params:
//   * ext/path are required
//   * If version is unspecified, it defaults to 1
//   * Version 1 clients only support HTTP redirection
//   * Version 2 clients MUST specify a redirection method (which MUST NOT be HTTP redirection)
func New(req *http.Request) (*requestpb.Request, error) {
	r := &requestpb.Request{
		Ext:     req.URL.Query().Get("ext"),
		Path:    req.URL.Query().Get("path"),
		Version: 1,
		Method:  requestpb.RedirectionMethod_HTTP_REDIRECT,
	}
	if r.Ext == "" || r.Path == "" {
		return nil, request.ErrBadRequest
	}

	method := req.URL.Query().Get("method")
	if version := req.URL.Query().Get("version"); version != "" {
		v, err := request.Uint(req, "version")
		if err != nil || v < 1 {
			return nil, request.ErrBadRequest
		}
		r.Version = int32(v)

		// V2 clients MUST specify version.
		if v >= 2 && method == "" {
			return nil, request.ErrBadRequest
		}
	}

	if method != "" {
		// V1 clients MUST NOT specify version.
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
