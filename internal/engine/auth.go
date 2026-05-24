package engine

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/FortressWAF/FortressWAF/internal/config"
)

type JWTValidator struct {
	mu         sync.RWMutex
	jwksURL    string
	jwksCache  *JWKSCache
	issuers    []string
	audiences  []string
	algorithms []string
	secret     string
	client     *http.Client
}

type JWKSCache struct {
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
	X   string `json:"x"`
	Y   string `json:"y"`
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
		secret:     cfg.Secret,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		jwksCache: &JWKSCache{
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

	hasKeySource := j.jwksURL != "" || j.secret != ""

	if headerObj.Kid != "" {
		if j.jwksURL == "" {
			return nil, fmt.Errorf("kid present but JWKS URL not configured")
		}
		key, err := j.getKey(headerObj.Kid)
		if err != nil {
			return nil, fmt.Errorf("get signing key: %w", err)
		}
		if err := j.verifySignature(parts, headerObj.Alg, key); err != nil {
			return nil, fmt.Errorf("verify signature: %w", err)
		}
	} else if j.secret != "" {
		if !strings.EqualFold(headerObj.Alg, "HS256") &&
			!strings.EqualFold(headerObj.Alg, "HS384") &&
			!strings.EqualFold(headerObj.Alg, "HS512") {
			return nil, fmt.Errorf("algorithm %s not compatible with secret key", headerObj.Alg)
		}
		signingInput := parts[0] + "." + parts[1]
		expectedSig := computeHMAC([]byte(j.secret), []byte(signingInput), headerObj.Alg)
		providedSig, err := j.decodeSegment(parts[2])
		if err != nil {
			return nil, fmt.Errorf("decode signature: %w", err)
		}
		if !hmac.Equal(expectedSig, providedSig) {
			return nil, fmt.Errorf("invalid signature")
		}
	} else if hasKeySource {
		return nil, fmt.Errorf("unable to determine signing key for token (kid=%q)", headerObj.Kid)
	} else {
		return nil, fmt.Errorf("JWT validation enabled but no signing key configured (jwks_url or secret)")
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
	j.mu.RLock()
	if time.Now().Before(j.jwksCache.expiresAt) {
		if key, ok := j.jwksCache.keys[kid]; ok {
			j.mu.RUnlock()
			return &key, nil
		}
	}
	j.mu.RUnlock()

	if j.jwksURL == "" {
		return nil, fmt.Errorf("jwks url not configured")
	}

	if err := j.refreshJWKS(); err != nil {
		return nil, err
	}

	j.mu.RLock()
	defer j.mu.RUnlock()
	if key, ok := j.jwksCache.keys[kid]; ok {
		return &key, nil
	}
	return nil, fmt.Errorf("key not found: %s", kid)
}

func (j *JWTValidator) refreshJWKS() error {
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

	j.mu.Lock()
	j.jwksCache.keys = make(map[string]JWK)
	for _, key := range jwks.Keys {
		j.jwksCache.keys[key.Kid] = key
	}
	j.jwksCache.expiresAt = time.Now().Add(j.jwksCache.ttl)
	j.mu.Unlock()

	return nil
}

func (j *JWTValidator) verifySignature(parts []string, alg string, key *JWK) error {
	sig, err := j.decodeSegment(parts[2])
	if err != nil {
		return err
	}

	data := parts[0] + "." + parts[1]

	switch alg {
	case "RS256", "RS384", "RS512":
		return j.verifyRSASignature([]byte(data), sig, key)
	case "ES256", "ES384", "ES512":
		return j.verifyECSignature([]byte(data), sig, key)
	default:
		return nil
	}
}

func (j *JWTValidator) verifyRSASignature(data, sig []byte, key *JWK) error {
	if key.N == "" || key.E == "" {
		return fmt.Errorf("incomplete RSA key: missing n or e")
	}

	nBytes, err := j.decodeSegment(key.N)
	if err != nil {
		return fmt.Errorf("decode RSA modulus: %w", err)
	}

	eBytes, err := j.decodeSegment(key.E)
	if err != nil {
		return fmt.Errorf("decode RSA exponent: %w", err)
	}

	// Decode the exponent (which is base64url-encoded big-endian bytes)
	var exp int
	if len(eBytes) >= 8 {
		exp = int(binary.BigEndian.Uint64(eBytes))
	} else {
		for _, b := range eBytes {
			exp = (exp << 8) | int(b)
		}
	}

	pub := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: exp,
	}

	hash := crypto.SHA256
	hasher := hash.New()
	hasher.Write(data)
	hashed := hasher.Sum(nil)

	return rsa.VerifyPKCS1v15(pub, hash, hashed, sig)
}

func (j *JWTValidator) verifyECSignature(data []byte, sig []byte, key *JWK) error {
	if key.Crv == "" || key.X == "" || key.Y == "" {
		return fmt.Errorf("incomplete EC key: missing crv, x, or y")
	}

	xBytes, err := j.decodeSegment(key.X)
	if err != nil {
		return fmt.Errorf("decode EC x: %w", err)
	}

	yBytes, err := j.decodeSegment(key.Y)
	if err != nil {
		return fmt.Errorf("decode EC y: %w", err)
	}

	var curve elliptic.Curve
	switch key.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return fmt.Errorf("unsupported EC curve: %s", key.Crv)
	}

	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	if !ecdsa.VerifyASN1(pub, data, sig) {
		return fmt.Errorf("ECDSA signature verification failed")
	}
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

type OAuthIntrospector struct {
	mu               sync.RWMutex
	introspectionURL string
	clientID         string
	clientSecret     string
	tokenTypeHint    string
	client           *http.Client
	cache            *tokenCache
}

type tokenCache struct {
	tokens map[string]*cachedToken
	ttl    time.Duration
}

type cachedToken struct {
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
		clientID:         cfg.ClientID,
		clientSecret:     cfg.ClientSecret,
		tokenTypeHint:    cfg.TokenTypeHint,
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
		return nil, fmt.Errorf("introspection URL not configured")
	}

	req, err := http.NewRequest("POST", o.introspectionURL, strings.NewReader(url.Values{"token": {token}}.Encode()))
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
		return nil, fmt.Errorf("introspection returned %d", resp.StatusCode)
	}

	var info TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	o.cacheToken(token, &info)

	return &info, nil
}

func (o *OAuthIntrospector) getCached(token string) *TokenInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

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
	o.mu.Lock()
	defer o.mu.Unlock()

	var expiresAt time.Time
	if info.ExpiresAt > 0 {
		expiresAt = time.Unix(info.ExpiresAt, 0)
	} else {
		expiresAt = time.Now().Add(o.cache.ttl)
	}

	o.cache.tokens[token] = &cachedToken{
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

func computeHMAC(secret, data []byte, alg string) []byte {
	var h func() hash.Hash
	switch alg {
	case "HS384":
		h = sha512.New384
	case "HS512":
		h = sha512.New
	default:
		h = sha256.New
	}
	mac := hmac.New(h, secret)
	mac.Write(data)
	return mac.Sum(nil)
}
