// ssh-relay-helper is an SSH proxy command compatible with the Google SSH-over-WebSocket Relay protocol.
// This binary is meant to be used as an ssh(1) ProxyCommand.
//
// Typical use in an ssh_config(5):
//
//	Host *.example.org
//	  ProxyCommand ssh-relay-helper --config=/etc/ssh-relay-helper.textproto --host=%h --port=%p
//
// NOTE: Options passed as flags override those from the config proto.
package main

import (
	"errors"
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/helper/agent"
	"github.com/hazaelsan/ssh-relay/helper/session"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/hazaelsan/ssh-relay/helper/proto/v1/configpb"
	"github.com/hazaelsan/ssh-relay/proto/v1/httppb"
)

var (
	cfgFile = flag.String("config", "", "path to a textproto config file")
	host    = flag.String("host", "", "destination SSH host")
	port    = flag.String("port", "22", "destination SSH port")
	csAddr  = flag.String("cookie_server_address", "", "address[:port] of the Cookie Server, port defaults to 8022")
)

// buildConfig builds and validates a proto config message.
func buildConfig(s string) (*configpb.Config, error) {
	cfg := new(configpb.Config)
	if s != "" {
		buf, err := os.ReadFile(s)
		if err != nil {
			return nil, err
		}
		if err := prototext.Unmarshal(buf, cfg); err != nil {
			return nil, err
		}
	}
	cfg.Host = *host
	cfg.Port = *port
	if cfg.Host == "" {
		return nil, errors.New("host must be specified")
	}
	if cfg.Port == "" {
		return nil, errors.New("port must be specified")
	}
	if *csAddr != "" {
		cfg.CookieServerAddress = *csAddr
	}
	if cfg.CookieServerAddress == "" {
		return nil, errors.New("cookie_server_address must be specified")
	}
	cfg.CookieServerAddress = session.AddDefaultPort(cfg.CookieServerAddress, session.DefaultPort)
	if cfg.CookieServerTransport == nil {
		cfg.CookieServerTransport = new(httppb.HttpTransport)
	}
	if cfg.SshRelayTransport == nil {
		cfg.SshRelayTransport = cfg.CookieServerTransport
	}
	return cfg, nil
}

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()
	cfg, err := buildConfig(*cfgFile)
	if err != nil {
		glog.Exit(err)
	}
	a, err := agent.New(cfg)
	if err != nil {
		glog.Exit(err)
	}
	if err := a.Run(); err != nil {
		glog.Exit(err)
	}
}
