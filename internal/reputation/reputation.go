package reputation

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

type FeedSource string

const (
	FeedAbuseIPDB       FeedSource = "abuseipdb"
	FeedSpamhaus        FeedSource = "spamhaus"
	FeedEmergingThreats FeedSource = "emerging_threats"
	FeedAlienVault      FeedSource = "alienvault_otx"
	FeedTorExitNodes    FeedSource = "tor_exit_nodes"
	FeedVPNRanges       FeedSource = "vpn_ranges"
	FeedProxyRanges     FeedSource = "proxy_ranges"
)

type IPRecord struct {
	IP           string     `json:"ip"`
	Score        float64    `json:"score"`
	LastSeen     time.Time  `json:"last_seen"`
	AbuseReports int        `json:"abuse_reports"`
	Source       FeedSource `json:"source"`
	Categories   []string   `json:"categories"`
	IsTor        bool       `json:"is_tor"`
	IsVPN        bool       `json:"is_vpn"`
	IsProxy      bool       `json:"is_proxy"`
	IsDatacenter bool       `json:"is_datacenter"`
}

type Engine struct {
	mu                 sync.RWMutex
	cache              map[string]*IPRecord
	cacheTTL           time.Duration
	allowlist          []*net.IPNet
	blocklist          []*net.IPNet
	allowlistASNs      map[uint]bool
	blocklistASNs      map[uint]bool
	allowlistCountries map[string]bool
	blocklistCountries map[string]bool
	torNodes           map[string]bool
	vpnRanges          []*net.IPNet
	proxyRanges        []*net.IPNet
	feedData           map[FeedSource]interface{}
	updaters           []func()
}

func NewEngine() *Engine {
	return &Engine{
		cache:              make(map[string]*IPRecord),
		cacheTTL:           30 * time.Minute,
		allowlist:          make([]*net.IPNet, 0),
		blocklist:          make([]*net.IPNet, 0),
		allowlistASNs:      make(map[uint]bool),
		blocklistASNs:      make(map[uint]bool),
		allowlistCountries: make(map[string]bool),
		blocklistCountries: make(map[string]bool),
		torNodes:           make(map[string]bool),
		vpnRanges:          make([]*net.IPNet, 0),
		proxyRanges:        make([]*net.IPNet, 0),
		feedData:           make(map[FeedSource]interface{}),
	}
}

func (e *Engine) Name() string { return "reputation" }

func (e *Engine) Inspect(ipStr string) (*IPRecord, float64) {
	if ipStr == "" {
		return nil, 0
	}

	e.mu.RLock()
	if record, ok := e.cache[ipStr]; ok && time.Since(record.LastSeen) < e.cacheTTL {
		e.mu.RUnlock()
		return record, record.Score
	}
	e.mu.RUnlock()

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, 0
	}

	record := &IPRecord{
		IP:       ipStr,
		LastSeen: time.Now(),
	}

	score := 0.0

	if e.isAllowlisted(ip) {
		record.Score = 0
		e.setCache(ipStr, record)
		return record, 0
	}

	if e.isBlocklisted(ip) {
		record.Score = 100
		e.setCache(ipStr, record)
		return record, 100
	}

	if e.torNodes[ipStr] {
		score += 60
		record.IsTor = true
		record.Categories = append(record.Categories, "tor")
	}

	if e.isInRanges(ip, e.vpnRanges) {
		score += 40
		record.IsVPN = true
		record.Categories = append(record.Categories, "vpn")
	}

	if e.isInRanges(ip, e.proxyRanges) {
		score += 50
		record.IsProxy = true
		record.Categories = append(record.Categories, "proxy")
	}

	if e.isDatacenterIP(ip) {
		score += 20
		record.IsDatacenter = true
		record.Categories = append(record.Categories, "datacenter")
	}

	record.Score = score
	e.setCache(ipStr, record)

	return record, score
}

func (e *Engine) isAllowlisted(ip net.IP) bool {
	for _, cidr := range e.allowlist {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (e *Engine) isBlocklisted(ip net.IP) bool {
	for _, cidr := range e.blocklist {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (e *Engine) isInRanges(ip net.IP, ranges []*net.IPNet) bool {
	for _, cidr := range ranges {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (e *Engine) isDatacenterIP(ip net.IP) bool {
	datacenterRanges := []string{
		"13.32.0.0/15", "13.104.0.0/14",
		"15.0.0.0/8", "35.0.0.0/8",
		"52.0.0.0/8", "54.0.0.0/8",
		"63.0.0.0/8", "64.0.0.0/8",
		"65.0.0.0/8", "66.0.0.0/8",
		"67.0.0.0/8", "68.0.0.0/8",
		"69.0.0.0/8", "70.0.0.0/8",
		"71.0.0.0/8", "72.0.0.0/8",
		"73.0.0.0/8", "74.0.0.0/8",
		"75.0.0.0/8", "76.0.0.0/8",
		"77.0.0.0/8", "78.0.0.0/8",
		"79.0.0.0/8", "80.0.0.0/5",
		"96.0.0.0/6", "100.0.0.0/8",
		"104.0.0.0/8", "108.0.0.0/8",
		"128.0.0.0/16",
	}

	for _, cidr := range datacenterRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func (e *Engine) setCache(ip string, record *IPRecord) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache[ip] = record
}

func (e *Engine) AddAllowlist(cidr string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid cidr: %w", err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowlist = append(e.allowlist, network)
	return nil
}

func (e *Engine) AddBlocklist(cidr string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid cidr: %w", err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.blocklist = append(e.blocklist, network)
	return nil
}

func (e *Engine) AllowlistASN(asn uint) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowlistASNs[asn] = true
}

func (e *Engine) BlocklistASN(asn uint) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.blocklistASNs[asn] = true
}

func (e *Engine) AllowlistCountry(code string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowlistCountries[code] = true
}

func (e *Engine) BlocklistCountry(code string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.blocklistCountries[code] = true
}

func (e *Engine) LoadTorNodes(nodes []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, node := range nodes {
		e.torNodes[node] = true
	}
}

func (e *Engine) LoadVPNRanges(cidrs []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid vpn cidr: %w", err)
		}
		e.vpnRanges = append(e.vpnRanges, network)
	}
	return nil
}

func (e *Engine) LoadProxyRanges(cidrs []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid proxy cidr: %w", err)
		}
		e.proxyRanges = append(e.proxyRanges, network)
	}
	return nil
}

func (e *Engine) GetScore(ipStr string) float64 {
	_, score := e.Inspect(ipStr)
	return score
}

func (e *Engine) Cleanup() {
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		for range ticker.C {
			e.mu.Lock()
			now := time.Now()
			for ip, record := range e.cache {
				if now.Sub(record.LastSeen) > e.cacheTTL*2 {
					delete(e.cache, ip)
				}
			}
			e.mu.Unlock()
		}
	}()
}

var _ = slog.Debug
