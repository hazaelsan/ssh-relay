package http

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"

	"github.com/hazaelsan/ssh-relay/proto/v1/httppb"
	"github.com/hazaelsan/ssh-relay/proto/v1/tlspb"
	"google.golang.org/protobuf/types/known/durationpb"
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
		name       string
		cfg        *httppb.HttpServerOptions
		server     *http.Server
		hstsMaxAge time.Duration
		ok         bool
	}{
		{
			name: "good",
			cfg: &httppb.HttpServerOptions{
				Addr: "::1",
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile:       "../testdata/test.crt",
					KeyFile:        "../testdata/test.key",
					ClientAuthType: tlspb.TlsConfig_REQUEST_CLIENT_CERT,
				},
				HttpServer: &httppb.HttpServer{
					ReadTimeout:       &durationpb.Duration{Seconds: 1},
					ReadHeaderTimeout: &durationpb.Duration{Seconds: 2},
					WriteTimeout:      &durationpb.Duration{Seconds: 3},
					IdleTimeout:       &durationpb.Duration{Seconds: 4},
					MaxHeaderBytes:    10,
				},
				HstsMaxAge:            &durationpb.Duration{Seconds: 3},
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
		{
			name: "no address",
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
		{
			name: "bad address",
			cfg: &httppb.HttpServerOptions{
				Addr: "1:2",
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile: "../testdata/test.crt",
					KeyFile:  "../testdata/test.key",
				},
			},
		},
		{
			name: "no port",
			cfg: &httppb.HttpServerOptions{
				Addr: "::1",
				TlsConfig: &tlspb.TlsConfig{
					CertFile: "../testdata/test.crt",
					KeyFile:  "../testdata/test.key",
				},
			},
		},
		{
			name: "no tls config",
			cfg: &httppb.HttpServerOptions{
				Port: "8022",
			},
		},
		{
			name: "no cert file",
			cfg: &httppb.HttpServerOptions{
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					KeyFile: "../testdata/test.key",
				},
			},
		},
		{
			name: "no client file",
			cfg: &httppb.HttpServerOptions{
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile: "../testdata/test.crt",
				},
			},
		},
		{
			name: "bad client ca",
			cfg: &httppb.HttpServerOptions{
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile:    "../testdata/test.crt",
					KeyFile:     "../testdata/test.key",
					RootCaCerts: []string{"invalid"},
				},
			},
		},
		{
			name: "bad duration",
			cfg: &httppb.HttpServerOptions{
				Addr: "::1",
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile:       "../testdata/test.crt",
					KeyFile:        "../testdata/test.key",
					ClientAuthType: tlspb.TlsConfig_REQUEST_CLIENT_CERT,
				},
				HttpServer: &httppb.HttpServer{
					ReadTimeout: &durationpb.Duration{Seconds: -1},
				},
			},
			server: &http.Server{
				Addr:              "[::1]:8022",
				ReadHeaderTimeout: -1 * time.Second,
			},
		},
		{
			name: "bad hstsMaxAge",
			cfg: &httppb.HttpServerOptions{
				Addr: "::1",
				Port: "8022",
				TlsConfig: &tlspb.TlsConfig{
					CertFile:       "../testdata/test.crt",
					KeyFile:        "../testdata/test.key",
					ClientAuthType: tlspb.TlsConfig_REQUEST_CLIENT_CERT,
				},
				HstsMaxAge: &durationpb.Duration{Seconds: -3},
			},
			server: &http.Server{
				Addr:              "[::1]:8022",
				ReadHeaderTimeout: -1 * time.Second,
			},
		},
	}
	for _, tt := range testdata {
		got, err := NewServer(tt.cfg)
		if err != nil {
			if tt.ok {
				t.Errorf("NewServer(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("NewServer(%v) error = nil", tt.name)
		}

		if got.hstsMaxAge != tt.hstsMaxAge {
			t.Errorf("hstsMaxAge(%v) = %v, want %v", tt.name, got.hstsMaxAge, tt.hstsMaxAge)
		}
		if diff := pretty.Compare(got.server, tt.server); diff != "" {
			t.Errorf("NewServer(%v) diff (-got +want)\n%v", tt.name, diff)
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if string(body) != wantMsg {
		t.Errorf("resp.Body = %v, want %v", string(body), wantMsg)
	}
}

func TestRun(t *testing.T) {
	cfg := &httppb.HttpServerOptions{
		Addr: "::1",
		Port: "8022",
		TlsConfig: &tlspb.TlsConfig{
			CertFile:       "../testdata/test.crt",
			KeyFile:        "../testdata/test.key",
			ClientAuthType: tlspb.TlsConfig_REQUEST_CLIENT_CERT,
		},
	}
	s, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	errc := make(chan error)
	go func() { errc <- s.Run() }()
	s.server.Shutdown(context.Background())
	if err = <-errc; !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("Run() error = %v, want %v", err, http.ErrServerClosed)
	}
}
