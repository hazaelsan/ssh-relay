package runner

import (
	"github.com/hazaelsan/ssh-relay/http"

	configpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/config_go_proto"
)

// New instantiates a Runner with a *configpb.Config.
func New(cfg *configpb.Config) (*Runner, error) {
	s, err := http.NewServer(cfg.ServerOptions)
	if err != nil {
		return nil, err
	}
	r := &Runner{
		cfg:    cfg,
		server: s,
	}
	s.HandleFunc("/cookie", r.handleCookie)
	return r, nil
}

// Runner is the main runner loop.
type Runner struct {
	cfg    *configpb.Config
	server *http.Server
}

// Run executes the main runner loop.
func (r *Runner) Run() error {
	return r.server.Run()
}
