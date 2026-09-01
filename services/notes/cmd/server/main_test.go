package main

import (
	"testing"
)

// A smoke test that run() returns a config error when required env vars are missing — proves
// the binary wires up without standing up Postgres or NATS. Every other service's cmd/server
// carries the same test; this one was the exception, so a wiring mistake in main.go would have
// been caught by nothing until a container failed to boot.
func TestRunMissingConfig(t *testing.T) {
	err := run()
	if err == nil {
		t.Fatal("expected error from missing config, got nil")
	}
}
