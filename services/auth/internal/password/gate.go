package password

import (
	"context"
	"errors"
	"runtime"
)

// ErrBusy means the gate was full for as long as the caller was willing to wait.
var ErrBusy = errors.New("password: too many hashes in flight")

// Gate bounds how many argon2id hashes run concurrently.
//
// Each hash allocates Params.Memory — 19 MiB at the defaults — for its whole duration, and
// Register and Login are both reachable without a token. Nothing else in this service holds
// memory proportional to request concurrency, so a hundred simultaneous logins is two gigabytes
// on a VPS that has less than that, and the process is killed rather than slowed.
//
// The gate converts that into queueing, which is the failure everyone would rather have: slow
// logins under load, and an explicit ErrBusy once the queue is longer than the caller's
// patience.
type Gate struct {
	tokens chan struct{}
}

// DefaultConcurrency is deliberately small. argon2id is tuned so one hash takes tens of
// milliseconds of CPU; more of them in parallel than there are cores buys no throughput and
// costs Memory bytes each.
func DefaultConcurrency() int {
	if n := runtime.GOMAXPROCS(0); n > 1 {
		return n
	}
	return 1
}

// NewGate returns a gate admitting n hashes at a time. n <= 0 means DefaultConcurrency.
func NewGate(n int) *Gate {
	if n <= 0 {
		n = DefaultConcurrency()
	}
	return &Gate{tokens: make(chan struct{}, n)}
}

// Do runs fn while holding a slot, returning ErrBusy if ctx is done before one frees up.
//
// A nil gate runs fn unbounded, so a zero-valued Options in a test behaves as it did before
// the gate existed.
func (g *Gate) Do(ctx context.Context, fn func() error) error {
	if g == nil {
		return fn()
	}
	select {
	case g.tokens <- struct{}{}:
		defer func() { <-g.tokens }()
		return fn()
	case <-ctx.Done():
		return ErrBusy
	}
}
