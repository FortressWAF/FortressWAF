package engine

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketInspector struct {
	mu                sync.RWMutex
	maxFrameSize      int
	maxMessageSize    int
	maxDepth          int
	maxFramesPerMin   int
	maxBytesPerMin    int
	blockOnLimit      bool
	frameLog          map[string]*FrameStats
	allowedTypes      map[int]bool
	strictMode        bool
	enablePing        bool
	enablePong        bool
	enableClose       bool
	connectionTimeout time.Duration
}

type FrameStats struct {
	Count    int
	Bytes    int64
	LastSeen time.Time
	Types    map[int]int
}

type Frame struct {
	Type     int
	Payload  []byte
	Finished bool
	Seq      int64
}

func NewWebSocketInspector(cfg WebSocketConfig) *WebSocketInspector {
	allowedTypes := make(map[int]bool)
	if len(cfg.AllowedTypes) == 0 {
		allowedTypes[websocket.TextMessage] = true
		allowedTypes[websocket.BinaryMessage] = true
	} else {
		for _, t := range cfg.AllowedTypes {
			allowedTypes[t] = true
		}
	}

	return &WebSocketInspector{
		maxFrameSize:      cfg.MaxFrameSize,
		maxMessageSize:    cfg.MaxMessageSize,
		maxDepth:          cfg.MaxDepth,
		maxFramesPerMin:   cfg.MaxFramesPerMin,
		maxBytesPerMin:    cfg.MaxBytesPerMin,
		blockOnLimit:      cfg.BlockOnLimit,
		frameLog:          make(map[string]*FrameStats),
		allowedTypes:      allowedTypes,
		strictMode:        cfg.StrictMode,
		enablePing:        cfg.EnablePing,
		enablePong:        cfg.EnablePong,
		enableClose:       cfg.EnableClose,
		connectionTimeout: cfg.ConnectionTimeout,
	}
}

type WebSocketConfig struct {
	Enabled           bool
	MaxFrameSize      int
	MaxMessageSize    int
	MaxDepth          int
	MaxFramesPerMin   int
	MaxBytesPerMin    int
	BlockOnLimit      bool
	AllowedTypes      []int
	StrictMode        bool
	EnablePing        bool
	EnablePong        bool
	EnableClose       bool
	ConnectionTimeout time.Duration
}

func (w *WebSocketInspector) Name() string { return "websocket_inspection" }

func (w *WebSocketInspector) Inspect(ctx *RequestContext) (*Decision, error) {
	if !w.isWebSocket(ctx) {
		return &Decision{Action: ActionAllow}, nil
	}

	stats := w.getStats(ctx.RealIP)

	if w.maxFramesPerMin > 0 && stats.Count > w.maxFramesPerMin {
		if w.blockOnLimit {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "WS-001",
				RuleName: "WebSocket frame rate limit exceeded",
				Severity: "high",
				Score:    80,
				Evidence: fmt.Sprintf("frames_per_min=%d, limit=%d", stats.Count, w.maxFramesPerMin),
			}, nil
		}
		return &Decision{
			Action:   ActionMonitor,
			RuleID:   "WS-001",
			RuleName: "WebSocket frame rate limit exceeded",
			Severity: "medium",
			Score:    40,
			Evidence: fmt.Sprintf("frames_per_min=%d, limit=%d", stats.Count, w.maxFramesPerMin),
		}, nil
	}

	return &Decision{Action: ActionAllow}, nil
}

func (w *WebSocketInspector) InspectMessage(ctx *RequestContext, frame Frame) (*Decision, error) {
	stats := w.getStats(ctx.RealIP)

	stats.Count++
	stats.Bytes += int64(len(frame.Payload))
	stats.LastSeen = time.Now()
	stats.Types[frame.Type]++

	if frame.Type == int(websocket.CloseMessage) && !w.enableClose {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "WS-002",
			RuleName: "WebSocket close not allowed",
			Severity: "medium",
			Score:    50,
		}, nil
	}

	if frame.Type == int(websocket.PingMessage) && !w.enablePing {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "WS-003",
			RuleName: "WebSocket PING not allowed",
			Severity: "low",
			Score:    30,
		}, nil
	}

	if frame.Type == int(websocket.PongMessage) && !w.enablePong {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "WS-004",
			RuleName: "WebSocket PONG not allowed",
			Severity: "low",
			Score:    30,
		}, nil
	}

	if !w.allowedTypes[frame.Type] {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "WS-005",
			RuleName: "WebSocket message type not allowed",
			Severity: "high",
			Score:    75,
			Evidence: fmt.Sprintf("type=%d", frame.Type),
		}, nil
	}

	if len(frame.Payload) > w.maxFrameSize {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "WS-006",
			RuleName: "WebSocket frame size exceeded",
			Severity: "high",
			Score:    70,
			Evidence: fmt.Sprintf("frame_size=%d, limit=%d", len(frame.Payload), w.maxFrameSize),
		}, nil
	}

	if w.strictMode && frame.Type == int(websocket.TextMessage) {
		if dec := w.checkInjection(frame.Payload); dec != nil {
			return dec, nil
		}
	}

	if w.maxDepth > 0 && frame.Type == int(websocket.TextMessage) {
		if dec := w.checkJSONDepth(frame.Payload); dec != nil {
			return dec, nil
		}
	}

	return &Decision{Action: ActionAllow}, nil
}

func (w *WebSocketInspector) isWebSocket(ctx *RequestContext) bool {
	upgrade := strings.ToLower(ctx.Headers["Upgrade"])
	connection := strings.ToLower(ctx.Headers["Connection"])
	wsKey := ctx.Headers["Sec-Websocket-Key"]
	wsVersion := ctx.Headers["Sec-Websocket-Version"]

	return strings.Contains(upgrade, "websocket") &&
		strings.Contains(connection, "upgrade") &&
		wsKey != "" &&
		wsVersion == "13"
}

func (w *WebSocketInspector) checkInjection(payload []byte) *Decision {
	injPatterns := []string{
		"<script", "javascript:", "onerror=", "onload=",
		"onclick=", "onmouseover=", "eval(", "document.",
		"window.", "expression(", "url(", "href=",
	}

	str := string(payload)
	lower := strings.ToLower(str)

	for _, pattern := range injPatterns {
		if strings.Contains(lower, pattern) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "WS-INJ-001",
				RuleName: "WebSocket injection pattern detected",
				Severity: "critical",
				Score:    95,
				Evidence: fmt.Sprintf("pattern=%s", pattern),
			}
		}
	}
	return nil
}

func (w *WebSocketInspector) checkJSONDepth(payload []byte) *Decision {
	var nested int
	for i := 0; i < len(payload); i++ {
		if payload[i] == '{' || payload[i] == '[' {
			nested++
			if nested > w.maxDepth {
				return &Decision{
					Action:   ActionBlock,
					RuleID:   "WS-JSON-001",
					RuleName: "WebSocket JSON depth exceeded",
					Severity: "high",
					Score:    75,
					Evidence: fmt.Sprintf("depth=%d, limit=%d", nested, w.maxDepth),
				}
			}
		}
	}
	return nil
}

func (w *WebSocketInspector) getStats(ip string) *FrameStats {
	w.mu.RLock()
	stats, ok := w.frameLog[ip]
	w.mu.RUnlock()

	if !ok {
		w.mu.Lock()
		stats = &FrameStats{Types: make(map[int]int)}
		w.frameLog[ip] = stats
		w.mu.Unlock()
	}

	now := time.Now()
	w.mu.Lock()
	if now.Sub(stats.LastSeen) > time.Minute {
		stats.Count = 0
		stats.Bytes = 0
	}
	w.mu.Unlock()

	return stats
}

func (w *WebSocketInspector) ParseFrame(data []byte) (Frame, error) {
	if len(data) < 2 {
		return Frame{}, fmt.Errorf("frame too small")
	}

	first := data[0]
	fin := first&0x80 != 0
	opcode := int(first & 0x0f)

	mask := data[1]&0x80 != 0
	payloadLen := int(data[1] & 0x7f)

	idx := 2
	if mask {
		idx += 4
	}

	if payloadLen == 126 {
		if len(data) < 4 {
			return Frame{}, fmt.Errorf("invalid extended length")
		}
		payloadLen = int(data[2])<<8 | int(data[3])
		idx = 4
	} else if payloadLen == 127 {
		if len(data) < 10 {
			return Frame{}, fmt.Errorf("invalid extended length")
		}
		payloadLen = int(data[6])<<56 | int(data[7])<<48 | int(data[8])<<40 | int(data[9])<<32 |
			int(data[10])<<24 | int(data[11])<<16 | int(data[12])<<8 | int(data[13])
		idx = 10
	}

	if len(data) < idx+payloadLen {
		return Frame{}, io.ErrUnexpectedEOF
	}

	payload := data[idx : idx+payloadLen]
	if mask {
		key := data[idx-4 : idx]
		for i := range payload {
			payload[i] ^= key[i%4]
		}
	}

	return Frame{
		Type:     opcode,
		Payload:  payload,
		Finished: fin,
	}, nil
}

func (w *WebSocketInspector) Upgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}
