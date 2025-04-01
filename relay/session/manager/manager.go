// Package manager implements an SSH-over-WebSocket Session manager for the SSH Relay.
package manager

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/hazaelsan/ssh-relay/session"
	"github.com/hazaelsan/ssh-relay/session/corprelay"
	"github.com/hazaelsan/ssh-relay/session/corprelayv4"
)

var (
	// ErrNoSuchSID is returned when a SID was not found in the session registry.
	ErrNoSuchSID = errors.New("no such SID")

	// ErrSessionLimit is returned when the maximum session limit is reached.
	ErrSessionLimit = errors.New("session limit reached")
)

// New instantiates a *Manager with a limit of sessions and individual session age.
func New(maxSessions int, maxAge time.Duration) *Manager {
	return &Manager{
		maxSessions: maxSessions,
		maxAge:      maxAge,
		sessions:    make(map[uuid.UUID]session.Session),
	}
}

// Manager is an SSH-over-WebSocket Session manager.
// It enforces a session limit as well as individual session lifetimes.
type Manager struct {
	maxAge      time.Duration
	maxSessions int
	sessions    map[uuid.UUID]session.Session
	mu          sync.RWMutex
}

// New creates and registers a Session from an SSH connection.
func (m *Manager) New(ssh net.Conn, v session.ProtocolVersion) (session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.maxSessions > 0 && len(m.sessions) >= m.maxSessions {
		return nil, ErrSessionLimit
	}
	var s session.Session
	switch v {
	case session.CorpRelay:
		s = corprelay.New(ssh)
	case session.CorpRelayV4:
		s = corprelayv4.New(ssh, session.Server)
	default:
		return nil, session.ErrBadProtocolVersion
	}
	session.SetDeadline(s, m.maxAge)
	go func() {
		<-s.Done()
		m.Delete(s.SID())
	}()
	m.sessions[s.SID()] = s
	glog.V(1).Infof("%v/%v active sessions", len(m.sessions), m.maxSessions)
	return s, nil
}

// Get retrieves the Session with the given UUID.
func (m *Manager) Get(sid uuid.UUID) (session.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sid]
	if !ok {
		return nil, ErrNoSuchSID
	}
	return s, nil
}

// Delete terminates the Session with the given UUID and de-registers it.
func (m *Manager) Delete(sid uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.sessions[sid]
	if !ok {
		return ErrNoSuchSID
	}
	delete(m.sessions, sid)
	glog.V(4).Infof("%v: Session terminated", sid)
	return nil
}
