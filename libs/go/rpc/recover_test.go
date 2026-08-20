package rpc

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
)

func panicking(v any) connect.UnaryFunc {
	return func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		panic(v)
	}
}

func TestRecoverTurnsAPanicIntoAnInternalError(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	next := Recover(log)(panicking(errors.New("nil map write")))
	_, err := next(context.Background(), connect.NewRequest(&struct{}{}))

	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal (err=%v)", connect.CodeOf(err), err)
	}
	if strings.Contains(err.Error(), "nil map write") {
		t.Fatalf("wire message leaked the panic value: %q", err.Error())
	}
	logged := buf.String()
	if !strings.Contains(logged, "nil map write") {
		t.Fatalf("log lost the panic value: %q", logged)
	}
	if !strings.Contains(logged, "stack") {
		t.Fatalf("log lost the stack: %q", logged)
	}
}

func TestRecoverPassesASuccessThrough(t *testing.T) {
	next := Recover(nil)(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(&struct{}{}), nil
	})
	resp, err := next(context.Background(), connect.NewRequest(&struct{}{}))
	if err != nil || resp == nil {
		t.Fatalf("got (%v, %v), want a response and no error", resp, err)
	}
}

func TestRecoverKeepsAnOrdinaryErrorIntact(t *testing.T) {
	want := connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	next := Recover(nil)(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, want
	})
	if _, err := next(context.Background(), connect.NewRequest(&struct{}{})); !errors.Is(err, want) {
		t.Fatalf("err = %v, want it passed through unchanged", err)
	}
}

// ErrAbortHandler means "drop the connection without a word". Converting it to a response
// would undo the only thing it is for, so it must keep unwinding.
func TestRecoverRepanicsOnErrAbortHandler(t *testing.T) {
	defer func() {
		err, _ := recover().(error)
		if !errors.Is(err, http.ErrAbortHandler) {
			t.Fatalf("recover() = %v, want it to keep unwinding as ErrAbortHandler", err)
		}
	}()
	next := Recover(nil)(panicking(http.ErrAbortHandler))
	_, _ = next(context.Background(), connect.NewRequest(&struct{}{}))
	t.Fatal("ErrAbortHandler was swallowed")
}

func TestPanicValue(t *testing.T) {
	if got := panicValue(errors.New("boom")); got != "boom" {
		t.Errorf("panicValue(error) = %q, want %q", got, "boom")
	}
	if got := panicValue(42); got != "42" {
		t.Errorf("panicValue(42) = %q, want %q", got, "42")
	}
}
