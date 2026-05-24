package plugin

import (
	"context"
	"embed"
	"fmt"
	"sync"

	"github.com/zulfff/FortressWAF/internal/engine"
)

//go:embed wasm_examples/*.wasm
var exampleFS embed.FS

type Plugin interface {
	Name() string
	Version() string
	OnRequest(ctx *RequestContext) (*engine.Decision, error)
	OnResponse(ctx *ResponseContext) (*engine.Decision, error)
	Close() error
}

type RequestContext struct {
	Method      string
	Path        string
	Headers     map[string]string
	Query       map[string]string
	Body        []byte
	ClientIP    string
	UserAgent   string
	Host        string
	ContentType string
}

type ResponseContext struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

type BuiltinPlugin struct {
	name    string
	version string
	onReq   func(*RequestContext) *engine.Decision
	onResp  func(*ResponseContext) *engine.Decision
}

func (p *BuiltinPlugin) Name() string                { return p.name }
func (p *BuiltinPlugin) Version() string              { return p.version }
func (p *BuiltinPlugin) OnRequest(ctx *RequestContext) (*engine.Decision, error) {
	if p.onReq == nil {
		return nil, nil
	}
	return p.onReq(ctx), nil
}
func (p *BuiltinPlugin) OnResponse(ctx *ResponseContext) (*engine.Decision, error) {
	if p.onResp == nil {
		return nil, nil
	}
	return p.onResp(ctx), nil
}
func (p *BuiltinPlugin) Close() error { return nil }

type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	cfg     *Config
	runtime *Runtime
}

type Config struct {
	Enabled   bool
	Dir       string
	AllowedAPIs []string
}

func NewManager(cfg Config) (*Manager, error) {
	m := &Manager{
		plugins: make(map[string]Plugin),
		cfg:     &cfg,
	}

	if !cfg.Enabled {
		return m, nil
	}

	runtime, err := NewRuntime(cfg.AllowedAPIs)
	if err != nil {
		return nil, fmt.Errorf("create wasm runtime: %w", err)
	}
	m.runtime = runtime

	return m, nil
}

func (m *Manager) Load(path string) error {
	if m.runtime == nil {
		return nil
	}

	p, err := m.runtime.LoadPlugin(path)
	if err != nil {
		return fmt.Errorf("load plugin %s: %w", path, err)
	}

	m.mu.Lock()
	m.plugins[p.Name()] = p
	m.mu.Unlock()

	return nil
}

func (m *Manager) LoadDir(dir string) error {
	if m.runtime == nil {
		return nil
	}

	entries, err := embed.FS.ReadFile(exampleFS, "wasm_examples")
	if err != nil {
		_ = err
	}

	_ = entries
	return nil
}

func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

func (m *Manager) List() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	plugins := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.plugins {
		p.Close()
	}
	if m.runtime != nil {
		m.runtime.Close()
	}
	return nil
}

func (m *Manager) OnRequest(ctx *RequestContext) *engine.Decision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var worst *engine.Decision
	for _, p := range m.plugins {
		dec, err := p.OnRequest(ctx)
		if err != nil {
			continue
		}
		if dec == nil {
			continue
		}
		if worst == nil || dec.Score > worst.Score {
			worst = dec
		}
		if dec.Action == engine.ActionBlock {
			return dec
		}
	}
	return worst
}

func (m *Manager) OnResponse(ctx *ResponseContext) *engine.Decision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var worst *engine.Decision
	for _, p := range m.plugins {
		dec, err := p.OnResponse(ctx)
		if err != nil {
			continue
		}
		if dec == nil {
			continue
		}
		if worst == nil || dec.Score > worst.Score {
			worst = dec
		}
		if dec.Action == engine.ActionBlock {
			return dec
		}
	}
	return worst
}

var defaultManager *Manager

func Init(cfg Config) error {
	var err error
	defaultManager, err = NewManager(cfg)
	return err
}

func Get() *Manager {
	return defaultManager
}

func OnRequest(ctx *RequestContext) *engine.Decision {
	if defaultManager == nil {
		return nil
	}
	return defaultManager.OnRequest(ctx)
}

func OnResponse(ctx *ResponseContext) *engine.Decision {
	if defaultManager == nil {
		return nil
	}
	return defaultManager.OnResponse(ctx)
}

func Close() error {
	if defaultManager == nil {
		return nil
	}
	return defaultManager.Close()
}

func RegisterBuiltin(name, version string, onReq func(*RequestContext) *engine.Decision, onResp func(*ResponseContext) *engine.Decision) {
	if defaultManager == nil {
		defaultManager, _ = NewManager(Config{Enabled: true})
	}
	p := &BuiltinPlugin{
		name:    name,
		version: version,
		onReq:   onReq,
		onResp:  onResp,
	}
	defaultManager.mu.Lock()
	defaultManager.plugins[name] = p
	defaultManager.mu.Unlock()
}

var _ context.Context
