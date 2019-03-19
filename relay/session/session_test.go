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

func TestNew(t *testing.T) {
	duration := time.Second
	a, _ := net.Pipe()
	conn := &testConn{Conn: a}
	_ = New(conn, duration)
	if conn.closed {
		t.Errorf("conn.closed = true before %v expired", duration)
	}
	time.Sleep(duration)
	if !conn.closed {
		t.Errorf("conn.closed = false after %v expired", duration)
	}
}
