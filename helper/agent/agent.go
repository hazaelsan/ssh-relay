// Package agent provides functionality to authenticate against the Cookie Server and set up an SSH-over-WebSocket Relay session.
package agent

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/helper/session/corprelay"
	rhttp "github.com/hazaelsan/ssh-relay/http"
	"github.com/hazaelsan/ssh-relay/response"

	pb "github.com/hazaelsan/ssh-relay/helper/proto/v1/config_go_proto"
)

const (
	// Dummy extension ID / path, both are required but otherwise unused.
	extID         = "sshRelayHelper"
	extPath       = "/"
	clientVersion = 2
	redirMethod   = "direct"

	// DefaultPort is the default port number for the Cookie Server / SSH Relay.
	DefaultPort = "8022"
)

// AddDefaultPort adds a port number to an address if one isn't specified.
func AddDefaultPort(addr, port string) string {
	u := url.URL{Host: addr}
	if u.Port() == "" {
		u.Host = net.JoinHostPort(u.Hostname(), port)
	}
	return net.JoinHostPort(u.Hostname(), u.Port())
}

// New creates an *Agent.
func New(cfg *pb.Config) (*Agent, error) {
	c, err := rhttp.NewClient(cfg.CookieServerTransport)
	if err != nil {
		return nil, fmt.Errorf("rhttp.NewClient() error: %w", err)
	}
	return &Agent{
		cfg:    cfg,
		client: c,
	}, nil
}

// An Agent authenticates against the Cookie Server and sets up an SSH-over-WebSocket session.
type Agent struct {
	cfg    *pb.Config
	client *http.Client
}

// Run authenticates against the Cookie Server and starts the SSH-over-WebSocket session.
func (a *Agent) Run() error {
	relay, cookies, err := a.auth()
	if err != nil {
		return fmt.Errorf("auth() error: %w", err)
	}
	opts := corprelay.Options{
		Relay:     relay,
		Host:      a.cfg.Host,
		Port:      a.cfg.Port,
		Origin:    fmt.Sprintf("chrome-extension://%v", extID),
		Cookies:   cookies,
		Transport: a.cfg.SshRelayTransport,
	}
	s := corprelay.New(opts)
	return s.Run()
}

// auth authenticates against the Cookie Server,
// returns the relay address and cookies to use for the WebSocket session.
func (a *Agent) auth() (string, []*http.Cookie, error) {
	u := a.authURL()
	glog.V(2).Infof("Authenticating against %v", u)
	resp, err := a.client.Get(u)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	r, err := response.FromReader(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("response.FromReader() error: %w", err)
	}
	return AddDefaultPort(r.Endpoint, DefaultPort), resp.Cookies(), nil
}

// authURL builds the correct URL for authenticating against the Cookie Server.
func (a *Agent) authURL() string {
	u := url.URL{
		Scheme: "https",
		Host:   a.cfg.CookieServerAddress,
		Path:   "/cookie",
	}
	q := u.Query()
	q.Set("ext", extID)
	q.Set("path", extPath)
	q.Set("version", strconv.Itoa(clientVersion))
	q.Set("method", redirMethod)
	u.RawQuery = q.Encode()
	return u.String()
}
