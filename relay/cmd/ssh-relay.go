// ssh-relay is an extensible SSH-over-WebSocket relay.
// The relay takes authenticated clients (by the Cookie Server) and relays client data to an SSH server.
package main

import (
	"errors"
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/hazaelsan/ssh-relay/relay/runner"

	pb "github.com/hazaelsan/ssh-relay/relay/proto/v1/config_go_proto"
)

var (
	cfgFile = flag.String("config", "", "path to a textproto config file")
)

func loadConfig(s string) (*pb.Config, error) {
	buf, err := os.ReadFile(s)
	if err != nil {
		return nil, err
	}
	cfg := new(pb.Config)
	if err := proto.UnmarshalText(string(buf), cfg); err != nil {
		return nil, err
	}
	if cfg.ServerOptions == nil {
		return nil, errors.New("server_options must be set")
	}
	if cfg.ServerOptions.Port == "" {
		cfg.ServerOptions.Port = "8022"
	}
	if cfg.OriginCookieName == "" {
		return nil, errors.New("origin_cookie_name must be set")
	}
	return cfg, nil
}

func main() {
	flag.Parse()
	if *cfgFile == "" {
		glog.Exitf("--config must be set")
	}
	cfg, err := loadConfig(*cfgFile)
	if err != nil {
		glog.Exit(err)
	}
	r, err := runner.New(cfg)
	if err != nil {
		glog.Exit(err)
	}
	glog.Exit(r.Run())
}
