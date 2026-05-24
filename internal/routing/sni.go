package routing

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
)

type SNIRouter struct {
	mu       sync.RWMutex
	certs    map[string]*tls.Certificate
	wildcard map[string]*tls.Certificate
	fallback string
}

func NewSNIRouter() *SNIRouter {
	return &SNIRouter{
		certs:    make(map[string]*tls.Certificate),
		wildcard: make(map[string]*tls.Certificate),
	}
}

func (s *SNIRouter) AddCert(hostname string, cert *tls.Certificate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.HasPrefix(hostname, "*.") {
		s.wildcard[hostname[2:]] = cert
	} else {
		s.certs[hostname] = cert
	}
}

func (s *SNIRouter) SetFallback(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallback = name
}

func (s *SNIRouter) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hostname := hello.ServerName
	if hostname == "" {
		hostname = s.fallback
	}

	if cert, ok := s.certs[hostname]; ok {
		return cert, nil
	}

	for suffix, cert := range s.wildcard {
		if strings.HasSuffix(hostname, suffix) {
			return cert, nil
		}
	}

	if s.fallback != "" {
		if cert, ok := s.certs[s.fallback]; ok {
			return cert, nil
		}
	}

	return nil, fmt.Errorf("no certificate found for hostname: %s", hostname)
}

func (s *SNIRouter) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: s.GetCertificate,
		MinVersion:     tls.VersionTLS12,
		MaxVersion:     tls.VersionTLS13,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		PreferServerCipherSuites: true,
	}
}

func (s *SNIRouter) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hosts := make([]string, 0, len(s.certs))
	for h := range s.certs {
		hosts = append(hosts, h)
	}
	for w := range s.wildcard {
		hosts = append(hosts, "*."+w)
	}
	return hosts
}

func (s *SNIRouter) Remove(hostname string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.certs, hostname)
	if strings.HasPrefix(hostname, "*.") {
		delete(s.wildcard, hostname[2:])
	}
}
