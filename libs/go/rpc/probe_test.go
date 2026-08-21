package rpc

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// listenerOn starts srv on loopback and returns the port, so the probe is exercised through a
// real socket rather than a handler call.
func listenerOn(t *testing.T, h http.Handler) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	_, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatalf("split %q: %v", srv.URL, err)
	}
	return port
}

func TestProbeSucceedsOnHealthy(t *testing.T) {
	port := listenerOn(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf("probed %q, want /healthz", r.URL.Path)
		}
		_, _ = w.Write([]byte("ok"))
	}))

	if err := Probe(port, time.Second); err != nil {
		t.Fatalf("Probe: %v", err)
	}
}

func TestProbeFailsOnUnhealthy(t *testing.T) {
	port := listenerOn(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "db unavailable", http.StatusServiceUnavailable)
	}))

	if err := Probe(port, time.Second); err == nil {
		t.Fatal("Probe() = nil, want an error for a 503")
	}
}

func TestProbeFailsWhenNothingIsListening(t *testing.T) {
	// Port 1 is privileged and nothing in a container binds it.
	if err := Probe("1", 200*time.Millisecond); err == nil {
		t.Fatal("Probe() = nil, want an error when the port is closed")
	}
}

// A probe that outlives the health handler's own database timeout reports "timeout" for
// something the service would have answered.
func TestProbeGivesUp(t *testing.T) {
	port := listenerOn(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(2 * time.Second)
	}))

	start := time.Now()
	if err := Probe(port, 100*time.Millisecond); err == nil {
		t.Fatal("Probe() = nil, want a timeout")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Probe took %v, want it bounded by the timeout", elapsed)
	}
}
