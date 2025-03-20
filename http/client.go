package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hazaelsan/ssh-relay/duration"
	"github.com/hazaelsan/ssh-relay/tls"

	"github.com/hazaelsan/ssh-relay/proto/v1/httppb"
	"github.com/hazaelsan/ssh-relay/proto/v1/tlspb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// NewClient generates an *http.Client from a config message.
func NewClient(cfg *httppb.HttpTransport) (*http.Client, error) {
	if cfg.GetTlsConfig() == nil {
		return http.DefaultClient, nil
	}

	t := &http.Transport{
		MaxResponseHeaderBytes: cfg.MaxResponseHeaderBytes,
	}
	for dst, src := range map[*time.Duration]*durationpb.Duration{
		&t.ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
	} {
		if err := duration.FromProto(dst, src); err != nil {
			return nil, fmt.Errorf("duration.FromProto(%v, %v) error: %w", dst, src, err)
		}
	}

	if cfg.GetTlsConfig().GetTlsMode() != tlspb.TlsConfig_TLS_MODE_DISABLED {
		tlsCfg, err := tls.CertConfig(cfg.TlsConfig)
		if err != nil {
			return nil, fmt.Errorf("tls.CertConfig() error: %w", err)
		}
		t.TLSClientConfig = tlsCfg
	}
	return &http.Client{Transport: t}, nil
}
