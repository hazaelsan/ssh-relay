// Package session defines an SSH Relay protocol implementation.
package session

import (
	"errors"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ProtocolVersion is the version of the SSH relay protocol to use in a session.
type ProtocolVersion int

const (
	// CorpRelay is the original corp-relay@google.com protocol version.
	CorpRelay ProtocolVersion = iota

	// CorpRelayV4 is the corp-relay-v4@google.com protocol version.
	CorpRelayV4

	// SSHFE is the ssh-fe@google.com protocol version.
	// NOTE: Not implemented.
	SSHFE
)

// Role indicates the role within a session.
type Role int

const (
	// Server indicates that this is the server side of a session.
	Server Role = iota

	// Client indicates that this is the client side of a session.
	Client
)

var (
	// ErrBadProtocolVersion is returned when a bad protocol version is requested.
	ErrBadProtocolVersion = errors.New("bad protocol version")
)

// A Session handles SSH-over-WebSocket Relay sessions.
type Session interface {
	// String returns the Session ID as a string, used for logging.
	String() string

	// SID returns the Session ID.
	SID() uuid.UUID

	// Version returns the protocol version in use for the session.
	Version() ProtocolVersion

	// Run starts bidirectional communication between the client and server.
	Run(ws *websocket.Conn) error

	// Close closes the SSH connection, causing the Session to be invalid.
	Close() error

	// Done notifies when the Session has terminated.
	Done() <-chan struct{}
}

// SetDeadline sets a maximum session deadline, after which the session will be terminated.
func SetDeadline(s Session, t time.Duration) {
	glog.V(2).Infof("%v: %v session expires in %v", s, s.Version(), t)
	go func() {
		select {
		case <-time.After(t):
			glog.V(2).Infof("%v: Session expired", s)
			s.Close()
		case <-s.Done():
			return
		}
	}()
}

func (v ProtocolVersion) String() string {
	switch v {
	case CorpRelay:
		return "corp-relay@google.com"
	case CorpRelayV4:
		return "corp-relay-v4@google.com"
	case SSHFE:
		return "ssh-fe@google.com"
	default:
		return "unknown"
	}
}
