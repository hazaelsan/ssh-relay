package session

import (
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/hazaelsan/ssh-relay/proto/v1/httppb"
)

const (
	// ExtID is the dummy Chrome extension ID to send on requests.
	ExtID = "sshRelayHelper"

	// DefaultPort is the default port number for the Cookie Server / SSH Relay.
	DefaultPort = "8022"
)

var (
	// ErrBadSessionID is returned if the Session ID could not be parsed.
	ErrBadSessionID = errors.New("bad Session ID")
)

// AddDefaultPort adds a port number to an address if one isn't specified.
func AddDefaultPort(addr, port string) string {
	u := url.URL{Host: addr}
	if u.Port() == "" {
		u.Host = net.JoinHostPort(u.Hostname(), port)
	}
	return net.JoinHostPort(u.Hostname(), u.Port())
}

// Options specifies a set of options to create a Session.
type Options struct {
	// Relay is the address:port of the WebSocket Relay.
	Relay string

	// Host is the address of the SSH host.
	Host string

	// Port is the port of the SSH host.
	Port string

	// Origin is the WebSocket origin.
	Origin string

	// Cookies is a list of cookies to send on outgoing requests.
	Cookies []*http.Cookie

	// Transport specifies settings for creating HTTP/WebSocket connections.
	Transport *httppb.HttpTransport
}

// A Session is an SSH-over-WebSocket Relay client session.
type Session interface {
	// Run authenticates against the Cookie Server and starts/resumes the
	// SSH-over-WebSocket session.
	Run() error

	// Done notifies when the Session has terminated.
	Done() <-chan struct{}
}
