// ssh-relay-cookie-server is the main Cookie Server binary.
// The Cookie Server is responsible for authenticating clients and redirecting them to an SSH-over-WebSocket Relay.
package main

import (
	"errors"
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/cookie-server/runner"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/configpb"
)

var (
	cfgFile = flag.String("config", "", "path to a text proto config file")
)

func loadConfig(s string) (*configpb.Config, error) {
	buf, err := os.ReadFile(s)
	if err != nil {
		return nil, err
	}
	cfg := new(configpb.Config)
	if err := prototext.Unmarshal(buf, cfg); err != nil {
		return nil, err
	}
	if cfg.ServerOptions == nil {
		return nil, errors.New("server_options must be set")
	}
	if cfg.ServerOptions.Port == "" {
		cfg.ServerOptions.Port = "8022"
	}
	if cfg.OriginCookie.Name == "" {
		return nil, errors.New("origin_cookie.name must be set")
	}
	return cfg, nil
}

func main() {
	flag.Parse()
	if *cfgFile == "" {
		glog.Exit("--config must be set")
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
