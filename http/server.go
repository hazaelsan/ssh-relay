package http

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/duration"
	"github.com/hazaelsan/ssh-relay/tls"

	"github.com/hazaelsan/ssh-relay/proto/v1/httppb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// HandlerFunc is the function signature for an HTTP handler.
type HandlerFunc func(http.ResponseWriter, *http.Request)

// NoCache is a helper to send no-cache headers to an *http.ResponseWriter.
func NoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
}

// SetHSTS is a helper to send HSTS headers to an *http.ResponseWriter.
func SetHSTS(w http.ResponseWriter, d time.Duration, subdomains bool) {
	w.Header().Add("Strict-Transport-Security", fmt.Sprintf("max-age=%d", int(d.Seconds())))
	if subdomains {
		w.Header().Add("Strict-Transport-Security", "includeSubDomains")
	}
}

// NewServer generates a *Server from a proto config message.
func NewServer(cfg *httppb.HttpServerOptions) (*Server, error) {
	if _, _, err := net.SplitHostPort(cfg.Addr); err == nil {
		return nil, ErrBadAddr
	}
	if cfg.Port == "" {
		return nil, ErrMissingPort
	}
	if cfg.HttpServer == nil {
		cfg.HttpServer = new(httppb.HttpServer)
	}
	if cfg.TlsConfig == nil {
		return nil, ErrNoTLSConfig
	}
	if cfg.TlsConfig.CertFile == "" {
		return nil, ErrNoCertFile
	}
	if cfg.TlsConfig.KeyFile == "" {
		return nil, ErrNoKeyFile
	}
	tlsConfig, err := tls.Config(cfg.TlsConfig)
	if err != nil {
		return nil, fmt.Errorf("tls.Config() error: %w", err)
	}
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:           net.JoinHostPort(cfg.Addr, cfg.Port),
		MaxHeaderBytes: int(cfg.HttpServer.GetMaxHeaderBytes()),
		TLSConfig:      tlsConfig,
		Handler:        mux,
	}
	for dst, src := range map[*time.Duration]*durationpb.Duration{
		&server.ReadTimeout:       cfg.HttpServer.ReadTimeout,
		&server.ReadHeaderTimeout: cfg.HttpServer.ReadHeaderTimeout,
		&server.WriteTimeout:      cfg.HttpServer.WriteTimeout,
		&server.IdleTimeout:       cfg.HttpServer.IdleTimeout,
	} {
		if err := duration.FromProto(dst, src); err != nil {
			return nil, fmt.Errorf("duration.FromProto(%v, %v) error: %w", dst, src, err)
		}
	}
	s := &Server{
		cfg:    cfg,
		server: server,
		mux:    mux,
	}
	if err := duration.FromProto(&s.hstsMaxAge, cfg.HstsMaxAge); err != nil {
		return nil, err
	}
	return s, nil
}

// Server is a wrapper around *http.Server that enforces common settings.
type Server struct {
	cfg        *httppb.HttpServerOptions
	server     *http.Server
	mux        *http.ServeMux
	hstsMaxAge time.Duration
}

// HandleFunc registers the handler function for the given pattern:
// * Disables caching
// * Sets the HSTS policy, if hstsMaxAge > 0
func (s *Server) HandleFunc(pattern string, handler HandlerFunc) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		NoCache(w)
		SetHSTS(w, s.hstsMaxAge, s.cfg.HstsIncludeSubdomains)
		handler(w, r)
	})
	glog.V(3).Infof("Registered handler for %v", pattern)
}

// Run starts the HTTPS server.
func (s *Server) Run() error {
	glog.V(4).Infof("HTTP server listening on %v", s.server.Addr)
	return s.server.ListenAndServeTLS(s.cfg.TlsConfig.CertFile, s.cfg.TlsConfig.KeyFile)
}
