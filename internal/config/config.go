package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type LoggingConfig struct {
	Level   string `yaml:"level"`
	Format  string `yaml:"format"`
	Output  string `yaml:"output"`
	Verbose bool   `yaml:"verbose"`
}

type TLSConfig struct {
	Enabled    bool   `yaml:"enabled"`
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	MinVersion string `yaml:"min_version"`
}

type AdminConfig struct {
	Port     int      `yaml:"port"`
	Enabled  bool     `yaml:"enabled"`
	MTLS     bool     `yaml:"mtls"`
	CACert   string   `yaml:"ca_cert"`
	CertFile string   `yaml:"cert_file"`
	KeyFile  string   `yaml:"key_file"`
	APIKeys  []string `yaml:"api_keys"`
}

type RedisConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
	TTL      int    `yaml:"ttl"`
}

type DBConfig struct {
	Driver  string `yaml:"driver"`
	DSN     string `yaml:"dsn"`
	MaxOpen int    `yaml:"max_open"`
	MaxIdle int    `yaml:"max_idle"`
}

type MLConfig struct {
	Enabled      bool    `yaml:"enabled"`
	Endpoint     string  `yaml:"endpoint"`
	Timeout      int     `yaml:"timeout"`
	MaxRetries   int     `yaml:"max_retries"`
	FallbackMode string  `yaml:"fallback_mode"`
	MinScore     float64 `yaml:"min_score"`
}

type SiteRuleOverride struct {
	Enabled *bool                `yaml:"enabled,omitempty"`
	Params  map[string]any `yaml:"params,omitempty"`
}

type SiteConfig struct {
	Name          string                      `yaml:"name"`
	Domains       []string                    `yaml:"domains"`
	Upstream      string                     `yaml:"upstream"`
	Port          int                         `yaml:"port"`
	TLS           bool                        `yaml:"tls"`
	CertFile      string                     `yaml:"cert_file"`
	KeyFile       string                     `yaml:"key_file"`
	RateLimit     *RateLimitSiteConfig       `yaml:"rate_limit,omitempty"`
	WAFEnabled   bool                       `yaml:"waf_enabled"`
	RuleOverrides map[string]SiteRuleOverride `yaml:"rule_overrides,omitempty"`
}

type RateLimitSiteConfig struct {
	RequestsPerSecond int `yaml:"requests_per_second"`
	Burst             int `yaml:"burst"`
}

type RuleConfig struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Enabled     bool              `yaml:"enabled"`
	Severity    string            `yaml:"severity"`
	Action      string            `yaml:"action"`
	Phase       string            `yaml:"phase"`
	Priority    int               `yaml:"priority"`
	Field       string            `yaml:"field"`
	Operator    string            `yaml:"operator"`
	Value       string            `yaml:"value"`
	Transform   []string          `yaml:"transform"`
	Tags        []string         `yaml:"tags"`
	Params      map[string]any `yaml:"params,omitempty"`
}

type Config struct {
	mu       sync.RWMutex
	filePath string

	Sites   []SiteConfig   `yaml:"sites"`
	Rules   []RuleConfig  `yaml:"rules"`
	ML      MLConfig      `yaml:"ml"`
	Redis   RedisConfig   `yaml:"redis"`
	DB      DBConfig      `yaml:"db"`
	Logging LoggingConfig `yaml:"logging"`
	TLS     TLSConfig     `yaml:"tls"`
	Admin   AdminConfig   `yaml:"admin"`
}

type Manager struct {
	mu       sync.RWMutex
	config   *Config
	watcher  *fsnotify.Watcher
	onChange []func(*Config)
}

func DefaultConfig() *Config {
	return &Config{
		Sites: []SiteConfig{},
		Rules: []RuleConfig{},
		ML: MLConfig{
			Enabled:      false,
			Endpoint:     "http://127.0.0.1:9090",
			Timeout:      5000,
			MaxRetries:   2,
			FallbackMode: "allow",
			MinScore:     0.5,
		},
		Redis: RedisConfig{
			Enabled:  false,
			Addr:     "127.0.0.1:6379",
			Password: "",
			DB:       0,
			PoolSize: 10,
			TTL:      3600,
		},
		DB: DBConfig{
			Driver:  "sqlite3",
			DSN:     "fortresswaf.db",
			MaxOpen: 25,
			MaxIdle: 5,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		TLS: TLSConfig{
			Enabled:    false,
			MinVersion: "1.2",
		},
		Admin: AdminConfig{
			Port:    8443,
			Enabled: true,
		},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	cfg.filePath = path

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Sites) == 0 {
		return fmt.Errorf("at least one site must be configured")
	}

	for i, site := range c.Sites {
		if site.Name == "" {
			return fmt.Errorf("site[%d]: name is required", i)
		}
		if len(site.Domains) == 0 {
			return fmt.Errorf("site[%d] %q: at least one domain is required", i, site.Name)
		}
		if site.Upstream == "" {
			return fmt.Errorf("site[%d] %q: upstream is required", i, site.Name)
		}
	}

	for i, rule := range c.Rules {
		if rule.ID == "" {
			return fmt.Errorf("rule[%d]: ID is required", i)
		}
		if rule.Field == "" {
			return fmt.Errorf("rule[%d] %q: field is required", i, rule.ID)
		}
		if rule.Operator == "" {
			return fmt.Errorf("rule[%d] %q: operator is required", i, rule.ID)
		}
	}

	if c.ML.Enabled && c.ML.Endpoint == "" {
		return fmt.Errorf("ml.endpoint is required when ml is enabled")
	}

	if c.Redis.Enabled && c.Redis.Addr == "" {
		return fmt.Errorf("redis.addr is required when redis is enabled")
	}

	return nil
}

func NewManager(path string) (*Manager, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		config: cfg,
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return m, nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("hot-reload watcher not available", "error", err)
		return m, nil
	}

	m.watcher = watcher

	dir := filepath.Dir(absPath)
	if err := watcher.Add(dir); err != nil {
		slog.Warn("cannot watch config directory", "dir", dir, "error", err)
		return m, nil
	}

	go m.watchLoop(absPath)

	return m, nil
}

func (m *Manager) watchLoop(path string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("config watcher panic", "recover", r)
		}
	}()

	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if event.Name == path {
					slog.Info("config file changed, reloading", "path", path)
					if err := m.Reload(); err != nil {
						slog.Error("config reload failed", "error", err)
					}
				}
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("config watcher error", "error", err)
		}
	}
}

func (m *Manager) Reload() error {
	newCfg, err := Load(m.config.filePath)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.config = newCfg
	m.mu.Unlock()

	for _, cb := range m.onChange {
		cb(newCfg)
	}

	return nil
}

func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *Manager) OnChange(cb func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = append(m.onChange, cb)
}

func (m *Manager) Close() error {
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}

func (c *Config) GetSite(name string) *SiteConfig {
	for i := range c.Sites {
		if c.Sites[i].Name == name {
			return &c.Sites[i]
		}
	}
	return nil
}

func (c *Config) FindSiteByDomain(domain string) *SiteConfig {
	for i := range c.Sites {
		for _, d := range c.Sites[i].Domains {
			if d == domain {
				return &c.Sites[i]
			}
		}
	}
	return nil
}

func (c *Config) GetRule(id string) *RuleConfig {
	for i := range c.Rules {
		if c.Rules[i].ID == id {
			return &c.Rules[i]
		}
	}
	return nil
}

func (c *Config) GetEnabledRules() []RuleConfig {
	var rules []RuleConfig
	for _, r := range c.Rules {
		if r.Enabled {
			rules = append(rules, r)
		}
	}
	return rules
}

func SaveToFile(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (m *Manager) UpdateConfig(fn func(*Config)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fn(m.config)

	if err := m.config.Validate(); err != nil {
		return fmt.Errorf("validate updated config: %w", err)
	}

	if m.config.filePath != "" {
		if err := SaveToFile(m.config.filePath, m.config); err != nil {
			return err
		}
	}

	for _, cb := range m.onChange {
		cb(m.config)
	}

	return nil
}

var DefaultManager *Manager

func SetDefaultManager(m *Manager) {
	DefaultManager = m
}

func GetConfig() *Config {
	if DefaultManager == nil {
		return DefaultConfig()
	}
	return DefaultManager.Get()
}

func ExpandEnvRefs(data []byte) ([]byte, error) {
	str := string(data)
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	matches := re.FindAllString(str, -1)

	for _, match := range matches {
		envVar := match[2 : len(match)-1]
		if val := os.Getenv(envVar); val != "" {
			str = strings.ReplaceAll(str, match, val)
		}
	}

	return []byte(str), nil
}

var _ = time.Second
