package auth

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/logger"
)

type TokenVerifier interface {
	Verify(ctx context.Context, token string) (*Claims, error)
}

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
			ctx = logger.WithUserID(WithClaims(ctx, claims), claims.UserID)
			return next(ctx, req)
		}
	}
}

func BearerToken(header string) string {
	const prefix = "bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func Require(ctx context.Context) (*Claims, error) {
	c, err := FromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return c, nil
}

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
