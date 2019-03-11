package tls

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"sort"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	pb "github.com/hazaelsan/ssh-relay/proto/tls_go_proto"
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
		cfg       *pb.TlsConfig
		want      *tls.Config
		rootCNs   []string
		clientCNs []string
		ok        bool
	}{
		{
			cfg: &pb.TlsConfig{
				ClientAuthType: pb.TlsConfig_REQUIRE_ANY_CLIENT_CERT,
			},
			want: &tls.Config{
				ClientAuth: tls.RequireAnyClientCert,
				MinVersion: TLSMinVersion,
			},
			ok: true,
		},
		{
			cfg: &pb.TlsConfig{
				ClientAuthType: pb.TlsConfig_REQUIRE_AND_VERIFY_CLIENT_CERT,
				RootCaCerts:    []string{"../testdata/test.crt"},
			},
			want: &tls.Config{
				ClientAuth: tls.RequireAndVerifyClientCert,
				MinVersion: TLSMinVersion,
			},
			rootCNs: []string{
				"CN=*.example.org,O=Internet Widgits Pty Ltd,ST=California,C=US",
			},
			ok: true,
		},
		{
			cfg: &pb.TlsConfig{
				ClientCaCerts: []string{"../testdata/test.crt"},
			},
			want: &tls.Config{
				ClientAuth: tls.NoClientCert,
				MinVersion: TLSMinVersion,
			},
			clientCNs: []string{
				"CN=*.example.org,O=Internet Widgits Pty Ltd,ST=California,C=US",
			},
			ok: true,
		},
		// Bad clientCaCerts.
		{
			cfg: &pb.TlsConfig{
				ClientCaCerts: []string{"invalid.crt"},
			},
		},
		// Bad rootCaCerts.
		{
			cfg: &pb.TlsConfig{
				RootCaCerts: []string{"invalid.crt"},
			},
		},
	}
	for i, tt := range tests {
		got, err := Config(tt.cfg)
		if err != nil {
			if tt.ok {
				t.Errorf("Config(%v) error = %v", i, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("Config(%v) error = nil", i)
		}

		if l := len(tt.cfg.RootCaCerts); l > 0 {
			var cns []string
			for j, s := range got.RootCAs.Subjects() {
				cn, err := subjectCN(s)
				if err != nil {
					t.Errorf("subjectCN(%v, %v) error = %v", i, j, err)
					continue
				}
				cns = append(cns, cn)
			}
			if diff := pretty.Compare(cns, tt.rootCNs); diff != "" {
				t.Errorf("RootCNs(%v) diff (-got +want):\n%v", i, diff)
			}
			got.RootCAs = nil
		}
		if l := len(tt.cfg.ClientCaCerts); l > 0 {
			var cns []string
			for j, s := range got.ClientCAs.Subjects() {
				cn, err := subjectCN(s)
				if err != nil {
					t.Errorf("subjectCN(%v, %v) error = %v", i, j, err)
					continue
				}
				cns = append(cns, cn)
			}
			if diff := pretty.Compare(cns, tt.clientCNs); diff != "" {
				t.Errorf("ClientCNs(%v) diff (-got +want):\n%v", i, diff)
			}
			got.ClientCAs = nil
		}
		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("Config(%v) diff (-got +want):\n%v", i, diff)
		}
	}
}

func TestClientConfig(t *testing.T) {
	tests := []struct {
		cfg      *pb.TlsConfig
		want     *tls.Config
		subjects []string
		names    []string
		ok       bool
	}{
		{
			cfg: &pb.TlsConfig{
				ClientAuthType: pb.TlsConfig_REQUIRE_ANY_CLIENT_CERT,
				CertFile:       "../testdata/test.crt",
				KeyFile:        "../testdata/test.key",
			},
			want: &tls.Config{
				ClientAuth: tls.RequireAnyClientCert,
				MinVersion: TLSMinVersion,
			},
			subjects: []string{
				"CN=*.example.org,O=Internet Widgits Pty Ltd,ST=California,C=US",
			},
			ok: true,
		},
		{
			cfg: &pb.TlsConfig{
				CertFile: "invalid.crt",
				KeyFile:  "../testdata/test.key",
			},
		},
		// No cert/key pair.
		{
			cfg: &pb.TlsConfig{},
		},
	}
	for i, tt := range tests {
		got, err := ClientConfig(tt.cfg)
		if err != nil {
			if tt.ok {
				t.Errorf("ClientConfig(%v) error = %v", i, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("ClientConfig(%v) error = nil", i)
		}

		var names []string
		var subjects []string
		for j, tlsCert := range got.Certificates {
			for k, tc := range tlsCert.Certificate {
				cert, err := x509.ParseCertificate(tc)
				if err != nil {
					t.Errorf("ParseCertificate(%v, %v, %v) error = %v", i, j, k, err)
				}
				subjects = append(subjects, cert.Subject.String())
				names = append(names, cert.Subject.CommonName)
			}
		}
		if diff := pretty.Compare(subjects, tt.subjects); diff != "" {
			t.Errorf("subjects(%v) diff (-got +want):\n%v", i, diff)
		}
		got.Certificates = nil

		var certNames []string
		for name := range got.NameToCertificate {
			certNames = append(certNames, name)
		}
		sort.Strings(names)
		sort.Strings(certNames)
		if diff := pretty.Compare(names, certNames); diff != "" {
			t.Errorf("NameToCertificate(%v) diff (-got +want):\n%v", i, diff)
		}
		got.NameToCertificate = nil

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("ClientConfig(%v) diff (-got +want):\n%v", i, diff)
		}
	}
}
