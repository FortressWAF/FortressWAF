package grpc_inspect

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/zulfff/FortressWAF/internal/engine"
)

type GRPCInspector struct {
	mu            sync.RWMutex
	maxBodySize   int
	maxDepth      int
	maxConcurrent int
	currentReqs   int
	contentType   string
	frameSize     int
	healthCheck   bool
	reflection    bool
	knownMethods  map[string]bool
}

func New(cfg Config) *GRPCInspector {
	known := make(map[string]bool)
	for _, m := range cfg.KnownMethods {
		known[m] = true
	}
	return &GRPCInspector{
		maxBodySize:   cfg.MaxBodySize,
		maxDepth:      cfg.MaxDepth,
		maxConcurrent: cfg.MaxConcurrent,
		contentType:   "application/grpc",
		frameSize:     16 * 1024,
		healthCheck:   cfg.HealthCheck,
		reflection:    cfg.Reflection,
		knownMethods:  known,
	}
}

type Config struct {
	MaxBodySize   int
	MaxDepth      int
	MaxConcurrent int
	HealthCheck   bool
	Reflection    bool
	KnownMethods  []string
}

func (g *GRPCInspector) Name() string { return "grpc_inspection" }

func (g *GRPCInspector) Inspect(ctx *engine.RequestContext) (*engine.Decision, error) {
	if ctx.ContentType != g.contentType && !strings.Contains(ctx.ContentType, "grpc") {
		return &engine.Decision{Action: engine.ActionAllow}, nil
	}

	g.mu.Lock()
	if g.currentReqs >= g.maxConcurrent {
		g.mu.Unlock()
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "GRPC-001",
			RuleName: "GRPC concurrent limit exceeded",
			Severity: "high",
			Score:    85,
			Evidence: fmt.Sprintf("concurrent_requests=%d, limit=%d", g.currentReqs, g.maxConcurrent),
		}, nil
	}
	g.currentReqs++
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		g.currentReqs--
		g.mu.Unlock()
	}()

	if g.healthCheck && g.isHealthCheck(ctx) {
		return &engine.Decision{Action: engine.ActionAllow}, nil
	}

	if !g.validateGRPCFrame(ctx) {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "GRPC-002",
			RuleName: "GRPC frame validation failed",
			Severity: "high",
			Score:    90,
			Evidence: "invalid grpc frame structure",
		}, nil
	}

	if ctx.Method != "" && !g.knownMethods[ctx.Method] && len(g.knownMethods) > 0 {
		slog.Debug("unknown grpc method", "method", ctx.Method)
	}

	return &engine.Decision{Action: engine.ActionAllow}, nil
}

func (g *GRPCInspector) isHealthCheck(ctx *engine.RequestContext) bool {
	path := strings.ToLower(ctx.Path)
	return strings.Contains(path, "grpc.health") ||
		strings.Contains(path, "/grpc.lb.v1.LoadBalancer") ||
		strings.Contains(path, "/grpc.reflection.v1alpha.ServerReflection") ||
		strings.Contains(path, "/grpc.reflection.v1.ServerReflection")
}

func (g *GRPCInspector) validateGRPCFrame(ctx *engine.RequestContext) bool {
	if ctx.Body == nil {
		return true
	}
	if len(ctx.Body) > g.frameSize {
		return false
	}
	if len(ctx.Body) < 5 {
		return false
	}
	return true
}

func (g *GRPCInspector) ParseMessage(body []byte) (method string, payload []byte, err error) {
	if len(body) < 5 {
		return "", nil, fmt.Errorf("frame too small")
	}

	compressed := body[0]&0x80 != 0
	_ = compressed

	msgLen := int(body[1])<<24 | int(body[2])<<16 | int(body[3])<<8 | int(body[4])
	if msgLen < 0 || msgLen > g.maxBodySize {
		return "", nil, fmt.Errorf("message length out of bounds: %d", msgLen)
	}

	if len(body) < 5+msgLen {
		return "", nil, io.ErrUnexpectedEOF
	}

	return "", body[5 : 5+msgLen], nil
}

func (g *GRPCInspector) CheckReflection(ctx *engine.RequestContext) *engine.Decision {
	if !g.reflection {
		return nil
	}

	if strings.Contains(ctx.Path, "ServerReflection") {
		return &engine.Decision{
			Action:   engine.ActionMonitor,
			RuleID:   "GRPC-003",
			RuleName: "GRPC reflection API accessed",
			Severity: "low",
			Score:    20,
			Evidence: fmt.Sprintf("path=%s", ctx.Path),
		}
	}
	return nil
}

func (g *GRPCInspector) SetMaxConcurrent(n int) {
	g.mu.Lock()
	g.maxConcurrent = n
	g.mu.Unlock()
}

func (g *GRPCInspector) Stats() (concurrent, total int) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.currentReqs, 0
}

var _ time.Time
