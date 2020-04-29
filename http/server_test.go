package http

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dpb "github.com/golang/protobuf/ptypes/duration"
	"github.com/kylelemons/godebug/pretty"

	httppb "github.com/hazaelsan/ssh-relay/proto/v1/http_go_proto"
	tlspb "github.com/hazaelsan/ssh-relay/proto/v1/tls_go_proto"
)

func getHeader(h http.Header, s string) string {
	return strings.Join(h[s], ", ")
}

func TestNoCache(t *testing.T) {
	testdata := map[string]string{
		"Cache-Control": "no-store, no-cache, must-revalidate, max-age=0",
		"Pragma":        "no-cache",
	}
	w := httptest.NewRecorder()
	NoCache(w)
	resp := w.Result()
	for hdr, want := range testdata {
		if got := getHeader(resp.Header, hdr); got != want {
			t.Errorf("NoCache(%v) = %v, want %v", hdr, got, want)
		}
	}
}

func TestSetHSTS(t *testing.T) {
	testdata := []struct {
		d          time.Duration
		subdomains bool
		want       string
	}{
		{
			d:          3 * time.Second,
			subdomains: true,
			want:       "max-age=3, includeSubDomains",
		},
		{
			d:    -5 * time.Second,
			want: "max-age=-5",
		},
		{
			want: "max-age=0",
		},
	}
	for _, tt := range testdata {
		w := httptest.NewRecorder()
		SetHSTS(w, tt.d, tt.subdomains)
		resp := w.Result()
		t.Log(pretty.Sprint(resp.Header))
		if got := getHeader(resp.Header, "Strict-Transport-Security"); got != tt.want {
			t.Errorf("SetHSTS() = %v, want %v", got, tt.want)
		}
	}
}

func TestNewServer(t *testing.T) {
	testdata := []struct {
		cfg        *httppb.HttpServerOptions
		server     *http.Server
		hstsMaxAge time.Duration
		ok         bool
	}{
		{
			cfg: &httppb.HttpServerOptions{
				Addr: "::1",
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile:       "../testdata/test.crt",
					KeyFile:        "../testdata/test.key",
					ClientAuthType: tlspb.TlsConfig_REQUEST_CLIENT_CERT,
				},
				HttpServer: &httppb.HttpServer{
					ReadTimeout:       &dpb.Duration{Seconds: 1},
					ReadHeaderTimeout: &dpb.Duration{Seconds: 2},
					WriteTimeout:      &dpb.Duration{Seconds: 3},
					IdleTimeout:       &dpb.Duration{Seconds: 4},
					MaxHeaderBytes:    10,
				},
				HstsMaxAge:            &dpb.Duration{Seconds: 3},
				HstsIncludeSubdomains: true,
			},
			server: &http.Server{
				Addr:              "[::1]:8022",
				ReadTimeout:       time.Second,
				ReadHeaderTimeout: 2 * time.Second,
				WriteTimeout:      3 * time.Second,
				IdleTimeout:       4 * time.Second,
				MaxHeaderBytes:    10,
				Handler:           http.NewServeMux(),
				TLSConfig: &tls.Config{
					ClientAuth: tls.RequestClientCert,
					MinVersion: tls.VersionTLS12,
				},
			},
			hstsMaxAge: 3 * time.Second,
			ok:         true,
		},
		// No address.
		{
			cfg: &httppb.HttpServerOptions{
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile: "../testdata/test.crt",
					KeyFile:  "../testdata/test.key",
				},
			},
			server: &http.Server{
				Addr: ":8022",
				TLSConfig: &tls.Config{
					ClientAuth: tls.RequireAndVerifyClientCert,
					MinVersion: tls.VersionTLS12,
				},
				Handler: http.NewServeMux(),
			},
			ok: true,
		},
		// Bad address.
		{
			cfg: &httppb.HttpServerOptions{
				Addr: "1:2",
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile: "../testdata/test.crt",
					KeyFile:  "../testdata/test.key",
				},
			},
		},
		// No port.
		{
			cfg: &httppb.HttpServerOptions{
				Addr: "::1",
				TlsConfig: &tlspb.TlsConfig{
					CertFile: "../testdata/test.crt",
					KeyFile:  "../testdata/test.key",
				},
			},
		},
		// No TLS config.
		{
			cfg: &httppb.HttpServerOptions{
				Port: "8022",
			},
		},
		// No CertFile.
		{
			cfg: &httppb.HttpServerOptions{
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					KeyFile: "../testdata/test.key",
				},
			},
		},
		// No KeyFile.
		{
			cfg: &httppb.HttpServerOptions{
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile: "../testdata/test.crt",
				},
			},
		},
		// Bad clientCAs.
		{
			cfg: &httppb.HttpServerOptions{
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile:    "../testdata/test.crt",
					KeyFile:     "../testdata/test.key",
					RootCaCerts: []string{"invalid"},
				},
			},
		},
	}
	for i, tt := range testdata {
		got, err := NewServer(tt.cfg)
		if err != nil {
			if tt.ok {
				t.Errorf("NewServer(%v) error = %v", i, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("NewServer(%v) error = nil", i)
		}

		if got.hstsMaxAge != tt.hstsMaxAge {
			t.Errorf("hstsMaxAge(%v) = %v, want %v", i, got.hstsMaxAge, tt.hstsMaxAge)
		}
		if diff := pretty.Compare(got.server, tt.server); diff != "" {
			t.Errorf("NewServer(%v) diff (-got +want)\n%v", i, diff)
		}
	}
}

func TestHandleFunc(t *testing.T) {
	wantMsg := "foo bar baz"
	mux := http.NewServeMux()
	s := &Server{
		cfg:    new(httppb.HttpServerOptions),
		server: &http.Server{Handler: mux},
		mux:    mux,
	}
	w := httptest.NewRecorder()
	dummyFunc := func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(wantMsg))
	}
	s.HandleFunc("/foo", dummyFunc)

	req := httptest.NewRequest("GET", "/foo", nil)
	s.mux.ServeHTTP(w, req)
	resp := w.Result()

	headers := map[string]string{
		"Cache-Control":             "no-store, no-cache, must-revalidate, max-age=0",
		"Pragma":                    "no-cache",
		"Strict-Transport-Security": "max-age=0",
	}
	for hdr, want := range headers {
		if got := getHeader(resp.Header, hdr); got != want {
			t.Errorf("getHeader(%v) = %v, want %v", hdr, got, want)
		}
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != wantMsg {
		t.Errorf("resp.Body = %v, want %v", string(body), wantMsg)
	}
}
