package schedule

import (
	"errors"
	"testing"
	"time"
)

func day(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC) }

func TestValidateBirthday(t *testing.T) {
	today := day(2026, 10, 5)
	cases := []struct {
		d, m, y, remind int
		err             error
	}{
		{14, 2, 0, 3, nil},
		{29, 2, 0, 0, nil},
		{29, 2, 2000, 0, nil},
		{29, 2, 2001, 0, ErrNotLeapYear},
		{31, 4, 0, 0, ErrBadDay},
		{0, 1, 0, 0, ErrBadDay},
		{1, 13, 0, 0, ErrBadMonth},
		{1, 1, 1899, 0, ErrBadYear},
		{1, 1, 2027, 0, ErrBadYear},
		{10, 12, 2026, 0, ErrBadYear},
		{5, 10, 2026, 0, nil},
		{1, 1, 0, 61, ErrBadRemindDays},
		{1, 1, 0, -1, ErrBadRemindDays},
	}
	for _, c := range cases {
		if err := ValidateBirthday(c.d, c.m, c.y, c.remind, today); !errors.Is(err, c.err) {
			t.Errorf("ValidateBirthday(%d/%d/%d, %d) = %v, want %v", c.d, c.m, c.y, c.remind, err, c.err)
		}
	}
}

func TestNextOccurrenceLaterThisYear(t *testing.T) {
	o := NextOccurrence(20, 10, 1990, day(2026, 10, 5))
	if !o.On.Equal(day(2026, 10, 20)) || o.DaysUntil != 15 || o.TurningAge != 36 {
		t.Fatalf("%+v", o)
	}
}

func TestNextOccurrenceToday(t *testing.T) {
	o := NextOccurrence(5, 10, 0, day(2026, 10, 5))
	if o.DaysUntil != 0 || o.HasAge {
		t.Fatalf("%+v", o)
	}
}

func TestNextOccurrenceRollsToNextYear(t *testing.T) {
	o := NextOccurrence(1, 1, 2000, day(2026, 10, 5))
	if !o.On.Equal(day(2027, 1, 1)) || o.DaysUntil != 88 || o.TurningAge != 27 {
		t.Fatalf("%+v", o)
	}
}

func TestLeapDayObservedOn28FebInCommonYears(t *testing.T) {
	o := NextOccurrence(29, 2, 2000, day(2026, 10, 5))
	if !o.On.Equal(day(2027, 2, 28)) || o.TurningAge != 27 {
		t.Fatalf("%+v", o)
	}
	o = NextOccurrence(29, 2, 2000, day(2027, 10, 5))
	if !o.On.Equal(day(2028, 2, 29)) || o.TurningAge != 28 {
		t.Fatalf("leap year should use 29 Feb: %+v", o)
	}
	o = NextOccurrence(29, 2, 0, day(2027, 2, 28))
	if o.DaysUntil != 0 {
		t.Fatalf("28 Feb in a common year is the observed day: %+v", o)
	}
}

func TestIsLeap(t *testing.T) {
	for y, want := range map[int]bool{2000: true, 1900: false, 2024: true, 2026: false} {
		if IsLeap(y) != want {
			t.Errorf("IsLeap(%d) != %v", y, want)
		}
	}
}

func TestNextOccurrenceNormalisesToday(t *testing.T) {
	o := NextOccurrence(5, 10, 0, time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC))
	if o.DaysUntil != 0 {
		t.Fatalf("a birthday today must not roll to next year: %+v", o)
	}
	ny, _ := time.LoadLocation("America/New_York")
	o = NextOccurrence(6, 10, 0, time.Date(2026, 10, 5, 22, 0, 0, 0, ny))
	if o.DaysUntil != 1 {
		t.Fatalf("today is 5 Oct in New York: %+v", o)
	}
}

func TestBornThisYearHasAgeZero(t *testing.T) {
	o := NextOccurrence(5, 10, 2026, day(2026, 10, 5))
	if !o.HasAge || o.TurningAge != 0 {
		t.Fatalf("%+v", o)
	}
}

func TestLeapDayFrom28FebInALeapYear(t *testing.T) {
	o := NextOccurrence(29, 2, 0, day(2028, 2, 28))
	if o.DaysUntil != 1 || !o.On.Equal(day(2028, 2, 29)) {
		t.Fatalf("%+v", o)
	}
}
