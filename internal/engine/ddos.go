package engine

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

type SlidingWindowCounter struct {
	mu         sync.Mutex
	timestamps []time.Time
	window     time.Duration
	maxCount   int
}

type EndpointCounter struct {
	mu       sync.Mutex
	counters map[string]*SlidingWindowCounter
}

type DDoSProtection struct {
	mu              sync.RWMutex
	devMode         bool
	globalRate      int
	perIPRate       int
	perEndpointRate int
	perSessionRate  int
	burstAllowance  int
	windowSize      time.Duration

	ipCounters       map[string]*SlidingWindowCounter
	sessionCounters  map[string]*SlidingWindowCounter
	endpointCounters map[string]*SlidingWindowCounter
	slowLorisTimers  map[string]time.Time
	slowPOSTTimers   map[string]time.Time

	globalMu      sync.RWMutex
	globalCounter *SlidingWindowCounter
}

func NewDDoSProtection(devMode bool) *DDoSProtection {
	return &DDoSProtection{
		devMode:          devMode,
		globalRate:       10000,
		perIPRate:        100,
		perEndpointRate:  500,
		perSessionRate:   200,
		burstAllowance:   20,
		windowSize:       time.Second,
		ipCounters:       make(map[string]*SlidingWindowCounter),
		sessionCounters:  make(map[string]*SlidingWindowCounter),
		endpointCounters: make(map[string]*SlidingWindowCounter),
		slowLorisTimers:  make(map[string]time.Time),
		slowPOSTTimers:   make(map[string]time.Time),
		globalCounter: &SlidingWindowCounter{
			timestamps: make([]time.Time, 0, 10020),
			window:     time.Second,
			maxCount:   10020,
		},
	}
}

func (d *DDoSProtection) Name() string { return "ddos_protection" }

func (d *DDoSProtection) Inspect(ctx *RequestContext) (*Decision, error) {
	if dec := d.detectHTTPFlood(ctx); dec != nil {
		return dec, nil
	}

	if dec := d.detectSlowloris(ctx); dec != nil {
		return dec, nil
	}

	if dec := d.detectSlowPOST(ctx); dec != nil {
		return dec, nil
	}

	if dec := d.detectCacheBusting(ctx); dec != nil {
		return dec, nil
	}

	return nil, nil
}

func (d *DDoSProtection) getOrCreateCounter(counters map[string]*SlidingWindowCounter, key string) *SlidingWindowCounter {
	if c, ok := counters[key]; ok {
		return c
	}
	c := &SlidingWindowCounter{
		timestamps: make([]time.Time, 0, d.perIPRate+d.burstAllowance),
		window:     d.windowSize,
		maxCount:   d.perIPRate + d.burstAllowance,
	}
	counters[key] = c
	return c
}

func (d *DDoSProtection) checkRate(counter *SlidingWindowCounter, limit int) bool {
	counter.mu.Lock()
	defer counter.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-d.windowSize)

	var valid []time.Time
	for _, t := range counter.timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	counter.timestamps = valid

	if len(valid) >= limit {
		return false
	}

	counter.timestamps = append(counter.timestamps, now)
	return true
}

func (d *DDoSProtection) detectHTTPFlood(ctx *RequestContext) *Decision {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if dec := d.checkGlobalRate(ctx); dec != nil {
		return dec
	}

	ipCounter := d.getOrCreateCounter(d.ipCounters, ctx.RealIP)
	if !d.checkRate(ipCounter, d.perIPRate) {
		return &Decision{
			Action:   ActionRateLimit,
			RuleID:   "DDoS001",
			RuleName: "HTTP Flood - IP",
			Severity: "high",
			Score:    65,
			Evidence: fmt.Sprintf("IP %s exceeded rate limit: %d req/s", ctx.RealIP, d.perIPRate),
		}
	}

	epCounter := d.getOrCreateCounter(d.endpointCounters, ctx.Path)
	if !d.checkRate(epCounter, d.perEndpointRate) {
		return &Decision{
			Action:   ActionRateLimit,
			RuleID:   "DDoS002",
			RuleName: "HTTP Flood - Endpoint",
			Severity: "high",
			Score:    60,
			Evidence: fmt.Sprintf("Endpoint %s exceeded rate: %d req/s", ctx.Path, d.perEndpointRate),
		}
	}

	if ctx.SessionID != "" {
		sessionCounter := d.getOrCreateCounter(d.sessionCounters, ctx.SessionID)
		if !d.checkRate(sessionCounter, d.perSessionRate) {
			return &Decision{
				Action:   ActionRateLimit,
				RuleID:   "DDoS003",
				RuleName: "HTTP Flood - Session",
				Severity: "medium",
				Score:    50,
				Evidence: fmt.Sprintf("Session %s exceeded rate: %d req/s", ctx.SessionID, d.perSessionRate),
			}
		}
	}

	return nil
}

func (d *DDoSProtection) checkGlobalRate(ctx *RequestContext) *Decision {
	d.globalMu.RLock()
	count := len(d.globalCounter.timestamps)
	d.globalMu.RUnlock()

	if int(count) > d.globalRate {
		return &Decision{
			Action:   ActionRateLimit,
			RuleID:   "DDoS000",
			RuleName: "HTTP Flood - Global",
			Severity: "critical",
			Score:    90,
			Evidence: fmt.Sprintf("global rate limit exceeded: %d req/s", count),
		}
	}
	return nil
}

func (d *DDoSProtection) detectSlowloris(ctx *RequestContext) *Decision {
	if ctx.Headers["Expect"] != "" {
		d.mu.Lock()
		d.slowLorisTimers[ctx.RealIP] = time.Now()
		d.mu.Unlock()
		return nil
	}

	contentLength := ctx.Request.Header.Get("Content-Length")
	if contentLength == "" && ctx.Method == http.MethodPost {
		d.mu.Lock()
		start, exists := d.slowLorisTimers[ctx.RealIP]
		if !exists {
			d.slowLorisTimers[ctx.RealIP] = time.Now()
			d.mu.Unlock()
			return nil
		}
		elapsed := time.Since(start)
		d.mu.Unlock()

		if elapsed > 30*time.Second {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "DDoS004",
				RuleName: "Slowloris Attack",
				Severity: "high",
				Score:    85,
				Evidence: fmt.Sprintf("slowloris detected from %s, elapsed: %v", ctx.RealIP, elapsed),
			}
		}
	}

	return nil
}

func (d *DDoSProtection) detectSlowPOST(ctx *RequestContext) *Decision {
	if ctx.Method != http.MethodPost || ctx.Body == nil {
		return nil
	}

	contentLengthStr := ctx.Request.Header.Get("Content-Length")
	if contentLengthStr == "" {
		return nil
	}

	var contentLength int64
	fmt.Sscanf(contentLengthStr, "%d", &contentLength)

	if contentLength > 1024*1024 {
		d.mu.Lock()
		start, exists := d.slowPOSTTimers[ctx.RealIP]
		if !exists {
			d.slowPOSTTimers[ctx.RealIP] = time.Now()
			d.mu.Unlock()
			return nil
		}
		received := len(ctx.Body)
		elapsed := time.Since(start)
		d.mu.Unlock()

		rate := float64(received) / elapsed.Seconds()
		if elapsed > 10*time.Second && rate < 1024 {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "DDoS005",
				RuleName: "Slow POST Attack",
				Severity: "high",
				Score:    80,
				Evidence: fmt.Sprintf("slow POST from %s, rate: %.0f b/s, elapsed: %v", ctx.RealIP, rate, elapsed),
			}
		}
	}

	return nil
}

func (d *DDoSProtection) detectCacheBusting(ctx *RequestContext) *Decision {
	queryKeys := make([]string, 0, len(ctx.QueryParams))
	for k := range ctx.QueryParams {
		queryKeys = append(queryKeys, k)
	}

	paramCount := len(queryKeys)
	randomPatterns := 0
	for k, v := range ctx.QueryParams {
		lowerK := strings.ToLower(k)
		if strings.Contains(lowerK, "cache") || strings.Contains(lowerK, "rand") ||
			strings.Contains(lowerK, "t") || strings.Contains(lowerK, "_") && len(k) <= 3 {
			for _, val := range v {
				if len(val) >= 8 && len(val) <= 32 {
					isHex := true
					for _, c := range val {
						if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
							isHex = false
							break
						}
					}
					if isHex || len(val) >= 16 {
						randomPatterns++
					}
				}
			}
		}
	}

	if paramCount >= 5 && randomPatterns >= paramCount/2 {
		return &Decision{
			Action:   ActionMonitor,
			RuleID:   "DDoS006",
			RuleName: "Cache Busting Attack",
			Severity: "medium",
			Score:    40,
			Evidence: fmt.Sprintf("cache busting detected, params: %d, random: %d", paramCount, randomPatterns),
		}
	}

	return nil
}

func (d *DDoSProtection) GetAdaptiveRate(ip string, currentRate int) int {
	d.mu.RLock()
	counter, exists := d.ipCounters[ip]
	d.mu.RUnlock()

	if !exists {
		return currentRate
	}

	counter.mu.Lock()
	count := len(counter.timestamps)
	counter.mu.Unlock()

	if count > currentRate*2 {
		return int(math.Max(float64(currentRate)*0.5, 10))
	}

	if count > currentRate {
		return int(math.Max(float64(currentRate)*0.8, 10))
	}

	return currentRate
}

func (d *DDoSProtection) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			d.mu.Lock()
			now := time.Now()
			for ip, counter := range d.ipCounters {
				counter.mu.Lock()
				if len(counter.timestamps) == 0 || now.Sub(counter.timestamps[len(counter.timestamps)-1]) > 10*time.Minute {
					delete(d.ipCounters, ip)
				}
				counter.mu.Unlock()
			}
			for ip, t := range d.slowLorisTimers {
				if now.Sub(t) > 5*time.Minute {
					delete(d.slowLorisTimers, ip)
				}
			}
			for ip, t := range d.slowPOSTTimers {
				if now.Sub(t) > 5*time.Minute {
					delete(d.slowPOSTTimers, ip)
				}
			}
			d.mu.Unlock()
		}
	}()
}

var _ = slog.Debug
