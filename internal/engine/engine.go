package engine

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Action string

const (
	ActionBlock    Action = "block"
	ActionAllow    Action = "allow"
	ActionChallenge Action = "challenge"
	ActionMonitor  Action = "monitor"
	ActionRateLimit Action = "rate_limit"
)

type Decision struct {
	Action    Action `json:"action"`
	RuleID    string `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	Severity  string `json:"severity"`
	Score     float64 `json:"score"`
	Evidence  string `json:"evidence"`
	Blocked   bool   `json:"blocked"`
}

type RequestContext struct {
	mu            sync.RWMutex
	Request       *http.Request
	Response      *http.Response
	Site          string
	RealIP        string
	UserAgent     string
	Path          string
	Method        string
	Headers       map[string]string
	Cookies       map[string]string
	QueryParams   map[string][]string
	FormParams    map[string][]string
	Body          []byte
	ContentType   string
	SessionID     string
	UserID        string
	APIKey        string
	Country       string
	ASN           uint
	BotScore      float64
	ThreatScore   float64
	AnomalyScore  float64
	Decisions     []Decision
	RiskScore     float64
	IsMobile      bool
	IsBot         bool
	IsTor         bool
	IsProxy       bool
	IsKnownAttack bool
	StartedAt     time.Time
	TLSVersion    string
	Fingerprint   string
	RequestID     string
	Tags          []string
	Context       context.Context
	Host          string
}

type ResponseContext struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Request    *RequestContext
}

func NewRequestContext(r *http.Request) *RequestContext {
	realIP := r.Header.Get("X-Forwarded-For")
	if realIP == "" {
		realIP = r.Header.Get("X-Real-IP")
	}
	if realIP == "" {
		realIP = r.RemoteAddr
	}

	ctx := &RequestContext{
		Request:     r,
		RealIP:      realIP,
		UserAgent:   r.UserAgent(),
		Path:        r.URL.Path,
		Method:      r.Method,
		Host:        r.Host,
		Headers:     make(map[string]string),
		Cookies:     make(map[string]string),
		QueryParams: make(map[string][]string),
		FormParams:  make(map[string][]string),
		StartedAt:   time.Now(),
		RequestID:   fmt.Sprintf("%x", time.Now().UnixNano()),
	}

	for k, v := range r.Header {
		ctx.Headers[k] = v[0]
	}

	for _, c := range r.Cookies() {
		ctx.Cookies[c.Name] = c.Value
	}

	qp := r.URL.Query()
	for k, v := range qp {
		ctx.QueryParams[k] = v
	}

	ctx.ContentType = r.Header.Get("Content-Type")

	return ctx
}

type Inspector interface {
	Name() string
	Inspect(ctx *RequestContext) (*Decision, error)
}

type Engine struct {
	mu          sync.RWMutex
	inspectors  []Inspector
	rules       Inspector
	ml          Inspector
	rateLimit   Inspector
	reputation  Inspector
	session     Inspector
	bot         Inspector
	ddos        Inspector
	sqli        Inspector
	xss         Inspector
	apiProtect  Inspector
	rce         Inspector
	protocol    Inspector
	upload      Inspector
	credential  Inspector
	geo         Inspector
	jwt         Inspector
	oauth       Inspector
	graphql     Inspector
	mtls        Inspector
	websocket   Inspector
	devMode     bool
}

type EngineConfig struct {
	DevMode    bool
	Rules      Inspector
	ML         Inspector
	RateLimit  Inspector
	Reputation Inspector
	Session    Inspector
	Bot        Inspector
	DDoS       Inspector
	SQLI       Inspector
	XSS        Inspector
	APIProtect Inspector
	RCE        Inspector
	Protocol   Inspector
	Upload     Inspector
	Credential Inspector
	Geo        Inspector
	JWT        Inspector
	OAuth      Inspector
	GraphQL    Inspector
	MTLS       Inspector
	WebSocket  Inspector
}

func New(cfg EngineConfig) *Engine {
	e := &Engine{
		devMode:    cfg.DevMode,
		rules:      cfg.Rules,
		ml:         cfg.ML,
		rateLimit:  cfg.RateLimit,
		reputation: cfg.Reputation,
		session:    cfg.Session,
		bot:        cfg.Bot,
		ddos:       cfg.DDoS,
		sqli:       cfg.SQLI,
		xss:        cfg.XSS,
		apiProtect: cfg.APIProtect,
		rce:        cfg.RCE,
		protocol:   cfg.Protocol,
		upload:     cfg.Upload,
		credential: cfg.Credential,
		geo:        cfg.Geo,
		jwt:        cfg.JWT,
		oauth:      cfg.OAuth,
		graphql:    cfg.GraphQL,
		mtls:       cfg.MTLS,
		websocket:  cfg.WebSocket,
	}

	e.inspectors = []Inspector{
		cfg.JWT,
		cfg.OAuth,
		cfg.MTLS,
		cfg.GraphQL,
		cfg.Reputation,
		cfg.RateLimit,
		cfg.Session,
		cfg.Rules,
		cfg.ML,
		cfg.Bot,
		cfg.DDoS,
		cfg.SQLI,
		cfg.XSS,
		cfg.APIProtect,
		cfg.RCE,
		cfg.Protocol,
		cfg.Upload,
		cfg.Credential,
		cfg.Geo,
		cfg.WebSocket,
	}

	return e
}

func (e *Engine) Inspect(ctx *RequestContext) (*Decision, error) {
	if e.devMode {
		slog.Debug("inspecting request",
			"method", ctx.Method,
			"path", ctx.Path,
			"ip", ctx.RealIP,
			"ua", ctx.UserAgent,
		)
	}

	for _, inspector := range e.inspectors {
		if inspector == nil {
			continue
		}

		dec, err := inspector.Inspect(ctx)
		if err != nil {
			slog.Error("inspector error",
				"inspector", inspector.Name(),
				"error", err,
				"request_id", ctx.RequestID,
			)
			continue
		}

		if dec == nil {
			continue
		}

		ctx.mu.Lock()
		ctx.Decisions = append(ctx.Decisions, *dec)
		ctx.ThreatScore += dec.Score
		if dec.Action == ActionBlock {
			ctx.IsKnownAttack = true
			ctx.BotScore = dec.Score
		}
		ctx.mu.Unlock()

		if e.devMode {
			slog.Debug("inspection decision",
				"inspector", inspector.Name(),
				"action", dec.Action,
				"rule_id", dec.RuleID,
				"score", dec.Score,
				"evidence", dec.Evidence,
				"request_id", ctx.RequestID,
			)
		}

		if dec.Action == ActionBlock {
			return dec, nil
		}
	}

	return e.finalDecision(ctx), nil
}

func (e *Engine) finalDecision(ctx *RequestContext) *Decision {
	ctx.mu.RLock()
	score := ctx.ThreatScore
	ctx.mu.RUnlock()

	if score >= 90 {
		return &Decision{Action: ActionBlock, Score: score, Evidence: "cumulative threat score exceeded threshold"}
	}
	if score >= 50 {
		return &Decision{Action: ActionChallenge, Score: score, Evidence: "elevated threat score requires challenge"}
	}

	ctx.mu.RLock()
	for _, d := range ctx.Decisions {
		if d.Action == ActionRateLimit {
			ctx.mu.RUnlock()
			return &Decision{Action: ActionRateLimit, Score: score, Evidence: "rate limit exceeded"}
		}
	}
	ctx.mu.RUnlock()

	return &Decision{Action: ActionAllow, Score: 0}
}

func (e *Engine) InspectRequest(r *http.Request) (*Decision, error) {
	ctx := NewRequestContext(r)

	if e.devMode {
		slog.Debug("request context created",
			"request_id", ctx.RequestID,
			"real_ip", ctx.RealIP,
		)
	}

	return e.Inspect(ctx)
}

func (e *Engine) InspectContext(ctx context.Context, r *http.Request) (*Decision, error) {
	return e.InspectRequest(r)
}

func (e *Engine) UpdateInspector(name string, inspector Inspector) {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch name {
	case "rules":
		e.rules = inspector
	case "ml":
		e.ml = inspector
	case "rate_limit":
		e.rateLimit = inspector
	case "reputation":
		e.reputation = inspector
	case "session":
		e.session = inspector
	case "bot":
		e.bot = inspector
	case "ddos":
		e.ddos = inspector
	case "sqli":
		e.sqli = inspector
	case "xss":
		e.xss = inspector
	case "api_protect":
		e.apiProtect = inspector
	case "rce":
		e.rce = inspector
	case "protocol":
		e.protocol = inspector
	case "upload":
		e.upload = inspector
	case "credential":
		e.credential = inspector
	case "geo":
		e.geo = inspector
	case "jwt":
		e.jwt = inspector
	case "oauth":
		e.oauth = inspector
	case "graphql":
		e.graphql = inspector
	case "mtls":
		e.mtls = inspector
	case "websocket":
		e.websocket = inspector
	}

	for i, ins := range e.inspectors {
		if ins != nil && ins.Name() == name {
			e.inspectors[i] = inspector
			return
		}
	}
}

func (e *Engine) Inspectors() []Inspector {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]Inspector, len(e.inspectors))
	copy(result, e.inspectors)
	return result
}

func ContextFromRequest(r *http.Request) *RequestContext {
	return NewRequestContext(r)
}
