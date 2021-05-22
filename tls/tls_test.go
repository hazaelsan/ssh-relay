package tls

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"sort"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	pb "github.com/hazaelsan/ssh-relay/proto/v1/tls_go_proto"
)

func subjectCN(b []byte) (string, error) {
	s := new(pkix.RDNSequence)
	if _, err := asn1.Unmarshal(b, s); err != nil {
		return "", err
	}
	return s.String(), nil
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *pb.TlsConfig
		want      *tls.Config
		rootCNs   []string
		clientCNs []string
		ok        bool
	}{
		{
			name: "good empty tls config",
			cfg:  new(pb.TlsConfig),
			want: &tls.Config{
				ClientAuth: tls.RequireAndVerifyClientCert,
				MinVersion: tls.VersionTLS12,
			},
			ok: true,
		},
		{
			name: "good require_any_client_cert",
			cfg: &pb.TlsConfig{
				ClientAuthType: pb.TlsConfig_REQUIRE_ANY_CLIENT_CERT,
			},
			want: &tls.Config{
				ClientAuth: tls.RequireAnyClientCert,
				MinVersion: tls.VersionTLS12,
			},
			ok: true,
		},
		{
			name: "good require_and_verify_client_cert",
			cfg: &pb.TlsConfig{
				ClientAuthType: pb.TlsConfig_REQUIRE_AND_VERIFY_CLIENT_CERT,
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
			cfg: &pb.TlsConfig{
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
			cfg: &pb.TlsConfig{
				ClientCaCerts: []string{"invalid.crt"},
			},
		},
		{
			name: "bad root ca certs",
			cfg: &pb.TlsConfig{
				RootCaCerts: []string{"invalid.crt"},
			},
		},
		{
			name: "no client ca certs",
			cfg: &pb.TlsConfig{
				ClientCaCerts: []string{"/dev/null"},
			},
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

		if l := len(tt.cfg.RootCaCerts); l > 0 {
			var cns []string
			for i, s := range got.RootCAs.Subjects() {
				cn, err := subjectCN(s)
				if err != nil {
					t.Errorf("subjectCN(%v, %v) error = %v", tt.name, i, err)
					continue
				}
				cns = append(cns, cn)
			}
			if diff := pretty.Compare(cns, tt.rootCNs); diff != "" {
				t.Errorf("RootCNs(%v) diff (-got +want):\n%v", tt.name, diff)
			}
			got.RootCAs = nil
		}
		if l := len(tt.cfg.ClientCaCerts); l > 0 {
			var cns []string
			for i, s := range got.ClientCAs.Subjects() {
				cn, err := subjectCN(s)
				if err != nil {
					t.Errorf("subjectCN(%v, %v) error = %v", tt.name, i, err)
					continue
				}
				cns = append(cns, cn)
			}
			if diff := pretty.Compare(cns, tt.clientCNs); diff != "" {
				t.Errorf("ClientCNs(%v) diff (-got +want):\n%v", tt.name, diff)
			}
			got.ClientCAs = nil
		}
		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("Config(%v) diff (-got +want):\n%v", tt.name, diff)
		}
	}
}

func TestCertConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *pb.TlsConfig
		want     *tls.Config
		subjects []string
		names    []string
		ok       bool
	}{
		{
			name: "good",
			cfg: &pb.TlsConfig{
				ClientAuthType: pb.TlsConfig_REQUIRE_ANY_CLIENT_CERT,
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
			cfg: &pb.TlsConfig{
				CertFile: "invalid.crt",
				KeyFile:  "../testdata/test.key",
			},
		},
		{
			name: "no cert/key pair",
			cfg:  new(pb.TlsConfig),
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

		var names []string
		var subjects []string
		for i, tlsCert := range got.Certificates {
			for k, tc := range tlsCert.Certificate {
				cert, err := x509.ParseCertificate(tc)
				if err != nil {
					t.Errorf("ParseCertificate(%v, %v, %v) error = %v", tt.name, i, k, err)
				}
				subjects = append(subjects, cert.Subject.String())
				names = append(names, cert.Subject.CommonName)
			}
		}
		if diff := pretty.Compare(subjects, tt.subjects); diff != "" {
			t.Errorf("subjects(%v) diff (-got +want):\n%v", tt.name, diff)
		}
		got.Certificates = nil

		var certNames []string
		for name := range got.NameToCertificate {
			certNames = append(certNames, name)
		}
		sort.Strings(names)
		sort.Strings(certNames)
		if diff := pretty.Compare(names, certNames); diff != "" {
			t.Errorf("NameToCertificate(%v) diff (-got +want):\n%v", tt.name, diff)
		}
		got.NameToCertificate = nil

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("CertConfig(%v) diff (-got +want):\n%v", tt.name, diff)
		}
	}
}

func TestCertConfig_invalidClientAuthType(t *testing.T) {
	m := clientAuthMap
	defer func() { clientAuthMap = m }()
	clientAuthMap = nil
	if _, err := CertConfig(new(pb.TlsConfig)); !errors.Is(err, ErrBadClientAuthType) {
		t.Errorf("CertConfig() error = %v, want %v", err, ErrBadClientAuthType)
	}
}
