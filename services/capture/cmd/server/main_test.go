package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/rpc"
	"github.com/nnc/family-manager/sdk/go/capture/v1/capturev1connect"
)

const testKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

type pinger struct{ err error }

func (p pinger) Ping(context.Context) error { return p.err }

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestRunMissingConfig(t *testing.T) {
	t.Setenv("CAPTURE_DATABASE_URL", "")
	if err := run(); err == nil {
		t.Fatal("expected error from missing config, got nil")
	}
}

func TestRunFailsWithoutSealKey(t *testing.T) {
	t.Setenv("CAPTURE_DATABASE_URL", "postgres://x")
	t.Setenv("CAPTURE_JWKS_URL", "http://auth/.well-known/jwks.json")
	t.Setenv("CAPTURE_SEAL_KEY", "")
	err := run()
	if err == nil || !strings.Contains(err.Error(), "CAPTURE_SEAL_KEY") {
		t.Fatalf("expected a seal key error, got %v", err)
	}
}

func TestRunFailsWithMalformedSealKey(t *testing.T) {
	t.Setenv("CAPTURE_DATABASE_URL", "postgres://x")
	t.Setenv("CAPTURE_JWKS_URL", "http://auth/.well-known/jwks.json")
	t.Setenv("CAPTURE_SEAL_KEY", "c2hvcnQ=")
	err := run()
	if err == nil || !strings.Contains(err.Error(), "CAPTURE_SEAL_KEY") {
		t.Fatalf("expected a seal key error, got %v", err)
	}
}

func verifier(t *testing.T) *fmauth.Verifier {
	t.Helper()
	v, err := fmauth.NewVerifier(fmauth.VerifierConfig{JWKSURL: "http://127.0.0.1:1/jwks.json"})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func get(t *testing.T, mux http.Handler, path string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code
}

func TestHealthzOnBothListeners(t *testing.T) {
	ok := pinger{}
	down := pinger{err: errors.New("db down")}
	cases := []struct {
		name string
		mux  http.Handler
		want int
	}{
		{"public up", publicMux(unimplemented{}, verifier(t), ok, quiet()), http.StatusOK},
		{"public down", publicMux(unimplemented{}, verifier(t), down, quiet()), http.StatusServiceUnavailable},
		{"internal up", internalMux(unimplemented{}, ok, quiet()), http.StatusOK},
		{"internal down", internalMux(unimplemented{}, down, quiet()), http.StatusServiceUnavailable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := get(t, c.mux, "/healthz"); got != c.want {
				t.Fatalf("status = %d, want %d", got, c.want)
			}
		})
	}
}

func TestMetricsOnlyOnInternal(t *testing.T) {
	if got := get(t, internalMux(unimplemented{}, pinger{}, quiet()), rpc.MetricsPath); got != http.StatusOK {
		t.Fatalf("internal metrics status = %d, want 200", got)
	}
	if got := get(t, publicMux(unimplemented{}, verifier(t), pinger{}, quiet()), rpc.MetricsPath); got != http.StatusNotFound {
		t.Fatalf("public metrics status = %d, want 404", got)
	}
}

func post(t *testing.T, mux http.Handler, procedure string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, procedure, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	return rec.Code
}

func TestPublicRequiresAuth(t *testing.T) {
	mux := publicMux(unimplemented{}, verifier(t), pinger{}, quiet())
	if got := post(t, mux, capturev1connect.CaptureServiceListSuggestionsProcedure); got != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", got)
	}
}

func TestInternalServesUnimplemented(t *testing.T) {
	mux := internalMux(unimplemented{}, pinger{}, quiet())
	if got := post(t, mux, capturev1connect.CaptureServiceListSuggestionsProcedure); got != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", got)
	}
}
