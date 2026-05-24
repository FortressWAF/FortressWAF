package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/zulfff/FortressWAF/internal/config"
	"github.com/zulfff/FortressWAF/internal/engine"
	"github.com/gorilla/mux"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
	startedAt time.Time

	totalRequests   atomic.Int64
	blockedRequests atomic.Int64
	allowedRequests atomic.Int64
	challengedReqs  atomic.Int64
	rateLimitedReqs atomic.Int64
	monitoredReqs   atomic.Int64
	activeConns     atomic.Int64
	bytesSent       atomic.Int64
	bytesReceived   atomic.Int64
)

func main() {
	configPath := flag.String("config", "deploy/config.yaml", "path to YAML config file")
	dev := flag.Bool("dev", false, "enable dev mode: verbose logging and rule debug")
	adminPort := flag.Int("admin-port", 8443, "admin API server port")
	proxyPort := flag.Int("proxy-port", 80, "reverse proxy listening port")
	flag.Parse()

	level := slog.LevelInfo
	if *dev {
		level = slog.LevelDebug
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))

	startedAt = time.Now()

	slog.Info("fortresswaf starting",
		"version", Version,
		"commit", Commit,
		"build_date", BuildDate,
		"dev", *dev,
		"config", *configPath,
	)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "path", *configPath, "error", err)
		os.Exit(1)
	}

	cfgMgr, err := config.NewManager(*configPath)
	if err != nil {
		slog.Error("failed to create config manager", "error", err)
		os.Exit(1)
	}
	defer cfgMgr.Close()

	slog.Info("configuration loaded",
		"sites", len(cfg.Sites),
		"rules", len(cfg.Rules),
		"admin_port", *adminPort,
		"proxy_port", *proxyPort,
	)

	for _, site := range cfg.Sites {
		slog.Info("site configured",
			"name", site.Name,
			"domains", strings.Join(site.Domains, ","),
			"upstream", site.Upstream,
			"waf_enabled", site.WAFEnabled,
		)
	}

	e := engine.New(engine.EngineConfig{
		DevMode: *dev,
	})

	cfgMgr.OnChange(func(newCfg *config.Config) {
		slog.Info("config reloaded",
			"sites", len(newCfg.Sites),
			"rules", len(newCfg.Rules),
		)
	})

	proxyHandler := newWAFHandler(cfgMgr, e, *dev)

	proxySrv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *proxyPort),
		Handler:           proxyHandler,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	adminRouter := newAdminRouter(cfgMgr, e, *adminPort)
	adminSrv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *adminPort),
		Handler:           adminRouter,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		slog.Info("received shutdown signal", "signal", sig.String())
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	proxyErr := make(chan error, 1)
	adminErr := make(chan error, 1)

	go func() {
		defer wg.Done()
		slog.Info("proxy server listening", "port", *proxyPort)
		if err := proxySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("proxy server fatal error", "error", err)
			proxyErr <- err
		}
	}()

	go func() {
		defer wg.Done()
		slog.Info("admin server listening", "port", *adminPort)
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("admin server fatal error", "error", err)
			adminErr <- err
		}
	}()

	select {
	case err := <-proxyErr:
		slog.Error("proxy server exited", "error", err)
	case err := <-adminErr:
		slog.Error("admin server exited", "error", err)
	case <-ctx.Done():
	}

	slog.Info("draining connections...", "active", activeConns.Load())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		if err := proxySrv.Shutdown(shutdownCtx); err != nil {
			slog.Error("proxy server shutdown error", "error", err)
		} else {
			slog.Info("proxy server shut down")
		}
		close(done)
	}()

	go func() {
		<-done
		if err := adminSrv.Shutdown(shutdownCtx); err != nil {
			slog.Error("admin server shutdown error", "error", err)
		} else {
			slog.Info("admin server shut down")
		}
	}()

	select {
	case <-done:
	case <-shutdownCtx.Done():
		slog.Error("graceful shutdown timed out, forcing exit")
	}

	wg.Wait()

	slog.Info("fortresswaf stopped",
		"uptime", time.Since(startedAt).Round(time.Second),
		"total_requests", totalRequests.Load(),
		"blocked", blockedRequests.Load(),
		"allowed", allowedRequests.Load(),
	)
}

type wafHandler struct {
	mu       sync.RWMutex
	cfgMgr   *config.Manager
	engine   *engine.Engine
	dev      bool
	proxies  map[string]*httputil.ReverseProxy
}

func newWAFHandler(cfgMgr *config.Manager, e *engine.Engine, dev bool) http.Handler {
	h := &wafHandler{
		cfgMgr:  cfgMgr,
		engine:  e,
		dev:     dev,
		proxies: make(map[string]*httputil.ReverseProxy),
	}

	for _, site := range cfgMgr.Get().Sites {
		if err := h.buildProxy(&site); err != nil {
			slog.Error("failed to build proxy for site", "site", site.Name, "error", err)
		}
	}

	cfgMgr.OnChange(func(newCfg *config.Config) {
		h.mu.Lock()
		defer h.mu.Unlock()
		for _, site := range newCfg.Sites {
			if _, ok := h.proxies[site.Name]; !ok {
				if err := h.buildProxy(&site); err != nil {
					slog.Error("failed to build proxy for new site", "site", site.Name, "error", err)
				}
			}
		}
	})

	return h
}

func (h *wafHandler) buildProxy(site *config.SiteConfig) error {
	upstream := site.Upstream
	if site.Port > 0 && !strings.Contains(upstream, ":") {
		upstream = fmt.Sprintf("%s:%d", upstream, site.Port)
	}

	target, err := url.Parse(upstream)
	if err != nil {
		return fmt.Errorf("parse upstream %q: %w", upstream, err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.SetXForwarded()
			r.Out.Host = r.In.Host
		},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
			ResponseHeaderTimeout: 30 * time.Second,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("upstream error",
				"error", err,
				"host", r.Host,
				"path", r.URL.Path,
			)
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  "bad_gateway",
				"detail": "upstream unreachable",
			})
		},
	}

	h.proxies[site.Name] = proxy
	return nil
}

func (h *wafHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	totalRequests.Add(1)
	activeConns.Add(1)
	defer activeConns.Add(-1)

	host := strings.Split(r.Host, ":")[0]
	cfg := h.cfgMgr.Get()
	site := cfg.FindSiteByDomain(host)

	if site == nil {
		if len(cfg.Sites) > 0 {
			site = &cfg.Sites[0]
		} else {
			writeJSON(w, http.StatusBadGateway, map[string]interface{}{
				"error":  "no_site_configured",
				"detail": fmt.Sprintf("no site configured for host %q", host),
			})
			blockedRequests.Add(1)
			return
		}
	}

	if !site.WAFEnabled {
		h.forwardRequest(w, r, site)
		return
	}

	decision, err := h.engine.InspectRequest(r)
	if err != nil {
		slog.Error("engine inspection error", "error", err, "host", r.Host, "path", r.URL.Path)
		if h.dev {
			h.forwardRequest(w, r, site)
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error":  "waf_error",
			"detail": "inspection engine error",
		})
		return
	}

	switch decision.Action {
	case engine.ActionAllow:
		allowedRequests.Add(1)
		h.forwardRequest(w, r, site)

	case engine.ActionBlock:
		blockedRequests.Add(1)
		slog.Warn("request blocked",
			"host", r.Host,
			"path", r.URL.Path,
			"ip", r.RemoteAddr,
			"rule_id", decision.RuleID,
			"rule_name", decision.RuleName,
			"severity", decision.Severity,
			"evidence", decision.Evidence,
			"score", decision.Score,
		)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-FortressWAF-Action", "block")
		w.Header().Set("X-FortressWAF-Rule", decision.RuleID)
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"blocked":    true,
			"action":     "block",
			"rule_id":    decision.RuleID,
			"rule_name":  decision.RuleName,
			"severity":  decision.Severity,
			"evidence":   decision.Evidence,
			"request_id": r.Header.Get("X-Request-ID"),
		})

	case engine.ActionChallenge:
		challengedReqs.Add(1)
		slog.Info("challenge issued",
			"host", r.Host,
			"path", r.URL.Path,
			"ip", r.RemoteAddr,
			"rule_id", decision.RuleID,
		)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-FortressWAF-Action", "challenge")
		w.WriteHeader(http.StatusForbidden)
		w.Write(challengePage(r))

	case engine.ActionMonitor:
		monitoredReqs.Add(1)
		slog.Info("monitor: request passed through",
			"host", r.Host,
			"path", r.URL.Path,
			"ip", r.RemoteAddr,
			"rule_id", decision.RuleID,
			"severity", decision.Severity,
			"score", decision.Score,
		)
		w.Header().Set("X-FortressWAF-Monitored", "true")
		w.Header().Set("X-FortressWAF-Rule", decision.RuleID)
		h.forwardRequest(w, r, site)

	case engine.ActionRateLimit:
		rateLimitedReqs.Add(1)
		slog.Warn("rate limited",
			"host", r.Host,
			"path", r.URL.Path,
			"ip", r.RemoteAddr,
		)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(60*time.Second).Unix()))
		w.Header().Set("X-FortressWAF-Action", "rate_limit")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "rate_limited",
			"detail":     "too many requests",
			"retry_after": 60,
		})

	default:
		allowedRequests.Add(1)
		h.forwardRequest(w, r, site)
	}
}

func (h *wafHandler) forwardRequest(w http.ResponseWriter, r *http.Request, site *config.SiteConfig) {
	h.mu.RLock()
	proxy, ok := h.proxies[site.Name]
	h.mu.RUnlock()

	if !ok {
		if err := h.buildProxy(site); err != nil {
			slog.Error("failed to build proxy on-the-fly", "site", site.Name, "error", err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		h.mu.RLock()
		proxy = h.proxies[site.Name]
		h.mu.RUnlock()
	}

	proxy.ServeHTTP(w, r)
}

func newAdminRouter(cfgMgr *config.Manager, e *engine.Engine, adminPort int) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/health", handleHealth).Methods("GET")
	r.HandleFunc("/metrics", handleMetrics).Methods("GET")
	r.HandleFunc("/ready", handleReady(cfgMgr)).Methods("GET")
	r.HandleFunc("/live", handleLive).Methods("GET")

	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", handleHealth).Methods("GET")
	api.HandleFunc("/status", handleStatus).Methods("GET")
	api.HandleFunc("/config", handleGetConfig(cfgMgr)).Methods("GET")
	api.HandleFunc("/reload", handleReload(cfgMgr)).Methods("POST")
	api.HandleFunc("/sites", handleListSites(cfgMgr)).Methods("GET")
	api.HandleFunc("/rules", handleListRules(cfgMgr)).Methods("GET")

	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"version":   Version,
		"commit":    Commit,
		"uptime":    time.Since(startedAt).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startedAt).Seconds()
	rps := float64(0)
	if uptime > 0 {
		rps = float64(totalRequests.Load()) / uptime
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintf(w, "# HELP fortresswaf_requests_total Total number of requests\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_requests_total counter\n")
	fmt.Fprintf(w, "fortresswaf_requests_total %d\n", totalRequests.Load())

	fmt.Fprintf(w, "# HELP fortresswaf_requests_allowed Total allowed requests\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_requests_allowed counter\n")
	fmt.Fprintf(w, "fortresswaf_requests_allowed %d\n", allowedRequests.Load())

	fmt.Fprintf(w, "# HELP fortresswaf_requests_blocked Total blocked requests\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_requests_blocked counter\n")
	fmt.Fprintf(w, "fortresswaf_requests_blocked %d\n", blockedRequests.Load())

	fmt.Fprintf(w, "# HELP fortresswaf_requests_challenged Total challenged requests\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_requests_challenged counter\n")
	fmt.Fprintf(w, "fortresswaf_requests_challenged %d\n", challengedReqs.Load())

	fmt.Fprintf(w, "# HELP fortresswaf_requests_rate_limited Total rate-limited requests\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_requests_rate_limited counter\n")
	fmt.Fprintf(w, "fortresswaf_requests_rate_limited %d\n", rateLimitedReqs.Load())

	fmt.Fprintf(w, "# HELP fortresswaf_requests_monitored Total monitored requests\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_requests_monitored counter\n")
	fmt.Fprintf(w, "fortresswaf_requests_monitored %d\n", monitoredReqs.Load())

	fmt.Fprintf(w, "# HELP fortresswaf_active_connections Current active connections\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_active_connections gauge\n")
	fmt.Fprintf(w, "fortresswaf_active_connections %d\n", activeConns.Load())

	fmt.Fprintf(w, "# HELP fortresswaf_uptime_seconds Uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_uptime_seconds gauge\n")
	fmt.Fprintf(w, "fortresswaf_uptime_seconds %f\n", uptime)

	fmt.Fprintf(w, "# HELP fortresswaf_requests_per_second Current requests per second\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_requests_per_second gauge\n")
	fmt.Fprintf(w, "fortresswaf_requests_per_second %f\n", rps)

	fmt.Fprintf(w, "# HELP fortresswaf_version_info FortressWAF version info\n")
	fmt.Fprintf(w, "# TYPE fortresswaf_version_info gauge\n")
	fmt.Fprintf(w, "fortresswaf_version_info{version=%q,commit=%q} 1\n", Version, Commit)
}

func handleReady(cfgMgr *config.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(cfgMgr.Get().Sites) == 0 {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				"status": "not_ready",
				"reason": "no sites configured",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "ready",
		})
	}
}

func handleLive(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "alive",
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startedAt)
	rps := float64(0)
	secs := uptime.Seconds()
	if secs > 0 {
		rps = float64(totalRequests.Load()) / secs
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"version":           Version,
		"commit":            Commit,
		"build_date":        BuildDate,
		"uptime":            uptime.String(),
		"uptime_seconds":    int(secs),
		"requests_per_sec":  rps,
		"total_requests":    totalRequests.Load(),
		"blocked_requests":  blockedRequests.Load(),
		"allowed_requests":  allowedRequests.Load(),
		"active_connections": activeConns.Load(),
		"challenged":        challengedReqs.Load(),
		"rate_limited":      rateLimitedReqs.Load(),
		"monitored":         monitoredReqs.Load(),
	})
}

func handleGetConfig(cfgMgr *config.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgMgr.Get()
		resp := map[string]interface{}{
			"sites_count": len(cfg.Sites),
			"rules_count": len(cfg.Rules),
			"ml_enabled":  cfg.ML.Enabled,
			"redis_enabled": cfg.Redis.Enabled,
			"admin_port":  cfg.Admin.Port,
			"sites":       make([]map[string]interface{}, 0, len(cfg.Sites)),
		}
		for _, s := range cfg.Sites {
			resp["sites"] = append(resp["sites"].([]map[string]interface{}), map[string]interface{}{
				"name":         s.Name,
				"domains":      s.Domains,
				"upstream":     s.Upstream,
				"waf_enabled":  s.WAFEnabled,
			})
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleReload(cfgMgr *config.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := cfgMgr.Reload(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error":  "reload_failed",
				"detail": err.Error(),
			})
			return
		}
		cfg := cfgMgr.Get()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":      "reloaded",
			"sites":       len(cfg.Sites),
			"rules":       len(cfg.Rules),
		})
	}
}

func handleListSites(cfgMgr *config.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgMgr.Get()
		sites := make([]map[string]interface{}, 0, len(cfg.Sites))
		for _, s := range cfg.Sites {
			sites = append(sites, map[string]interface{}{
				"name":        s.Name,
				"domains":     s.Domains,
				"upstream":    s.Upstream,
				"port":        s.Port,
				"tls":         s.TLS,
				"waf_enabled": s.WAFEnabled,
			})
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"sites": sites,
			"count": len(sites),
		})
	}
}

func handleListRules(cfgMgr *config.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgMgr.Get()
		rules := make([]map[string]interface{}, 0, len(cfg.Rules))
		for _, r := range cfg.Rules {
			rules = append(rules, map[string]interface{}{
				"id":          r.ID,
				"name":        r.Name,
				"description": r.Description,
				"enabled":     r.Enabled,
				"severity":    r.Severity,
				"action":      r.Action,
				"tags":        r.Tags,
			})
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"rules": rules,
			"count": len(rules),
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func challengePage(r *http.Request) []byte {
	challengeToken := fmt.Sprintf("%x", time.Now().UnixNano())
	return []byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Security Challenge</title></head>
<body style="display:flex;justify-content:center;align-items:center;height:100vh;margin:0;font-family:monospace;background:#1a1a2e;color:#e0e0e0;">
<div style="text-align:center;padding:40px;border:1px solid #333;border-radius:8px;background:#16213e;">
<h2>Security Verification</h2>
<p>Please wait while we verify your browser...</p>
<form id="cf-form" action="/__challenge" method="POST">
<input type="hidden" name="challenge_token" value="%s">
<input type="hidden" name="original_path" value="%s">
</form>
<script>
setTimeout(function(){
  var elapsed = (Date.now() / 1000 | 0) - %d;
  if(elapsed > 2) {
    document.getElementById("cf-form").submit();
  }
}, 2500);
</script>
<noscript><p>JavaScript is required. Please enable JavaScript and try again.</p>
<button type="submit" form="cf-form">Continue</button></noscript>
</div>
</body>
</html>`, challengeToken, r.URL.Path, time.Now().Unix()))
}


