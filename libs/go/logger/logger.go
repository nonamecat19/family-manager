// Package logger is the single logging entry point for every Go service in this repo.
//
// Text handler in dev, JSON in prod, and a request id + user id carried on the context so
// every line emitted while handling a request is correlated. See docs/adr/0006-observability.md.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	userIDKey
)

// Options configures the process-wide logger.
type Options struct {
	// Service is the service name stamped on every record, e.g. "finance".
	Service string
	// Level is one of debug, info, warn, error. Empty means info.
	Level string
	// JSON selects the JSON handler. Use it in prod; text is easier to read in dev.
	JSON bool
}

// New builds a logger. It never fails: an unknown level falls back to info.
func New(opts Options) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{Level: parseLevel(opts.Level)}

	var h slog.Handler
	if opts.JSON {
		h = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		h = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	l := slog.New(ContextHandler(h))
	if opts.Service != "" {
		l = l.With(slog.String("service", opts.Service))
	}
	return l
}

// SetDefault installs the logger as the slog default so libraries pick it up too.
func SetDefault(l *slog.Logger) { slog.SetDefault(l) }

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID returns a context whose log records carry request_id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// WithUserID returns a context whose log records carry user_id.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// RequestID reads the request id back out, "" when unset.
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// UserID reads the user id back out, "" when unset.
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// ContextHandler wraps h so the correlation ids on a context are copied onto every record made
// with it. New already does this; it is exported for the case New cannot serve — a test that
// needs the same correlation behaviour over a buffer instead of stdout.
func ContextHandler(h slog.Handler) slog.Handler { return &contextHandler{Handler: h} }

// contextHandler copies the correlation ids from the context onto each record, so callers
// never have to pass them explicitly.
type contextHandler struct{ slog.Handler }

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if id := UserID(ctx); id != "" {
		r.AddAttrs(slog.String("user_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}
