package rpc

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/logger"
)

const RequestIDHeader = "X-Request-Id"

func WithRequestID(ctx context.Context, id string) context.Context {
	return logger.WithRequestID(ctx, id)
}

func RequestID(ctx context.Context) string {
	return logger.RequestID(ctx)
}

const maxRequestIDLength = 64

func Observe(log *slog.Logger) connect.UnaryInterceptorFunc {
	if log == nil {
		log = slog.Default()
	}
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			id := sanitizeRequestID(req.Header().Get(RequestIDHeader))
			if id == "" {
				id = reference()
			}
			ctx = WithRequestID(ctx, id)

			started := time.Now()
			resp, err := next(ctx, req)
			elapsed := time.Since(started)

			attrs := []any{
				slog.String("procedure", req.Spec().Procedure),
				slog.Duration("elapsed", elapsed),
				slog.String("code", codeOf(err)),
			}
			if err != nil {
				log.WarnContext(ctx, "rpc failed", attrs...)
			} else {
				log.DebugContext(ctx, "rpc", attrs...)
			}

			if resp != nil {
				resp.Header().Set(RequestIDHeader, id)
			}
			return resp, err
		}
	}
}

func codeOf(err error) string {
	if err == nil {
		return "ok"
	}
	return connect.CodeOf(err).String()
}

func sanitizeRequestID(s string) string {
	if len(s) == 0 || len(s) > maxRequestIDLength {
		return ""
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7e {
			return ""
		}
	}
	return s
}

func ForwardRequestID() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if id := RequestID(ctx); id != "" {
				req.Header().Set(RequestIDHeader, id)
			}
			return next(ctx, req)
		}
	}
}
