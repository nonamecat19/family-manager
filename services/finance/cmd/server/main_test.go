package main

import (
	"testing"
)

func TestRunMissingConfig(t *testing.T) {
	err := run()
	if err == nil {
		t.Fatal("expected error from missing config, got nil")
	}
}
