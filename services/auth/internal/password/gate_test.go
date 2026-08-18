package password

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGateLimitsConcurrency(t *testing.T) {
	const limit = 3
	g := NewGate(limit)

	var inFlight, peak int64
	release := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = g.Do(context.Background(), func() error {
				n := atomic.AddInt64(&inFlight, 1)
				for {
					p := atomic.LoadInt64(&peak)
					if n <= p || atomic.CompareAndSwapInt64(&peak, p, n) {
						break
					}
				}
				<-release
				atomic.AddInt64(&inFlight, -1)
				return nil
			})
		}()
	}

	// Give the goroutines time to pile up against the gate before letting any finish.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := atomic.LoadInt64(&peak); got > limit {
		t.Fatalf("peak concurrency = %d, want at most %d", got, limit)
	}
	if got := atomic.LoadInt64(&peak); got == 0 {
		t.Fatal("nothing ran")
	}
}

func TestGateReturnsBusyWhenTheCallerGivesUp(t *testing.T) {
	g := NewGate(1)

	held := make(chan struct{})
	done := make(chan struct{})
	go func() {
		_ = g.Do(context.Background(), func() error {
			close(held)
			<-done
			return nil
		})
	}()
	<-held
	defer close(done)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := g.Do(ctx, func() error {
		t.Fatal("ran while the only slot was held")
		return nil
	})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
}

func TestGateReturnsTheFunctionsError(t *testing.T) {
	want := errors.New("boom")
	if err := NewGate(1).Do(context.Background(), func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

// A nil gate is the zero value an Options with no gate produces; it must not be a nil
// dereference, and it must not silently skip the work.
func TestNilGateRunsUnbounded(t *testing.T) {
	var g *Gate
	ran := false
	if err := g.Do(context.Background(), func() error { ran = true; return nil }); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if !ran {
		t.Fatal("nil gate skipped the work")
	}
}

func TestGateReleasesItsSlotOnPanic(t *testing.T) {
	g := NewGate(1)

	func() {
		defer func() { _ = recover() }()
		_ = g.Do(context.Background(), func() error { panic("boom") })
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := g.Do(ctx, func() error { return nil }); err != nil {
		t.Fatalf("slot was not released after a panic: %v", err)
	}
}

func TestDefaultConcurrencyIsAtLeastOne(t *testing.T) {
	if n := DefaultConcurrency(); n < 1 {
		t.Fatalf("DefaultConcurrency() = %d, want >= 1", n)
	}
}
