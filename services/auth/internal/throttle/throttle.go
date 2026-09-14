package throttle

import (
	"sync"
	"time"
)

type Params struct {
	Threshold int
	Base      time.Duration
	Max       time.Duration
	Forget    time.Duration
}

func DefaultParams() Params {
	return Params{
		Threshold: 10,
		Base:      10 * time.Second,
		Max:       5 * time.Minute,
		Forget:    time.Hour,
	}
}

type entry struct {
	failures int
	lockedTo time.Time
	seen     time.Time
}

type Throttle struct {
	params Params
	now    func() time.Time

	mu      sync.Mutex
	entries map[string]*entry
}

func New(p Params, now func() time.Time) *Throttle {
	d := DefaultParams()
	if p.Threshold <= 0 {
		p.Threshold = d.Threshold
	}
	if p.Base <= 0 {
		p.Base = d.Base
	}
	if p.Max <= 0 {
		p.Max = d.Max
	}
	if p.Forget <= 0 {
		p.Forget = d.Forget
	}
	if now == nil {
		now = time.Now
	}
	return &Throttle{params: p, now: now, entries: map[string]*entry{}}
}

func (t *Throttle) Retry(key string) time.Duration {
	if t == nil || key == "" {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	e, ok := t.entries[key]
	if !ok {
		return 0
	}
	now := t.now()
	if wait := e.lockedTo.Sub(now); wait > 0 {
		return wait
	}
	return 0
}

func (t *Throttle) Failed(key string) time.Duration {
	if t == nil || key == "" {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	t.sweepLocked(now)

	e, ok := t.entries[key]
	if !ok {
		e = &entry{}
		t.entries[key] = e
	}
	e.failures++
	e.seen = now

	over := e.failures - t.params.Threshold + 1
	if over <= 0 {
		return 0
	}

	wait := t.params.Base
	for i := 1; i < over && wait < t.params.Max; i++ {
		wait *= 2
	}
	if wait > t.params.Max {
		wait = t.params.Max
	}
	e.lockedTo = now.Add(wait)
	return wait
}

func (t *Throttle) Succeeded(key string) {
	if t == nil || key == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, key)
}

func (t *Throttle) sweepLocked(now time.Time) {
	for k, e := range t.entries {
		if now.Sub(e.seen) > t.params.Forget && !e.lockedTo.After(now) {
			delete(t.entries, k)
		}
	}
}
