package manager

import (
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hazaelsan/ssh-relay/relay/session"
)

func TestNew(t *testing.T) {
	testdata := []struct {
		maxAge      time.Duration
		maxSessions int
		sessions    int
		hasLimit    bool
	}{
		{
			maxAge:      100 * time.Millisecond,
			maxSessions: 1,
			sessions:    1,
			hasLimit:    true,
		},
		{
			maxAge:   100 * time.Millisecond,
			sessions: 1000,
		},
		{
			maxAge:      100 * time.Millisecond,
			maxSessions: 10,
			sessions:    10,
			hasLimit:    true,
		},
		{
			maxAge:      100 * time.Millisecond,
			maxSessions: -1,
			sessions:    10,
		},
	}
	p, _ := net.Pipe()
	for i, tt := range testdata {
		m := &Manager{
			maxAge:      tt.maxAge,
			maxSessions: tt.maxSessions,
			sessions:    make(map[uuid.UUID]*session.Session),
		}
		// Test up to session limits.
		for j := 0; j < tt.sessions; j++ {
			if _, err := m.New(p); err != nil {
				t.Errorf("New(%v, %v) error = %v", i, j, err)
			}
		}
		// Test one past the session limit.
		if _, err := m.New(p); err != nil {
			if !tt.hasLimit {
				t.Errorf("New(%v, %v) error = %v", i, tt.sessions, err)
			}
		} else if tt.hasLimit {
			t.Errorf("New(%v, %v) error = nil", i, tt.sessions)
		}
		// Test limits after sessions have expired.
		if tt.hasLimit {
			time.Sleep(2 * tt.maxAge)
			if _, err := m.New(p); err != nil {
				t.Errorf("New(%v) error = %v", i, err)
			}
		}
	}
}
