// Package token mints the access tokens every other service verifies, and publishes the
// public half as a JWKS.
//
// ES256 (docs/adr/0005-auth.md): the private key lives only here, and siblings verify with a
// public key they fetch. A shared HMAC secret would mean every service could mint tokens, and
// a single leaked config would compromise the whole system.
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

// Claims carried by an access token. Everything a sibling service needs to authorize a
// request must be here, or it will have to make a network call per request to find out.
type Claims struct {
	UserID string
	Email  string
	// FamilyID is empty for a user who has not created or joined a household yet.
	FamilyID string
}

// Signer mints access tokens and exposes the matching JWKS.
type Signer struct {
	key      *ecdsa.PrivateKey
	kid      string
	issuer   string
	audience string
	ttl      time.Duration
	now      func() time.Time
}

type Config struct {
	// PrivateKeyPEM is a PKCS#8 or SEC1 EC private key on the P-256 curve.
	PrivateKeyPEM string
	Issuer        string
	Audience      string
	// TTL is the access-token lifetime. Short by design: revocation is handled by the
	// refresh chain, so an access token only has to be short enough that a stolen one
	// expires before it is worth much.
	TTL time.Duration
	// Now is injected by tests.
	Now func() time.Time
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

// TTL is the access-token lifetime, which the response reports as expires_in.
func (s *Signer) TTL() time.Duration { return s.ttl }

// Sign returns a signed ES256 access token.
func (s *Signer) Sign(c Claims) (string, error) {
	now := s.now()

	claims := jwt.MapClaims{
		"sub": c.UserID,
		"iss": s.issuer,
		"aud": s.audience,
		"iat": now.Unix(),
		"exp": now.Add(s.ttl).Unix(),
		// nbf guards against clock skew making a fresh token invalid on a sibling host.
		"nbf": now.Add(-30 * time.Second).Unix(),
	}
	if c.Email != "" {
		claims["email"] = c.Email
	}
	// Absent rather than empty: a family-less user has no family, and "" would read as one.
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

// JWKS is the public key set siblings fetch, in the shape libs/go/auth parses.
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

// JWKS returns the public half. The private key is never reachable from this type.
func (s *Signer) JWKS() JWKS {
	pub := s.key.PublicKey
	// Coordinates are fixed-width for P-256: left-padding matters, since a coordinate with a
	// leading zero byte would otherwise decode to the wrong number.
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

// keyID is a stable thumbprint of the public key, so a rotated key gets a new kid and
// verifiers can hold both during the overlap.
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
