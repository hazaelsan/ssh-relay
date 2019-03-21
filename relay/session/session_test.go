package session

import (
	"net"
	"testing"
	"time"
)

// testConn is a wrapper to test a net.Conn to see if the underlying connection is closed.
type testConn struct {
	net.Conn
	closed bool
}

func (t *testConn) Close() error {
	t.closed = true
	return t.Conn.Close()
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
	duration := 100 * time.Millisecond
	a, _ := net.Pipe()
	conn := &testConn{Conn: a}
	_, c := New(conn, duration)
	if isDone(c, duration) {
		t.Errorf("done = true before %v expired", duration)
	}
	if conn.closed {
		t.Errorf("conn.closed = true before %v expired", duration)
	}

	time.Sleep(2 * duration)
	if !isDone(c, duration) {
		t.Errorf("done = false after %v expired", duration)
	}
	if !conn.closed {
		t.Errorf("conn.closed = false after %v expired", duration)
	}
}
