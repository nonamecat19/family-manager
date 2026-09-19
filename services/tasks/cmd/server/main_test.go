package main

import "testing"

func TestRunMissingConfig(t *testing.T) {
	t.Setenv("TASKS_DATABASE_URL", "")
	if err := run(); err == nil {
		t.Fatal("expected error from missing config, got nil")
	}
}
