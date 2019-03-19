// Package session implements a container for an SSH-over-WebSocket session with a limited lifetime.
package session

import (
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hazaelsan/ssh-relay/session"
)

// New creates a *Session from an SSH connection with the given lifetime,
// returns a channel that's closed when the session has expired.
func New(ssh net.Conn, t time.Duration) (*Session, <-chan struct{}) {
	s := &Session{
		SID: uuid.New(),
		s:   session.New(ssh),
	}
	glog.V(2).Infof("%v: Creating session with maximum lifetime: %v", s, t)
	done := make(chan struct{})
	go func() {
		select {
		case <-time.After(t):
			glog.V(2).Infof("%v: Session expired", s)
			s.s.Close()
			close(done)
		}
	}()
	return s, done
}

// A Session is a container for an SSH session.
type Session struct {
	SID uuid.UUID
	s   *session.Session
}

func (s Session) String() string {
	return s.SID.String()
}

// Close closes the SSH connection, causing the Session to be invalid.
func (s *Session) Close() error {
	return s.s.Close()
}

// Run starts bidirectional communication between the WebSocket and SSH connections.
func (s *Session) Run(ws *websocket.Conn) error {
	return s.s.Run(ws)
}
