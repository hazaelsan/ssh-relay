package runner

import (
	"time"

	"github.com/hazaelsan/ssh-relay/duration"
	"github.com/hazaelsan/ssh-relay/http"
	"github.com/hazaelsan/ssh-relay/relay/session/manager"

	pb "github.com/hazaelsan/ssh-relay/relay/proto/v1/config_go_proto"
)

// New instantiates a Runner with a *pb.Config.
func New(cfg *pb.Config) (*Runner, error) {
	s, err := http.NewServer(cfg.ServerOptions)
	if err != nil {
		return nil, err
	}
	var maxAge time.Duration
	if err := duration.FromProto(&maxAge, cfg.MaxSessionAge); err != nil {
		return nil, err
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
	cfg    *pb.Config
	mgr    *manager.Manager
	server *http.Server
}

// Run executes the runner, listens for incoming client connections.
func (r *Runner) Run() error {
	return r.server.Run()
}
