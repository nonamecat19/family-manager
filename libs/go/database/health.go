package database

import (
	"context"
	"net/http"
	"time"
)

// Pinger is the one method a health check needs from a pool. Narrowing it keeps the handler
// testable without a database and keeps this file from importing pgxpool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// DefaultHealthTimeout bounds the ping. Compose and the reverse proxy both retry, so a slow
// answer is worse than a failed one: an unbounded check turns "the database is wedged" into
// "the health endpoint never replies", and an orchestrator waiting on a reply cannot restart
// anything.
const DefaultHealthTimeout = 2 * time.Second

// HealthHandler reports whether the service can reach its database.
//
// The ping inherits the request's context, so a client that gives up releases the check with
// it, and it is capped at timeout regardless. Zero means DefaultHealthTimeout.
func HealthHandler(p Pinger, timeout time.Duration) http.HandlerFunc {
	if timeout <= 0 {
		timeout = DefaultHealthTimeout
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		if err := p.Ping(ctx); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	}
}
