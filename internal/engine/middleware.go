package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type CAPTCHAVerifier struct {
	enabled  bool
	provider string
	secret   string
	siteKey  string
	score    float64
	client   *http.Client
}

func NewCAPTCHAVerifier(provider, secret, siteKey string, score float64) *CAPTCHAVerifier {
	return &CAPTCHAVerifier{
		enabled:  true,
		provider: provider,
		secret:   secret,
		siteKey:  siteKey,
		score:    score,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (cv *CAPTCHAVerifier) Name() string { return "captcha" }

func (cv *CAPTCHAVerifier) Inspect(ctx *RequestContext) (*Decision, error) {
	if !cv.enabled {
		return nil, nil
	}
	token := ctx.Request.Header.Get("X-CAPTCHA-Token")
	if token == "" {
		token = ctx.Request.Header.Get("X-Recaptcha-Token")
	}
	if token == "" {
		return nil, nil
	}
	ok, score, err := cv.verify(token)
	if err != nil {
		return nil, fmt.Errorf("captcha verify: %w", err)
	}
	if !ok {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "CAPTCHA001",
			RuleName: "CAPTCHA Verification Failed",
			Severity: "low",
			Score:    30,
			Evidence: fmt.Sprintf("CAPTCHA score %f below threshold %f", score, cv.score),
		}, nil
	}
	return nil, nil
}

func (cv *CAPTCHAVerifier) verify(token string) (bool, float64, error) {
	switch cv.provider {
	case "recaptcha":
		return cv.verifyRecaptcha(token)
	case "hcaptcha":
		return cv.verifyHCaptcha(token)
	default:
		return false, 0, fmt.Errorf("unsupported captcha provider: %s", cv.provider)
	}
}

func (cv *CAPTCHAVerifier) verifyRecaptcha(token string) (bool, float64, error) {
	data := url.Values{
		"secret":   {cv.secret},
		"response": {token},
	}
	resp, err := cv.client.PostForm("https://www.google.com/recaptcha/api/siteverify", data)
	if err != nil {
		return false, 0, fmt.Errorf("recaptcha verify: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool    `json:"success"`
		Score   float64 `json:"score"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, 0, fmt.Errorf("recaptcha decode: %w", err)
	}
	return result.Success && result.Score >= cv.score, result.Score, nil
}

func (cv *CAPTCHAVerifier) verifyHCaptcha(token string) (bool, float64, error) {
	data := url.Values{
		"secret":   {cv.secret},
		"response": {token},
	}
	resp, err := cv.client.PostForm("https://hcaptcha.com/siteverify", data)
	if err != nil {
		return false, 0, fmt.Errorf("hcaptcha verify: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool    `json:"success"`
		Score   float64 `json:"score"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, 0, fmt.Errorf("hcaptcha decode: %w", err)
	}
	return result.Success && result.Score >= cv.score, result.Score, nil
}

type ResponseWriter struct {
	http.ResponseWriter
	StatusCode  int
	Body        []byte
	InspectBody bool
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if rw.InspectBody {
		rw.Body = append(rw.Body, b...)
	}
	return rw.ResponseWriter.Write(b)
}

type ResponseInspector struct {
	enabled bool
}

func NewResponseInspector() *ResponseInspector {
	return &ResponseInspector{enabled: true}
}

func (ri *ResponseInspector) Name() string { return "response_inspect" }

func (ri *ResponseInspector) Inspect(ctx *RequestContext) (*Decision, error) {
	return nil, nil
}

type SOAPValidator struct {
	enabled      bool
	strictSchema bool
	maxDepth     int
}

func NewSOAPValidator(strictSchema bool, maxDepth int) *SOAPValidator {
	if maxDepth <= 0 {
		maxDepth = 10
	}
	return &SOAPValidator{enabled: true, strictSchema: strictSchema, maxDepth: maxDepth}
}

func (sv *SOAPValidator) Name() string { return "soap" }

func (sv *SOAPValidator) Inspect(ctx *RequestContext) (*Decision, error) {
	if !sv.enabled {
		return nil, nil
	}
	if ctx.ContentType != "text/xml" && ctx.ContentType != "application/soap+xml" {
		return nil, nil
	}
	depth := 0
	openTags := 0
	for _, b := range ctx.Body {
		if b == '<' {
			openTags++
			depth++
			if depth > sv.maxDepth {
				return &Decision{
					Action:   ActionBlock,
					RuleID:   "SOAP001",
					RuleName: "SOAP/XML Depth Exceeded",
					Severity: "medium",
					Score:    50,
					Evidence: fmt.Sprintf("XML nesting depth exceeded max of %d", sv.maxDepth),
				}, nil
			}
		}
		if b == '>' {
			openTags--
			if openTags < 0 {
				return &Decision{
					Action:   ActionBlock,
					RuleID:   "SOAP002",
					RuleName: "Malformed XML",
					Severity: "medium",
					Score:    50,
					Evidence: "unexpected closing tag",
				}, nil
			}
		}
	}
	return nil, nil
}

type GRPCInspector struct {
	enabled    bool
	maxMsgSize int
	rateLimit  int
	counters   map[string]*grpcCounter
	mu         sync.Mutex
}

type grpcCounter struct {
	count     int
	resetTime time.Time
}

func NewGRPCInspector(maxMsgSize, rateLimit int) *GRPCInspector {
	if maxMsgSize <= 0 {
		maxMsgSize = 4 * 1024 * 1024
	}
	if rateLimit <= 0 {
		rateLimit = 100
	}
	return &GRPCInspector{
		enabled:    true,
		maxMsgSize: maxMsgSize,
		rateLimit:  rateLimit,
		counters:   make(map[string]*grpcCounter),
	}
}

func (gi *GRPCInspector) Name() string { return "grpc" }

func (gi *GRPCInspector) Inspect(ctx *RequestContext) (*Decision, error) {
	if !gi.enabled {
		return nil, nil
	}
	if !strings.HasPrefix(ctx.ContentType, "application/grpc") {
		return nil, nil
	}

	service := ctx.Path
	gi.mu.Lock()
	defer gi.mu.Unlock()

	now := time.Now()
	counter, exists := gi.counters[service]
	if !exists || now.Sub(counter.resetTime) > time.Minute {
		gi.counters[service] = &grpcCounter{count: 1, resetTime: now}
		return nil, nil
	}

	counter.count++
	if counter.count > gi.rateLimit {
		return &Decision{
			Action:   ActionRateLimit,
			RuleID:   "GRPC001",
			RuleName: "gRPC Rate Limit",
			Severity: "medium",
			Score:    60,
			Evidence: fmt.Sprintf("gRPC %s exceeded rate limit of %d req/min", service, gi.rateLimit),
		}, nil
	}

	if ctx.Request != nil && ctx.Request.ContentLength > int64(gi.maxMsgSize) {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "GRPC002",
			RuleName: "gRPC Message Too Large",
			Severity: "medium",
			Score:    40,
			Evidence: fmt.Sprintf("gRPC message size %d exceeds max %d", ctx.Request.ContentLength, gi.maxMsgSize),
		}, nil
	}

	return nil, nil
}
