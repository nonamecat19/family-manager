package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestContextHandlerAddsCorrelationIDs(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(&contextHandler{Handler: slog.NewJSONHandler(&buf, nil)})

	ctx := WithUserID(WithRequestID(context.Background(), "req-1"), "user-1")
	l.InfoContext(ctx, "hello")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec["request_id"] != "req-1" {
		t.Errorf("request_id = %v, want req-1", rec["request_id"])
	}
	if rec["user_id"] != "user-1" {
		t.Errorf("user_id = %v, want user-1", rec["user_id"])
	}
}

func TestContextHandlerOmitsUnsetIDs(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(&contextHandler{Handler: slog.NewJSONHandler(&buf, nil)})

	l.InfoContext(context.Background(), "hello")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := rec["request_id"]; ok {
		t.Error("request_id present on a bare context")
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"WARN":  slog.LevelWarn,
		"error": slog.LevelError,
		"":      slog.LevelInfo,
		"bogus": slog.LevelInfo,
	}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}
