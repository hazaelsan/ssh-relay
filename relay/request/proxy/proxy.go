// Package proxy represents a /proxy request to the SSH Relay.
package proxy

import (
	"net/http"

	"github.com/hazaelsan/ssh-relay/request"
)

// New creates a *Request from an *http.Request.
func New(req *http.Request) (*Request, error) {
	r := &Request{
		Host: req.URL.Query().Get("host"),
		Port: req.URL.Query().Get("port"),
	}
	if r.Host == "" || r.Port == "" {
		return nil, request.ErrBadRequest
	}
	return r, nil
}

// A Request is a normalized request to /proxy.
type Request struct {
	Host string
	Port string
}
