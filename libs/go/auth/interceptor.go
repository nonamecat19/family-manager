package auth

import (
	"context"
	"strings"

	"connectrpc.com/connect"
)

// TokenVerifier is what an interceptor needs; *Verifier satisfies it, and tests substitute
// a stub instead of standing up a JWKS server.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (*Claims, error)
}

// Interceptor verifies the Authorization header on every Connect/gRPC call and puts the
// claims on the context.
//
// public lists fully-qualified procedure names that skip verification, e.g.
// "/auth.v1.AuthService/Login". Anything not listed requires a valid token.
func Interceptor(v TokenVerifier, public ...string) connect.UnaryInterceptorFunc {
	skip := make(map[string]struct{}, len(public))
	for _, p := range public {
		skip[p] = struct{}{}
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := skip[req.Spec().Procedure]; ok {
				return next(ctx, req)
			}

			token := BearerToken(req.Header().Get("Authorization"))
			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoToken)
			}

			claims, err := v.Verify(ctx, token)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}
			return next(WithClaims(ctx, claims), req)
		}
	}
}

// BearerToken extracts the token from an Authorization header value, "" when the header is
// missing or not a bearer.
func BearerToken(header string) string {
	const prefix = "bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

// Require is the handler-side guard: it returns the caller's claims or a Connect
// Unauthenticated error, so handlers never repeat the error mapping.
func Require(ctx context.Context) (*Claims, error) {
	c, err := FromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return c, nil
}

// RequireFamily is Require plus the check that the caller actually belongs to a family —
// every family-scoped resource in this repo needs it.
func RequireFamily(ctx context.Context) (*Claims, error) {
	c, err := Require(ctx)
	if err != nil {
		return nil, err
	}
	if c.FamilyID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrNoFamily)
	}
	return c, nil
}
