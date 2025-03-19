// Package corprelayv4 implements the corp-relay-v4@google.com protocol, see
// https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/docs/relay-protocol.md#corp-relay-v4.
package corprelayv4

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hazaelsan/ssh-relay/session"
	"github.com/hazaelsan/ssh-relay/session/corprelayv4/command"
)

// newWriter defines a function to get a WebSocket writer, used for ease of testing.
type newWriter func(int) (io.WriteCloser, error)

var (
	// ErrInvalidSession is returned when a session is in an invalid state.
	ErrInvalidSession = errors.New("invalid session")
)

// New creates a *Session from a given SSH connection.
func New(ssh io.ReadWriteCloser, role session.Role) *Session {
	s := &Session{
		ssh:  ssh,
		role: role,
		done: make(chan struct{}),
	}
	if s.role == session.Server {
		s.sid = uuid.New()
	}
	return s
}

// A Session is a V4 SSH-over-Websocket Relay session.
// TODO: Implement reconnect logic.
type Session struct {
	sid    uuid.UUID
	ssh    io.ReadWriteCloser
	ws     *websocket.Conn
	rCount uint64
	wCount uint64
	mu     sync.RWMutex
	wFunc  newWriter
	role   session.Role
	done   chan struct{}
}

func (s *Session) String() string {
	return s.sid.String()
}

// SID returns the Session ID.
func (s *Session) SID() uuid.UUID {
	return s.sid
}

// Version returns the protocol version in use for the session.
func (s *Session) Version() session.ProtocolVersion {
	return session.CorpRelayV4
}

// Close closes the SSH connection, causing the Session to be invalid.
func (s *Session) Close() error {
	err := s.ssh.Close()
	s.done <- struct{}{}
	return err
}

// Done notifies when a session has terminated.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

// Run starts a new session between the WebSocket and SSH connections.
func (s *Session) Run(ws *websocket.Conn) error {
	s.mu.Lock()
	defer func() {
		s.mu.Unlock()
		s.Close()
	}()
	s.ws = ws
	s.wFunc = s.ws.NextWriter
	errc := make(chan error)
	go s.runWS(errc)
	go s.runSSH(errc)
	return <-errc
}

// recvCmd parses an in-band command from a WebSocket stream.
func (s *Session) recvCmd(b []byte) error {
	c, err := command.Unmarshal(b)
	if err != nil {
		return fmt.Errorf("command.Unmarshal(%v) error: %w", b, err)
	}
	switch c.Tag() {
	case command.TagConnectSuccess:
		b := c.(command.ConnectSuccess).SID()
		// CONNECT_SUCCESS can only be sent to a client as the first command.
		if s.role != session.Client || !bytes.Equal(s.sid[:], uuid.Nil[:]) {
			break
		}
		s.sid, err = uuid.Parse(b)
		if err != nil {
			return fmt.Errorf("uuid.Parse(%v) error: %w", b, err)
		}
		return nil
	case command.TagReconnectSuccess:
		return errors.New("not implemented")
	case command.TagData:
		if err := s.readData(c.(command.Data)); err != nil {
			return err
		}
		return s.sendAck()
	case command.TagAck:
		return s.readAck(c.(command.Ack))
	}
	return fmt.Errorf("%w: %v", command.ErrBadCommand, c.Tag())
}

// establishConn sends the initial CONNECT_SUCCESS/RECONNECT_SUCCESS command to establish a connection.
func (s *Session) establishConn(ack uint64) error {
	if ack > 0 {
		return s.sendReconnect(ack)
	}
	return s.sendConnect()
}

// sendConnect sends a CONNECT_SUCCESS command.
func (s *Session) sendConnect() error {
	sid, err := s.sid.MarshalText()
	if err != nil {
		return err
	}
	w, err := s.wFunc(websocket.BinaryMessage)
	if err != nil {
		return fmt.Errorf("wFunc() error: %w", err)
	}
	defer w.Close()
	cs, err := command.NewConnectSuccess(sid)
	if err != nil {
		return err
	}
	return cs.Write(w)
}

// sendReconnect sends a RECONNECT_SUCCESS command.
func (s *Session) sendReconnect(ack uint64) error {
	return errors.New("not implemented")
}

// readData processes an incoming DATA command.
func (s *Session) readData(d command.Data) error {
	data := d.Data()
	s.rCount += uint64(len(data))
	glog.V(5).Infof("%v: ws->ssh read %v bytes", s, len(data))
	if _, err := s.ssh.Write(data); err != nil {
		return err
	}
	return nil
}

// sendAck sends an ACK command in response to received data from one or more DATA commands.
func (s *Session) sendAck() error {
	w, err := s.wFunc(websocket.BinaryMessage)
	if err != nil {
		return fmt.Errorf("wFunc() error: %w", err)
	}
	defer w.Close()
	a := command.NewAck(s.rCount)
	return a.Write(w)
}

// readAck processes an incoming ACK command.
func (s *Session) readAck(a command.Ack) error {
	ack := a.Ack()
	diff := ack - s.wCount
	if diff == 0 {
		return nil
	}
	if diff < 0 {
		return fmt.Errorf("reverse ack %v -> %v", s.wCount, ack)
	}
	s.wCount = ack
	return nil
}

// runSSH handles reads from SSH, as well as the initial session connection establishment.
func (s *Session) runSSH(errc chan<- error) {
	if s.role == session.Server {
		if err := s.establishConn(0); err != nil {
			errc <- err
			return
		}
	}
	for {
		err := func() error {
			r := bufio.NewReader(s.ssh)
			b := make([]byte, command.MaxArrayLen)
			n, err := r.Read(b)
			data := b[0:n]
			glog.V(5).Infof("%v: ssh->ws read %v bytes", s, n)
			if err != nil {
				return err
			}

			w, err := s.wFunc(websocket.BinaryMessage)
			if err != nil {
				return fmt.Errorf("wFunc() error: %w", err)
			}
			defer w.Close()

			return s.copySSH(w, data)
		}()
		if err != nil {
			errc <- err
			return
		}
	}
}

// copySSH copies the actual data bytes from ssh->ws.
func (s *Session) copySSH(w io.Writer, b []byte) error {
	d, err := command.NewData(b)
	if err != nil {
		return err
	}
	return d.Write(w)
}

// runWS handles reads from the WebSocket.
func (s *Session) runWS(errc chan<- error) {
	for {
		t, r, err := s.ws.NextReader()
		if err != nil {
			errc <- fmt.Errorf("NextReader() error: %w", err)
			return
		}

		err = func() error {
			switch t {
			case websocket.BinaryMessage:
				return s.parseBinary(r)
			default:
				return fmt.Errorf("unsupported message type: %v", t)
			}
		}()
		if err != nil {
			errc <- err
			return
		}
	}
}

// parseBinary handles a ws->ssh message.
func (s *Session) parseBinary(r io.Reader) error {
	b := new(bytes.Buffer)
	if _, err := b.ReadFrom(r); err != nil {
		return err
	}
	return s.recvCmd(b.Bytes())
}
