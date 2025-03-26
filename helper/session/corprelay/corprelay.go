// Package corprelay implements a corp-relay@google.com SSH-over-WebSocket Relay client session.
//
// Sessions are established in two parts:
// * /proxy: Tells the Relay to set up the SSH connection, returns a Session ID
// * /connect: SSH-over-WebSocket Relay session
//
// NOTE: Reconnections are not implemented.
package corprelay

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	hsession "github.com/hazaelsan/ssh-relay/helper/session"
	rhttp "github.com/hazaelsan/ssh-relay/http"
	"github.com/hazaelsan/ssh-relay/session"
	"github.com/hazaelsan/ssh-relay/session/corprelay"
	"github.com/hazaelsan/ssh-relay/tls"

	"github.com/hazaelsan/ssh-relay/proto/v1/tlspb"
)

// New creates a *Session.
// Communication to ssh(1) is done via stdin/stdout.
func New(opts hsession.Options) *Session {
	ssh := hsession.NewWrapper(os.Stdin, os.Stdout)
	insecure := opts.Transport.GetTlsConfig().GetTlsMode() == tlspb.TlsConfig_TLS_MODE_DISABLED
	return &Session{
		opts:     opts,
		s:        corprelay.New(ssh),
		insecure: insecure,
	}
}

// A Session is a corp-relay@google.com SSH-over-WebSocket Relay client session.
type Session struct {
	opts     hsession.Options
	s        session.Session
	ws       *websocket.Conn
	query    url.Values
	insecure bool
}

// Run copies I/O to an SSH host through a WebSocket Relay via /connect.
func (s *Session) Run() error {
	u := s.proxyURL()
	if err := s.dial(u); err != nil {
		return fmt.Errorf("dial(%v) error: %w", u, err)
	}
	return s.s.Run(s.ws)
}

// Done notifies when the Session has terminated.
func (s *Session) Done() <-chan struct{} {
	return s.s.Done()
}

// connectHeader builds an http.Header for a /connect request.
func (s *Session) connectHeader() http.Header {
	h := http.Header{}
	h.Add("Origin", s.opts.Origin)
	for _, c := range s.opts.Cookies {
		h.Add("Cookie", c.String())
	}
	return h
}

// connectURL builds the correct URL for /connect requests.
func (s *Session) connectURL() string {
	q := s.query
	q.Add("ack", "0")
	q.Add("pos", "0")
	q.Add("try", "1")
	scheme := "wss"
	if s.insecure {
		scheme = "ws"
	}
	u := url.URL{
		Scheme:   scheme,
		Host:     s.opts.Relay,
		Path:     "/connect",
		RawQuery: q.Encode(),
	}
	return u.String()
}

// cookieReq builds an *http.Request with all cookies loaded.
func (s *Session) cookieReq(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for _, c := range s.opts.Cookies {
		req.AddCookie(c)
	}
	return req, nil
}

// proxyURL builds the correct URL for /proxy requests.
func (s *Session) proxyURL() string {
	scheme := "https"
	if s.insecure {
		scheme = "http"
	}
	u := url.URL{
		Scheme: scheme,
		Host:   s.opts.Relay,
		Path:   "/proxy",
	}
	q := u.Query()
	q.Set("host", s.opts.Host)
	q.Set("port", s.opts.Port)
	u.RawQuery = q.Encode()
	return u.String()
}

// dial initiates the SSH session and sets up the WebSocket for I/O.
func (s *Session) dial(proxyURL string) error {
	glog.V(2).Infof("Setting up SSH session via %v", proxyURL)
	c, err := rhttp.NewClient(s.opts.Transport)
	if err != nil {
		return fmt.Errorf("rhttp.NewClient() error: %w", err)
	}
	req, err := s.cookieReq(proxyURL)
	if err != nil {
		return fmt.Errorf("cookieReq(%v) error: %w", proxyURL, err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	if err := s.parseProxyResp(resp.Body); err != nil {
		return fmt.Errorf("parseProxyResp() error: %w", err)
	}
	connectURL := s.connectURL()
	glog.V(2).Infof("Copying I/O via %v", connectURL)
	tlsCfg, err := tls.Config(s.opts.Transport.TlsConfig)
	if err != nil {
		return fmt.Errorf("tls.Config() error: %w", err)
	}
	d := &websocket.Dialer{TLSClientConfig: tlsCfg}
	s.ws, _, err = d.Dial(connectURL, s.connectHeader())
	if err != nil {
		return fmt.Errorf("Dial(%v) error: %w", connectURL, err)
	}
	return nil
}

// parseProxyResp parses a /proxy response, loading the Session ID and other optional query string arguments.
func (s *Session) parseProxyResp(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	// Body is a query string minus the leading "sid=".
	// e.g., 4b2fbe8f4eff640b&host=foovpn-1.system.example.com
	u := url.URL{RawQuery: fmt.Sprintf("sid=%v", string(b))}
	s.query = u.Query()
	if s.query.Get("sid") == "" {
		return hsession.ErrBadSessionID
	}
	return nil
}
