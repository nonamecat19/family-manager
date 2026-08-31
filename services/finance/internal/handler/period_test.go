package handler

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

func mustDay(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(dateLayout, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return d
}

func TestResolvePeriod(t *testing.T) {
	now := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		period   *financev1.Period
		from, to string
	}{
		{"unset is the current month", nil, "2026-08-01", "2026-08-31"},
		{"day", &financev1.Period{
			Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_DAY,
			Anchor:      "2026-08-15",
		}, "2026-08-15", "2026-08-15"},
		{"week starts on the household's weekday", &financev1.Period{
			Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_WEEK,
			Anchor:      "2026-08-29", // a Saturday
		}, "2026-08-24", "2026-08-30"},
		{"month", &financev1.Period{
			Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_MONTH,
			Anchor:      "2026-02-14",
		}, "2026-02-01", "2026-02-28"},
		{"year", &financev1.Period{
			Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_YEAR,
			Anchor:      "2026-08-15",
		}, "2026-01-01", "2026-12-31"},
		{"custom takes the explicit range", &financev1.Period{
			Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_CUSTOM,
			Range:       &financev1.DateRange{From: "2026-03-05", To: "2026-04-09"},
		}, "2026-03-05", "2026-04-09"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolvePeriod(tc.period, now, time.UTC, "monday")
			if err != nil {
				t.Fatalf("resolvePeriod: %v", err)
			}
			if got.fromString() != tc.from || got.toString() != tc.to {
				t.Errorf("window = %s..%s, want %s..%s",
					got.fromString(), got.toString(), tc.from, tc.to)
			}
		})
	}
}

func TestResolvePeriodRejectsAnUnusableCustomRange(t *testing.T) {
	now := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)

	for name, period := range map[string]*financev1.Period{
		"no range": {Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_CUSTOM},
		"backwards": {
			Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_CUSTOM,
			Range:       &financev1.DateRange{From: "2026-04-09", To: "2026-03-05"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := resolvePeriod(period, now, time.UTC, "monday"); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

// The rolling window is what lets a household budget from the 5th without being pushed onto
// calendar months, and the clamped month arithmetic is what stops an anchor on the 31st from
// skipping February.
func TestBudgetWindow(t *testing.T) {
	cases := []struct {
		name             string
		period           string
		start, asOf      string
		wantFrom, wantTo string
	}{
		{"month from the 1st", budgetPeriodMonth, "2026-01-01", "2026-08-15", "2026-08-01", "2026-08-31"},
		{"month from the 5th", budgetPeriodMonth, "2026-01-05", "2026-08-15", "2026-08-05", "2026-09-04"},
		{"month from the 5th, before the anchor day", budgetPeriodMonth, "2026-01-05", "2026-08-03", "2026-07-05", "2026-08-04"},
		{"month anchored on the 31st does not skip February", budgetPeriodMonth, "2026-01-31", "2026-02-15", "2026-01-31", "2026-02-27"},
		{"week", budgetPeriodWeek, "2026-08-03", "2026-08-19", "2026-08-17", "2026-08-23"},
		{"year", budgetPeriodYear, "2024-03-01", "2026-08-15", "2026-03-01", "2027-02-28"},
		{"a budget that has not begun reports its first window", budgetPeriodMonth, "2026-09-01", "2026-08-15", "2026-09-01", "2026-09-30"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := budgetWindow(tc.period, mustDay(t, tc.start), mustDay(t, tc.asOf))
			if got.fromString() != tc.wantFrom || got.toString() != tc.wantTo {
				t.Errorf("window = %s..%s, want %s..%s",
					got.fromString(), got.toString(), tc.wantFrom, tc.wantTo)
			}
		})
	}
}

func TestSeriesBucketsEndOnTheCurrentBucket(t *testing.T) {
	now := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	buckets := seriesBuckets(financev1.PeriodGranularity_PERIOD_GRANULARITY_MONTH, 7, now, time.UTC, "monday")

	if len(buckets) != 7 {
		t.Fatalf("buckets = %d, want 7", len(buckets))
	}
	if buckets[0].fromString() != "2026-02-01" {
		t.Errorf("first bucket = %s, want 2026-02-01", buckets[0].fromString())
	}
	last := buckets[len(buckets)-1]
	if last.fromString() != "2026-08-01" || last.toString() != "2026-08-31" {
		t.Errorf("last bucket = %s..%s, want August", last.fromString(), last.toString())
	}
	if !last.contains(now) {
		t.Error("the last bucket must contain today — the chart outlines it by index")
	}
}

// advanceDue is the whole recurrence engine; the day anchor is the part that has an edge case
// in every month with fewer than 31 days.
func TestAdvanceDue(t *testing.T) {
	cases := []struct {
		name       string
		from       string
		interval   int32
		unit       string
		dayOfMonth int32
		want       string
	}{
		{"monthly", "2026-08-05", 1, "month", 5, "2026-09-05"},
		{"monthly on the 31st lands on the last day of a short month", "2026-01-31", 1, "month", 31, "2026-02-28"},
		{"and comes back to the 31st the month after", "2026-02-28", 1, "month", 31, "2026-03-31"},
		{"weekly", "2026-08-05", 1, "week", 0, "2026-08-12"},
		{"every two weeks", "2026-08-05", 2, "week", 0, "2026-08-19"},
		{"daily", "2026-08-05", 10, "day", 0, "2026-08-15"},
		{"yearly", "2026-08-05", 1, "year", 5, "2027-08-05"},
		{"a zero interval is treated as one", "2026-08-05", 0, "month", 5, "2026-09-05"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := advanceDue(mustDay(t, tc.from), tc.interval, tc.unit, tc.dayOfMonth)
			if got.Format(dateLayout) != tc.want {
				t.Errorf("advanceDue = %s, want %s", got.Format(dateLayout), tc.want)
			}
		})
	}
}

func TestBudgetStatusArithmetic(t *testing.T) {
	window := dayRange{from: mustDay(t, "2026-08-01"), to: mustDay(t, "2026-08-31")}
	today := mustDay(t, "2026-08-30")
	budget := budgetFixture(200_00)

	cases := map[int64]struct {
		share    float64
		exceeded bool
		left     int64
	}{
		0:      {0, false, 200_00},
		100_00: {0.5, false, 100_00},
		200_00: {1, false, 0},
		512_00: {2.56, true, -312_00},
	}
	for spent, want := range cases {
		status := budgetStatusFrom(budget, window, spent, today)
		if status.Share != want.share {
			t.Errorf("spent %d: share = %v, want %v", spent, status.Share, want.share)
		}
		if status.Exceeded != want.exceeded {
			t.Errorf("spent %d: exceeded = %v, want %v", spent, status.Exceeded, want.exceeded)
		}
		if status.Remaining.AmountMinor != want.left {
			t.Errorf("spent %d: remaining = %d, want %d", spent, status.Remaining.AmountMinor, want.left)
		}
		// 30 August in a window ending 31 August is two days including today.
		if status.DaysRemaining != 2 {
			t.Errorf("spent %d: days_remaining = %d, want 2", spent, status.DaysRemaining)
		}
	}
}

// budgetFixture is a group budget with only the fields the arithmetic reads.
func budgetFixture(limit int64) db.Budget {
	return db.Budget{
		TargetKind: targetGroup, LimitMinor: limit, CurrencyCode: "UAH",
		Period: budgetPeriodMonth, StartOn: pgtype.Date{Time: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}
}
