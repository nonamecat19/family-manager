// Package rpc holds the Connect-side concerns every service repeats: turning an internal
// failure into a response that says nothing, and (see interceptor.go) the interceptors that
// wrap every procedure.
//
// The rule this package exists to enforce: an error that crosses the wire is written for the
// caller, and an error that goes to the log is written for us. They are not the same string.
package rpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
)

// opaque is what a client sees for any unexpected failure. It is deliberately useless: the
// caller cannot act on "constraint recipes_family_id_fkey violated" and an attacker should not
// receive it. The reference is the thread back to the log line that has the real story.
const opaque = "internal error"

// Internal logs err at ERROR with what as the operation, and returns a CodeInternal Connect
// error carrying only an opaque message and a short reference.
//
// Connect serialises the wrapped error's Error() into the response body, so wrapping the
// database error — which is what every service used to do — published SQLSTATE codes, table
// names and constraint names to whoever called the endpoint. Callers get the reference
// instead; `grep <ref>` finds the line with the cause.
func Internal(ctx context.Context, log *slog.Logger, err error, what string) error {
	if log == nil {
		log = slog.Default()
	}
	ref := reference()
	log.ErrorContext(ctx, what,
		slog.String("error", err.Error()),
		slog.String("ref", ref),
	)
	return connect.NewError(connect.CodeInternal, errors.New(opaque+" (ref "+ref+")"))
}

// reference is a short random tag, not an identifier anything stores. Six bytes is plenty to
// pick one request out of a day of logs, and short enough that a person can read it off a
// phone screen into a chat message.
func reference() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// Losing the reference is not worth failing the response over; the log line still
		// carries the cause, it is just harder to find.
		return "unref"
	}
	return hex.EncodeToString(buf[:])
}
