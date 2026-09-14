package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nnc/family-manager/sdk/go/auth/v1/authv1connect"
	"github.com/nnc/family-manager/services/auth/internal/handler"
	"github.com/nnc/family-manager/services/auth/internal/token"
)

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

func testSigner(t *testing.T) *token.Signer {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	pemText := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	signer, err := token.NewSigner(token.Config{PrivateKeyPEM: pemText})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return signer
}

func TestJWKSEndpointServesThePublicKey(t *testing.T) {
	signer := testSigner(t)
	mux := newMux(handler.New(handler.Options{}), signer, okPinger{}, nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/jwk-set+json" {
		t.Errorf("content-type = %q", ct)
	}

	var set token.JWKS
	if err := json.Unmarshal(rec.Body.Bytes(), &set); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(set.Keys) != 1 {
		t.Fatalf("got %d keys, want 1", len(set.Keys))
	}
	k := set.Keys[0]
	if k.Kty != "EC" || k.Crv != "P-256" || k.Kid == "" || k.X == "" || k.Y == "" {
		t.Errorf("jwk = %+v", k)
	}
}

func TestJWKSEndpointNeverLeaksThePrivateKey(t *testing.T) {
	mux := newMux(handler.New(handler.Options{}), testSigner(t), okPinger{}, nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))

	body := rec.Body.String()
	for _, forbidden := range []string{`"d"`, "PRIVATE", "BEGIN"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("JWKS response contains %q: %s", forbidden, body)
		}
	}
}

func TestJWKSEndpointIsReadOnly(t *testing.T) {
	mux := newMux(handler.New(handler.Options{}), testSigner(t), okPinger{}, nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/.well-known/jwks.json", nil))

	if rec.Code == http.StatusOK {
		t.Error("POST to the JWKS endpoint succeeded; it is registered GET-only")
	}
}

func TestAuthProceduresAreRegistered(t *testing.T) {
	mux := newMux(handler.New(handler.Options{}), testSigner(t), okPinger{}, nil)

	for _, procedure := range []string{
		authv1connect.AuthServiceLoginProcedure,
		authv1connect.AuthServiceRegisterProcedure,
		authv1connect.AuthServiceRefreshProcedure,
		authv1connect.AuthServiceLogoutProcedure,
	} {
		_, pattern := mux.Handler(httptest.NewRequest(http.MethodPost, procedure, nil))
		if pattern == "" {
			t.Errorf("%s is not routed", procedure)
		}
	}
}

func TestHealthzReportsOK(t *testing.T) {
	mux := newMux(handler.New(handler.Options{}), testSigner(t), okPinger{}, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("healthz = %d, want 200", rec.Code)
	}
}

func TestOversizeRequestIsRejected(t *testing.T) {
	mux := newMux(handler.New(handler.Options{}), testSigner(t), okPinger{}, nil)

	body := `{"email":"a@b.co","password":"correct horse","name":"` +
		strings.Repeat("x", maxRequestBytes+1) + `"}`
	req := httptest.NewRequest(http.MethodPost,
		authv1connect.AuthServiceRegisterProcedure, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 (body=%s)", rec.Code, rec.Body.String())
	}
}

type countingSweeper struct {
	calls chan struct{}
	err   error
}

func (c *countingSweeper) DeleteExpiredRefreshTokens(context.Context) (int64, error) {
	c.calls <- struct{}{}
	return 3, c.err
}

func TestSweepRunsBeforeTheFirstTick(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &countingSweeper{calls: make(chan struct{}, 4)}
	go sweepExpiredTokens(ctx, s, slog.Default(), time.Hour)

	select {
	case <-s.calls:
	case <-time.After(2 * time.Second):
		t.Fatal("no sweep before the first tick")
	}
}

func TestSweepStopsWithTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &countingSweeper{calls: make(chan struct{}, 64)}

	done := make(chan struct{})
	go func() {
		sweepExpiredTokens(ctx, s, slog.Default(), 10*time.Millisecond)
		close(done)
	}()

	<-s.calls
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sweep outlived its context")
	}
}

func TestSweepKeepsGoingAfterAnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &countingSweeper{calls: make(chan struct{}, 64), err: errors.New("db down")}
	go sweepExpiredTokens(ctx, s, slog.Default(), 10*time.Millisecond)

	for i := 0; i < 2; i++ {
		select {
		case <-s.calls:
		case <-time.After(2 * time.Second):
			t.Fatalf("sweep stopped after %d call(s)", i)
		}
	}
}
