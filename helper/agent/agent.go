// Package agent provides functionality to authenticate against the Cookie Server and set up an SSH-over-WebSocket Relay session.
package agent

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/hazaelsan/ssh-relay/helper/session"
	"github.com/hazaelsan/ssh-relay/helper/session/cookie"
	"github.com/hazaelsan/ssh-relay/helper/session/corprelay"
	"github.com/hazaelsan/ssh-relay/helper/session/corprelayv4"
	rhttp "github.com/hazaelsan/ssh-relay/http"

	pb "github.com/hazaelsan/ssh-relay/helper/proto/v1/config"
)

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
	relay, cookies, err := cookie.Authenticate(a.cfg.CookieServerAddress, a.client)
	if err != nil {
		return fmt.Errorf("cookie.Authenticate(%v) error: %w", a.cfg.CookieServerAddress, err)
	}
	opts := session.Options{
		Relay:     relay,
		Host:      a.cfg.Host,
		Port:      a.cfg.Port,
		Origin:    fmt.Sprintf("chrome-extension://%v", session.ExtID),
		Cookies:   cookies,
		Transport: a.cfg.SshRelayTransport,
	}
	var s session.Session
	switch a.cfg.GetProtocolVersion() {
	case pb.Config_CORP_RELAY, pb.Config_PROTOCOL_VERSION_UNSPECIFIED:
		s = corprelay.New(opts)
	case pb.Config_CORP_RELAY_V4:
		s = corprelayv4.New(opts)
	default:
		return errors.New("unsupported protocol version")
	}
	go func() {
		<-s.Done()
	}()
	return s.Run()
}
