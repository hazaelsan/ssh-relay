// Package request implements common functionality to all HTTP requests.
package request

import (
	"errors"
	"net/http"
	"strconv"
)

var (
	// ErrBadRequest is returned when a request is not valid.
	ErrBadRequest = errors.New("bad request")
)

// Uint parses the given URL parameter and returns is as an uint.
func Uint(req *http.Request, key string) (uint, error) {
	i, err := strconv.ParseUint(req.URL.Query().Get(key), 10, 64)
	return uint(i), err
}
