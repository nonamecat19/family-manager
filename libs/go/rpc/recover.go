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

func recovered(ctx context.Context, log *slog.Logger, r any, procedure string) error {
	if err, ok := r.(error); ok && errors.Is(err, http.ErrAbortHandler) {
		panic(r)
	}

	ref := refFrom(ctx)
	log.ErrorContext(ctx, "panic in handler",
		slog.String("procedure", procedure),
		slog.String("panic", panicValue(r)),
		slog.String("stack", stack()),
		slog.String("ref", ref),
	)
	return connect.NewError(connect.CodeInternal,
		errors.New(opaque+" (ref "+ref+")"))
}

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

func panicValue(r any) string {
	if err, ok := r.(error); ok {
		return err.Error()
	}
	return fmt.Sprint(r)
}
