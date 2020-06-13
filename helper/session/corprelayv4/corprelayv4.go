// Package corprelayv4 implements a corp-relay-v4@google.com SSH-over-WebSocket Relay client session.
//
// NOTE: Reconnections are not implemented.
package corprelayv4

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	hsession "github.com/hazaelsan/ssh-relay/helper/session"
	"github.com/hazaelsan/ssh-relay/session"
	"github.com/hazaelsan/ssh-relay/session/corprelayv4"
	"github.com/hazaelsan/ssh-relay/tls"
)

// New creates a *Session.
// Communication to ssh(1) is done via stdin/stdout.
func New(opts hsession.Options) *Session {
	ssh := hsession.NewWrapper(os.Stdin, os.Stdout)
	return &Session{
		opts: opts,
		s:    corprelayv4.New(ssh, session.Client),
	}
}

// A Session is a corp-relay-v4@google.com SSH-over-WebSocket Relay client session.
type Session struct {
	opts hsession.Options
	s    session.Session
	ws   *websocket.Conn
}

// Run copies I/O to an SSH host through a WebSocket Relay via /v4/connect.
func (s *Session) Run() error {
	u := s.connectURL()
	if err := s.dial(u); err != nil {
		return fmt.Errorf("dial(%v) error: %w", u, err)
	}
	return s.s.Run(s.ws)
}

// Done notifies when the Session has terminated.
func (s *Session) Done() <-chan struct{} {
	return s.s.Done()
}

// connectHeader builds an http.Header for a /v4/connect request.
func (s *Session) connectHeader() http.Header {
	h := http.Header{}
	h.Add("Origin", s.opts.Origin)
	for _, c := range s.opts.Cookies {
		h.Add("Cookie", c.String())
	}
	return h
}

// connectURL builds the correct URL for /v4/connect requests.
func (s *Session) connectURL() string {
	u := url.URL{
		Scheme: "wss",
		Host:   s.opts.Relay,
		Path:   "/v4/connect",
	}
	q := u.Query()
	q.Set("host", s.opts.Host)
	q.Set("port", s.opts.Port)
	u.RawQuery = q.Encode()
	return u.String()
}

// dial initiates the SSH session and sets up the WebSocket for I/O.
func (s *Session) dial(u string) error {
	glog.V(2).Infof("Copying I/O via %v", u)
	tlsCfg, err := tls.Config(s.opts.Transport.TlsConfig)
	if err != nil {
		return fmt.Errorf("tls.Config() error: %w", err)
	}
	d := &websocket.Dialer{TLSClientConfig: tlsCfg}
	s.ws, _, err = d.Dial(u, s.connectHeader())
	if err != nil {
		return fmt.Errorf("Dial(%v) error: %w", u, err)
	}
	return nil
}
