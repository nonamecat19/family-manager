package database

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type pingFunc func(context.Context) error

func (f pingFunc) Ping(ctx context.Context) error { return f(ctx) }

func TestHealthHandlerOK(t *testing.T) {
	rec := httptest.NewRecorder()
	HealthHandler(pingFunc(func(context.Context) error { return nil }), 0).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

func TestHealthHandlerReportsUnavailable(t *testing.T) {
	rec := httptest.NewRecorder()
	HealthHandler(pingFunc(func(context.Context) error { return errors.New("down") }), 0).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

// The point of the timeout: a wedged database must produce a 503, not a handler that never
// returns. A check that hangs is one an orchestrator cannot act on.
func TestHealthHandlerBoundsAHangingPing(t *testing.T) {
	ping := pingFunc(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	done := make(chan int, 1)
	go func() {
		rec := httptest.NewRecorder()
		HealthHandler(ping, 20*time.Millisecond).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		done <- rec.Code
	}()

	select {
	case code := <-done:
		if code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("health check did not return; the ping is unbounded")
	}
}

// A client that hangs up must release the check, not leave it pinning a pool connection until
// the timeout expires.
func TestHealthHandlerFollowsTheRequestContext(t *testing.T) {
	released := make(chan struct{})
	ping := pingFunc(func(ctx context.Context) error {
		<-ctx.Done()
		close(released)
		return ctx.Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil).WithContext(ctx)

	go HealthHandler(ping, time.Minute).ServeHTTP(httptest.NewRecorder(), req)
	cancel()

	select {
	case <-released:
	case <-time.After(2 * time.Second):
		t.Fatal("ping outlived the cancelled request")
	}
}
