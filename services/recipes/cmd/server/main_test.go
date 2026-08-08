package main

import (
	"testing"
)

// A smoke test that run() returns a config error when required env vars are missing — proves
// the binary wires up without standing up Postgres or NATS.
func TestRunMissingConfig(t *testing.T) {
	err := run()
	if err == nil {
		t.Fatal("expected error from missing config, got nil")
	}
}