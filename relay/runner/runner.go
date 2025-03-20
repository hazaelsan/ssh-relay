package runner

import (
	"fmt"
	"time"

	"github.com/hazaelsan/ssh-relay/duration"
	"github.com/hazaelsan/ssh-relay/http"
	"github.com/hazaelsan/ssh-relay/relay/session/manager"

	"github.com/hazaelsan/ssh-relay/relay/proto/v1/configpb"
)

// New instantiates a Runner with a *configpb.Config.
func New(cfg *configpb.Config) (*Runner, error) {
	s, err := http.NewServer(cfg.ServerOptions)
	if err != nil {
		return nil, fmt.Errorf("http.NewServer() error = %w", err)
	}
	var maxAge time.Duration
	if err := duration.FromProto(&maxAge, cfg.MaxSessionAge); err != nil {
		return nil, fmt.Errorf("duration.FromProto(%v) error = %w", cfg.MaxSessionAge, err)
	}
	r := &Runner{
		cfg:    cfg,
		mgr:    manager.New(int(cfg.MaxSessions), maxAge),
		server: s,
	}
	for path, fun := range map[string]http.HandlerFunc{
		"/connect":    r.connectHandle,
		"/proxy":      r.proxyHandle,
		"/v4/connect": r.connectHandleV4,
	} {
		s.HandleFunc(path, fun)
	}
	return r, nil
}

// Runner is the main SSH-over-WebSocket Relay connection handler.
type Runner struct {
	cfg    *configpb.Config
	mgr    *manager.Manager
	server *http.Server
}

// Run executes the runner, listens for incoming client connections.
func (r *Runner) Run() error {
	return r.server.Run()
}
