package billing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

var CommunityPublicKey *ecdsa.PublicKey

func init() {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	CommunityPublicKey = &privKey.PublicKey
}

type LicenseClaims struct {
	Tier       string    `json:"tier"`
	Org        string    `json:"org"`
	Seats      int       `json:"seats"`
	Features   []string  `json:"features"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	GraceUntil time.Time `json:"grace_until"`
	Version    string    `json:"version"`
	LicenseID  string    `json:"license_id"`
}

type License struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

func NewLicense() *License {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return &License{
		PrivateKey: privKey,
		PublicKey:  &privKey.PublicKey,
	}
}

func (l *License) Generate(claims LicenseClaims) (string, error) {
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(claimsJSON)
	signature, err := ecdsa.SignASN1(rand.Reader, l.PrivateKey, hash[:])
	if err != nil {
		return "", err
	}

	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := base64.RawURLEncoding.EncodeToString(signature)

	return fmt.Sprintf("FWL-%s.%s", payload, sig), nil
}

func (l *License) Validate(token string) (*LicenseClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid license format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(payload)
	if !ecdsa.VerifyASN1(l.PublicKey, hash[:], sig) {
		return nil, errors.New("invalid signature")
	}

	var claims LicenseClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	if time.Now().After(claims.GraceUntil) {
		return nil, errors.New("license expired")
	}

	return &claims, nil
}

func (l *License) ValidateWithPublicKey(token string, pubKey *ecdsa.PublicKey) (*LicenseClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid license format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(payload)
	if !ecdsa.VerifyASN1(pubKey, hash[:], sig) {
		return nil, errors.New("invalid signature")
	}

	var claims LicenseClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	return &claims, nil
}

func (l *License) ValidateCommunityLicense(token string) (*LicenseClaims, error) {
	return l.ValidateWithPublicKey(token, CommunityPublicKey)
}

func (l *License) ValidateWithoutSignature(token string) (*LicenseClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid license format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	var claims LicenseClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	return &claims, nil
}

func GenerateTrialLicense() (string, string, error) {
	l := NewLicense()
	issued := time.Now()
	expires := issued.Add(30 * 24 * time.Hour)
	grace := expires.Add(7 * 24 * time.Hour)

	claims := LicenseClaims{
		Tier:       "professional",
		Org:        "Trial",
		Seats:      1,
		Features:   []string{"ml_engine", "api_protection", "dlp", "compliance_reports", "advanced_analytics"},
		IssuedAt:   issued,
		ExpiresAt:  expires,
		GraceUntil: grace,
		Version:    "1.0.0",
		LicenseID:  fmt.Sprintf("TRIAL-%d", time.Now().UnixNano()),
	}

	token, err := l.Generate(claims)
	return token, claims.LicenseID, err
}

func (c *LicenseClaims) IsFeatureEnabled(feature string) bool {
	for _, f := range c.Features {
		if f == feature {
			return true
		}
	}
	return false
}

func ParseLicenseID(token string) (string, error) {
	claims, err := NewLicense().ValidateWithoutSignature(token)
	if err != nil {
		return "", err
	}
	return claims.LicenseID, nil
}

func GetLicenseExpiry(token string) (time.Time, error) {
	claims, err := NewLicense().ValidateWithoutSignature(token)
	if err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt, nil
}

func GetLicenseTier(token string) (string, error) {
	claims, err := NewLicense().ValidateWithoutSignature(token)
	if err != nil {
		return "", err
	}
	return claims.Tier, nil
}

func SignClaims(claims LicenseClaims, privateKey *ecdsa.PrivateKey) (string, error) {
	l := &License{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}
	return l.Generate(claims)
}

func VerifyToken(token string, publicKey *ecdsa.PublicKey) (*LicenseClaims, error) {
	l := &License{PublicKey: publicKey}
	return l.ValidateWithPublicKey(token, publicKey)
}

func (l *License) PublicKeyToBytes() ([]byte, error) {
	return EncodePublicKey(l.PublicKey)
}

func EncodePublicKey(pub *ecdsa.PublicKey) ([]byte, error) {
	xBytes := EncodeBigInt(pub.X)
	yBytes := EncodeBigInt(pub.Y)
	result := make([]byte, 0, len(xBytes)+1+len(yBytes))
	result = append(result, xBytes...)
	result = append(result, ',')
	result = append(result, yBytes...)
	return result, nil
}

func EncodeBigInt(n *big.Int) []byte {
	return []byte(n.String())
}

func DecodeBigInt(s []byte) *big.Int {
	var n big.Int
	n.SetString(string(s), 10)
	return &n
}
