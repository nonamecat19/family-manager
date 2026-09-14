package password

import (
	"context"
	"errors"
	"runtime"
)

var ErrBusy = errors.New("password: too many hashes in flight")

type Gate struct {
	tokens chan struct{}
}

func DefaultConcurrency() int {
	if n := runtime.GOMAXPROCS(0); n > 1 {
		return n
	}
	return 1
}

func NewGate(n int) *Gate {
	if n <= 0 {
		n = DefaultConcurrency()
	}
	return &Gate{tokens: make(chan struct{}, n)}
}

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
