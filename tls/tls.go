// Package tls provides helpers around the crypto/tls package.
// Configuration is done via protobuf messages.
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"

	pb "github.com/hazaelsan/ssh-relay/proto/v1/tls_go_proto"
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
	clientAuthMap = map[pb.TlsConfig_ClientAuthType]tls.ClientAuthType{
		pb.TlsConfig_CLIENT_AUTH_TYPE_UNSPECIFIED:   tls.RequireAndVerifyClientCert,
		pb.TlsConfig_NO_CLIENT_CERT:                 tls.NoClientCert,
		pb.TlsConfig_REQUEST_CLIENT_CERT:            tls.RequestClientCert,
		pb.TlsConfig_REQUIRE_ANY_CLIENT_CERT:        tls.RequireAnyClientCert,
		pb.TlsConfig_VERIFY_CLIENT_CERT_IF_GIVEN:    tls.VerifyClientCertIfGiven,
		pb.TlsConfig_REQUIRE_AND_VERIFY_CLIENT_CERT: tls.RequireAndVerifyClientCert,
	}
)

// ClientAuthType converts a proto ClientAuthType to its tls package equivalent.
func ClientAuthType(t pb.TlsConfig_ClientAuthType) (tls.ClientAuthType, error) {
	if val, ok := clientAuthMap[t]; ok {
		return val, nil
	}
	return 0, ErrBadClientAuthType
}

// Config creates a *tls.Config directive from a proto message.
func Config(cfg *pb.TlsConfig) (*tls.Config, error) {
	cat, err := ClientAuthType(cfg.ClientAuthType)
	if err != nil {
		return nil, err
	}
	c := &tls.Config{
		ClientAuth: cat,
		MinVersion: TLSMinVersion,
	}
	clientCAs, err := loadCerts(cfg.ClientCaCerts)
	if err != nil {
		return nil, err
	}
	c.ClientCAs = clientCAs
	rootCAs, err := loadCerts(cfg.RootCaCerts)
	if err != nil {
		return nil, err
	}
	c.RootCAs = rootCAs
	return c, nil
}

// CertConfig creates a *tls.Config directive from a proto message,
// loading an X.509 certificate from the cert/key files specified.
func CertConfig(cfg *pb.TlsConfig) (*tls.Config, error) {
	c, err := Config(cfg)
	if err != nil {
		return nil, err
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}
	c.Certificates = []tls.Certificate{cert}
	c.BuildNameToCertificate()
	return c, nil
}

// loadCerts loads all the public certificates into a CertPool.
func loadCerts(certs []string) (*x509.CertPool, error) {
	if len(certs) == 0 {
		return nil, nil
	}
	pool := x509.NewCertPool()
	for _, f := range certs {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("ReadFile(%v) error: %w", f, err)
		}
		if !pool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("AppendCertsFromPEM() error: %w", err)
		}
	}
	return pool, nil
}
