package rpc

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/logger"
)

// The logger under test is the repo's own, over a buffer: the point of routing ids through
// logger is that its handler stamps them, and a plain slog handler would not show that.
func debugLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(logger.ContextHandler(h)), &buf
}

func TestObserveMintsAnIDAndLogsTheCall(t *testing.T) {
	log, buf := debugLogger()
	next := Observe(log)(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		if RequestID(ctx) == "" {
			t.Error("handler ran without a request id in its context")
		}
		return connect.NewResponse(&struct{}{}), nil
	})

	resp, err := next(context.Background(), connect.NewRequest(&struct{}{}))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if resp.Header().Get(RequestIDHeader) == "" {
		t.Error("response carries no request id")
	}
	if logged := buf.String(); !strings.Contains(logged, "request_id") ||
		!strings.Contains(logged, "code=ok") {
		t.Errorf("log line = %q", logged)
	}
}

// A trace that starts in the app or in Caddy has to survive the hop, or it is not a trace.
func TestObserveKeepsTheCallersID(t *testing.T) {
	log, _ := debugLogger()
	req := connect.NewRequest(&struct{}{})
	req.Header().Set(RequestIDHeader, "abc123")

	var seen string
	next := Observe(log)(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		seen = RequestID(ctx)
		return connect.NewResponse(&struct{}{}), nil
	})
	resp, _ := next(context.Background(), req)

	if seen != "abc123" {
		t.Fatalf("request id in context = %q, want %q", seen, "abc123")
	}
	if got := resp.Header().Get(RequestIDHeader); got != "abc123" {
		t.Fatalf("echoed id = %q, want %q", got, "abc123")
	}
}

// The id is written verbatim into every log line for the request, so a caller must not be able
// to put a newline, a control character or a kilobyte of text there.
func TestObserveRejectsAHostileCallerID(t *testing.T) {
	hostile := []string{
		"abc\nlevel=ERROR msg=\"forged line\"",
		"abc\x00def",
		strings.Repeat("x", maxRequestIDLength+1),
		"  ",
	}
	for _, in := range hostile {
		req := connect.NewRequest(&struct{}{})
		req.Header().Set(RequestIDHeader, in)

		var seen string
		log, _ := debugLogger()
		next := Observe(log)(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			seen = RequestID(ctx)
			return connect.NewResponse(&struct{}{}), nil
		})
		if _, err := next(context.Background(), req); err != nil {
			t.Fatalf("err = %v", err)
		}
		if seen == in {
			t.Fatalf("accepted a hostile request id %q", in)
		}
		if seen == "" {
			t.Fatalf("rejected %q without minting a replacement", in)
		}
	}
}

func TestObserveLogsTheCodeOfAFailure(t *testing.T) {
	log, buf := debugLogger()
	next := Observe(log)(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("nope"))
	})

	if _, err := next(context.Background(), connect.NewRequest(&struct{}{})); err == nil {
		t.Fatal("error was swallowed")
	}
	if logged := buf.String(); !strings.Contains(logged, "code=not_found") {
		t.Errorf("log line = %q", logged)
	}
}

// A handler's own log lines must carry the id without the handler doing anything.
func TestObserveCorrelatesAHandlersOwnLogLines(t *testing.T) {
	log, buf := debugLogger()
	req := connect.NewRequest(&struct{}{})
	req.Header().Set(RequestIDHeader, "abc123")

	next := Observe(log)(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		log.InfoContext(ctx, "handler did something")
		return connect.NewResponse(&struct{}{}), nil
	})
	if _, err := next(context.Background(), req); err != nil {
		t.Fatalf("err = %v", err)
	}

	line, _, _ := strings.Cut(buf.String(), "\n")
	if !strings.Contains(line, "handler did something") || !strings.Contains(line, "request_id=abc123") {
		t.Fatalf("handler line = %q, want it stamped with the request id", line)
	}
}

// The reference in an opaque error is only useful if it names the same trace as the logs.
func TestInternalReusesTheRequestID(t *testing.T) {
	log, _ := debugLogger()
	ctx := WithRequestID(context.Background(), "abc123")

	err := Internal(ctx, log, errors.New("boom"), "do thing")
	if !strings.Contains(err.Error(), "abc123") {
		t.Fatalf("err = %q, want it to carry the request id", err.Error())
	}
}
