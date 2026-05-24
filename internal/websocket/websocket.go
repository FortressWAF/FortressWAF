package websocket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FortressWAF/FortressWAF/internal/engine"
	"github.com/gorilla/websocket"
)

type Inspector struct {
	mu               sync.RWMutex
	maxFrameSize     int
	maxMessageSize   int
	maxDepth         int
	maxFramesPerMin  int
	maxBytesPerMin   int
	blockOnLimit     bool
	frameLog         map[string]*FrameStats
	allowedTypes     map[websocket.MessageType]bool
	strictMode       bool
	enablePING       bool
	enablePONG       bool
	enableClose      bool
	pingInterval     time.Duration
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

type Message struct {
	Type     int
	Payload  []byte
	Depth    int
	IsArray  bool
	Path     string
}

func New(cfg Config) *Inspector {
	allowedTypes := make(map[websocket.MessageType]bool)
	if len(cfg.AllowedTypes) == 0 {
		allowedTypes[websocket.TextMessage] = true
		allowedTypes[websocket.BinaryMessage] = true
	} else {
		for _, t := range cfg.AllowedTypes {
			allowedTypes[websocket.MessageType(t)] = true
		}
	}

	return &Inspector{
		maxFrameSize:     cfg.MaxFrameSize,
		maxMessageSize:    cfg.MaxMessageSize,
		maxDepth:          cfg.MaxDepth,
		maxFramesPerMin:   cfg.MaxFramesPerMin,
		maxBytesPerMin:    cfg.MaxBytesPerMin,
		blockOnLimit:      cfg.BlockOnLimit,
		frameLog:          make(map[string]*FrameStats),
		allowedTypes:      allowedTypes,
		strictMode:        cfg.StrictMode,
		enablePING:        cfg.EnablePING,
		enablePONG:        cfg.EnablePONG,
		enableClose:       cfg.EnableClose,
		pingInterval:      cfg.PingInterval,
		connectionTimeout: cfg.ConnectionTimeout,
	}
}

type Config struct {
	MaxFrameSize      int
	MaxMessageSize    int
	MaxDepth          int
	MaxFramesPerMin   int
	MaxBytesPerMin    int
	BlockOnLimit      bool
	AllowedTypes      []int
	StrictMode        bool
	EnablePING        bool
	EnablePONG        bool
	EnableClose       bool
	PingInterval      time.Duration
	ConnectionTimeout time.Duration
}

func (w *Inspector) Name() string { return "websocket_inspection" }

func (w *Inspector) Inspect(ctx *engine.RequestContext) (*engine.Decision, error) {
	if !w.isWebSocket(ctx) {
		return &engine.Decision{Action: engine.ActionAllow}, nil
	}

	stats := w.getStats(ctx.RealIP)

	if w.maxFramesPerMin > 0 && stats.Count > w.maxFramesPerMin {
		if w.blockOnLimit {
			return &engine.Decision{
				Action:   engine.ActionBlock,
				RuleID:   "WS-001",
				RuleName: "WebSocket frame rate limit exceeded",
				Severity: "high",
				Score:    80,
				Evidence: fmt.Sprintf("frames_per_min=%d, limit=%d", stats.Count, w.maxFramesPerMin),
			}, nil
		}
		return &engine.Decision{
			Action:   engine.ActionMonitor,
			RuleID:   "WS-001",
			RuleName: "WebSocket frame rate limit exceeded",
			Severity: "medium",
			Score:    40,
			Evidence: fmt.Sprintf("frames_per_min=%d, limit=%d", stats.Count, w.maxFramesPerMin),
		}, nil
	}

	return &engine.Decision{Action: engine.ActionAllow}, nil
}

func (w *Inspector) InspectMessage(ctx *engine.RequestContext, frame Frame) (*engine.Decision, error) {
	stats := w.getStats(ctx.RealIP)

	stats.Count++
	stats.Bytes += int64(len(frame.Payload))
	stats.LastSeen = time.Now()
	stats.Types[frame.Type]++

	if frame.Type == websocket.CloseMessage && !w.enableClose {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "WS-002",
			RuleName: "WebSocket close not allowed",
			Severity: "medium",
			Score:    50,
		}, nil
	}

	if frame.Type == websocket.PingMessage && !w.enablePING {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "WS-003",
			RuleName: "WebSocket PING not allowed",
			Severity: "low",
			Score:    30,
		}, nil
	}

	if frame.Type == websocket.PongMessage && !w.enablePONG {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "WS-004",
			RuleName: "WebSocket PONG not allowed",
			Severity: "low",
			Score:    30,
		}, nil
	}

	if !w.allowedTypes[websocket.MessageType(frame.Type)] {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "WS-005",
			RuleName: "WebSocket message type not allowed",
			Severity: "high",
			Score:    75,
			Evidence: fmt.Sprintf("type=%d", frame.Type),
		}, nil
	}

	if len(frame.Payload) > w.maxFrameSize {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "WS-006",
			RuleName: "WebSocket frame size exceeded",
			Severity: "high",
			Score:    70,
			Evidence: fmt.Sprintf("frame_size=%d, limit=%d", len(frame.Payload), w.maxFrameSize),
		}, nil
	}

	if w.strictMode && frame.Type == websocket.TextMessage {
		if dec := w.checkInjection(frame.Payload); dec != nil {
			return dec, nil
		}
	}

	if w.maxDepth > 0 && frame.Type == websocket.TextMessage {
		if dec := w.checkJSONDepth(frame.Payload); dec != nil {
			return dec, nil
		}
	}

	return &engine.Decision{Action: engine.ActionAllow}, nil
}

func (w *Inspector) isWebSocket(ctx *engine.RequestContext) bool {
	upgrade := strings.ToLower(ctx.Headers["Upgrade"])
	connection := strings.ToLower(ctx.Headers["Connection"])
	wsKey := ctx.Headers["Sec-Websocket-Key"]
	wsVersion := ctx.Headers["Sec-Websocket-Version"]

	return strings.Contains(upgrades, "websocket") &&
		strings.Contains(connection, "upgrade") &&
		wsKey != "" &&
		wsVersion == "13"
}

var upgrades string = "websocket"

func (w *Inspector) checkInjection(payload []byte) *engine.Decision {
	injPatterns := []string{
		"<script", "javascript:", "onerror=", "onload=",
		"onclick=", "onmouseover=", "eval(", "document.",
		"window.", "expression(", "url(", "href=",
	}

	str := string(payload)
	lower := strings.ToLower(str)

	for _, pattern := range injPatterns {
		if strings.Contains(lower, pattern) {
			return &engine.Decision{
				Action:   engine.ActionBlock,
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

func (w *Inspector) checkJSONDepth(payload []byte) *engine.Decision {
	var nested int
	for i := 0; i < len(payload); i++ {
		if payload[i] == '{' || payload[i] == '[' {
			nested++
			if nested > w.maxDepth {
				return &engine.Decision{
					Action:   engine.ActionBlock,
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

func (w *Inspector) getStats(ip string) *FrameStats {
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

func (w *Inspector) ParseFrame(data []byte) (Frame, error) {
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

func (w *Inspector) EncodeFrame(frame Frame) []byte {
	length := len(frame.Payload)
	headerLen := 2
	if length > 65535 {
		headerLen += 8
	} else if length > 125 {
		headerLen += 2
	}

	if length <= 125 {
		headerLen = 2
	} else if length <= 65535 {
		headerLen = 4
	} else {
		headerLen = 10
	}

	data := make([]byte, headerLen+length)
	data[0] = byte(0x80 | frame.Type)

	switch {
	case length <= 125:
		data[1] = byte(length)
	case length <= 65535:
		data[1] = 126
		binary.BigEndian.PutUint16(data[2:4], uint16(length))
	default:
		data[1] = 127
		binary.BigEndian.PutUint64(data[2:10], uint64(length))
	}

	copy(data[headerLen:], frame.Payload)
	return data
}

func (w *Inspector) ParseMessage(frame Frame) ([]Message, error) {
	var msgs []Message

	if frame.Type == websocket.TextMessage || frame.Type == websocket.BinaryMessage {
		if w.isJSON(frame.Payload) {
			msg, err := w.parseJSONMessage(frame.Payload, 0, "")
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, *msg)
		} else {
			msgs = append(msgs, Message{
				Type:    frame.Type,
				Payload: frame.Payload,
				Depth:   0,
			})
		}
	}

	return msgs, nil
}

func (w *Inspector) isJSON(payload []byte) bool {
	trimmed := bytes.TrimSpace(payload)
	return len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
}

func (w *Inspector) parseJSONMessage(data []byte, depth int, path string) (*Message, error) {
	if depth > w.maxDepth {
		return nil, fmt.Errorf("max depth exceeded")
	}

	var js interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		return &Message{
			Type:    websocket.TextMessage,
			Payload: data,
			Depth:   depth,
			Path:    path,
		}, nil
	}

	return &Message{
		Type:    websocket.TextMessage,
		Payload: data,
		Depth:   depth,
		IsArray: false,
		Path:    path,
	}, nil
}

func (w *Inspector) Upgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (w *Inspector) ServeWS(wr http.ResponseWriter, r *http.Request) *websocket.Conn {
	upgrader := w.Upgrader()
	conn, err := upgrader.Upgrade(wr, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return nil
	}
	return conn
}

func (w *Inspector) TrackConnection(ip string, add bool) {
	stats := w.getStats(ip)
	w.mu.Lock()
	if add {
		stats.Count++
	} else {
		if stats.Count > 0 {
			stats.Count--
		}
	}
	w.mu.Unlock()
}

func (w *Inspector) SetMaxFramesPerMin(n int) {
	w.mu.Lock()
	w.maxFramesPerMin = n
	w.mu.Unlock()
}

func (w *Inspector) SetMaxBytesPerMin(n int) {
	w.mu.Lock()
	w.maxBytesPerMin = n
	w.mu.Unlock()
}

var _ = strconv.IntSize
var _ = math.MaxInt
var _ = bufio.NewReader
