package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	"github.com/hazaelsan/ssh-relay/proto/v1/tlspb"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *tlspb.TlsConfig
		want      *tls.Config
		rootCNs   []string
		clientCNs []string
		ok        bool
	}{
		{
			name: "good require_any_client_cert",
			cfg: &tlspb.TlsConfig{
				CertFile:       "cert",
				KeyFile:        "key",
				ClientAuthType: tlspb.TlsConfig_REQUIRE_ANY_CLIENT_CERT,
			},
			want: &tls.Config{
				ClientAuth: tls.RequireAnyClientCert,
				MinVersion: tls.VersionTLS12,
			},
			ok: true,
		},
		{
			name: "good require_and_verify_client_cert",
			cfg: &tlspb.TlsConfig{
				CertFile:       "cert",
				KeyFile:        "key",
				ClientAuthType: tlspb.TlsConfig_REQUIRE_AND_VERIFY_CLIENT_CERT,
				RootCaCerts:    []string{"../testdata/test.crt"},
			},
			want: &tls.Config{
				ClientAuth: tls.RequireAndVerifyClientCert,
				MinVersion: tls.VersionTLS12,
			},
			rootCNs: []string{
				"CN=*.example.org,O=Internet Widgits Pty Ltd,ST=California,C=US",
			},
			ok: true,
		},
		{
			name: "good client ca certs",
			cfg: &tlspb.TlsConfig{
				CertFile:      "cert",
				KeyFile:       "key",
				ClientCaCerts: []string{"../testdata/test.crt"},
			},
			want: &tls.Config{
				ClientAuth: tls.RequireAndVerifyClientCert,
				MinVersion: tls.VersionTLS12,
			},
			clientCNs: []string{
				"CN=*.example.org,O=Internet Widgits Pty Ltd,ST=California,C=US",
			},
			ok: true,
		},
		{
			name: "bad client ca certs",
			cfg: &tlspb.TlsConfig{
				CertFile:      "cert",
				KeyFile:       "key",
				ClientCaCerts: []string{"invalid.crt"},
			},
		},
		{
			name: "bad root ca certs",
			cfg: &tlspb.TlsConfig{
				CertFile:    "cert",
				KeyFile:     "key",
				RootCaCerts: []string{"invalid.crt"},
			},
		},
		{
			name: "no client ca certs",
			cfg: &tlspb.TlsConfig{
				CertFile:      "cert",
				KeyFile:       "key",
				ClientCaCerts: []string{"/dev/null"},
			},
		},
		{
			name: "empty tls config",
			cfg:  new(tlspb.TlsConfig),
			want: &tls.Config{
				ClientAuth: tls.RequireAndVerifyClientCert,
				MinVersion: tls.VersionTLS12,
			},
			ok: true,
		},
	}
	for _, tt := range tests {
		got, err := Config(tt.cfg)
		if err != nil {
			if tt.ok {
				t.Errorf("Config(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("Config(%v) error = nil", tt.name)
		}

		got.RootCAs = nil
		got.ClientCAs = nil
		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("Config(%v) diff (-got +want):\n%v", tt.name, diff)
		}
	}
}

func TestCertConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *tlspb.TlsConfig
		want     *tls.Config
		subjects []string
		names    []string
		ok       bool
	}{
		{
			name: "good",
			cfg: &tlspb.TlsConfig{
				ClientAuthType: tlspb.TlsConfig_REQUIRE_ANY_CLIENT_CERT,
				CertFile:       "../testdata/test.crt",
				KeyFile:        "../testdata/test.key",
			},
			want: &tls.Config{
				ClientAuth: tls.RequireAnyClientCert,
				MinVersion: tls.VersionTLS12,
			},
			subjects: []string{
				"CN=*.example.org,O=Internet Widgits Pty Ltd,ST=California,C=US",
			},
			ok: true,
		},
		{
			name: "invalid cert",
			cfg: &tlspb.TlsConfig{
				CertFile: "invalid.crt",
				KeyFile:  "../testdata/test.key",
			},
		},
		{
			name: "no cert/key pair",
			cfg:  new(tlspb.TlsConfig),
		},
	}
	for _, tt := range tests {
		got, err := CertConfig(tt.cfg)
		if err != nil {
			if tt.ok {
				t.Errorf("CertConfig(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("CertConfig(%v) error = nil", tt.name)
		}

		var subjects []string
		for i, tlsCert := range got.Certificates {
			for k, tc := range tlsCert.Certificate {
				cert, err := x509.ParseCertificate(tc)
				if err != nil {
					t.Errorf("ParseCertificate(%v, %v, %v) error = %v", tt.name, i, k, err)
				}
				subjects = append(subjects, cert.Subject.String())
			}
		}
		if diff := pretty.Compare(subjects, tt.subjects); diff != "" {
			t.Errorf("subjects(%v) diff (-got +want):\n%v", tt.name, diff)
		}
		got.Certificates = nil

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("CertConfig(%v) diff (-got +want):\n%v", tt.name, diff)
		}
	}
}

func TestCertConfig_invalidClientAuthType(t *testing.T) {
	m := clientAuthMap
	defer func() { clientAuthMap = m }()
	clientAuthMap = nil
	cfg := &tlspb.TlsConfig{
		CertFile: "cert",
		KeyFile:  "key",
	}
	if _, err := CertConfig(cfg); !errors.Is(err, ErrBadClientAuthType) {
		t.Errorf("CertConfig() error = %v, want %v", err, ErrBadClientAuthType)
	}
}
