package rpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"

	"connectrpc.com/connect"
)

// Recover returns an interceptor that turns a panic in a handler into a CodeInternal response.
//
// Without it a panic unwinds into net/http, which kills the connection: the client sees a
// transport error rather than an RPC error, every other in-flight stream on that HTTP/2
// connection dies with it, and the only record is net/http's own stack on stderr — not the
// service logger, so not the JSON stream anything collects.
//
// The response carries the same opaque message as any other internal failure. The stack goes
// to the log, next to the procedure that produced it.
func Recover(log *slog.Logger) connect.UnaryInterceptorFunc {
	if log == nil {
		log = slog.Default()
	}
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = recovered(ctx, log, r, req.Spec().Procedure)
				}
			}()
			return next(ctx, req)
		}
	}
}

// recovered builds the error a recovered panic becomes, or re-panics for the one value that
// must not be swallowed.
func recovered(ctx context.Context, log *slog.Logger, r any, procedure string) error {
	// http.ErrAbortHandler is how a handler says "drop this connection, silently". Turning it
	// into a response would defeat the only thing it does.
	if r == http.ErrAbortHandler {
		panic(r)
	}

	ref := reference()
	log.ErrorContext(ctx, "panic in handler",
		slog.String("procedure", procedure),
		slog.String("panic", panicValue(r)),
		slog.String("stack", stack()),
		slog.String("ref", ref),
	)
	return connect.NewError(connect.CodeInternal,
		errors.New(opaque+" (ref "+ref+")"))
}

// stack renders the goroutine's stack, growing the buffer until it fits. A truncated stack of
// a panic is the one log line where the missing part is usually the part you need.
func stack() string {
	buf := make([]byte, 8<<10)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		if len(buf) >= 1<<20 {
			return string(buf) + "\n... stack truncated at 1 MiB"
		}
		buf = make([]byte, 2*len(buf))
	}
}

// panicValue renders a recovered value for a message; errors keep their text, everything else
// gets %v.
func panicValue(r any) string {
	if err, ok := r.(error); ok {
		return err.Error()
	}
	return fmt.Sprint(r)
}
