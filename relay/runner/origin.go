package runner

import (
	"net/http"
	"strings"
)

// origin returns the validated value of the origin cookie from an *http.Request.
// TODO: Improve validation, current logic is just a Proof of Concept.
func (r *Runner) origin(req *http.Request) (string, error) {
	if r.cfg.OriginCookieName == "" {
		return "", ErrBadOrigin
	}
	cookie, err := req.Cookie(r.cfg.OriginCookieName)
	if err != nil {
		return "", err
	}
	if !strings.Contains(cookie.Value, "://") {
		return "", ErrBadOrigin
	}
	return cookie.Value, nil
}
