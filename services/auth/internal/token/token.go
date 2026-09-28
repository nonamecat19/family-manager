package token

import (
	"context"
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

	fmauth "github.com/nnc/family-manager/libs/go/auth"
)

type Claims struct {
	UserID   string
	Email    string
	FamilyID string
	Locale   string
	ChainID  string
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
	if c.Locale != "" {
		claims["locale"] = c.Locale
	}
	if c.ChainID != "" {
		claims["sid"] = c.ChainID
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

func (s *Signer) Verify(_ context.Context, signed string) (*fmauth.Claims, error) {
	claims, err := s.parse(signed)
	if err != nil {
		return nil, err
	}

	sub, err := claims.GetSubject()
	if err != nil || sub == "" {
		return nil, fmauth.ErrInvalidToken
	}
	out := &fmauth.Claims{UserID: sub}
	if v, ok := claims["email"].(string); ok {
		out.Email = v
	}
	if v, ok := claims["family_id"].(string); ok {
		out.FamilyID = v
	}
	if v, ok := claims["locale"].(string); ok {
		out.Locale = v
	}
	return out, nil
}

func (s *Signer) ChainOf(_ context.Context, signed string) (string, error) {
	claims, err := s.parse(signed)
	if err != nil {
		return "", err
	}
	sid, _ := claims["sid"].(string)
	return sid, nil
}

func (s *Signer) parse(signed string) (jwt.MapClaims, error) {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"ES256"}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(s.now),
	}
	if s.issuer != "" {
		opts = append(opts, jwt.WithIssuer(s.issuer))
	}
	if s.audience != "" {
		opts = append(opts, jwt.WithAudience(s.audience))
	}

	claims := jwt.MapClaims{}
	if _, err := jwt.NewParser(opts...).ParseWithClaims(signed, claims,
		func(*jwt.Token) (any, error) { return &s.key.PublicKey, nil },
	); err != nil {
		return nil, fmt.Errorf("%w: %w", fmauth.ErrInvalidToken, err)
	}
	return claims, nil
}
