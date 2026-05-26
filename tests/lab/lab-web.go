package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

//go:embed lab.html
var labHTML string

var (
	totalAllowed atomic.Int64
	totalBlocked atomic.Int64
)

var attacks = []struct {
	Group     string
	Name      string
	Method    string
	Path      string
	Body      string
	DirectURL string
	ProxyURL  string
}{
	// Legitimate
	{"Legitimate", "Health Check", "GET", "/health", "", "/health", "/health"},
	{"Legitimate", "Home Page", "GET", "/", "", "/", "/"},
	{"Legitimate", "Normal Login", "POST", "/login", "username=admin&password=test", "/login", "/login"},
	{"Legitimate", "Search Query", "GET", "/search?q=laptop", "", "/search?q=laptop", "/search?q=laptop"},
	{"Legitimate", "Add Comment", "POST", "/comment", "author=User&message=Nice+site", "/comment", "/comment"},
	// SQL Injection
	{"SQL Injection", "OR 1=1 (username)", "POST", "/login", "username=' OR '1'='1&password=test", "/login", "/login"},
	{"SQL Injection", "OR 1=1 (password)", "POST", "/login", "username=admin&password=' OR '1'='1", "/login", "/login"},
	{"SQL Injection", "UNION SELECT", "GET", "/search?q=' UNION SELECT * FROM users", "", "/search?q=' UNION SELECT * FROM users", "/search?q=' UNION SELECT * FROM users"},
	{"SQL Injection", "DROP TABLE", "GET", "/search?q=';DROP TABLE users", "", "/search?q=';DROP TABLE users", "/search?q=';DROP TABLE users"},
	{"SQL Injection", "Comment (#)", "GET", "/search?q=admin'--", "", "/search?q=admin'--", "/search?q=admin'--"},
	// XSS
	{"XSS", "<script> in search", "GET", "/search?q=<script>alert(1)</script>", "", "/search?q=<script>alert(1)</script>", "/search?q=<script>alert(1)</script>"},
	{"XSS", "<script> in comment", "POST", "/comment", "author=XSS&message=<script>alert('x')</script>", "/comment", "/comment"},
	{"XSS", "onerror payload", "GET", "/search?q=<img src=x onerror=alert(1)>", "", "/search?q=<img src=x onerror=alert(1)>", "/search?q=<img src=x onerror=alert(1)>"},
	{"XSS", "prompt() attack", "POST", "/comment", "author=XSS&message=prompt('x')", "/comment", "/comment"},
	// RCE
	{"RCE/LFI", "; cat /etc/passwd", "POST", "/ping", "addr=8.8.8.8; cat /etc/passwd", "/ping", "/ping"},
	{"RCE/LFI", "; ls -la", "POST", "/ping", "addr=8.8.8.8; ls -la", "/ping", "/ping"},
	{"RCE/LFI", "LFI ../../../etc/passwd", "GET", "/file?name=../../../etc/passwd", "", "/file?name=../../../etc/passwd", "/file?name=../../../etc/passwd"},
	{"RCE/LFI", "LFI /etc/shadow", "GET", "/file?name=../../etc/shadow", "", "/file?name=../../etc/shadow", "/file?name=../../etc/shadow"},
	// Scanner
	{"Scanner", "sqlmap User-Agent", "GET", "/", "", "/", "/"},
	{"Scanner", "nikto User-Agent", "GET", "/", "", "/", "/"},
	{"Scanner", "nmap User-Agent", "GET", "/", "", "/", "/"},
	{"Scanner", "curl (default UA)", "GET", "/", "", "/", "/"},
}

var scannerUAs = map[string]string{
	"sqlmap User-Agent":   "sqlmap/1.8",
	"nikto User-Agent":    "nikto/2.5",
	"nmap User-Agent":     "nmap script",
	"curl (default UA)":   "curl/8.0",
}

type attackResult struct {
	Name       string `json:"name"`
	Group      string `json:"group"`
	DirectURL  string `json:"directUrl"`
	ProxyURL   string `json:"proxyUrl"`
	Direct     *resp  `json:"direct,omitempty"`
	Proxy      *resp  `json:"proxy,omitempty"`
	DirectErr  string `json:"directErr,omitempty"`
	ProxyErr   string `json:"proxyErr,omitempty"`
}

type resp struct {
	StatusCode int    `json:"statusCode"`
	Body       string `json:"body"`
	Header     string `json:"header"`
	Duration   string `json:"duration"`
}

func main() {
	vulnHost := getEnv("VULN_HOST", "http://localhost:9099")
	proxyHost := getEnv("PROXY_HOST", "http://localhost:8081")
	port := getEnv("LAB_PORT", "5000")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(labHTML))
	})

	mux.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", 405)
			return
		}
		var a struct {
			Index int `json:"idx"`
		}
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			idx := r.URL.Query().Get("idx")
			if idx == "" {
				http.Error(w, "missing idx", 400)
				return
			}
			a.Index = atoi(idx)
		}
		if a.Index < 0 || a.Index >= len(attacks) {
			http.Error(w, "invalid index", 400)
			return
		}
		att := attacks[a.Index]
		res := attackResult{
			Name:      att.Name,
			Group:     att.Group,
			DirectURL: vulnHost + att.DirectURL,
			ProxyURL:  proxyHost + att.ProxyURL,
		}

		ua := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
		if override, ok := scannerUAs[att.Name]; ok {
			ua = override
		}

		d, derr := doReq(vulnHost, att.Method, att.Path, att.Body, ua)
		p, perr := doReq(proxyHost, att.Method, att.Path, att.Body, ua)
		res.Direct = d
		res.Proxy = p
		if derr != nil {
			res.DirectErr = derr.Error()
		}
		if perr != nil {
			res.ProxyErr = perr.Error()
		}

		if p != nil && p.StatusCode == 403 {
			totalBlocked.Add(1)
		} else if p != nil && p.StatusCode < 400 {
			totalAllowed.Add(1)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			totalAllowed.Store(0)
			totalBlocked.Store(0)
			w.WriteHeader(204)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{
			"allowed": totalAllowed.Load(),
			"blocked": totalBlocked.Load(),
		})
	})

	mux.HandleFunc("/api/attacks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		type attackInfo struct {
			Index int    `json:"index"`
			Group string `json:"group"`
			Name  string `json:"name"`
		}
		var list []attackInfo
		for i, a := range attacks {
			list = append(list, attackInfo{Index: i, Group: a.Group, Name: a.Name})
		}
		json.NewEncoder(w).Encode(list)
	})

	fmt.Printf("🌐 Lab Web Dashboard: http://localhost:%s\n", port)
	fmt.Printf("   VulnApp (direct):  %s\n", vulnHost)
	fmt.Printf("   FortressWAF:       %s\n", proxyHost)
	http.ListenAndServe(":"+port, mux)
}

func doReq(base, method, path, body, ua string) (*resp, error) {
	start := time.Now()
	var req *http.Request
	var err error
	url := base + path
	if method == "POST" && body != "" {
		req, err = http.NewRequest(method, url, bytes.NewBufferString(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, url, err)
	}
	req.Header.Set("User-Agent", ua)

	client := &http.Client{Timeout: 5 * time.Second}
	respRaw, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer respRaw.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(respRaw.Body, 4096))
	dur := time.Since(start)

	headerStr := ""
	for k, v := range respRaw.Header {
		if len(v) > 0 {
			headerStr += fmt.Sprintf("%s: %s\n", k, v[0])
		}
	}

	return &resp{
		StatusCode: respRaw.StatusCode,
		Body:       string(respBody),
		Header:     headerStr,
		Duration:   fmt.Sprintf("%.0fms", float64(dur.Microseconds())/1000),
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}
