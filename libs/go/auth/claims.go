package auth

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string
	FamilyID string
	Email    string
}

type ctxKey int

const claimsKey ctxKey = 0

var (
	ErrNoToken      = errors.New("auth: no bearer token")
	ErrInvalidToken = errors.New("auth: invalid token")
	ErrNoClaims     = errors.New("auth: no verified claims on context")
	ErrNoFamily     = errors.New("auth: caller belongs to no family")
)

func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func FromContext(ctx context.Context) (*Claims, error) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	if !ok || c == nil {
		return nil, ErrNoClaims
	}
	return c, nil
}

func UserID(ctx context.Context) string {
	c, err := FromContext(ctx)
	if err != nil {
		return ""
	}
	return c.UserID
}

func FamilyID(ctx context.Context) string {
	c, err := FromContext(ctx)
	if err != nil {
		return ""
	}
	return c.FamilyID
}

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
