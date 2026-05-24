package engine

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zulfff/FortressWAF/internal/config"
)

type JWTValidator struct {
	mu         sync.RWMutex
	jwksURL    string
	jwksCache  *jwksCache
	issuers    []string
	audiences  []string
	algorithms []string
	client     *http.Client
}

type jwksCache struct {
	mu        sync.RWMutex
	keys      map[string]JWK
	expiresAt time.Time
	ttl       time.Duration
}

type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWTClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	NotBefore int64    `json:"nbf"`
	JWTID     string   `json:"jti"`
	Scope     string   `json:"scope"`
	Roles     []string `json:"roles"`
}

func NewJWTValidator(cfg config.JWTConfig) *JWTValidator {
	return &JWTValidator{
		jwksURL:    cfg.JWKSURL,
		issuers:    cfg.Issuers,
		audiences:  cfg.Audiences,
		algorithms: cfg.Algorithms,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		jwksCache: &jwksCache{
			keys: make(map[string]JWK),
			ttl:  time.Hour,
		},
	}
}

func (j *JWTValidator) Name() string { return "jwt_validation" }

func (j *JWTValidator) Inspect(ctx *RequestContext) (*Decision, error) {
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

	claims, err := j.Validate(token)
	if err != nil {
		slog.Debug("jwt validation failed", "error", err, "token_prefix", token[:min(10, len(token))])
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "JWT-001",
			RuleName: "JWT validation failed",
			Severity: "high",
			Score:    85,
			Evidence: err.Error(),
		}, nil
	}

	ctx.UserID = claims.Subject

	return &Decision{Action: ActionAllow}, nil
}

func (j *JWTValidator) Validate(tokenString string) (*JWTClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	header, err := j.decodeSegment(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}

	var headerObj struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(header, &headerObj); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	if !j.isAllowedAlgorithm(headerObj.Alg) {
		return nil, fmt.Errorf("algorithm not allowed: %s", headerObj.Alg)
	}

	if headerObj.Alg == "none" {
		return nil, fmt.Errorf("algorithm 'none' not allowed")
	}

	payload, err := j.decodeSegment(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	if err := j.validateClaims(&claims); err != nil {
		return nil, err
	}

	if headerObj.Kid != "" && j.jwksURL != "" {
		key, err := j.getKey(headerObj.Kid)
		if err != nil {
			return nil, fmt.Errorf("get key %s: %w", headerObj.Kid, err)
		}

		if err := j.verifySignature(parts, headerObj.Alg, key); err != nil {
			return nil, fmt.Errorf("verify signature: %w", err)
		}
	}

	return &claims, nil
}

func (j *JWTValidator) decodeSegment(seg string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(seg)
		if err != nil {
			return nil, err
		}
	}
	return decoded, nil
}

func (j *JWTValidator) isAllowedAlgorithm(alg string) bool {
	if len(j.algorithms) == 0 {
		return true
	}
	for _, a := range j.algorithms {
		if a == alg {
			return true
		}
	}
	return false
}

func (j *JWTValidator) validateClaims(claims *JWTClaims) error {
	now := time.Now().Unix()

	if claims.ExpiresAt > 0 && claims.ExpiresAt < now {
		return fmt.Errorf("token expired")
	}

	if claims.NotBefore > 0 && claims.NotBefore > now {
		return fmt.Errorf("token not yet valid")
	}

	if len(j.issuers) > 0 {
		found := false
		for _, iss := range j.issuers {
			if claims.Issuer == iss {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("issuer not allowed: %s", claims.Issuer)
		}
	}

	if len(j.audiences) > 0 {
		found := false
		for _, aud := range j.audiences {
			for _, c := range claims.Audience {
				if c == aud {
					found = true
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("audience not allowed")
		}
	}

	return nil
}

func (j *JWTValidator) getKey(kid string) (*JWK, error) {
	j.jwksCache.mu.RLock()
	if time.Now().Before(j.jwksCache.expiresAt) {
		if key, ok := j.jwksCache.keys[kid]; ok {
			j.jwksCache.mu.RUnlock()
			return &key, nil
		}
	}
	j.jwksCache.mu.RUnlock()

	if err := j.refreshJWKS(); err != nil {
		return nil, err
	}

	j.jwksCache.mu.RLock()
	defer j.jwksCache.mu.RUnlock()
	if key, ok := j.jwksCache.keys[kid]; ok {
		return &key, nil
	}
	return nil, fmt.Errorf("key not found: %s", kid)
}

func (j *JWTValidator) refreshJWKS() error {
	if j.jwksURL == "" {
		return fmt.Errorf("jwks url not configured")
	}

	resp, err := j.client.Get(j.jwksURL)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	j.jwksCache.mu.Lock()
	j.jwksCache.keys = make(map[string]JWK)
	for _, key := range jwks.Keys {
		j.jwksCache.keys[key.Kid] = key
	}
	j.jwksCache.expiresAt = time.Now().Add(j.jwksCache.ttl)
	j.jwksCache.mu.Unlock()

	return nil
}

func (j *JWTValidator) verifySignature(parts []string, alg string, key *JWK) error {
	sig, err := j.decodeSegment(parts[2])
	if err != nil {
		return err
	}

	data := parts[0] + "." + parts[1]

	switch alg {
	case "RS256":
		return j.verifyRS256(data, sig, key)
	case "RS384":
		return j.verifyRS384(data, sig, key)
	case "RS512":
		return j.verifyRS512(data, sig, key)
	default:
		return fmt.Errorf("unsupported algorithm: %s", alg)
	}
}

func (j *JWTValidator) verifyRS256(data string, sig, keyBytes []byte) error {
	return nil
}

func (j *JWTValidator) verifyRS384(data string, sig, keyBytes []byte) error {
	return nil
}

func (j *JWTValidator) verifyRS512(data string, sig, keyBytes []byte) error {
	return nil
}

func (j *JWTValidator) parseRSAPublicKey(key *JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

func (j *JWTValidator) HasScope(claims *JWTClaims, scope string) bool {
	if claims.Scope == "" {
		return false
	}
	scopes := strings.Split(claims.Scope, " ")
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func (j *JWTValidator) HasRole(claims *JWTClaims, role string) bool {
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}

var _ = x509.Certificate{}
