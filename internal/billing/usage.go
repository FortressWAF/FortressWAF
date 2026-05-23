package billing

import (
	"fmt"
	"sync"
	"time"
)

type UsageTracker struct {
	mu       sync.RWMutex
	counters map[string]*TenantUsage
	limits   map[string]int64
}

type TenantUsage struct {
	RequestsToday     int64
	RequestsThisMonth int64
	BlocksToday       int64
	LastReset         time.Time
	MonthStart        time.Time
}

func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		counters: make(map[string]*TenantUsage),
		limits:   make(map[string]int64),
	}
}

func (u *UsageTracker) SetLimit(tenantID string, limit int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.limits[tenantID] = limit
}

func (u *UsageTracker) RecordRequest(tenantID string, blocked bool) {
	u.mu.Lock()
	defer u.mu.Unlock()

	now := time.Now()
	usage, ok := u.counters[tenantID]
	if !ok {
		usage = &TenantUsage{
			LastReset:  now,
			MonthStart: now,
		}
		u.counters[tenantID] = usage
	}

	if now.Sub(usage.LastReset) > 24*time.Hour {
		usage.RequestsToday = 0
		usage.BlocksToday = 0
		usage.LastReset = now
	}

	if now.Sub(usage.MonthStart) > 30*24*time.Hour {
		usage.RequestsThisMonth = 0
		usage.MonthStart = now
	}

	usage.RequestsToday++
	usage.RequestsThisMonth++

	if blocked {
		usage.BlocksToday++
	}
}

func (u *UsageTracker) GetUsage(tenantID string) (today, thisMonth, limit int64, blockedToday int64) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if usage, ok := u.counters[tenantID]; ok {
		today = usage.RequestsToday
		thisMonth = usage.RequestsThisMonth
		blockedToday = usage.BlocksToday
	}

	if l, ok := u.limits[tenantID]; ok {
		limit = l
	}

	return
}

func (u *UsageTracker) CheckLimit(tenantID string) error {
	u.mu.RLock()
	limit := u.limits[tenantID]
	usage := u.counters[tenantID]
	u.mu.RUnlock()

	if limit <= 0 {
		return nil
	}

	if usage != nil && usage.RequestsToday >= limit {
		return fmt.Errorf("daily request limit exceeded: %d/%d", usage.RequestsToday, limit)
	}

	return nil
}

func (u *UsageTracker) ResetDaily(tenantID string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if usage, ok := u.counters[tenantID]; ok {
		usage.RequestsToday = 0
		usage.BlocksToday = 0
		usage.LastReset = time.Now()
	}
}

func (u *UsageTracker) ResetMonthly(tenantID string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if usage, ok := u.counters[tenantID]; ok {
		usage.RequestsThisMonth = 0
		usage.MonthStart = time.Now()
	}
}

func (u *UsageTracker) GetAllUsage() map[string]TenantUsage {
	u.mu.RLock()
	defer u.mu.RUnlock()

	result := make(map[string]TenantUsage)
	for k, v := range u.counters {
		result[k] = *v
	}
	return result
}

func (u *UsageTracker) RemoveTenant(tenantID string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.counters, tenantID)
	delete(u.limits, tenantID)
}

func (u *UsageTracker) CopyCounters() map[string]TenantUsage {
	u.mu.RLock()
	defer u.mu.RUnlock()

	result := make(map[string]TenantUsage, len(u.counters))
	for k, v := range u.counters {
		result[k] = TenantUsage{
			RequestsToday:     v.RequestsToday,
			RequestsThisMonth: v.RequestsThisMonth,
			BlocksToday:       v.BlocksToday,
			LastReset:         v.LastReset,
			MonthStart:        v.MonthStart,
		}
	}
	return result
}

func (u *UsageTracker) MergeCounters(other map[string]TenantUsage) {
	u.mu.Lock()
	defer u.mu.Unlock()

	for tenantID, usage := range other {
		if existing, ok := u.counters[tenantID]; ok {
			existing.RequestsToday += usage.RequestsToday
			existing.RequestsThisMonth += usage.RequestsThisMonth
			existing.BlocksToday += usage.BlocksToday
		} else {
			u.counters[tenantID] = &TenantUsage{
				RequestsToday:     usage.RequestsToday,
				RequestsThisMonth: usage.RequestsThisMonth,
				BlocksToday:       usage.BlocksToday,
				LastReset:         usage.LastReset,
				MonthStart:        usage.MonthStart,
			}
		}
	}
}

type UsageSnapshot struct {
	TenantID          string    `json:"tenant_id"`
	RequestsToday     int64     `json:"requests_today"`
	RequestsThisMonth int64     `json:"requests_this_month"`
	BlocksToday       int64     `json:"blocks_today"`
	Limit             int64     `json:"limit"`
	CapturedAt        time.Time `json:"captured_at"`
}

func (u *UsageTracker) Snapshot(tenantID string) UsageSnapshot {
	today, thisMonth, limit, blocked := u.GetUsage(tenantID)
	return UsageSnapshot{
		TenantID:          tenantID,
		RequestsToday:     today,
		RequestsThisMonth: thisMonth,
		BlocksToday:       blocked,
		Limit:             limit,
		CapturedAt:        time.Now(),
	}
}

func (s *UsageSnapshot) UsagePercent() float64 {
	if s.Limit <= 0 {
		return 0
	}
	return float64(s.RequestsToday) / float64(s.Limit) * 100
}

func (s *UsageSnapshot) RemainingRequests() int64 {
	if s.Limit <= 0 {
		return -1
	}
	remaining := s.Limit - s.RequestsToday
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s *UsageSnapshot) IsOverLimit() bool {
	return s.Limit > 0 && s.RequestsToday >= s.Limit
}
