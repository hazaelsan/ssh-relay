package http

import (
	"errors"
)

var (
	// ErrBadAddr is returned when an address is invalid (e.g., it includes a port).
	ErrBadAddr = errors.New("bad address specified")

	// ErrMissingPort is returned if the port is required but not specified.
	ErrMissingPort = errors.New("no port specified")

	// ErrNoTLSConfig is returned if tls_config is not specified.
	ErrNoTLSConfig = errors.New("tls_config must be specified")
)
