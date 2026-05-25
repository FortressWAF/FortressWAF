package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Server struct {
	mu          sync.RWMutex
	router      *mux.Router
	adminRouter *mux.Router
	srv         *http.Server
	adminSrv    *http.Server
	config      *ServerConfig
	handlers    *Handlers
	wsUpgrader  websocket.Upgrader
	done        chan struct{}
	baseCtx     context.Context
	cancel      context.CancelFunc
}

type ServerConfig struct {
	Port         int
	AdminPort    int
	AdminEnabled bool
	MTLSEnabled  bool
	CACertFile   string
	CertFile     string
	KeyFile      string
	APIKeys      []string
	JWTSecret    string
	DevMode      bool
}

func NewServer(cfg *ServerConfig, handlers *Handlers) *Server {
	s := &Server{
		config:   cfg,
		handlers: handlers,
		router:   mux.NewRouter(),
		done:     make(chan struct{}),
		wsUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	s.setupRoutes()
	s.setupAdminRoutes()

	return s
}

func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	api.Use(s.authMiddleware)
	api.Use(s.rateLimitMiddleware)
	api.Use(s.loggingMiddleware)

	sites := api.PathPrefix("/sites").Subrouter()
	sites.HandleFunc("", s.handlers.ListSites).Methods("GET")
	sites.HandleFunc("", s.handlers.CreateSite).Methods("POST")
	sites.HandleFunc("/{id}", s.handlers.GetSite).Methods("GET")
	sites.HandleFunc("/{id}", s.handlers.UpdateSite).Methods("PUT")
	sites.HandleFunc("/{id}", s.handlers.DeleteSite).Methods("DELETE")

	rules := api.PathPrefix("/rules").Subrouter()
	rules.HandleFunc("", s.handlers.ListRules).Methods("GET")
	rules.HandleFunc("", s.handlers.CreateRule).Methods("POST")
	rules.HandleFunc("/{id}", s.handlers.GetRule).Methods("GET")
	rules.HandleFunc("/{id}", s.handlers.UpdateRule).Methods("PUT")
	rules.HandleFunc("/{id}", s.handlers.DeleteRule).Methods("DELETE")
	rules.HandleFunc("/{id}/test", s.handlers.TestRule).Methods("POST")
	rules.HandleFunc("/{id}/reorder", s.handlers.ReorderRule).Methods("PUT")

	logs := api.PathPrefix("/logs").Subrouter()
	logs.HandleFunc("", s.handlers.QueryLogs).Methods("GET")
	logs.HandleFunc("/tail", s.handlers.TailLogs).Methods("GET")
	logs.HandleFunc("/export", s.handlers.ExportLogs).Methods("GET")

	analytics := api.PathPrefix("/analytics").Subrouter()
	analytics.HandleFunc("/traffic", s.handlers.TrafficStats).Methods("GET")
	analytics.HandleFunc("/attacks", s.handlers.AttackStats).Methods("GET")

	patches := api.PathPrefix("/patches").Subrouter()
	patches.HandleFunc("", s.handlers.ListPatches).Methods("GET")
	patches.HandleFunc("", s.handlers.CreatePatch).Methods("POST")
	patches.HandleFunc("/{id}/apply", s.handlers.ApplyPatch).Methods("POST")
	patches.HandleFunc("/{id}/revoke", s.handlers.RevokePatch).Methods("POST")

	config := api.PathPrefix("/config").Subrouter()
	config.HandleFunc("/export", s.handlers.ExportConfig).Methods("GET")
	config.HandleFunc("/import", s.handlers.ImportConfig).Methods("POST")
	config.HandleFunc("/validate", s.handlers.ValidateConfig).Methods("POST")
	config.HandleFunc("/diff", s.handlers.DiffConfig).Methods("POST")

	api.HandleFunc("/health", s.handlers.Health).Methods("GET")
	api.HandleFunc("/alerts", s.handlers.ListAlerts).Methods("GET")
	api.HandleFunc("/alerts", s.handlers.CreateAlert).Methods("POST")
	api.HandleFunc("/alerts/{id}", s.handlers.UpdateAlert).Methods("PUT")

	tenants := api.PathPrefix("/tenants").Subrouter()
	tenants.HandleFunc("", s.handlers.ListTenants).Methods("GET")
	tenants.HandleFunc("", s.handlers.CreateTenant).Methods("POST")
	tenants.HandleFunc("/{id}", s.handlers.GetTenant).Methods("GET")
	tenants.HandleFunc("/{id}", s.handlers.UpdateTenant).Methods("PUT")
	tenants.HandleFunc("/{id}", s.handlers.DeleteTenant).Methods("DELETE")

	s.router.HandleFunc("/ws", s.handleWebSocket)
}

func (s *Server) setupAdminRoutes() {
	s.adminRouter = mux.NewRouter()
	s.adminRouter.Use(s.authMiddleware)
	s.adminRouter.Use(s.loggingMiddleware)

	admin := s.adminRouter.PathPrefix("/admin").Subrouter()
	admin.HandleFunc("/status", s.handlers.AdminStatus).Methods("GET")
	admin.HandleFunc("/config", s.handlers.AdminConfig).Methods("GET")
	admin.HandleFunc("/reload", s.handlers.AdminReload).Methods("POST")
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			for _, key := range s.config.APIKeys {
				if key == apiKey {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token := ""
			fmt.Sscanf(authHeader, "Bearer %s", &token)
			if token != "" {
				// Validate Bearer token against session store
				s.handlers.sessionsMu.RLock()
				session, ok := s.handlers.sessions[token]
				s.handlers.sessionsMu.RUnlock()

				if ok && time.Now().Before(session.ExpiresAt) {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		if s.config.DevMode {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	})
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("api request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"duration", time.Since(start),
		)
	})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	slog.Info("websocket connected", "remote", r.RemoteAddr)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func (s *Server) Start() error {
	s.baseCtx, s.cancel = context.WithCancel(context.Background())
	s.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext:  func(_ net.Listener) context.Context { return s.baseCtx },
	}

	go func() {
		slog.Info("api server starting", "port", s.config.Port)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("api server error", "error", err)
		}
	}()

	if s.config.AdminEnabled {
		s.adminSrv = &http.Server{
			Addr:         fmt.Sprintf(":%d", s.config.AdminPort),
			Handler:      s.adminRouter,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
			BaseContext:  func(_ net.Listener) context.Context { return s.baseCtx },
		}
		go func() {
			slog.Info("admin server starting", "port", s.config.AdminPort)
			var err error
			if s.config.MTLSEnabled {
				err = s.startMTLSServer()
			} else {
				err = s.adminSrv.ListenAndServe()
			}
			if err != nil && err != http.ErrServerClosed {
				slog.Error("admin server error", "error", err)
			}
		}()
	}

	return nil
}

func (s *Server) startMTLSServer() error {
	caCert, err := os.ReadFile(s.config.CACertFile)
	if err != nil {
		return fmt.Errorf("read ca cert: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
		MinVersion: tls.VersionTLS12,
	}

	s.adminSrv.TLSConfig = tlsConfig
	return s.adminSrv.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
}

func (s *Server) Shutdown(ctx context.Context) error {
	var errs []error

	if s.cancel != nil {
		s.cancel()
	}

	if s.srv != nil {
		if err := s.srv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("api server shutdown: %w", err))
		}
	}

	if s.adminSrv != nil {
		if err := s.adminSrv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("admin server shutdown: %w", err))
		}
	}

	close(s.done)

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (s *Server) Routes() *mux.Router {
	return s.router
}

func (s *Server) Broadcast(event string, data interface{}) {
}

var _ = slog.Debug
