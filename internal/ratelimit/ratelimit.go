package ratelimit

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type Algorithm string

const (
	FixedWindow   Algorithm = "fixed_window"
	SlidingWindow Algorithm = "sliding_window"
	TokenBucket   Algorithm = "token_bucket"
	LeakyBucket   Algorithm = "leaky_bucket"
)

type Granularity string

const (
	PerIP       Granularity = "ip"
	PerUser     Granularity = "user"
	PerSession  Granularity = "session"
	PerAPIKey   Granularity = "api_key"
	PerEndpoint Granularity = "endpoint"
	PerGeo      Granularity = "geo"
)

type windowEntry struct {
	timestamp time.Time
	count     int
}

type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	refillRate float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu           sync.RWMutex
	algorithm    Algorithm
	defaultRate  int
	defaultBurst int
	windowSize   time.Duration
	cleanupTick  time.Duration

	fixedWindows   map[string]*windowEntry
	slidingWindows map[string][]time.Time
	tokenBuckets   map[string]*tokenBucket
	leakyBuckets   map[string]*leakyBucket

	perIPLimits       map[string]int
	perUserLimits     map[string]int
	perEndpointLimits map[string]int
	perGeoLimits      map[string]int
	perAPIKeyLimits   map[string]int
	priorityQueues    map[string]bool
}

type leakyBucket struct {
	mu       sync.Mutex
	queue    []time.Time
	capacity int
	leakRate time.Duration
	lastLeak time.Time
}

func NewRateLimiter(algorithm Algorithm, defaultRate, defaultBurst int) *RateLimiter {
	rl := &RateLimiter{
		algorithm:         algorithm,
		defaultRate:       defaultRate,
		defaultBurst:      defaultBurst,
		windowSize:        time.Second,
		cleanupTick:       5 * time.Minute,
		fixedWindows:      make(map[string]*windowEntry),
		slidingWindows:    make(map[string][]time.Time),
		tokenBuckets:      make(map[string]*tokenBucket),
		leakyBuckets:      make(map[string]*leakyBucket),
		perIPLimits:       make(map[string]int),
		perUserLimits:     make(map[string]int),
		perEndpointLimits: make(map[string]int),
		perGeoLimits:      make(map[string]int),
		perAPIKeyLimits:   make(map[string]int),
		priorityQueues:    make(map[string]bool),
	}

	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Name() string { return "rate_limiter" }

func (rl *RateLimiter) Inspect(key string, granularity Granularity) (bool, *Decision) {
	rate := rl.getRate(key, granularity)
	burst := rl.getBurst(key)

	switch rl.algorithm {
	case FixedWindow:
		return rl.checkFixedWindow(key, rate, burst)
	case SlidingWindow:
		return rl.checkSlidingWindow(key, rate, burst)
	case TokenBucket:
		return rl.checkTokenBucket(key, rate, burst)
	case LeakyBucket:
		return rl.checkLeakyBucket(key, rate, burst)
	default:
		return rl.checkFixedWindow(key, rate, burst)
	}
}

type Decision struct {
	Allowed    bool
	RetryAfter int
	Remaining  int
	Limit      int
	ResetAt    time.Time
}

func (rl *RateLimiter) checkFixedWindow(key string, rate, burst int) (bool, *Decision) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowKey := fmt.Sprintf("%s:%d", key, now.Unix())

	entry, exists := rl.fixedWindows[windowKey]
	if !exists {
		rl.fixedWindows[windowKey] = &windowEntry{timestamp: now, count: 1}
		return true, &Decision{Allowed: true, Remaining: rate + burst - 1, Limit: rate + burst, ResetAt: now.Add(time.Second)}
	}

	if now.Sub(entry.timestamp) > time.Second {
		entry.timestamp = now
		entry.count = 1
		return true, &Decision{Allowed: true, Remaining: rate + burst - 1, Limit: rate + burst, ResetAt: now.Add(time.Second)}
	}

	entry.count++
	limit := rate + burst

	if entry.count > limit {
		return false, &Decision{
			Allowed: false, RetryAfter: 1, Remaining: 0,
			Limit: limit, ResetAt: now.Add(time.Second),
		}
	}

	return true, &Decision{
		Allowed: true, Remaining: limit - entry.count,
		Limit: limit, ResetAt: now.Add(time.Second),
	}
}

func (rl *RateLimiter) checkSlidingWindow(key string, rate, burst int) (bool, *Decision) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	entries := rl.slidingWindows[key]
	var valid []time.Time
	for _, t := range entries {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	count := len(valid)
	limit := rate + burst

	if count >= limit {
		rl.slidingWindows[key] = valid
		oldest := valid[0]
		retryAfter := int(rl.windowSize - now.Sub(oldest))
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, &Decision{
			Allowed: false, RetryAfter: retryAfter, Remaining: 0,
			Limit: limit, ResetAt: oldest.Add(rl.windowSize),
		}
	}

	valid = append(valid, now)
	rl.slidingWindows[key] = valid

	return true, &Decision{
		Allowed: true, Remaining: limit - count - 1,
		Limit: limit, ResetAt: now.Add(rl.windowSize),
	}
}

func (rl *RateLimiter) checkTokenBucket(key string, rate, burst int) (bool, *Decision) {
	rl.mu.Lock()

	bucket, exists := rl.tokenBuckets[key]
	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(burst),
			capacity:   float64(burst),
			refillRate: float64(rate),
			lastRefill: time.Now(),
		}
		rl.tokenBuckets[key] = bucket
	}
	bucket.mu.Lock()
	rl.mu.Unlock()

	defer bucket.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = math.Min(bucket.capacity, bucket.tokens+elapsed*bucket.refillRate)
	bucket.lastRefill = now

	if bucket.tokens < 1 {
		refillTime := time.Duration(1.0/bucket.refillRate*1000) * time.Millisecond
		return false, &Decision{
			Allowed: false, RetryAfter: int(refillTime.Milliseconds()),
			Remaining: 0, Limit: rate + burst,
			ResetAt: now.Add(refillTime),
		}
	}

	bucket.tokens--
	return true, &Decision{
		Allowed: true, Remaining: int(bucket.tokens),
		Limit: rate + burst, ResetAt: now.Add(time.Second),
	}
}

func (rl *RateLimiter) checkLeakyBucket(key string, rate, burst int) (bool, *Decision) {
	rl.mu.Lock()

	bucket, exists := rl.leakyBuckets[key]
	if !exists {
		bucket = &leakyBucket{
			queue:    make([]time.Time, 0, burst),
			capacity: burst,
			leakRate: time.Second / time.Duration(rate),
			lastLeak: time.Now(),
		}
		rl.leakyBuckets[key] = bucket
	}
	bucket.mu.Lock()
	rl.mu.Unlock()

	defer bucket.mu.Unlock()

	now := time.Now()
	leakCount := int(now.Sub(bucket.lastLeak) / bucket.leakRate)
	if leakCount > 0 {
		if leakCount >= len(bucket.queue) {
			bucket.queue = bucket.queue[:0]
		} else {
			bucket.queue = bucket.queue[leakCount:]
		}
		bucket.lastLeak = now
	}

	if len(bucket.queue) >= bucket.capacity {
		oldest := bucket.queue[0]
		retryAfter := int(bucket.leakRate*time.Duration(len(bucket.queue)-bucket.capacity+1)) / int(time.Millisecond)
		return false, &Decision{
			Allowed: false, RetryAfter: retryAfter, Remaining: 0,
			Limit: burst, ResetAt: oldest.Add(bucket.leakRate * time.Duration(burst)),
		}
	}

	bucket.queue = append(bucket.queue, now)
	return true, &Decision{
		Allowed: true, Remaining: bucket.capacity - len(bucket.queue),
		Limit: burst, ResetAt: now.Add(bucket.leakRate * time.Duration(bucket.capacity-len(bucket.queue))),
	}
}

func (rl *RateLimiter) getRate(key string, granularity Granularity) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	switch granularity {
	case PerIP:
		if rate, ok := rl.perIPLimits[key]; ok {
			return rate
		}
	case PerUser:
		if rate, ok := rl.perUserLimits[key]; ok {
			return rate
		}
	case PerEndpoint:
		if rate, ok := rl.perEndpointLimits[key]; ok {
			return rate
		}
	case PerGeo:
		if rate, ok := rl.perGeoLimits[key]; ok {
			return rate
		}
	case PerAPIKey:
		if rate, ok := rl.perAPIKeyLimits[key]; ok {
			return rate
		}
	}

	return rl.defaultRate
}

func (rl *RateLimiter) getBurst(key string) int {
	if rl.priorityQueues[key] {
		return rl.defaultBurst * 2
	}
	return rl.defaultBurst
}

func (rl *RateLimiter) SetRateLimit(granularity Granularity, key string, rate int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	switch granularity {
	case PerIP:
		rl.perIPLimits[key] = rate
	case PerUser:
		rl.perUserLimits[key] = rate
	case PerEndpoint:
		rl.perEndpointLimits[key] = rate
	case PerGeo:
		rl.perGeoLimits[key] = rate
	case PerAPIKey:
		rl.perAPIKeyLimits[key] = rate
	}
}

func (rl *RateLimiter) SetPriority(key string, premium bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.priorityQueues[key] = premium
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupTick)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()

		for key, entry := range rl.fixedWindows {
			if now.Sub(entry.timestamp) > rl.windowSize*2 {
				delete(rl.fixedWindows, key)
			}
		}

		for key, entries := range rl.slidingWindows {
			cutoff := now.Add(-rl.windowSize * 2)
			var valid []time.Time
			for _, t := range entries {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.slidingWindows, key)
			} else {
				rl.slidingWindows[key] = valid
			}
		}

		for key, bucket := range rl.tokenBuckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastRefill) > rl.cleanupTick*2 {
				delete(rl.tokenBuckets, key)
			}
			bucket.mu.Unlock()
		}

		for key, bucket := range rl.leakyBuckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastLeak) > rl.cleanupTick*2 && len(bucket.queue) == 0 {
				delete(rl.leakyBuckets, key)
			}
			bucket.mu.Unlock()
		}

		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allowed(key string, granularity Granularity) (bool, *Decision) {
	return rl.Inspect(key, granularity)
}
