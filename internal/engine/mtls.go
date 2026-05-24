package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type MTLSInspector struct {
	mu              sync.RWMutex
	config          MTLSConfig
	caCert          *x509.CertPool
	verifyDepth     int
	failOnError     bool
	earlyAuth       bool
	usernameHeader  string
}

type MTLSConfig struct {
	Enabled        bool
	CAFile         string
	ClientAuth     string
	SkipVerify     bool
	PolicyOID      string
	VerifyDepth    int
	FailOnError    bool
	EarlyAuth      bool
	UsernameHeader string
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

func NewMTLSInspector(cfg MTLSConfig) (*MTLSInspector, error) {
	i := &MTLSInspector{
		config:         cfg,
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
	return nil, nil
}

func (m *MTLSInspector) Name() string { return "mtls_inspection" }

func (m *MTLSInspector) Inspect(ctx *RequestContext) (*Decision, error) {
	if !m.config.Enabled {
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
		if m.config.ClientAuth == "require-and-verify-client-cert" || m.config.ClientAuth == "require-any-client-cert" {
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

	if m.config.SkipVerify {
		return &Decision{Action: ActionAllow}, nil
	}

	if m.config.PolicyOID != "" {
		if !m.validateCertificatePolicy(cert) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "MTLS-004",
				RuleName: "client certificate policy violation",
				Severity: "high",
				Score:    85,
				Evidence: fmt.Sprintf("required policy: %s", m.config.PolicyOID),
			}, nil
		}
	}

	return &Decision{Action: ActionAllow}, nil
}

func (m *MTLSInspector) validateCertificatePolicy(cert *x509.Certificate) bool {
	if m.config.PolicyOID == "" {
		return true
	}

	for _, ext := range cert.Extensions {
		if ext.Id.String() == "2.5.29.32" {
			return strings.Contains(string(ext.Value), m.config.PolicyOID)
		}
	}
	return false
}

func extractCertInfo(cert *x509.Certificate) ClientCertInfo {
	return ClientCertInfo{
		Subject:      cert.Subject.String(),
		Issuer:      cert.Issuer.String(),
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
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
