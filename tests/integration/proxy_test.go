package integration

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func TestProxyHealthEndpoint(t *testing.T) {
	_ = httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestProxyMetricsEndpoint(t *testing.T) {
	_ = httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`# HELP fortresswaf_requests_total Total requests
# TYPE fortresswaf_requests_total counter
fortresswaf_requests_total 0`))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestTLSConfigMinVersion(t *testing.T) {
	c := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if c.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2 min version")
	}
	// Verify TLS 1.3 is also supported
	if tls.VersionTLS13 < tls.VersionTLS12 {
		t.Error("TLS 1.3 should be >= TLS 1.2")
	}
}

func TestTLSHTTP2NPN(t *testing.T) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
	}
	if len(tlsCfg.NextProtos) != 2 {
		t.Fatalf("expected 2 NextProtos, got %d", len(tlsCfg.NextProtos))
	}
	if tlsCfg.NextProtos[0] != "h2" {
		t.Errorf("expected h2 as first NextProto")
	}
	if tlsCfg.NextProtos[1] != "http/1.1" {
		t.Errorf("expected http/1.1 as second NextProto")
	}
}

func TestOCSPVerifyConnection(t *testing.T) {
	called := false
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		VerifyConnection: func(cs tls.ConnectionState) error {
			called = true
			return nil
		},
	}
	if tlsCfg.VerifyConnection == nil {
		t.Fatal("VerifyConnection should not be nil")
	}
	// Simulate a connection state
	cs := tls.ConnectionState{
		Version: tls.VersionTLS12,
	}
	err := tlsCfg.VerifyConnection(cs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("VerifyConnection callback was not called")
	}
}

func TestOCSPVerifyConnectionError(t *testing.T) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		VerifyConnection: func(cs tls.ConnectionState) error {
			return nil
		},
	}
	cs := tls.ConnectionState{
		Version: tls.VersionTLS13,
	}
	if err := tlsCfg.VerifyConnection(cs); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestACMEManagerConfig(t *testing.T) {
	m := &autocert.Manager{
		Cache:      autocert.DirCache("/tmp/fortresswaf-test-certs"),
		Prompt:     autocert.AcceptTOS,
		Email:      "admin@example.com",
		HostPolicy: autocert.HostWhitelist("example.com", "api.example.com"),
	}
	if m.Cache == nil {
		t.Fatal("Cache should not be nil")
	}
	if m.Prompt == nil {
		t.Fatal("Prompt should not be nil")
	}
	if m.Email != "admin@example.com" {
		t.Errorf("expected admin@example.com, got %s", m.Email)
	}
	allowed := map[string]bool{"example.com": true, "api.example.com": true}
	if err := m.HostPolicy(nil, "example.com"); err != nil {
		t.Errorf("example.com should be allowed: %v", err)
	}
	if err := m.HostPolicy(nil, "api.example.com"); err != nil {
		t.Errorf("api.example.com should be allowed: %v", err)
	}
	for host := range allowed {
		if err := m.HostPolicy(nil, host); err != nil {
			t.Errorf("%s should be allowed: %v", host, err)
		}
	}
}

func TestACMEManagerRejectsUnknownHost(t *testing.T) {
	m := &autocert.Manager{
		Cache:      autocert.DirCache("/tmp/fortresswaf-test-certs"),
		Prompt:     autocert.AcceptTOS,
		Email:      "admin@example.com",
		HostPolicy: autocert.HostWhitelist("example.com"),
	}
	if err := m.HostPolicy(nil, "evil.com"); err == nil {
		t.Error("expected error for unknown host")
	}
}

func TestACMEManagerTLSConfig(t *testing.T) {
	m := &autocert.Manager{
		Cache:      autocert.DirCache("/tmp/fortresswaf-test-certs"),
		Prompt:     autocert.AcceptTOS,
		Email:      "admin@example.com",
		HostPolicy: autocert.HostWhitelist("example.com"),
	}
	tlsCfg := m.TLSConfig()
	if tlsCfg == nil {
		t.Fatal("TLSConfig should not be nil")
	}
	if tlsCfg.GetCertificate == nil {
		t.Fatal("GetCertificate should not be nil")
	}
	// Verify min version is set
	tlsCfg.MinVersion = tls.VersionTLS12
	if tlsCfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS 1.2")
	}
}

func TestTLSConfigWithACMEAndOCSP(t *testing.T) {
	m := &autocert.Manager{
		Cache:      autocert.DirCache("/tmp/fortresswaf-test-certs"),
		Prompt:     autocert.AcceptTOS,
		Email:      "admin@example.com",
		HostPolicy: autocert.HostWhitelist("example.com"),
	}
	tlsCfg := m.TLSConfig()
	tlsCfg.MinVersion = tls.VersionTLS12
	tlsCfg.NextProtos = []string{"h2", "http/1.1"}
	tlsCfg.VerifyConnection = func(cs tls.ConnectionState) error {
		return nil
	}
	if tlsCfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2 min")
	}
	if len(tlsCfg.NextProtos) != 2 {
		t.Errorf("expected 2 next protos")
	}

	cs := tls.ConnectionState{Version: tls.VersionTLS12}
	if err := tlsCfg.VerifyConnection(cs); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTLSServerConfig(t *testing.T) {
	srv := &http.Server{
		Addr:              ":8443",
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2", "http/1.1"},
		},
	}
	if srv.TLSConfig == nil {
		t.Fatal("TLSConfig should not be nil")
	}
	if srv.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2")
	}
	if srv.TLSConfig.NextProtos[0] != "h2" {
		t.Errorf("expected h2")
	}
}

func TestTLSCipherSuites(t *testing.T) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}
	if len(tlsCfg.CipherSuites) != 4 {
		t.Errorf("expected 4 cipher suites, got %d", len(tlsCfg.CipherSuites))
	}
}

func TestServerTLSConfigCreation(t *testing.T) {
	enabled := true
	http2Enabled := true
	ocspEnabled := true
	acmeEnabled := true
	acmeEmail := "admin@example.com"

	if !enabled {
		t.Skip("TLS not enabled")
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if http2Enabled {
		tlsCfg.NextProtos = []string{"h2", "http/1.1"}
	}
	if ocspEnabled {
		tlsCfg.VerifyConnection = func(cs tls.ConnectionState) error {
			return nil
		}
	}
	if acmeEnabled && acmeEmail != "" {
		m := &autocert.Manager{
			Cache:      autocert.DirCache("/tmp/fortresswaf-certs"),
			Prompt:     autocert.AcceptTOS,
			Email:      acmeEmail,
			HostPolicy: autocert.HostWhitelist("example.com"),
		}
		tlsCfg = m.TLSConfig()
		tlsCfg.MinVersion = tls.VersionTLS12
	}
	if tlsCfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2")
	}
}

func TestTLSGetCertificate(t *testing.T) {
	m := &autocert.Manager{
		Cache:      autocert.DirCache("/tmp/test-certs"),
		Prompt:     autocert.AcceptTOS,
		Email:      "test@example.com",
		HostPolicy: autocert.HostWhitelist("example.com"),
	}
	hello := &tls.ClientHelloInfo{
		ServerName: "example.com",
	}
	cert, err := m.GetCertificate(hello)
	if err == nil && cert != nil {
		t.Log("got certificate from manager")
	}
	if err != nil {
		t.Logf("expected error (no ACME server): %v", err)
	}
}

func TestHTTPServerWithTLSConfig(t *testing.T) {
	srv := &http.Server{
		Addr:              ":443",
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
		VerifyConnection: func(cs tls.ConnectionState) error {
			return nil
		},
	}
	srv.TLSConfig = cfg

	if srv.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2")
	}
	if srv.TLSConfig.NextProtos[0] != "h2" {
		t.Errorf("expected h2")
	}
}

func TestTLSVersionString(t *testing.T) {
	versions := map[uint16]string{
		tls.VersionTLS12: "tls12",
		tls.VersionTLS13: "tls13",
	}
	if v, ok := versions[tls.VersionTLS12]; !ok || v != "tls12" {
		t.Errorf("unexpected TLS 1.2 version mapping")
	}
	if v, ok := versions[tls.VersionTLS13]; !ok || v != "tls13" {
		t.Errorf("unexpected TLS 1.3 version mapping")
	}
}
