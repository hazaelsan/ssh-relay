package session

import (
	"net"
	"sync"
	"testing"
	"time"
)

// testConn is a wrapper to test a net.Conn to see if the underlying connection is closed.
type testConn struct {
	net.Conn
	c  bool
	mu sync.RWMutex
}

func (t *testConn) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.c = true
	return t.Conn.Close()
}

func (t *testConn) closed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.c
}

func isDone(c <-chan struct{}, d time.Duration) bool {
	select {
	case <-c:
		return true
	case <-time.After(d):
		return false
	}
}

func TestNew(t *testing.T) {
	timeout := 300 * time.Millisecond
	a, _ := net.Pipe()
	conn := &testConn{Conn: a}
	s := New(conn, 2*timeout)
	if isDone(s.Done(), timeout) {
		t.Errorf("s.Done() = true before %v expired", timeout)
	}
	if conn.closed() {
		t.Errorf("conn.closed() = true before %v expired", timeout)
	}
	time.Sleep(timeout)
	if !isDone(s.Done(), timeout) {
		t.Errorf("s.Done() = false after %v expired", timeout)
	}
	if !conn.closed() {
		t.Errorf("conn.closed() = false after %v expired", timeout)
	}
}
