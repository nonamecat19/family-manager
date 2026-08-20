package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// VerifierConfig configures token verification.
type VerifierConfig struct {
	// JWKSURL is services/auth's public key set, e.g. http://auth:8080/.well-known/jwks.json.
	JWKSURL string
	// Issuer, when set, must match the token's iss claim.
	Issuer string
	// Audience, when set, must match the token's aud claim.
	Audience string
	// CacheTTL bounds how long a fetched key set is trusted. Zero means 10 minutes.
	CacheTTL time.Duration
	// HTTPClient overrides the default client (tests, mTLS).
	HTTPClient *http.Client
}

// Verifier parses and validates access tokens against a cached JWKS. It is safe for
// concurrent use.
type Verifier struct {
	cfg    VerifierConfig
	client *http.Client
	parser *jwt.Parser

	mu        sync.RWMutex
	keys      map[string]*ecdsa.PublicKey
	fetchedAt time.Time
}

// NewVerifier builds a verifier. The key set is fetched lazily on the first token.
func NewVerifier(cfg VerifierConfig) (*Verifier, error) {
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("auth: empty JWKSURL")
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 10 * time.Minute
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	// ES256 only: accepting more algorithms is how "alg: none" bugs get in.
	opts := []jwt.ParserOption{jwt.WithValidMethods([]string{"ES256"}), jwt.WithExpirationRequired()}
	if cfg.Issuer != "" {
		opts = append(opts, jwt.WithIssuer(cfg.Issuer))
	}
	if cfg.Audience != "" {
		opts = append(opts, jwt.WithAudience(cfg.Audience))
	}

	return &Verifier{cfg: cfg, client: client, parser: jwt.NewParser(opts...), keys: map[string]*ecdsa.PublicKey{}}, nil
}

// Verify parses the token and returns its claims, or ErrInvalidToken.
func (v *Verifier) Verify(ctx context.Context, token string) (*Claims, error) {
	if token == "" {
		return nil, ErrNoToken
	}

	mc := jwt.MapClaims{}
	_, err := v.parser.ParseWithClaims(token, mc, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return v.keyForKID(ctx, kid)
	})
	if err != nil {
		// Both verbs wrap: a caller matching on ErrInvalidToken keeps working, and one
		// matching on jwt.ErrTokenExpired — which is the interesting half — now can.
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	return claimsFromJWT(mc)
}

// keyForKID serves the key from cache, refetching once when the kid is unknown — that is
// what a key rotation looks like from here.
func (v *Verifier) keyForKID(ctx context.Context, kid string) (*ecdsa.PublicKey, error) {
	if key, ok := v.cachedKey(kid); ok {
		return key, nil
	}
	if err := v.refresh(ctx); err != nil {
		return nil, err
	}
	if key, ok := v.cachedKey(kid); ok {
		return key, nil
	}
	return nil, fmt.Errorf("auth: no key for kid %q", kid)
}

func (v *Verifier) cachedKey(kid string) (*ecdsa.PublicKey, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if time.Since(v.fetchedAt) > v.cfg.CacheTTL {
		return nil, false
	}
	// A single-key JWKS may omit kid; fall back to the only key we hold.
	if kid == "" && len(v.keys) == 1 {
		for _, k := range v.keys {
			return k, true
		}
	}
	k, ok := v.keys[kid]
	return k, ok
}

func (v *Verifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return fmt.Errorf("auth: jwks request: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth: fetch jwks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: jwks status %d", resp.StatusCode)
	}

	var set jwks
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return fmt.Errorf("auth: decode jwks: %w", err)
	}

	keys, err := set.ecdsaKeys()
	if err != nil {
		return err
	}

	v.mu.Lock()
	v.keys, v.fetchedAt = keys, time.Now()
	v.mu.Unlock()
	return nil
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Kid string `json:"kid"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (s jwks) ecdsaKeys() (map[string]*ecdsa.PublicKey, error) {
	out := map[string]*ecdsa.PublicKey{}
	for _, k := range s.Keys {
		if k.Kty != "EC" || k.Crv != "P-256" {
			continue // ES256 only; anything else is not ours to trust
		}
		x, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, fmt.Errorf("auth: jwk %q bad x: %w", k.Kid, err)
		}
		y, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, fmt.Errorf("auth: jwk %q bad y: %w", k.Kid, err)
		}
		out[k.Kid] = &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(x),
			Y:     new(big.Int).SetBytes(y),
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("auth: jwks contained no P-256 keys")
	}
	return out, nil
}
