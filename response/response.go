// Package response implements a JSON response for client redirection, see
// https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/doc/relay-protocol.md#corp-relay-method.
package response

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
)

const (
	xssiHeader = ")]}'"
)

var (
	// ErrNoXSSIHeader is returned if no XSSI header was found.
	ErrNoXSSIHeader = errors.New("XSSI header not found")
)

// XSSI returns an XSSI header to send on a Response.
func XSSI() string {
	return xssiHeader + "\n"
}

// FromReader creates a *Response from an io.Reader by first stripping out the XSSI header.
func FromReader(r io.Reader) (*Response, error) {
	b := make([]byte, len(xssiHeader))
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	if string(b) != xssiHeader {
		return nil, ErrNoXSSIHeader
	}
	dec := json.NewDecoder(r)
	resp := new(Response)
	err := dec.Decode(resp)
	return resp, err
}

// FromEndpoint creates a *Response with a given Endpoint.
func FromEndpoint(endpoint string) *Response {
	return &Response{Endpoint: endpoint}
}

// FromError creates a *Response with a given Error message.
func FromError(msg string) *Response {
	return &Response{Error: msg}
}

// A Response is a configuration response sent to clients.
type Response struct {
	Endpoint string `json:"endpoint"`
	Error    string `json:"error,omitempty"`
}

// Encode performs URL base64 encoding on the Response.
func (r *Response) Encode() (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Marshal marshals the Response.
func (r *Response) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// MarshalXSSI marshals the Response with an XSSI header prepended.
func (r *Response) MarshalXSSI() ([]byte, error) {
	b, err := r.Marshal()
	if err != nil {
		return nil, err
	}
	return append([]byte(XSSI()), b...), nil
}
