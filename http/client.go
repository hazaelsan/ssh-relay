package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hazaelsan/ssh-relay/duration"
	"github.com/hazaelsan/ssh-relay/tls"

	"github.com/hazaelsan/ssh-relay/proto/v1/httppb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// NewClient generates an *http.Client from a config message.
func NewClient(cfg *httppb.HttpTransport) (*http.Client, error) {
	if cfg.TlsConfig == nil {
		return http.DefaultClient, nil
	}
	tlsCfg, err := tls.CertConfig(cfg.TlsConfig)
	if err != nil {
		return nil, fmt.Errorf("tls.CertConfig() error: %w", err)
	}
	t := &http.Transport{
		TLSClientConfig:        tlsCfg,
		MaxResponseHeaderBytes: cfg.MaxResponseHeaderBytes,
	}
	for dst, src := range map[*time.Duration]*durationpb.Duration{
		&t.ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
	} {
		if err := duration.FromProto(dst, src); err != nil {
			return nil, fmt.Errorf("duration.FromProto(%v, %v) error: %w", dst, src, err)
		}
	}
	c := new(http.Client)
	c.Transport = t
	return c, nil
}
