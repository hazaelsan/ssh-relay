package runner

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/http"
	"github.com/hazaelsan/ssh-relay/tls"
	"google.golang.org/grpc"

	"github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/configpb"
	"github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/servicepb"
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
	c      servicepb.CookieServerClient
}

// Run executes the main runner loop.
func (r *Runner) Run() error {
	addr := net.JoinHostPort(r.cfg.GetGrpcOptions().GetAddr(), r.cfg.GetGrpcOptions().GetPort())
	creds, err := tls.TransportCreds(r.cfg.GetGrpcOptions().GetTlsConfig())
	if err != nil {
		return err
	}
	glog.V(1).Infof("Connecting to gRPC backend %v", addr)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return fmt.Errorf("grpc.NewClient(%v) error: %w", addr, err)
	}
	defer conn.Close()
	r.c = servicepb.NewCookieServerClient(conn)
	return r.server.Run()
}
