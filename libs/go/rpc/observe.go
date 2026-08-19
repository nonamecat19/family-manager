package rpc

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/logger"
)

// RequestIDHeader carries the id across a hop. Apps may send one; if they do not, the first
// service to handle the call mints it and every service downstream keeps it.
const RequestIDHeader = "X-Request-Id"

// WithRequestID returns a context carrying id.
//
// It delegates to libs/go/logger rather than keeping a key of its own. logger's handler already
// copies request_id onto every record emitted with that context — a feature written in
// docs/adr/0006-observability.md and never once called — so routing the id through it means a
// handler's own log lines are correlated without any handler being changed.
func WithRequestID(ctx context.Context, id string) context.Context {
	return logger.WithRequestID(ctx, id)
}

// RequestID returns the id carried by ctx, or "" if there is none.
func RequestID(ctx context.Context) string {
	return logger.RequestID(ctx)
}

// maxRequestIDLength bounds what we will echo from a caller. The id is written into every log
// line for the request, so an unbounded one is a way for a client to write as much as it likes
// into our logs.
const maxRequestIDLength = 64

// Observe returns an interceptor that gives every call an id and an access-log line.
//
// The id comes from the caller's X-Request-Id when there is one, so a trace that starts in the
// app or in Caddy stays one trace across the services it fans out to; otherwise it is minted
// here. It goes into the context, into the response header, and into the log line — and
// rpc.Internal uses it as the reference it hands back, so the opaque error a user reports
// names the same id as the logs of every service that touched the request.
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

			// request_id is not in this list: logger's handler puts it on every record made
			// with this context, including the ones handlers write themselves.
			attrs := []any{
				slog.String("procedure", req.Spec().Procedure),
				slog.Duration("elapsed", elapsed),
				slog.String("code", codeOf(err)),
			}
			if err != nil {
				// The cause is already logged by whoever produced it; this line records that
				// the call failed and with what code, which is what a latency or error-rate
				// question needs.
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

// codeOf renders the Connect code of err, or "ok".
func codeOf(err error) string {
	if err == nil {
		return "ok"
	}
	return connect.CodeOf(err).String()
}

// sanitizeRequestID accepts only what is safe to write into a log line and a response header:
// printable ASCII, bounded length, no control characters that could forge a second line.
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

// ForwardRequestID returns a client interceptor that copies the request id from the outgoing
// call's context onto its header.
//
// Without it a trace stops at the first hop. auth resolving a household from family during a
// login is one request as far as anyone debugging it is concerned, but family would mint its
// own id and the two halves would never be findable together.
//
// It sets nothing when the context carries no id, so a call made outside a request — a
// startup probe, a background job — does not invent a trace it is not part of.
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
