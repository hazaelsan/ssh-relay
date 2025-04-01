// Package tls provides helpers around the crypto/tls package.
// Configuration is done via protobuf messages.
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/hazaelsan/ssh-relay/proto/v1/tlspb"
)

const (
	// TLSMinVersion is the minimum SSL/TLS version supported.
	TLSMinVersion = tls.VersionTLS12
)

var (
	// ErrBadClientAuthType is returned if the corresponding ClientAuthType could not be found.
	ErrBadClientAuthType = errors.New("bad ClientAuthType")
)

var (
	clientAuthMap = map[tlspb.TlsConfig_ClientAuthType]tls.ClientAuthType{
		tlspb.TlsConfig_CLIENT_AUTH_TYPE_UNSPECIFIED:   tls.RequireAndVerifyClientCert,
		tlspb.TlsConfig_NO_CLIENT_CERT:                 tls.NoClientCert,
		tlspb.TlsConfig_REQUEST_CLIENT_CERT:            tls.RequestClientCert,
		tlspb.TlsConfig_REQUIRE_ANY_CLIENT_CERT:        tls.RequireAnyClientCert,
		tlspb.TlsConfig_VERIFY_CLIENT_CERT_IF_GIVEN:    tls.VerifyClientCertIfGiven,
		tlspb.TlsConfig_REQUIRE_AND_VERIFY_CLIENT_CERT: tls.RequireAndVerifyClientCert,
	}
)

// ClientAuthType converts a proto ClientAuthType to its tls package equivalent.
func ClientAuthType(t tlspb.TlsConfig_ClientAuthType) (tls.ClientAuthType, error) {
	if val, ok := clientAuthMap[t]; ok {
		return val, nil
	}
	return 0, ErrBadClientAuthType
}

// Config creates a *tls.Config directive from a proto message.
// Returns nil if TLS is disabled in the config.
func Config(cfg *tlspb.TlsConfig) (*tls.Config, error) {
	if cfg.GetTlsMode() == tlspb.TlsConfig_TLS_MODE_DISABLED {
		return nil, nil
	}
	cat, err := ClientAuthType(cfg.GetClientAuthType())
	if err != nil {
		return nil, err
	}
	c := &tls.Config{
		ClientAuth: cat,
		MinVersion: TLSMinVersion,
	}
	clientCAs, err := loadCerts(cfg.GetClientCaCerts())
	if err != nil {
		return nil, err
	}
	c.ClientCAs = clientCAs
	rootCAs, err := loadCerts(cfg.GetRootCaCerts())
	if err != nil {
		return nil, err
	}
	c.RootCAs = rootCAs
	return c, nil
}

// CertConfig creates a *tls.Config directive from a proto message,
// loading an X.509 certificate from the cert/key files specified.
func CertConfig(cfg *tlspb.TlsConfig) (*tls.Config, error) {
	c, err := Config(cfg)
	if err != nil {
		return nil, err
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}
	c.Certificates = []tls.Certificate{cert}
	return c, nil
}

// TransportCreds returns the correct credentials for a gRPC connection.
func TransportCreds(cfg *tlspb.TlsConfig) (credentials.TransportCredentials, error) {
	if cfg.GetTlsMode() == tlspb.TlsConfig_TLS_MODE_DISABLED {
		return insecure.NewCredentials(), nil
	}
	tlsCfg, err := CertConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("CertConfig() error: %w", err)
	}
	return credentials.NewTLS(tlsCfg), nil
}

// loadCerts loads all the public certificates into a CertPool.
func loadCerts(certs []string) (*x509.CertPool, error) {
	if len(certs) == 0 {
		return nil, nil
	}
	pool := x509.NewCertPool()
	for _, f := range certs {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("ReadFile(%v) error: %w", f, err)
		}
		if !pool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("AppendCertsFromPEM() error: %w", err)
		}
	}
	return pool, nil
}
