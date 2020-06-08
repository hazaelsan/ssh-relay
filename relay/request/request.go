// Package request implements common functionality to all SSH Relay HTTP requests.
package request

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	// ErrBadOrigin is returned when the origin cookie is missing or invalid.
	ErrBadOrigin = errors.New("bad origin")
)

// Origin returns the validated value of the origin cookie from an *http.Request.
// TODO: Improve validation, current logic is just a Proof of Concept.
func Origin(req *http.Request, name string) (string, error) {
	if name == "" {
		return "", ErrBadOrigin
	}
	cookie, err := req.Cookie(name)
	if err != nil {
		return "", fmt.Errorf("req.Cookie(%v) error: %w", name, err)
	}
	if !strings.Contains(cookie.Value, "://") {
		return "", ErrBadOrigin
	}
	return cookie.Value, nil
}
