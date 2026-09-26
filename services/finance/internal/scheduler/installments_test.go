package scheduler

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

type countingPoster struct{ calls atomic.Int32 }

func (p *countingPoster) PostDueInstallments(context.Context) (int, error) {
	p.calls.Add(1)
	return 0, nil
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestRunPostsAtStartAndOnEveryTickUntilCancelled(t *testing.T) {
	poster := &countingPoster{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		NewInstallments(poster, 5*time.Millisecond, quiet()).Run(ctx)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for poster.calls.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("calls = %d after 2s, want at least 3", poster.calls.Load())
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestRunWithZeroIntervalIsDisabled(t *testing.T) {
	poster := &countingPoster{}
	NewInstallments(poster, 0, quiet()).Run(context.Background())
	if poster.calls.Load() != 0 {
		t.Errorf("calls = %d, want 0 when disabled", poster.calls.Load())
	}
}
