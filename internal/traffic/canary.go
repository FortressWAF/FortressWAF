package traffic

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zulfff/FortressWAF/internal/config"
)

type CanaryRouter struct {
	mu    sync.RWMutex
	rules []CanaryRule
}

type CanaryRule struct {
	Name          string
	Match         RuleMatch
	Primary       string
	Canary        string
	Weight        int
	StickyHeader  string
	StickyCookie  string
	SessionTTL    time.Duration
	headerMatch   string
	cookieName    string
}

type RuleMatch struct {
	PathPrefix   string
	PathContains string
	HeaderName   string
	HeaderValue  string
	QueryName    string
	QueryValue   string
	IPCIDR       []string
}

type CanaryResult struct {
	Upstream     string
	IsCanary     bool
	CanaryWeight int
	StickyID     string
}

func NewCanaryRouter(cfg []config.CanaryConfig) *CanaryRouter {
	r := &CanaryRouter{
		rules: make([]CanaryRule, 0),
	}

	for _, c := range cfg {
		rule := CanaryRule{
			Name:         c.Name,
			Primary:      c.Primary,
			Canary:       c.Canary,
			Weight:       c.Weight,
			StickyHeader: c.StickyHeader,
			StickyCookie: c.CookieName,
			SessionTTL:   c.SessionTTL,
		}

		if c.Match.PathPrefix != "" {
			rule.Match.PathPrefix = c.Match.PathPrefix
		}
		if c.Match.PathContains != "" {
			rule.Match.PathContains = c.Match.PathContains
		}
		if c.Match.HeaderName != "" {
			rule.headerMatch = c.Match.HeaderName
			rule.Match.HeaderValue = c.Match.HeaderValue
		}
		if c.Match.QueryName != "" {
			rule.Match.QueryName = c.Match.QueryName
			rule.Match.QueryValue = c.Match.QueryValue
		}

		r.rules = append(r.rules, rule)
	}

	return r
}

func (c *CanaryRouter) Name() string { return "canary_routing" }

func (c *CanaryRouter) Route(ctx RequestContext) CanaryResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, rule := range c.rules {
		if !c.matchRule(rule, ctx) {
			continue
		}

		stickyID := c.getStickyID(ctx, rule)

		if stickyID != "" {
			canary := c.hashToBool(stickyID, rule.Weight)
			if canary {
				return CanaryResult{
					Upstream:     rule.Canary,
					IsCanary:     true,
					CanaryWeight: rule.Weight,
					StickyID:     stickyID,
				}
			}
			return CanaryResult{
				Upstream:     rule.Primary,
				IsCanary:     false,
				CanaryWeight: rule.Weight,
				StickyID:     stickyID,
			}
		}

		upstream := c.selectUpstream(rule)
		return CanaryResult{
			Upstream:     upstream,
			IsCanary:     upstream == rule.Canary,
			CanaryWeight: rule.Weight,
		}
	}

	return CanaryResult{}
}

func (c *CanaryRouter) matchRule(rule CanaryRule, ctx RequestContext) bool {
	if rule.Match.PathPrefix != "" {
		if !strings.HasPrefix(ctx.Path, rule.Match.PathPrefix) {
			return false
		}
	}

	if rule.Match.PathContains != "" {
		if !strings.Contains(ctx.Path, rule.Match.PathContains) {
			return false
		}
	}

	if rule.headerMatch != "" {
		if v, ok := ctx.Headers[rule.headerMatch]; !ok || (rule.Match.HeaderValue != "" && v != rule.Match.HeaderValue) {
			return false
		}
	}

	if rule.Match.QueryName != "" {
		if v, ok := ctx.Query[rule.Match.QueryName]; !ok || (rule.Match.QueryValue != "" && v != rule.Match.QueryValue) {
			return false
		}
	}

	return true
}

func (c *CanaryRouter) getStickyID(ctx RequestContext, rule CanaryRule) string {
	if rule.StickyHeader != "" {
		if v, ok := ctx.Headers[rule.StickyHeader]; ok {
			return v
		}
	}

	if rule.StickyCookie != "" {
		for _, cookie := range ctx.Cookies {
			if cookie.Name == rule.StickyCookie {
				return cookie.Value
			}
		}
	}

	return ""
}

func (c *CanaryRouter) hashToBool(id string, weight int) bool {
	if weight <= 0 {
		return false
	}
	if weight >= 100 {
		return true
	}

	h := fnv.New32a()
	h.Write([]byte(id))
	hash := h.Sum32()
	return int(hash%100) < weight
}

func (c *CanaryRouter) selectUpstream(rule CanaryRule) string {
	if rule.Weight <= 0 {
		return rule.Primary
	}
	if rule.Weight >= 100 {
		return rule.Canary
	}

	r := rand.Intn(100)
	if r < rule.Weight {
		return rule.Canary
	}
	return rule.Primary
}

type RequestContext struct {
	Method   string
	Path     string
	Headers  map[string]string
	Query    map[string]string
	Cookies  []http.Cookie
	ClientIP string
}

func (c *CanaryRouter) GetStats() map[string]CanaryStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]CanaryStats)
	for _, rule := range c.rules {
		stats[rule.Name] = CanaryStats{
			PrimaryRequests: 0,
			CanaryRequests:  0,
		}
	}
	return stats
}

type CanaryStats struct {
	PrimaryRequests int64
	CanaryRequests  int64
	CanaryRatio     float64
}

type ABMux struct {
	mu     sync.RWMutex
	tests  map[string]*ABTest
	sticky map[string]*ABStickySession
}

type ABMuxConfig struct {
	Tests []struct {
		Name        string
		VariantA    string
		VariantB    string
		Percentage  int
		Metrics     []string
		Sticky      bool
		Match       RuleMatch
	}
}

type ABDTest struct {
	Name        string
	VariantA    string
	VariantB    string
	Percentage  int
	Metrics     []string
	Sticky      bool
	Match       RuleMatch
}

func NewABMux(cfg ABMuxConfig) *ABMux {
	m := &ABMux{
		tests:  make(map[string]*ABTest),
		sticky: make(map[string]*ABStickySession),
	}

	for _, t := range cfg.Tests {
		m.tests[t.Name] = &ABTest{
			Name:       t.Name,
			VariantA:   t.VariantA,
			VariantB:   t.VariantB,
			Percentage: t.Percentage,
			Metrics:    t.Metrics,
			Sticky:     t.Sticky,
		}
	}

	return m
}

func (m *ABMux) Route(ctx RequestContext, testName string) string {
	m.mu.RLock()
	test, ok := m.tests[testName]
	m.mu.RUnlock()

	if !ok {
		return ""
	}

	if test.Sticky {
		stickyID := m.getStickyID(ctx, testName)
		if stickyID != "" {
			variant := m.getStickyVariant(stickyID, testName)
			if variant != "" {
				return variant
			}
		}
	}

	variant := m.selectVariant(test.Percentage)
	if test.Sticky {
		m.setStickyVariant(ctx, testName, variant)
	}

	return variant
}

func (m *ABMux) selectVariant(percentage int) string {
	if rand.Intn(100) < percentage {
		return "B"
	}
	return "A"
}

func (m *ABMux) getStickyID(ctx RequestContext, testName string) string {
	cookieName := fmt.Sprintf("ab_%s", testName)
	for _, c := range ctx.Cookies {
		if c.Name == cookieName {
			return c.Value
		}
	}
	return ""
}

func (m *ABMux) getStickyVariant(id, testName string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", testName, id)
	session, ok := m.sticky[key]
	if !ok {
		return ""
	}
	return session.Variant
}

func (m *ABMux) setStickyVariant(ctx RequestContext, testName, variant string) {
	id := ctx.ClientIP
	if id == "" {
		id = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	m.mu.Lock()
	key := fmt.Sprintf("%s:%s", testName, id)
	m.sticky[key] = &ABStickySession{
		Variant: variant,
		ExpAt:   time.Now().Add(24 * time.Hour),
	}
	m.mu.Unlock()
}

type ABStickySession struct {
	Variant string
	ExpAt   time.Time
}

type ShadowRouter struct {
	mu    sync.RWMutex
	rules []ShadowRule
}

type ShadowRule struct {
	Name             string
	Match            RuleMatch
	Shadow           string
	Percent          int
	HeaderInjection  bool
	ResponseTimeout  time.Duration
	RetryOnError     bool
	IgnoreResponse   bool
}

func NewShadowRouter(cfg []config.ShadowConfig) *ShadowRouter {
	r := &ShadowRouter{
		rules: make([]ShadowRule, 0),
	}

	for _, s := range cfg {
		rule := ShadowRule{
			Name:            s.Name,
			Shadow:          s.Shadow,
			Percent:         s.Percent,
			HeaderInjection: s.HeaderInjection,
			ResponseTimeout: s.ResponseTimeout,
			RetryOnError:    s.RetryOnError,
			IgnoreResponse:  s.IgnoreResponse,
		}

		if s.Match.PathPrefix != "" {
			rule.Match.PathPrefix = s.Match.PathPrefix
		}
		if s.Match.PathContains != "" {
			rule.Match.PathContains = s.Match.PathContains
		}

		r.rules = append(r.rules, rule)
	}

	return r
}

func (s *ShadowRouter) ShouldShadow(ctx RequestContext) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, rule := range s.rules {
		if !s.matchRule(rule, ctx) {
			continue
		}

		if rand.Intn(100) < rule.Percent {
			return true, rule.Shadow
		}
		return false, ""
	}

	return false, ""
}

func (s *ShadowRouter) matchRule(rule ShadowRule, ctx RequestContext) bool {
	if rule.Match.PathPrefix != "" {
		if !strings.HasPrefix(ctx.Path, rule.Match.PathPrefix) {
			return false
		}
	}

	if rule.Match.PathContains != "" {
		if !strings.Contains(ctx.Path, rule.Match.PathContains) {
			return false
		}
	}

	return true
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var _ = sha256.New
var _ = binary.BigEndian
var _ = math.MaxInt64
