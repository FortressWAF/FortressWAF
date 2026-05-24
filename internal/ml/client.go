package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type InspectionResult struct {
	ThreatScore  float64            `json:"threat_score"`
	IsMalicious  bool               `json:"is_malicious"`
	Category     string             `json:"category"`
	Confidence   float64            `json:"confidence"`
	Features     map[string]float64 `json:"features"`
	BotScore     float64            `json:"bot_score"`
	AnomalyScore float64            `json:"anomaly_score"`
}

type ClassificationResult struct {
	Label         string             `json:"label"`
	Score         float64            `json:"score"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type FingerprintResult struct {
	Fingerprint string  `json:"fingerprint"`
	Confidence  float64 `json:"confidence"`
	IsKnown     bool    `json:"is_known"`
}

type BotScoreResult struct {
	Score   float64 `json:"score"`
	IsBot   bool    `json:"is_bot"`
	BotType string  `json:"bot_type"`
}

type Client struct {
	mu         sync.RWMutex
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
	fallback   string
	available  bool
	cbState    circuitBreaker
}

type circuitBreaker struct {
	mu        sync.Mutex
	failures  int
	lastError time.Time
	threshold int
	cooldown  time.Duration
	open      bool
}

type InspectRequest struct {
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Headers     map[string]string   `json:"headers"`
	Body        string              `json:"body"`
	ContentType string              `json:"content_type"`
	RealIP      string              `json:"real_ip"`
	UserAgent   string              `json:"user_agent"`
	QueryParams map[string][]string `json:"query_params"`
}

func NewClient(endpoint string, timeoutSec, maxRetries int, fallbackMode string) *Client {
	return &Client{
		baseURL: endpoint,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
		timeout:    time.Duration(timeoutSec) * time.Second,
		maxRetries: maxRetries,
		fallback:   fallbackMode,
		available:  true,
		cbState: circuitBreaker{
			threshold: 5,
			cooldown:  30 * time.Second,
		},
	}
}

func (c *Client) Name() string { return "ml" }

func (c *Client) Inspect(ctx context.Context, req *InspectRequest) (*InspectionResult, error) {
	if !c.isAvailable() {
		return c.fallbackResult(), fmt.Errorf("ml client unavailable")
	}

	var lastErr error
	for i := 0; i <= c.maxRetries; i++ {
		result, err := c.callInspect(ctx, req)
		if err == nil {
			c.recordSuccess()
			return result, nil
		}

		lastErr = err
		c.recordFailure()

		if i < c.maxRetries {
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
		}
	}

	return c.fallbackResult(), fmt.Errorf("ml inspect failed after %d retries: %w", c.maxRetries, lastErr)
}

func (c *Client) callInspect(ctx context.Context, req *InspectRequest) (*InspectionResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/inspect", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ml service returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result InspectionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) Classify(ctx context.Context, data interface{}) (*ClassificationResult, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/classify", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ClassificationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) Fingerprint(ctx context.Context, data interface{}) (*FingerprintResult, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/fingerprint", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result FingerprintResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) BotScore(ctx context.Context, data interface{}) (*BotScoreResult, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/bot-score", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result BotScoreResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) isAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.available {
		return false
	}

	c.cbState.mu.Lock()
	defer c.cbState.mu.Unlock()

	if c.cbState.open {
		if time.Since(c.cbState.lastError) > c.cbState.cooldown {
			c.cbState.open = false
			c.cbState.failures = 0
			return true
		}
		return false
	}

	return true
}

func (c *Client) recordSuccess() {
	c.cbState.mu.Lock()
	defer c.cbState.mu.Unlock()
	c.cbState.failures = 0
}

func (c *Client) recordFailure() {
	c.cbState.mu.Lock()
	defer c.cbState.mu.Unlock()
	c.cbState.failures++
	if c.cbState.failures >= c.cbState.threshold {
		c.cbState.open = true
		c.cbState.lastError = time.Now()
		slog.Warn("ml circuit breaker opened", "failures", c.cbState.failures)
	}
}

func (c *Client) fallbackResult() *InspectionResult {
	switch c.fallback {
	case "block":
		return &InspectionResult{ThreatScore: 100, IsMalicious: true, BotScore: 100}
	case "monitor":
		return &InspectionResult{ThreatScore: 50, IsMalicious: false, BotScore: 50}
	default:
		return &InspectionResult{ThreatScore: 0, IsMalicious: false, BotScore: 0}
	}
}

func (c *Client) SetAvailable(avail bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.available = avail
}

func (c *Client) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/v1/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

var _ = slog.Debug
