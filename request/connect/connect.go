// Package connect represents a /connect request.
package connect

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/hazaelsan/ssh-relay/request"
)

// New creates a *Request from an *http.Request.
// TODO: Session resumption is not supported, pos/try MUST be 0.
func New(req *http.Request) (*Request, error) {
	var err error
	r := new(Request)
	r.SID, err = uuid.Parse(req.URL.Query().Get("sid"))
	if err != nil {
		return nil, request.ErrBadRequest
	}
	r.ack, err = request.Uint(req, "ack")
	if err != nil || r.ack != 0 {
		return nil, request.ErrBadRequest
	}
	r.pos, err = request.Uint(req, "pos")
	if err != nil || r.pos != 0 {
		return nil, request.ErrBadRequest
	}
	r.try, err = request.Uint(req, "try")
	if err != nil {
		return nil, request.ErrBadRequest
	}
	return r, nil
}

// A Request is a normalized request to /connect.
type Request struct {
	SID uuid.UUID
	ack uint
	pos uint
	try uint
}

func (r Request) String() string {
	return r.SID.String()
}
