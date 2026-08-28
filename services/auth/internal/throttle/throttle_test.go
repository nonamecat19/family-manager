package throttle

import (
	"sync"
	"testing"
	"time"
)

func testParams() Params {
	return Params{Threshold: 3, Base: time.Second, Max: 4 * time.Second, Forget: time.Minute}
}

// A person mistyping their own password must not be locked out, which is the failure mode a
// throttle most easily creates.
func TestUnderTheThresholdNothingIsLocked(t *testing.T) {
	th := New(testParams(), time.Now)
	for i := 0; i < testParams().Threshold-1; i++ {
		if wait := th.Failed("ada@example.test"); wait != 0 {
			t.Fatalf("locked after %d failure(s), threshold is %d", i+1, testParams().Threshold)
		}
	}
	if wait := th.Retry("ada@example.test"); wait != 0 {
		t.Fatalf("Retry = %v below the threshold, want 0", wait)
	}
}

func TestLockoutDoublesAndIsCapped(t *testing.T) {
	p := testParams()
	th := New(p, time.Now)

	for i := 0; i < p.Threshold-1; i++ {
		th.Failed("ada@example.test")
	}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second}
	for i, w := range want {
		if got := th.Failed("ada@example.test"); got != w {
			t.Fatalf("failure %d past the threshold locked for %v, want %v", i+1, got, w)
		}
	}
}

func TestLockoutExpires(t *testing.T) {
	p := testParams()
	now := time.Unix(1_800_000_000, 0)
	th := New(p, func() time.Time { return now })

	for i := 0; i < p.Threshold; i++ {
		th.Failed("ada@example.test")
	}
	if th.Retry("ada@example.test") == 0 {
		t.Fatal("not locked after passing the threshold")
	}

	now = now.Add(p.Base + time.Millisecond)
	if wait := th.Retry("ada@example.test"); wait != 0 {
		t.Fatalf("still locked for %v after the lockout expired", wait)
	}
}

// A correct password is the strongest evidence the attempts before it were the owner.
func TestSuccessClearsTheKey(t *testing.T) {
	p := testParams()
	th := New(p, time.Now)
	for i := 0; i < p.Threshold; i++ {
		th.Failed("ada@example.test")
	}
	th.Succeeded("ada@example.test")

	if wait := th.Retry("ada@example.test"); wait != 0 {
		t.Fatalf("Retry = %v after a success, want 0", wait)
	}
	if wait := th.Failed("ada@example.test"); wait != 0 {
		t.Fatalf("the counter survived a success: locked for %v on the next failure", wait)
	}
}

// One account's failures must not lock another out — otherwise the throttle is a way to lock
// anyone out of their own household.
func TestKeysAreIndependent(t *testing.T) {
	p := testParams()
	th := New(p, time.Now)
	for i := 0; i < p.Threshold+3; i++ {
		th.Failed("victim@example.test")
	}
	if wait := th.Retry("someone.else@example.test"); wait != 0 {
		t.Fatalf("an unrelated account is locked for %v", wait)
	}
}

func TestIdleEntriesAreForgotten(t *testing.T) {
	p := testParams()
	now := time.Unix(1_800_000_000, 0)
	th := New(p, func() time.Time { return now })

	for i := 0; i < p.Threshold; i++ {
		th.Failed("old@example.test")
	}
	now = now.Add(p.Forget + time.Minute)
	th.Failed("someone.else@example.test") // the write path is where the sweep runs

	th.mu.Lock()
	_, still := th.entries["old@example.test"]
	th.mu.Unlock()
	if still {
		t.Fatal("an idle entry was kept; the map is a leak keyed by every address ever guessed")
	}
}

func TestNilThrottleAllowsEverything(t *testing.T) {
	var th *Throttle
	if th.Retry("a@b.co") != 0 || th.Failed("a@b.co") != 0 {
		t.Fatal("a nil throttle blocked something")
	}
	th.Succeeded("a@b.co")
}

func TestEmptyKeyIsIgnored(t *testing.T) {
	th := New(testParams(), time.Now)
	for i := 0; i < 100; i++ {
		if th.Failed("") != 0 {
			t.Fatal("an empty key was throttled")
		}
	}
}

func TestConcurrentUse(t *testing.T) {
	th := New(DefaultParams(), time.Now)
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			th.Failed("ada@example.test")
			th.Retry("ada@example.test")
			th.Succeeded("other@example.test")
		}()
	}
	wg.Wait()
}

// Setting one field must not silently reset the others: a whole-struct fallback would have
// done exactly that the first time someone configured only the threshold.
func TestPartialParamsFallBackFieldByField(t *testing.T) {
	d := DefaultParams()
	th := New(Params{Threshold: 2}, time.Now)

	if th.params.Threshold != 2 {
		t.Fatalf("Threshold = %d, want the configured 2", th.params.Threshold)
	}
	if th.params.Base != d.Base || th.params.Max != d.Max || th.params.Forget != d.Forget {
		t.Fatalf("params = %+v, want the remaining fields defaulted", th.params)
	}
}
