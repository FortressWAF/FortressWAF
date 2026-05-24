package cert

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

type AutoCertConfig struct {
	Enabled     bool
	Email       string
	CacheDir    string
	Hosts       []string
	RenewalDays int
}

type AutoCertManager struct {
	mu         sync.RWMutex
	config     *AutoCertConfig
	certManager *autocert.Manager
	cache      *certCache
}

type certCache struct {
	mu    sync.RWMutex
	certs map[string]*tls.Certificate
	dir   string
}

func NewAutoCertManager(cfg AutoCertConfig) (*AutoCertManager, error) {
	m := &AutoCertManager{
		config: &cfg,
		cache: &certCache{
			certs: make(map[string]*tls.Certificate),
			dir:   cfg.CacheDir,
		},
	}

	if !cfg.Enabled {
		return m, nil
	}

	cacheDir := cfg.CacheDir
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "fortresswaf-autocert")
	}

	m.certManager = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Email:      cfg.Email,
		Cache:      autocert.DirCache(cacheDir),
		HostPolicy: autocert.HostWhitelist(cfg.Hosts...),
	}

	return m, nil
}

func (m *AutoCertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if m.certManager == nil {
		return nil, fmt.Errorf("autocert not enabled")
	}

	cert, err := m.certManager.GetCertificate(hello)
	if err != nil {
		return nil, err
	}

	m.cacheCert(hello.ServerName, cert)
	return cert, nil
}

func (m *AutoCertManager) cacheCert(hostname string, cert *tls.Certificate) {
	m.cache.mu.Lock()
	defer m.cache.mu.Unlock()
	m.cache.certs[hostname] = cert
}

func (m *AutoCertManager) GetTLSCert() *tls.Config {
	if m.certManager == nil {
		return nil
	}

	return &tls.Config{
		GetCertificate: m.GetCertificate,
		MinVersion:     tls.VersionTLS12,
		MaxVersion:     tls.VersionTLS13,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		PreferServerCipherSuites: true,
	}
}

func (m *AutoCertManager) ServeHTTP(addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: m.GetTLSCert(),
	}

	slog.Info("autocert server listening", "addr", addr)
	return server.ListenAndServe()
}

func (m *AutoCertManager) StartRenewalLoop(ctx context.Context) {
	if m.certManager == nil {
		return
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.renewCertificates()
		case <-ctx.Done():
			return
		}
	}
}

func (m *AutoCertManager) renewCertificates() {
	slog.Info("checking certificates for renewal")

	hosts := m.config.Hosts
	for _, host := range hosts {
		if m.shouldRenew(host) {
			slog.Info("renewing certificate", "host", host)
			m.renewCertificate(host)
		}
	}
}

func (m *AutoCertManager) shouldRenew(hostname string) bool {
	m.cache.mu.RLock()
	cert, ok := m.cache.certs[hostname]
	m.cache.mu.RUnlock()

	if !ok {
		return true
	}

	renewalDays := m.config.RenewalDays
	if renewalDays <= 0 {
		renewalDays = 30
	}

	expiry := cert.Leaf.NotAfter
	renewalTime := expiry.AddDate(0, 0, -renewalDays)

	return time.Now().After(renewalTime)
}

func (m *AutoCertManager) renewCertificate(hostname string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cert, err := m.certManager.GetCertificate(&tls.ClientHelloInfo{
		ServerName: hostname,
	})
	if err != nil {
		slog.Error("certificate renewal failed", "host", hostname, "error", err)
		return
	}

	m.cacheCert(hostname, cert)
	slog.Info("certificate renewed", "host", hostname, "expires", cert.Leaf.NotAfter)
}

func (m *AutoCertManager) GetCertStats() map[string]CertStats {
	m.cache.mu.RLock()
	defer m.cache.mu.RUnlock()

	stats := make(map[string]CertStats)
	for host, cert := range m.cache.certs {
		stats[host] = CertStats{
			Issuer:     cert.Leaf.Issuer.String(),
			NotBefore:  cert.Leaf.NotBefore,
			NotAfter:   cert.Leaf.NotAfter,
			WantRenew:  m.shouldRenew(host),
		}
	}
	return stats
}

type CertStats struct {
	Issuer    string
	NotBefore time.Time
	NotAfter  time.Time
	WantRenew bool
}

func (m *AutoCertManager) Close() error {
	return nil
}

func isIPAddress(s string) bool {
	return net.ParseIP(s) != nil
}

func isWildcard(s string) bool {
	return strings.HasPrefix(s, "*.")
}

var _ = context.Background
