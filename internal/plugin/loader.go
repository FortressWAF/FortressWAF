package plugin

import (
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

type Loader struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	dir     string
}

func NewLoader(dir string) *Loader {
	return &Loader{
		plugins: make(map[string]Plugin),
		dir:     dir,
	}
}

func (l *Loader) LoadAll() error {
	if l.dir == "" {
		return nil
	}

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".wasm" {
			if err := l.loadWASM(filepath.Join(l.dir, entry.Name())); err != nil {
				slog.Warn("failed to load wasm plugin", "file", entry.Name(), "error", err)
			}
		} else if ext == ".so" {
			if err := l.loadSO(filepath.Join(l.dir, entry.Name())); err != nil {
				slog.Warn("failed to load so plugin", "file", entry.Name(), "error", err)
			}
		}
	}
	return nil
}

func (l *Loader) loadWASM(path string) error {
	mgr := Get()
	if mgr == nil {
		return nil
	}
	return mgr.Load(path)
}

func (l *Loader) loadSO(path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return err
	}

	sym, err := p.Lookup("Plugin")
	if err != nil {
		return err
	}

	plug, ok := sym.(Plugin)
	if !ok {
		return nil
	}

	l.mu.Lock()
	l.plugins[plug.Name()] = plug
	l.mu.Unlock()

	return nil
}

func (l *Loader) Get(name string) (Plugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	p, ok := l.plugins[name]
	return p, ok
}

func (l *Loader) List() []Plugin {
	l.mu.RLock()
	defer l.mu.RUnlock()
	plugins := make([]Plugin, 0, len(l.plugins))
	for _, p := range l.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

func (l *Loader) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, p := range l.plugins {
		p.Close()
	}
	return nil
}

var _ unsafe.Pointer
