package rpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
)

const opaque = "internal error"

func Internal(ctx context.Context, log *slog.Logger, err error, what string) error {
	if log == nil {
		log = slog.Default()
	}
	ref := refFrom(ctx)
	log.ErrorContext(ctx, what,
		slog.String("error", err.Error()),
		slog.String("ref", ref),
	)
	return connect.NewError(connect.CodeInternal, errors.New(opaque+" (ref "+ref+")"))
}

func refFrom(ctx context.Context) string {
	if id := RequestID(ctx); id != "" {
		return id
	}
	return reference()
}

func reference() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "unref"
	}
	return hex.EncodeToString(buf[:])
}
