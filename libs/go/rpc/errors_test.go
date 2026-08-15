package rpc

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect"
)

func TestInternalKeepsTheCauseOutOfTheResponse(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	cause := errors.New(`ERROR: relation "recipes" does not exist (SQLSTATE 42P01)`)
	err := Internal(context.Background(), log, cause, "list recipes")

	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal", connect.CodeOf(err))
	}
	if strings.Contains(err.Error(), "SQLSTATE") || strings.Contains(err.Error(), "recipes") {
		t.Fatalf("wire message leaked the cause: %q", err.Error())
	}
	if logged := buf.String(); !strings.Contains(logged, "SQLSTATE") ||
		!strings.Contains(logged, "list recipes") {
		t.Fatalf("log line lost the cause: %q", logged)
	}
}

func TestInternalReferenceLinksResponseToLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	err := Internal(context.Background(), log, errors.New("boom"), "do thing")

	// The reference is the only shared token between the two; if it does not appear in both,
	// the response is untraceable and the whole trade is a loss.
	_, ref, ok := strings.Cut(err.Error(), "ref ")
	if !ok {
		t.Fatalf("no reference in %q", err.Error())
	}
	ref = strings.TrimSuffix(ref, ")")
	if !strings.Contains(buf.String(), ref) {
		t.Fatalf("reference %q missing from log %q", ref, buf.String())
	}
}

func TestInternalFallsBackToTheDefaultLogger(t *testing.T) {
	if err := Internal(context.Background(), nil, errors.New("boom"), "do thing"); err == nil {
		t.Fatal("Internal(nil logger) = nil, want an error")
	}
}

func TestReferenceIsDistinctPerCall(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 64; i++ {
		r := reference()
		if seen[r] {
			t.Fatalf("reference() repeated %q", r)
		}
		seen[r] = true
	}
}
