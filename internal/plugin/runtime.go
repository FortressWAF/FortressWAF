package plugin

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Runtime struct {
	mu          sync.RWMutex
	runtime     wazero.Runtime
	allowedAPIs map[string]bool
	modules     map[string]api.Module
}

func NewRuntime(allowedAPIs []string) (*Runtime, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	_, err := r.NewHostModuleBuilder("fortresswaf").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) (int32, int32) {
		return 0, 0
	}).Export("log").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, size int32) int32 {
		return 0
	}).Export("get_header").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, namePtr, nameSize, valuePtr int32) int32 {
		return 0
	}).Export("set_header").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, namePtr, nameSize, valuePtr, valueSize int32) int32 {
		return 0
	}).Export("block").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, reasonPtr, reasonSize int32) int32 {
		return 1
	}).Export("allow").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) int32 {
		return 1
	}).Instantiate(ctx)
	if err != nil {
		return nil, fmt.Errorf("host module: %w", err)
	}

	allowed := make(map[string]bool)
	for _, a := range allowedAPIs {
		allowed[a] = true
	}

	return &Runtime{
		runtime:     r,
		allowedAPIs: allowed,
		modules:     make(map[string]api.Module),
	}, nil
}

func (r *Runtime) LoadPlugin(path string) (Plugin, error) {
	ctx := context.Background()

	compiled, err := r.runtime.CompileGuestModule(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("compile module: %w", err)
	}

	mod, err := r.runtime.InstantiateModule(ctx, compiled)
	if err != nil {
		return nil, fmt.Errorf("instantiate module: %w", err)
	}

	r.mu.Lock()
	r.modules[path] = mod
	r.mu.Unlock()

	return &wasmPlugin{
		name:   "wasm-plugin",
		module: mod,
		runtime: r,
	}, nil
}

func (r *Runtime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.modules {
		m.Close(context.Background())
	}
	return r.runtime.Close(context.Background())
}

type wasmPlugin struct {
	name   string
	module api.Module
	runtime *Runtime
}

func (p *wasmPlugin) Name() string { return p.name }
func (p *wasmPlugin) Version() string { return "1.0.0" }

func (p *wasmPlugin) OnRequest(ctx *RequestContext) (*engine.Decision, error) {
	return nil, nil
}

func (p *wasmPlugin) OnResponse(ctx *ResponseContext) (*engine.Decision, error) {
	return nil, nil
}

func (p *wasmPlugin) Close() error {
	p.module.Close(context.Background())
	return nil
}

func wasmString(mem api.Memory, ptr, size int32) string {
	if ptr == 0 || size == 0 {
		return ""
	}
	buf, ok := mem.Read(uint32(ptr), uint32(size))
	if !ok {
		return ""
	}
	return string(buf)
}

func writeWasmString(mem api.Memory, str string, ptr int32) int32 {
	if ptr == 0 {
		return 0
	}
	data := []byte(str)
	mem.Write(uint32(ptr), data)
	return int32(len(data))
}

var _ engine.Decision
