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

type Options struct {
	Service string
	Level   string
	JSON    bool
}

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

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

func ContextHandler(h slog.Handler) slog.Handler { return &contextHandler{Handler: h} }

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
