package main

import (
	"io"
	"strings"
	"testing"

	"github.com/nnc/family-manager/services/auth/internal/password"
)

func TestReadPasswordTakesTheFirstLine(t *testing.T) {
	got, err := readPassword(strings.NewReader("correct horse\nignored\n"), io.Discard)
	if err != nil {
		t.Fatalf("readPassword: %v", err)
	}
	if got != "correct horse" {
		t.Fatalf("got %q, want %q", got, "correct horse")
	}
}

// A password may legitimately begin or end with a space. Trimming it would hash something the
// user did not type, and they would find out at the sign-in that fails.
func TestReadPasswordKeepsSurroundingSpaces(t *testing.T) {
	got, err := readPassword(strings.NewReader("  spaced  \n"), io.Discard)
	if err != nil {
		t.Fatalf("readPassword: %v", err)
	}
	if got != "  spaced  " {
		t.Fatalf("got %q, want %q", got, "  spaced  ")
	}
}

func TestReadPasswordAcceptsInputWithoutATrailingNewline(t *testing.T) {
	got, err := readPassword(strings.NewReader("no newline"), io.Discard)
	if err != nil {
		t.Fatalf("readPassword: %v", err)
	}
	if got != "no newline" {
		t.Fatalf("got %q, want %q", got, "no newline")
	}
}

func TestReadPasswordRejectsEmptyInput(t *testing.T) {
	for _, in := range []string{"", "\n"} {
		if _, err := readPassword(strings.NewReader(in), io.Discard); err == nil {
			t.Fatalf("readPassword(%q) = nil error, want a refusal", in)
		}
	}
}

// The whole point of the tool: what it prints must verify against what the service checks.
func TestHashOutputVerifies(t *testing.T) {
	hash, err := password.Hash("correct horse", password.DefaultParams())
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if err := password.Verify("correct horse", hash); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}
