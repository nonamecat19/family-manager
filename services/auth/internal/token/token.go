package token

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string
	Email    string
	FamilyID string
}

type Signer struct {
	key      *ecdsa.PrivateKey
	kid      string
	issuer   string
	audience string
	ttl      time.Duration
	now      func() time.Time
}

type Config struct {
	PrivateKeyPEM string
	Issuer        string
	Audience      string
	TTL           time.Duration
	Now           func() time.Time
}

var ErrNotP256 = errors.New("token: signing key must be an ECDSA P-256 key")

func NewSigner(cfg Config) (*Signer, error) {
	key, err := parseECPrivateKey(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, err
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 15 * time.Minute
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	return &Signer{
		key:      key,
		kid:      keyID(&key.PublicKey),
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		ttl:      cfg.TTL,
		now:      cfg.Now,
	}, nil
}

func (s *Signer) TTL() time.Duration { return s.ttl }

func (s *Signer) Sign(c Claims) (string, error) {
	now := s.now()

	claims := jwt.MapClaims{
		"sub": c.UserID,
		"iss": s.issuer,
		"aud": s.audience,
		"iat": now.Unix(),
		"exp": now.Add(s.ttl).Unix(),
		"nbf": now.Add(-30 * time.Second).Unix(),
	}
	if c.Email != "" {
		claims["email"] = c.Email
	}
	if c.FamilyID != "" {
		claims["family_id"] = c.FamilyID
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = s.kid

	signed, err := tok.SignedString(s.key)
	if err != nil {
		return "", fmt.Errorf("token: sign: %w", err)
	}
	return signed, nil
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (s *Signer) JWKS() JWKS {
	pub := s.key.PublicKey
	return JWKS{Keys: []JWK{{
		Kty: "EC",
		Crv: "P-256",
		Kid: s.kid,
		Alg: "ES256",
		Use: "sig",
		X:   base64.RawURLEncoding.EncodeToString(pub.X.FillBytes(make([]byte, 32))),
		Y:   base64.RawURLEncoding.EncodeToString(pub.Y.FillBytes(make([]byte, 32))),
	}}}
}

func keyID(pub *ecdsa.PublicKey) string {
	sum := sha256.Sum256(elliptic.MarshalCompressed(elliptic.P256(), pub.X, pub.Y))
	return base64.RawURLEncoding.EncodeToString(sum[:8])
}

func parseECPrivateKey(pemText string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil, errors.New("token: signing key is not PEM")
	}

	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return checkCurve(key)
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("token: parse signing key: %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, ErrNotP256
	}
	return checkCurve(key)
}

func checkCurve(key *ecdsa.PrivateKey) (*ecdsa.PrivateKey, error) {
	if key.Curve != elliptic.P256() {
		return nil, ErrNotP256
	}
	return key, nil
}
