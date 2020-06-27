package runner

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/http"
	"github.com/hazaelsan/ssh-relay/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	configpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/config_go_proto"
	servicepb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/service_go_proto"
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
	tlsCfg, err := tls.CertConfig(r.cfg.GetGrpcOptions().GetTlsConfig())
	if err != nil {
		return fmt.Errorf("tls.CertConfig() error: %w", err)
	}
	addr := net.JoinHostPort(r.cfg.GetGrpcOptions().GetAddr(), r.cfg.GetGrpcOptions().GetPort())
	glog.V(4).Infof("Connecting to gRPC backend %v", addr)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("grpc.Dial(%v) error: %w", addr, err)
	}
	defer conn.Close()
	r.c = servicepb.NewCookieServerClient(conn)
	return r.server.Run()
}
