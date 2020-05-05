// Package session defines an SSH Relay protocol implementation.
package session

import (
	"github.com/gorilla/websocket"
)

// A Session is an SSH-over-WebSocket Relay session.
type Session interface {
	// Run starts bidirectional communication between the client and server.
	Run(ws *websocket.Conn) error

	// Close closes the SSH connection, causing the Session to be invalid.
	Close() error

	// Done notifies when the Session has terminated.
	Done() <-chan struct{}
}
