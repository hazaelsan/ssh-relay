package http

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"

	dpb "github.com/golang/protobuf/ptypes/duration"
	httppb "github.com/hazaelsan/ssh-relay/proto/v1/http"
	tlspb "github.com/hazaelsan/ssh-relay/proto/v1/tls"
)

func subjectCN(b []byte) (string, error) {
	s := new(pkix.RDNSequence)
	if _, err := asn1.Unmarshal(b, s); err != nil {
		return "", err
	}
	return s.String(), nil
}

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
				ResponseHeaderTimeout:  &dpb.Duration{Seconds: 3},
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
				ResponseHeaderTimeout: &dpb.Duration{Seconds: -1},
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
		if l := len(tt.cfg.TlsConfig.RootCaCerts); l > 0 {
			var cns []string
			for j, s := range transport.TLSClientConfig.RootCAs.Subjects() {
				cn, err := subjectCN(s)
				if err != nil {
					t.Errorf("subjectCN(%v, %v) error = %v", tt.name, j, err)
					continue
				}
				cns = append(cns, cn)
			}
			if diff := pretty.Compare(cns, tt.rootCNs); diff != "" {
				t.Errorf("RootCNs(%v) diff (-got +want):\n%v", tt.name, diff)
			}
			transport.TLSClientConfig.RootCAs = nil
		}

		var names []string
		var subjects []string
		for j, tlsCert := range transport.TLSClientConfig.Certificates {
			for k, tc := range tlsCert.Certificate {
				cert, err := x509.ParseCertificate(tc)
				if err != nil {
					t.Errorf("ParseCertificate(%v, %v, %v) error = %v", tt.name, j, k, err)
				}
				subjects = append(subjects, cert.Subject.String())
				names = append(names, cert.Subject.CommonName)
			}
		}
		if diff := pretty.Compare(subjects, tt.rootCNs); diff != "" {
			t.Errorf("subjects(%v) diff (-got +want):\n%v", tt.name, diff)
		}
		transport.TLSClientConfig.Certificates = nil

		var certNames []string
		for name := range transport.TLSClientConfig.NameToCertificate {
			certNames = append(certNames, name)
		}
		sort.Strings(names)
		sort.Strings(certNames)
		if diff := pretty.Compare(names, certNames); diff != "" {
			t.Errorf("NameToCertificate(%v) diff (-got +want):\n%v", tt.name, diff)
		}
		transport.TLSClientConfig.NameToCertificate = nil

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("NewClient(%v) diff (-got +want):\n%v", tt.name, diff)
		}
	}
}
