// Package corprelay implements the corp-relay@google.com protocol, see
// https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/doc/relay-protocol.md#corp-relay.
package corprelay

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hazaelsan/ssh-relay/session"
)

const (
	// AckByteSize is the size in bytes for READ_ACK/WRITE_ACKs.
	AckByteSize = 4

	// AckErrMask is the bit mask for error ack ranges.
	AckErrMask = 0xff000000

	// ChunkSize is the size in bytes for valid read/write requests.
	ChunkSize = 0xffffff
)

var (
	// ErrInvalidAck is returned when an ack has any error bits set.
	ErrInvalidAck = errors.New("invalid ack range")
)

// New creates a *Session from a plain SSH connection.
func New(ssh io.ReadWriteCloser) *Session {
	return &Session{
		sid:  uuid.New(),
		ssh:  ssh,
		done: make(chan struct{}),
	}
}

// A Session is an SSH-over-WebSocket Relay session.
// One leg of the session is a WebSocket, the other is an io.Reader/io.Writer pair that talks plain SSH.
type Session struct {
	sid  uuid.UUID
	ssh  io.ReadWriteCloser
	ws   *websocket.Conn
	c    uint32
	mu   sync.RWMutex
	done chan struct{}
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
	return session.CorpRelay
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

// incCounter increments the counter by n, wrapping every 24 bits.
func (s *Session) incCounter(n int) {
	s.c = (s.c + uint32(n)) & ChunkSize
}

// Run starts bidirectional communication between the WebSocket and SSH connections.
func (s *Session) Run(ws *websocket.Conn) error {
	s.mu.Lock()
	defer func() {
		s.mu.Unlock()
		s.Close()
	}()
	s.ws = ws
	errc := make(chan error)
	go s.runSSH(errc)
	go s.runWS(errc)

	// From here on, we can't do anything about failures, and we want to report the original error.
	err := <-errc
	w, wErr := s.ws.NextWriter(websocket.BinaryMessage)
	if wErr != nil {
		return err
	}
	defer w.Close()

	// Inform the WebSocket the connection is in an error state.
	s.c |= AckErrMask
	_ = writeAck(w, s.c)
	return err
}

// runSSH handles ssh->ws writes.
// Data is sent in 32KiB chunks, the first 4 bytes are the ack.
// Only the lower 3 bytes in the ack are used, a non-zero high byte indicates a connection error.
func (s *Session) runSSH(errc chan<- error) {
	for {
		err := func() error {
			r := bufio.NewReader(s.ssh)
			b := make([]byte, ChunkSize)
			n, err := r.Read(b)
			data := b[0:n]
			glog.V(5).Infof("ssh->ws read %v bytes", n)
			if err != nil {
				return err
			}

			w, err := s.ws.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return fmt.Errorf("NextWriter() error: %w", err)
			}
			defer w.Close()
			if err := writeAck(w, s.c); err != nil {
				return fmt.Errorf("writeAck(%v) error: %w", s.c, err)
			}
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
	n, err := w.Write(b)
	glog.V(5).Infof("ssh->ws wrote %v bytes", n)
	return err
}

// runWS handles ws->ssh writes.
// Data is sent in 32KiB chunks, the first 4 bytes are the ack.
// Only the lower 3 bytes in the ack are used, a non-zero high byte indicates a connection error.
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
			case websocket.TextMessage:
				return s.parseText(r)
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
	if _, err := readAck(r); err != nil {
		return fmt.Errorf("readAck() error: %w", err)
	}
	return s.copyWS(r)
}

// parseText handles non-SSH control messages:
//   ack latency:     A:integer
//   reply latency:   R:integer
// TODO: Do something useful with these.
func (s *Session) parseText(r io.Reader) error {
	b := new(bytes.Buffer)
	if _, err := b.ReadFrom(r); err != nil {
		return err
	}
	glog.V(1).Infof("Ignoring TEXT: %v", b)
	return nil
}

// copyWS copies the actual data bytes from ws->ssh.
func (s *Session) copyWS(r io.Reader) error {
	b := new(bytes.Buffer)
	n, err := b.ReadFrom(r)
	if err != nil {
		return err
	}
	s.incCounter(int(n))
	glog.V(5).Infof("ws->ssh read %v bytes", n)
	_, err = s.ssh.Write([]byte(b.String()))
	return err
}

// readAck consumes the ack from a WebSocket reader.
func readAck(r io.Reader) (uint32, error) {
	b := make([]byte, AckByteSize)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, fmt.Errorf("ReadFull() error: %w", err)
	}
	ack := binary.BigEndian.Uint32(b)
	if ack > ChunkSize {
		return ack, ErrInvalidAck
	}
	return ack, nil
}

// writeAck serializes and writes an ack to a WebSocket writer.
func writeAck(w io.Writer, ack uint32) error {
	b := make([]byte, AckByteSize)
	binary.BigEndian.PutUint32(b, ack)
	_, err := w.Write(b)
	return err
}
