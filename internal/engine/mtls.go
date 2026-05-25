package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/FortressWAF/FortressWAF/internal/config"
)

type MTLSInspector struct {
	mu             sync.RWMutex
	mtlsCfg        config.MTLSConfig
	caCert         *x509.CertPool
	verifyDepth    int
	failOnError    bool
	earlyAuth      bool
	usernameHeader string
}

type ClientCertInfo struct {
	Subject      string
	Issuer       string
	NotBefore    time.Time
	NotAfter     time.Time
	SerialNumber string
	Fingerprint  string
	PEM          string
}

func NewMTLSInspector(cfg config.MTLSConfig) (*MTLSInspector, error) {
	i := &MTLSInspector{
		mtlsCfg:        cfg,
		verifyDepth:    cfg.VerifyDepth,
		failOnError:    cfg.FailOnError,
		earlyAuth:      cfg.EarlyAuth,
		usernameHeader: cfg.UsernameHeader,
	}

	if cfg.CAFile != "" {
		caCert, err := loadCAFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("load CA cert: %w", err)
		}
		i.caCert = caCert
	}

	return i, nil
}

func loadCAFile(path string) (*x509.CertPool, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("no valid CA certificates found in %s", path)
	}
	return pool, nil
}

func (m *MTLSInspector) Name() string { return "mtls_inspection" }

func (m *MTLSInspector) Inspect(ctx *RequestContext) (*Decision, error) {
	if !m.mtlsCfg.Enabled {
		return &Decision{Action: ActionAllow}, nil
	}

	conn := GetConnFromContext(ctx.Request)
	if conn == nil {
		if m.failOnError {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "MTLS-001",
				RuleName: "mTLS connection info unavailable",
				Severity: "high",
				Score:    85,
			}, nil
		}
		return &Decision{Action: ActionAllow}, nil
	}

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		if m.failOnError {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "MTLS-002",
				RuleName: "connection is not TLS",
				Severity: "high",
				Score:    90,
			}, nil
		}
		return &Decision{Action: ActionAllow}, nil
	}

	peerCerts := tlsConn.ConnectionState().PeerCertificates
	if len(peerCerts) == 0 {
		if m.mtlsCfg.ClientAuth == "require-and-verify-client-cert" || m.mtlsCfg.ClientAuth == "require-any-client-cert" {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "MTLS-003",
				RuleName: "client certificate required",
				Severity: "high",
				Score:    90,
				Evidence: "no client certificate provided",
			}, nil
		}
		return &Decision{Action: ActionAllow}, nil
	}

	cert := peerCerts[0]
	certInfo := extractCertInfo(cert)

	if m.usernameHeader != "" {
		ctx.Headers[m.usernameHeader] = certInfo.Subject
	}

	if m.mtlsCfg.SkipVerify {
		return &Decision{Action: ActionAllow}, nil
	}

	if m.mtlsCfg.PolicyOID != "" {
		if !m.validateCertificatePolicy(cert) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "MTLS-004",
				RuleName: "client certificate policy violation",
				Severity: "high",
				Score:    85,
				Evidence: fmt.Sprintf("required policy: %s", m.mtlsCfg.PolicyOID),
			}, nil
		}
	}

	return &Decision{Action: ActionAllow}, nil
}

func (m *MTLSInspector) validateCertificatePolicy(cert *x509.Certificate) bool {
	if m.mtlsCfg.PolicyOID == "" {
		return true
	}

	for _, ext := range cert.Extensions {
		if ext.Id.String() == "2.5.29.32" {
			var policies []asn1.ObjectIdentifier
			if _, err := asn1.Unmarshal(ext.Value, &policies); err != nil {
				slog.Warn("failed to unmarshal certificate policies", "error", err)
				return false
			}
			for _, policy := range policies {
				if policy.String() == m.mtlsCfg.PolicyOID {
					return true
				}
			}
		}
	}
	return false
}

func extractCertInfo(cert *x509.Certificate) ClientCertInfo {
	return ClientCertInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		SerialNumber: cert.SerialNumber.String(),
	}
}

type connContextKey struct{}

func GetConnFromContext(r *http.Request) net.Conn {
	if r.Context().Value(connContextKey{}) != nil {
		return r.Context().Value(connContextKey{}).(net.Conn)
	}
	return nil
}

func SetConnInContext(r *http.Request, conn net.Conn) *http.Request {
	return r.WithContext(
		context.WithValue(r.Context(), connContextKey{}, conn),
	)
}

func (m *MTLSInspector) GetClientCertInfo(r *http.Request) *ClientCertInfo {
	conn := GetConnFromContext(r)
	if conn == nil {
		return nil
	}

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return nil
	}

	peerCerts := tlsConn.ConnectionState().PeerCertificates
	if len(peerCerts) == 0 {
		return nil
	}

	info := extractCertInfo(peerCerts[0])
	return &info
}
