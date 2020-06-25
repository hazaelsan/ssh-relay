package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hazaelsan/ssh-relay/duration"
	"github.com/hazaelsan/ssh-relay/tls"

	dpb "github.com/golang/protobuf/ptypes/duration"
	httppb "github.com/hazaelsan/ssh-relay/proto/v1/http_go_proto"
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
	for dst, src := range map[*time.Duration]*dpb.Duration{
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
