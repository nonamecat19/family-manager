package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/logger"
)

type stubVerifier struct {
	claims *Claims
	err    error
	seen   string
}

func (s *stubVerifier) Verify(_ context.Context, token string) (*Claims, error) {
	s.seen = token
	return s.claims, s.err
}

func TestBearerToken(t *testing.T) {
	cases := map[string]string{
		"Bearer abc":   "abc",
		"bearer abc":   "abc",
		"BEARER  abc ": "abc",
		"Basic abc":    "",
		"":             "",
		"Bearer":       "",
	}
	for header, want := range cases {
		if got := BearerToken(header); got != want {
			t.Errorf("BearerToken(%q) = %q, want %q", header, got, want)
		}
	}
}

func TestInterceptorRejectsMissingToken(t *testing.T) {
	v := &stubVerifier{claims: &Claims{UserID: "u1"}}
	next := func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		t.Fatal("handler ran for an unauthenticated request")
		return nil, nil
	}

	_, err := Interceptor(v)(next)(context.Background(), connect.NewRequest(&struct{}{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated (err=%v)", connect.CodeOf(err), err)
	}
}

func TestInterceptorRejectsBadToken(t *testing.T) {
	v := &stubVerifier{err: ErrInvalidToken}
	req := connect.NewRequest(&struct{}{})
	req.Header().Set("Authorization", "Bearer nope")

	next := func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		t.Fatal("handler ran for an invalid token")
		return nil, nil
	}

	_, err := Interceptor(v)(next)(context.Background(), req)
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
	if v.seen != "nope" {
		t.Errorf("verifier saw %q, want %q", v.seen, "nope")
	}
}

func TestInterceptorPutsClaimsOnContext(t *testing.T) {
	v := &stubVerifier{claims: &Claims{UserID: "u1", FamilyID: "f1"}}
	req := connect.NewRequest(&struct{}{})
	req.Header().Set("Authorization", "Bearer good")

	var gotUser, gotFamily string
	next := func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		gotUser, gotFamily = UserID(ctx), FamilyID(ctx)
		return connect.NewResponse(&struct{}{}), nil
	}

	if _, err := Interceptor(v)(next)(context.Background(), req); err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if gotUser != "u1" || gotFamily != "f1" {
		t.Errorf("claims on context = (%q,%q), want (u1,f1)", gotUser, gotFamily)
	}
}

func TestFromContextWithoutClaims(t *testing.T) {
	if _, err := FromContext(context.Background()); !errors.Is(err, ErrNoClaims) {
		t.Fatalf("err = %v, want ErrNoClaims", err)
	}
	if UserID(context.Background()) != "" {
		t.Error("UserID on a bare context should be empty")
	}
}

func TestRequireFamily(t *testing.T) {
	noFamily := WithClaims(context.Background(), &Claims{UserID: "u1"})
	if _, err := RequireFamily(noFamily); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want failed_precondition", connect.CodeOf(err))
	}

	withFamily := WithClaims(context.Background(), &Claims{UserID: "u1", FamilyID: "f1"})
	c, err := RequireFamily(withFamily)
	if err != nil {
		t.Fatalf("RequireFamily: %v", err)
	}
	if c.FamilyID != "f1" {
		t.Errorf("FamilyID = %q, want f1", c.FamilyID)
	}
}

// The interceptor is the only place that knows who the caller is. If it does not put the id on
// the context, every log line downstream is anonymous.
func TestInterceptorStampsTheUserIDForLogging(t *testing.T) {
	v := &stubVerifier{claims: &Claims{UserID: "u1", FamilyID: "f1"}}
	req := connect.NewRequest(&struct{}{})
	req.Header().Set("Authorization", "Bearer good")

	var buf bytes.Buffer
	log := slog.New(logger.ContextHandler(slog.NewTextHandler(&buf, nil)))

	next := func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		if got := logger.UserID(ctx); got != "u1" {
			t.Errorf("logger.UserID(ctx) = %q, want %q", got, "u1")
		}
		log.InfoContext(ctx, "handler ran")
		return connect.NewResponse(&struct{}{}), nil
	}

	if _, err := Interceptor(v)(next)(context.Background(), req); err != nil {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(buf.String(), "user_id=u1") {
		t.Fatalf("log line = %q, want it stamped with the user id", buf.String())
	}
}
