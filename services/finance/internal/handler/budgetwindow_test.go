package handler

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func checkWindow(t *testing.T, period, startOn, asOf, wantFrom, wantTo string) {
	t.Helper()
	got, err := windowFor(period, day(startOn), day(asOf))
	if err != nil {
		t.Fatalf("windowFor(%s, %s, %s): %v", period, startOn, asOf, err)
	}
	if got.from.Format(time.DateOnly) != wantFrom || got.to.Format(time.DateOnly) != wantTo {
		t.Errorf("windowFor(%s, start=%s, asOf=%s) = %s..%s, want %s..%s",
			period, startOn, asOf,
			got.from.Format(time.DateOnly), got.to.Format(time.DateOnly), wantFrom, wantTo)
	}
}

func TestMonthlyWindowFollowsTheAnchorDay(t *testing.T) {
	// Anchored on the 15th: the window runs the 15th to the 14th, which is what someone paid
	// mid-month means by "a month".
	checkWindow(t, periodMonth, "2026-01-15", "2026-01-15", "2026-01-15", "2026-02-14")
	checkWindow(t, periodMonth, "2026-01-15", "2026-02-14", "2026-01-15", "2026-02-14")
	checkWindow(t, periodMonth, "2026-01-15", "2026-02-15", "2026-02-15", "2026-03-14")
	checkWindow(t, periodMonth, "2026-01-15", "2026-08-03", "2026-07-15", "2026-08-14")
}

func TestMonthlyWindowFromTheFirstIsACalendarMonth(t *testing.T) {
	checkWindow(t, periodMonth, "2026-01-01", "2026-03-17", "2026-03-01", "2026-03-31")
	checkWindow(t, periodMonth, "2026-01-01", "2026-02-28", "2026-02-01", "2026-02-28")
}

// The case that breaks naive date arithmetic: Go's AddDate turns 31 January plus one month
// into 3 March, which would walk the anchor forward every short month.
func TestMonthlyWindowClampsTheAnchorToShortMonths(t *testing.T) {
	checkWindow(t, periodMonth, "2026-01-31", "2026-01-31", "2026-01-31", "2026-02-27")
	checkWindow(t, periodMonth, "2026-01-31", "2026-02-28", "2026-02-28", "2026-03-30")
	checkWindow(t, periodMonth, "2026-01-31", "2026-03-31", "2026-03-31", "2026-04-29")

	// And the anchor is not lost: after a short month it returns to the 31st.
	checkWindow(t, periodMonth, "2026-01-31", "2026-05-15", "2026-04-30", "2026-05-30")
}

func TestMonthlyWindowInALeapYear(t *testing.T) {
	checkWindow(t, periodMonth, "2028-01-30", "2028-02-29", "2028-02-29", "2028-03-29")
}

func TestWeeklyWindow(t *testing.T) {
	checkWindow(t, periodWeek, "2026-03-02", "2026-03-02", "2026-03-02", "2026-03-08")
	checkWindow(t, periodWeek, "2026-03-02", "2026-03-08", "2026-03-02", "2026-03-08")
	checkWindow(t, periodWeek, "2026-03-02", "2026-03-09", "2026-03-09", "2026-03-15")
	checkWindow(t, periodWeek, "2026-03-02", "2026-04-01", "2026-03-30", "2026-04-05")
}

func TestYearlyWindow(t *testing.T) {
	checkWindow(t, periodYear, "2026-01-01", "2026-06-15", "2026-01-01", "2026-12-31")
	checkWindow(t, periodYear, "2026-04-06", "2026-04-05", "2026-04-06", "2027-04-05")
	checkWindow(t, periodYear, "2026-04-06", "2027-01-01", "2026-04-06", "2027-04-05")
	// 29 February anchors clamp rather than jumping to 1 March.
	checkWindow(t, periodYear, "2028-02-29", "2029-06-01", "2029-02-28", "2030-02-27")
}

// A budget starting next month should read as a full, untouched window — not an empty or
// backwards one.
func TestWindowBeforeTheAnchorIsTheFirstOccurrence(t *testing.T) {
	checkWindow(t, periodMonth, "2026-09-01", "2026-08-07", "2026-09-01", "2026-09-30")
	checkWindow(t, periodWeek, "2026-09-01", "2026-08-07", "2026-09-01", "2026-09-07")
	checkWindow(t, periodYear, "2026-09-01", "2026-08-07", "2026-09-01", "2027-08-31")
}

func TestWindowsTileWithoutGapOrOverlap(t *testing.T) {
	// Walking a year day by day, every day must fall inside exactly one window, and
	// consecutive windows must abut.
	for _, period := range []string{periodWeek, periodMonth, periodYear} {
		start := day("2026-01-31") // the nastiest anchor
		var prev budgetWindow

		for d := day("2026-01-01"); d.Before(day("2027-06-01")); d = d.AddDate(0, 0, 1) {
			w, err := windowFor(period, start, d)
			if err != nil {
				t.Fatalf("windowFor: %v", err)
			}
			if d.Before(start) {
				continue // before the anchor every day maps to the first window by design
			}
			if d.Before(w.from) || d.After(w.to) {
				t.Fatalf("%s: %s is outside its own window %s..%s", period,
					d.Format(time.DateOnly), w.from.Format(time.DateOnly), w.to.Format(time.DateOnly))
			}
			if !prev.from.IsZero() && !w.from.Equal(prev.from) {
				if !w.from.Equal(prev.to.AddDate(0, 0, 1)) {
					t.Fatalf("%s: gap or overlap between %s..%s and %s..%s", period,
						prev.from.Format(time.DateOnly), prev.to.Format(time.DateOnly),
						w.from.Format(time.DateOnly), w.to.Format(time.DateOnly))
				}
			}
			prev = w
		}
	}
}

func TestUnknownPeriodIsAnError(t *testing.T) {
	if _, err := windowFor("fortnight", day("2026-01-01"), day("2026-01-02")); err == nil {
		t.Fatal("expected an error for an unknown period")
	}
}

func TestDaysRemainingCountsToday(t *testing.T) {
	w, err := windowFor(periodMonth, day("2026-03-01"), day("2026-03-15"))
	if err != nil {
		t.Fatalf("windowFor: %v", err)
	}

	if got := w.daysRemaining(day("2026-03-31")); got != 1 {
		t.Errorf("last day of the window = %d days remaining, want 1", got)
	}
	if got := w.daysRemaining(day("2026-03-30")); got != 2 {
		t.Errorf("penultimate day = %d, want 2", got)
	}
	if got := w.daysRemaining(day("2026-03-01")); got != 31 {
		t.Errorf("first day = %d, want 31", got)
	}
	if got := w.daysRemaining(day("2026-04-05")); got != 0 {
		t.Errorf("after the window = %d, want 0", got)
	}
}
