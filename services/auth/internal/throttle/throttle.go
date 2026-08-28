// Package throttle rate-limits repeated failed sign-ins for one account.
//
// The floor on password length is 8 characters (services/auth/internal/handler), which is the
// right floor and is not, on its own, enough: an attacker who can try passwords as fast as the
// service will answer gets a very large number of guesses against a family's accounts. The
// argon2 gate bounds how much memory that costs us; it does nothing about how many guesses
// they get.
//
// This is deliberately per-account rather than per-IP. The address is what an attacker is
// attacking, they can change IP freely, and a household behind one NAT would share an IP limit
// with each other.
//
// It is in-process. The deployment is one container per service on one box (ADR 0004), so a
// shared store would be a Redis nobody else needs; the cost is that a restart forgets, which
// an attacker cannot cause.
package throttle

import (
	"sync"
	"time"
)

// Params configures a Throttle.
type Params struct {
	// Threshold is how many consecutive failures are allowed before the next attempt is
	// refused. It is set
	// well above what a person mistyping their own password reaches, because locking out the
	// legitimate owner is the failure mode a throttle most easily creates.
	Threshold int
	// Base is the first lockout. Each further failure past the threshold doubles it.
	Base time.Duration
	// Max caps the lockout, so an account is never permanently unreachable.
	Max time.Duration
	// Forget drops an idle entry, which is what keeps the map from being a slow leak keyed by
	// every address anyone has ever guessed at.
	Forget time.Duration
}

// DefaultParams tolerates ten consecutive failures, then backs off from ten seconds to five
// minutes.
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

// Throttle tracks consecutive failures per key. Safe for concurrent use.
type Throttle struct {
	params Params
	now    func() time.Time

	mu      sync.Mutex
	entries map[string]*entry
}

// New returns a Throttle. now is injected by tests.
//
// Each zero field falls back to its default individually, so configuring one setting does not
// silently reset the others — which a whole-struct fallback would do the first time someone set
// only AUTH_LOGIN_FAILURE_THRESHOLD.
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

// Retry reports how long the key must wait, or zero when an attempt is allowed now.
//
// A nil Throttle allows everything, so a handler built without one behaves as it did before
// this package existed.
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

// Failed records a failed attempt and returns how long the key is now locked out for, which is
// zero while it is still under the threshold.
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

	// The Threshold-th failure is the one that locks: "ten tolerated" means the eleventh
	// attempt is refused, not that the eleventh failure is still allowed to happen.
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

// Succeeded clears the key. A correct password is the strongest possible evidence that the
// attempts before it were the owner mistyping.
func (t *Throttle) Succeeded(key string) {
	if t == nil || key == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, key)
}

// sweepLocked drops entries nothing has touched for Forget. Called on the write path, which is
// the only place the map grows, so there is no goroutine to stop.
func (t *Throttle) sweepLocked(now time.Time) {
	for k, e := range t.entries {
		if now.Sub(e.seen) > t.params.Forget && !e.lockedTo.After(now) {
			delete(t.entries, k)
		}
	}
}
