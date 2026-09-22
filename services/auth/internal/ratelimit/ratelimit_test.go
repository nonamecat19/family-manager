package ratelimit

import (
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func TestTakeAllowsTheLimitThenWaitsOutTheWindow(t *testing.T) {
	c := &clock{t: time.Unix(1_700_000_000, 0)}
	l := New(2, time.Minute, c.now)

	for i := range 2 {
		if wait := l.Take("1.2.3.4"); wait != 0 {
			t.Fatalf("take %d waited %v", i, wait)
		}
	}
	if wait := l.Take("1.2.3.4"); wait != time.Minute {
		t.Fatalf("over the limit waited %v, want 1m", wait)
	}
	if wait := l.Take("5.6.7.8"); wait != 0 {
		t.Fatalf("another key waited %v", wait)
	}

	c.t = c.t.Add(time.Minute)
	if wait := l.Take("1.2.3.4"); wait != 0 {
		t.Fatalf("a new window waited %v", wait)
	}
}

func TestSweepForgetsClosedWindows(t *testing.T) {
	c := &clock{t: time.Unix(1_700_000_000, 0)}
	l := New(1, time.Minute, c.now)

	l.Take("a")
	l.Take("b")
	c.t = c.t.Add(2 * time.Minute)
	l.Take("c")

	if len(l.windows) != 1 {
		t.Fatalf("windows = %d, want only the live one", len(l.windows))
	}
}

func TestZeroLimitNeverRefuses(t *testing.T) {
	l := New(0, time.Minute, nil)
	for range 100 {
		if l.Take("x") != 0 {
			t.Fatal("a disabled limiter refused")
		}
	}
	var nilLimiter *Limiter
	if nilLimiter.Take("x") != 0 {
		t.Fatal("a nil limiter refused")
	}
}

func TestCountReportsTheCurrentWindow(t *testing.T) {
	c := &clock{t: time.Unix(1_700_000_000, 0)}
	l := New(0, time.Minute, c.now)

	for range 3 {
		l.Take("net")
	}
	if n := l.Count("net"); n != 3 {
		t.Fatalf("count = %d, want 3 with no limit", n)
	}
	if n := l.Count("other"); n != 0 {
		t.Fatalf("unseen key count = %d", n)
	}
	c.t = c.t.Add(time.Minute)
	if n := l.Count("net"); n != 0 {
		t.Fatalf("count after the window = %d, want 0", n)
	}
}
