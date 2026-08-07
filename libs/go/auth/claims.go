// Package auth verifies the access tokens minted by services/auth and exposes the verified
// identity to handlers. Services never parse a JWT themselves.
//
// Tokens are ES256 (see docs/adr/0005-auth.md). Only services/auth holds the private key;
// everyone else verifies with the public JWKS.
package auth

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the verified payload of an access token.
type Claims struct {
	// UserID is the token subject.
	UserID string
	// FamilyID is the family the user belongs to, "" when they are in none yet.
	FamilyID string
	// Email is informational; authorization never keys off it.
	Email string
}

type ctxKey int

const claimsKey ctxKey = 0

// Errors callers are expected to branch on.
var (
	ErrNoToken      = errors.New("auth: no bearer token")
	ErrInvalidToken = errors.New("auth: invalid token")
	ErrNoClaims     = errors.New("auth: no verified claims on context")
	ErrNoFamily     = errors.New("auth: caller belongs to no family")
)

// WithClaims stores verified claims on the context. Only the interceptors call this.
func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

// FromContext returns the verified claims, or ErrNoClaims when the request was not
// authenticated. A handler that needs identity must treat the error as unauthenticated.
func FromContext(ctx context.Context) (*Claims, error) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	if !ok || c == nil {
		return nil, ErrNoClaims
	}
	return c, nil
}

// UserID is the common case of FromContext: the caller's id, "" when unauthenticated.
func UserID(ctx context.Context) string {
	c, err := FromContext(ctx)
	if err != nil {
		return ""
	}
	return c.UserID
}

// FamilyID is the caller's family, "" when unauthenticated or not in a family.
func FamilyID(ctx context.Context) string {
	c, err := FromContext(ctx)
	if err != nil {
		return ""
	}
	return c.FamilyID
}

// claimsFromJWT maps the registered + custom claims onto our struct.
func claimsFromJWT(mc jwt.MapClaims) (*Claims, error) {
	sub, err := mc.GetSubject()
	if err != nil || sub == "" {
		return nil, ErrInvalidToken
	}
	c := &Claims{UserID: sub}
	if v, ok := mc["family_id"].(string); ok {
		c.FamilyID = v
	}
	if v, ok := mc["email"].(string); ok {
		c.Email = v
	}
	return c, nil
}
