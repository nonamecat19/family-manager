package schedule

import (
	"errors"
	"testing"
	"time"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestParseDeadline(t *testing.T) {
	cases := []struct {
		on, at string
		err    error
	}{
		{"", "", nil},
		{"2026-10-05", "", nil},
		{"2026-10-05", "18:30", nil},
		{"", "18:30", ErrTimeWithoutDate},
		{"2026-13-01", "", ErrBadDate},
		{"05.10.2026", "", ErrBadDate},
		{"2026-10-05", "25:00", ErrBadTime},
		{"2026-10-05", "6pm", ErrBadTime},
		{"2026-10-05", "9:05", nil},
		{"2026-10-05", "18:30:00", nil},
	}
	for _, c := range cases {
		_, err := ParseDeadline(c.on, c.at)
		if !errors.Is(err, c.err) {
			t.Errorf("ParseDeadline(%q, %q) = %v, want %v", c.on, c.at, err, c.err)
		}
	}
}

func TestDateOnlyDeadlineIsEndOfDayInFamilyTimezone(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	d, _ := ParseDeadline("2026-10-05", "")
	at := d.At(kyiv)
	want := time.Date(2026, 10, 5, 23, 59, 59, 0, kyiv)
	if !at.Equal(want) {
		t.Fatalf("At = %v, want %v", at, want)
	}
	if at.UTC().Hour() != 20 {
		t.Fatalf("Kyiv end of day should be 20:59 UTC in October, got %v", at.UTC())
	}
}

func TestTimedDeadlineUsesWallClock(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	d, _ := ParseDeadline("2026-07-01", "09:15")
	if got := d.At(ny).UTC(); !got.Equal(time.Date(2026, 7, 1, 13, 15, 0, 0, time.UTC)) {
		t.Fatalf("At = %v", got)
	}
}

func TestDeadlineInsideSpringForwardGap(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	d, _ := ParseDeadline("2026-03-29", "03:30")
	at := d.At(kyiv)
	if at.IsZero() || at.In(kyiv).Day() != 29 {
		t.Fatalf("a wall time skipped by DST must still land on the same day, got %v", at)
	}
}

func TestDeadlineAcrossFallBack(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	d, _ := ParseDeadline("2026-10-25", "")
	end := d.At(kyiv)
	start := time.Date(2026, 10, 25, 0, 0, 0, 0, kyiv)
	if hours := end.Sub(start).Hours(); hours < 24 || hours > 25 {
		t.Fatalf("the fall-back day is 25 hours long, end-of-day is %v hours after midnight", hours)
	}
}

func TestOverdue(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	d, _ := ParseDeadline("2026-10-05", "")
	if d.Overdue(time.Date(2026, 10, 5, 20, 0, 0, 0, time.UTC), kyiv) {
		t.Fatal("still 23:00 in Kyiv, not overdue")
	}
	if !d.Overdue(time.Date(2026, 10, 5, 21, 0, 0, 0, time.UTC), kyiv) {
		t.Fatal("already 00:00 next day in Kyiv, should be overdue")
	}
	if (Deadline{}).Overdue(time.Now(), kyiv) {
		t.Fatal("no deadline is never overdue")
	}
}

func TestTodayFollowsTimezone(t *testing.T) {
	now := time.Date(2026, 10, 5, 22, 30, 0, 0, time.UTC)
	if got := Today(now, mustLoc(t, "Europe/Kyiv")); got != "2026-10-06" {
		t.Fatalf("Kyiv today = %s", got)
	}
	if got := Today(now, mustLoc(t, "America/Los_Angeles")); got != "2026-10-05" {
		t.Fatalf("LA today = %s", got)
	}
}

func TestDeadlineTimeIsNormalised(t *testing.T) {
	d, _ := ParseDeadline("2026-10-05", "9:05")
	if d.Time != "09:05" {
		t.Fatalf("time = %q", d.Time)
	}
	d, _ = ParseDeadline("2026-10-05", "18:30:00")
	if d.Time != "18:30" {
		t.Fatalf("a Postgres TIME reads back as HH:MM:SS: %q", d.Time)
	}
}

func TestMidnightGapKeepsTheDay(t *testing.T) {
	santiago := mustLoc(t, "America/Santiago")
	d, _ := ParseDeadline("2026-09-06", "00:30")
	at := d.At(santiago).In(santiago)
	if at.Format("2006-01-02") != "2026-09-06" || at.Hour() != 1 || at.Minute() != 30 {
		t.Fatalf("00:30 does not exist on 6 Sep in Santiago; expected 01:30 that day, got %v", at)
	}
}

func TestEndOfDayWhenTheGapStartsAt2300(t *testing.T) {
	dhaka := mustLoc(t, "Asia/Dhaka")
	d, _ := ParseDeadline("2009-06-19", "")
	end := d.At(dhaka)
	if end.In(dhaka).Format("2006-01-02") != "2009-06-19" {
		t.Fatalf("end of day leaked into the next day: %v", end.In(dhaka))
	}
}

func TestStoredTimeWithSecondsIsNotOverdue(t *testing.T) {
	d := Deadline{On: "2026-10-05", Time: "18:30:00"}
	if d.Overdue(time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC), time.UTC) {
		t.Fatal("HH:MM:SS must parse, not read as zero time")
	}
	bad := Deadline{On: "2026-10-05", Time: "nonsense"}
	if bad.Overdue(time.Now(), time.UTC) {
		t.Fatal("an unparseable deadline is never overdue")
	}
}

func TestDaysBetweenIgnoresTimeOfDay(t *testing.T) {
	a := time.Date(2026, 10, 5, 23, 0, 0, 0, time.UTC)
	b := time.Date(2026, 10, 6, 1, 0, 0, 0, time.UTC)
	if DaysBetween(a, b) != 1 {
		t.Fatalf("%d", DaysBetween(a, b))
	}
}

func TestOverlapPicksTheEarlierInstant(t *testing.T) {
	london := mustLoc(t, "Europe/London")
	d, _ := ParseDeadline("2026-10-25", "01:30")
	at := d.At(london)
	if _, offset := at.In(london).Zone(); offset != 3600 {
		t.Fatalf("01:30 happens twice on 25 Oct in London; the deadline is the first (BST), got offset %d", offset)
	}
}

func TestEndOfDayWhenMidnightRepeats(t *testing.T) {
	amman := mustLoc(t, "Asia/Amman")
	d, _ := ParseDeadline("2015-10-29", "")
	if got := d.At(amman).In(amman).Format("2006-01-02"); got != "2015-10-29" {
		t.Fatalf("end of day leaked to %s", got)
	}
}
