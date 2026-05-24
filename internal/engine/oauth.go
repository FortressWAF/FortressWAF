package engine

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zulfff/FortressWAF/internal/config"
)

type OAuthIntrospector struct {
	mu              sync.RWMutex
	introspectionURL string
	clientID        string
	clientSecret    string
	tokenTypeHint   string
	client          *http.Client
	cache           *tokenCache
}

type tokenCache struct {
	mu    sync.RWMutex
	tokens map[string]*cachedToken
	ttl    time.Duration
}

type cachedToken struct {
	token     string
	info      *TokenInfo
	expiresAt time.Time
}

type TokenInfo struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope"`
	ClientID  string   `json:"client_id"`
	Username  string   `json:"username"`
	TokenType string   `json:"token_type"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	NotBefore int64    `json:"nbf"`
	Subject   string   `json:"sub"`
	Audience  string   `json:"aud"`
	Roles     []string `json:"roles"`
}

func NewOAuthIntrospector(cfg config.OAuthConfig) *OAuthIntrospector {
	return &OAuthIntrospector{
		introspectionURL: cfg.IntrospectionURL,
		clientID:        cfg.ClientID,
		clientSecret:    cfg.ClientSecret,
		tokenTypeHint:   cfg.TokenTypeHint,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: &tokenCache{
			tokens: make(map[string]*cachedToken),
			ttl:    time.Minute * 5,
		},
	}
}

func (o *OAuthIntrospector) Name() string { return "oauth_introspection" }

func (o *OAuthIntrospector) Inspect(ctx *RequestContext) (*Decision, error) {
	auth := ctx.Headers["Authorization"]
	if auth == "" {
		auth = ctx.Headers["authorization"]
	}
	if auth == "" {
		return &Decision{Action: ActionAllow}, nil
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		return &Decision{Action: ActionAllow}, nil
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		return &Decision{Action: ActionAllow}, nil
	}

	info, err := o.Introspect(token)
	if err != nil {
		slog.Debug("oauth introspection failed", "error", err)
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "OAUTH-001",
			RuleName: "OAuth token introspection failed",
			Severity: "high",
			Score:    80,
			Evidence: err.Error(),
		}, nil
	}

	if !info.Active {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "OAUTH-002",
			RuleName: "OAuth token inactive",
			Severity: "high",
			Score:    90,
			Evidence: "token is not active",
		}, nil
	}

	ctx.UserID = info.Subject

	return &Decision{Action: ActionAllow}, nil
}

func (o *OAuthIntrospector) Introspect(token string) (*TokenInfo, error) {
	if info := o.getCached(token); info != nil {
		return info, nil
	}

	if o.introspectionURL == "" {
		return nil, &introspectionError{message: "introspection URL not configured"}
	}

	req, err := http.NewRequest("POST", o.introspectionURL, strings.NewReader("token="+token))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(o.clientID, o.clientSecret)

	if o.tokenTypeHint != "" {
		req.Header.Set("Token-Type-Hint", o.tokenTypeHint)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &introspectionError{message: resp.Status}
	}

	var info TokenInfo
	if err := decodeJSONResponse(resp, &info); err != nil {
		return nil, err
	}

	o.cacheToken(token, &info)

	return &info, nil
}

func (o *OAuthIntrospector) getCached(token string) *TokenInfo {
	o.cache.mu.RLock()
	defer o.cache.mu.RUnlock()

	cached, ok := o.cache.tokens[token]
	if !ok {
		return nil
	}

	if time.Now().After(cached.expiresAt) {
		return nil
	}

	return cached.info
}

func (o *OAuthIntrospector) cacheToken(token string, info *TokenInfo) {
	o.cache.mu.Lock()
	defer o.cache.mu.Unlock()

	var expiresAt time.Time
	if info.ExpiresAt > 0 {
		expiresAt = time.Unix(info.ExpiresAt, 0)
	} else {
		expiresAt = time.Now().Add(o.cache.ttl)
	}

	o.cache.tokens[token] = &cachedToken{
		token:     token,
		info:      info,
		expiresAt: expiresAt,
	}
}

func (o *OAuthIntrospector) HasScope(info *TokenInfo, scope string) bool {
	if info.Scope == "" {
		return false
	}
	scopes := strings.Split(info.Scope, " ")
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func (o *OAuthIntrospector) HasRole(info *TokenInfo, role string) bool {
	for _, r := range info.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type introspectionError struct {
	message string
}

func (e *introspectionError) Error() string {
	return e.message
}

func decodeJSONResponse(resp *http.Response, v interface{}) error {
	return nil
}

var _ = time.Second
