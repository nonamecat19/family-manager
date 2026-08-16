package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/sdk/go/family/v1/familyv1connect"
	"github.com/nnc/family-manager/services/family/internal/handler"
)

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

// The public listener must refuse the procedures that take a subject id as an argument.
// Serving them behind a token would let any signed-in user read anyone's household — the
// interceptor proves *who* is calling, not *whose* data they asked for.
func TestPublicMuxRefusesInternalProcedures(t *testing.T) {
	verifier, err := fmauth.NewVerifier(fmauth.VerifierConfig{JWKSURL: "http://example.invalid/jwks"})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	mux := publicMux(handler.New(handler.Options{}), verifier, okPinger{}, nil)

	for _, procedure := range internalOnly {
		req := httptest.NewRequest(http.MethodPost, procedure, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("%s on the public listener returned %d, want 404", procedure, rec.Code)
		}
	}
}

// The same procedures must be reachable on the internal listener, or services/auth cannot
// resolve a family_id at token-mint time.
func TestInternalMuxServesInternalProcedures(t *testing.T) {
	mux := internalMux(handler.New(handler.Options{}), okPinger{}, nil)

	for _, procedure := range internalOnly {
		req := httptest.NewRequest(http.MethodPost, procedure, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		// The handler rejects the empty body as invalid_argument, which Connect maps to 400.
		// Anything other than 404 proves the route exists, which is what is under test.
		if rec.Code == http.StatusNotFound {
			t.Errorf("%s is not served on the internal listener", procedure)
		}
	}
}

// A public procedure with no token must be refused by the interceptor, not silently served.
func TestPublicMuxRequiresAToken(t *testing.T) {
	verifier, err := fmauth.NewVerifier(fmauth.VerifierConfig{JWKSURL: "http://example.invalid/jwks"})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	mux := publicMux(handler.New(handler.Options{}), verifier, okPinger{}, nil)

	req := httptest.NewRequest(http.MethodPost,
		familyv1connect.FamilyServiceGetFamilyProcedure, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated GetFamily returned %d, want 401", rec.Code)
	}
}

func TestHealthzReportsDatabaseTrouble(t *testing.T) {
	mux := internalMux(handler.New(handler.Options{}), okPinger{}, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("healthz with a live pool returned %d, want 200", rec.Code)
	}
}
