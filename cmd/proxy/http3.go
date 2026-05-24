package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go/http3"
)

type HTTP3Server struct {
	addr      string
	certFile  string
	keyFile   string
	tlsConfig *tls.Config
	handler   http.Handler
	server    *http3.Server
	ln        *quicListener
	wg        sync.WaitGroup
	shutdown  bool
	mu        sync.Mutex
}

type quicListener struct {
	ln  net.PacketConn
	tls *tls.Config
}

func NewHTTP3Server(addr, certFile, keyFile string, handler http.Handler, tlsConfig *tls.Config) *HTTP3Server {
	return &HTTP3Server{
		addr:      addr,
		certFile:  certFile,
		keyFile:  keyFile,
		handler:  handler,
		tlsConfig: tlsConfig,
	}
}

func (s *HTTP3Server) ListenAndServe() error {
	if s.shutdown {
		return fmt.Errorf("server closed")
	}

	cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
	if err != nil {
		return fmt.Errorf("load cert: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"h3", "h3-34", "h3-35", "h3-36", "h3-38", "h3-39"},
	}
	if s.tlsConfig != nil {
		tlsConfig = s.tlsConfig
	}

	ln, err := net.ListenPacket("udp", s.addr)
	if err != nil {
		return fmt.Errorf("listen udp: %w", err)
	}

	s.ln = &quicListener{ln: ln, tls: tlsConfig}

	s.server = &http3.Server{
		Addr:         s.addr,
		Handler:      s.handler,
		TLSConfig:    tlsConfig,
		MaxHeaderBytes: 1 << 20,
		QuicConfig: &quic.Config{
			MaxBidirectionalStreams: 100,
			MaxUnidirectionalStreams: 100,
			HandshakeTimeout: 10 * time.Second,
			IdleTimeout: 30 * time.Second,
		},
	}

	slog.Info("http/3 server listening", "addr", s.addr)
	return s.server.Serve(s.ln)
}

func (s *HTTP3Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.shutdown = true
	s.mu.Unlock()

	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *HTTP3Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.shutdown = true

	if s.server != nil {
		s.server.Close()
	}
	if s.ln != nil {
		return s.ln.ln.Close()
	}
	return nil
}

func (s *quicListener) Accept(ctx context.Context) (net.PacketConn, *tls.Config, error) {
	return s.ln, s.tls, nil
}

type HTTP3Config struct {
	Enabled  bool
	Port     int
	CertFile string
	KeyFile  string
}

func (cfg *HTTP3Config) IsEnabled() bool {
	return cfg != nil && cfg.Enabled
}

func GetHTTP3Addr(cfg *HTTP3Config, fallback int) string {
	if cfg != nil && cfg.Port > 0 {
		return fmt.Sprintf(":%d", cfg.Port)
	}
	return fmt.Sprintf(":%d", fallback)
}

var _ net.PacketConn = (*net.UDPConn)(nil)
