package http

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"

	"github.com/hazaelsan/ssh-relay/proto/v1/httppb"
	"github.com/hazaelsan/ssh-relay/proto/v1/tlspb"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestNewClient(t *testing.T) {
	testdata := []struct {
		name    string
		cfg     *httppb.HttpTransport
		want    *http.Client
		rootCNs []string
		ok      bool
	}{
		{
			name: "good",
			cfg: &httppb.HttpTransport{
				TlsConfig: &tlspb.TlsConfig{
					CertFile:       "../testdata/test.crt",
					KeyFile:        "../testdata/test.key",
					RootCaCerts:    []string{"../testdata/test.crt"},
					ClientAuthType: tlspb.TlsConfig_REQUEST_CLIENT_CERT,
				},
				MaxResponseHeaderBytes: 5,
				ResponseHeaderTimeout:  &durationpb.Duration{Seconds: 3},
			},
			want: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						ClientAuth: tls.RequestClientCert,
						MinVersion: tls.VersionTLS12,
					},
					MaxResponseHeaderBytes: 5,
					ResponseHeaderTimeout:  3 * time.Second,
				},
			},
			rootCNs: []string{
				"CN=*.example.org,O=Internet Widgits Pty Ltd,ST=California,C=US",
			},
			ok: true,
		},
		{
			name: "good defaults",
			cfg:  new(httppb.HttpTransport),
			want: http.DefaultClient,
			ok:   true,
		},
		{
			name: "bad client cert",
			cfg: &httppb.HttpTransport{
				TlsConfig: &tlspb.TlsConfig{
					CertFile: "invalid.crt",
					KeyFile:  "../testdata/test.key",
				},
			},
		},
		{
			name: "bad client key",
			cfg: &httppb.HttpTransport{
				TlsConfig: &tlspb.TlsConfig{
					CertFile: "../testdata/test.crt",
					KeyFile:  "invalid.key",
				},
			},
		},
		{
			name: "bad duration",
			cfg: &httppb.HttpTransport{
				TlsConfig: &tlspb.TlsConfig{
					CertFile:       "../testdata/test.crt",
					KeyFile:        "../testdata/test.key",
					RootCaCerts:    []string{"../testdata/test.crt"},
					ClientAuthType: tlspb.TlsConfig_REQUEST_CLIENT_CERT,
				},
				ResponseHeaderTimeout: &durationpb.Duration{Seconds: -1},
			},
		},
	}
	for _, tt := range testdata {
		got, err := NewClient(tt.cfg)
		if err != nil {
			if tt.ok {
				t.Errorf("NewClient(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("NewClient(%v) error = nil", tt.name)
		}

		transport, ok := got.Transport.(*http.Transport)
		if !ok {
			if tt.cfg.TlsConfig != nil {
				t.Errorf("got.Transport(%v) bad type assertion", tt.name)
			}
			continue
		}
		transport.TLSClientConfig.RootCAs = nil

		var subjects []string
		for j, tlsCert := range transport.TLSClientConfig.Certificates {
			for k, tc := range tlsCert.Certificate {
				cert, err := x509.ParseCertificate(tc)
				if err != nil {
					t.Errorf("ParseCertificate(%v, %v, %v) error = %v", tt.name, j, k, err)
				}
				subjects = append(subjects, cert.Subject.String())
			}
		}
		if diff := pretty.Compare(subjects, tt.rootCNs); diff != "" {
			t.Errorf("subjects(%v) diff (-got +want):\n%v", tt.name, diff)
		}
		transport.TLSClientConfig.Certificates = nil

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("NewClient(%v) diff (-got +want):\n%v", tt.name, diff)
		}
	}
}
