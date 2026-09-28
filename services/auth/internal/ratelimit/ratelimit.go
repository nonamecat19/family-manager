package ratelimit

import (
	"sync"
	"time"
)

type window struct {
	start time.Time
	count int
}

type Limiter struct {
	limit  int
	period time.Duration
	now    func() time.Time

	mu      sync.Mutex
	windows map[string]*window
	swept   time.Time
}

func New(limit int, period time.Duration, now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{limit: limit, period: period, now: now, windows: map[string]*window{}}
}

func (l *Limiter) Take(key string) time.Duration {
	if l == nil || l.limit <= 0 || l.period <= 0 {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweepLocked(now)

	w, ok := l.windows[key]
	if !ok || !now.Before(w.start.Add(l.period)) {
		w = &window{start: now}
		l.windows[key] = w
	}
	if w.count >= l.limit {
		return w.start.Add(l.period).Sub(now)
	}
	w.count++
	return 0
}

func (l *Limiter) sweepLocked(now time.Time) {
	if now.Sub(l.swept) < l.period {
		return
	}
	l.swept = now
	for k, w := range l.windows {
		if !now.Before(w.start.Add(l.period)) {
			delete(l.windows, k)
		}
	}
}
