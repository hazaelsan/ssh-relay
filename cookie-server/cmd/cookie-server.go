// ssh-relay-cookie-server is the main Cookie Server binary.
// The Cookie Server is responsible for authenticating clients and redirecting them to an SSH-over-WebSocket Relay.
package main

import (
	"errors"
	"flag"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/hazaelsan/ssh-relay/cookie-server/runner"

	pb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/config_go_proto"
)

var (
	cfgFile = flag.String("config", "", "path to a textpb config file")
)

func loadConfig(s string) (*pb.Config, error) {
	buf, err := ioutil.ReadFile(s)
	if err != nil {
		return nil, err
	}
	cfg := new(pb.Config)
	if err := proto.UnmarshalText(string(buf), cfg); err != nil {
		glog.Exit(err)
	}
	if cfg.ServerOptions == nil {
		return nil, errors.New("server_options must be set")
	}
	if cfg.ServerOptions.Port == "" {
		cfg.ServerOptions.Port = "8022"
	}
	if cfg.FallbackRelayHost == "" {
		return nil, errors.New("fallback_relay_host must be set")
	}
	if cfg.OriginCookie.Name == "" {
		return nil, errors.New("origin_cookie.name must be set")
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
