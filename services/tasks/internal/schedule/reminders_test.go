package schedule

import (
	"errors"
	"testing"
	"time"
)

func TestTimedTaskRemindsAnHourBefore(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	d, _ := ParseDeadline("2026-10-05", "18:00")
	if got := TaskRemindAt(d, kyiv); !got.Equal(time.Date(2026, 10, 5, 17, 0, 0, 0, kyiv)) {
		t.Fatalf("remind at %v", got)
	}
}

func TestDateOnlyTaskRemindsThatMorning(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	d, _ := ParseDeadline("2026-10-05", "")
	if got := TaskRemindAt(d, kyiv); !got.Equal(time.Date(2026, 10, 5, 9, 0, 0, 0, kyiv)) {
		t.Fatalf("remind at %v", got)
	}
	if !TaskRemindAt(Deadline{}, kyiv).IsZero() {
		t.Fatal("no deadline, no reminder")
	}
}

func TestBirthdayRemindsEarlyAndOnTheDay(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	occ := NextOccurrence(20, 10, 0, day(2026, 10, 5))
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, kyiv)
	got := BirthdayRemindAt(occ, 3, kyiv, now)
	want := []time.Time{time.Date(2026, 10, 17, 9, 0, 0, 0, kyiv), time.Date(2026, 10, 20, 9, 0, 0, 0, kyiv)}
	if len(got) != 2 || !got[0].Equal(want[0]) || !got[1].Equal(want[1]) {
		t.Fatalf("%v", got)
	}
	if only := BirthdayRemindAt(occ, 0, kyiv, now); len(only) != 1 || !only[0].Equal(want[1]) {
		t.Fatalf("0 days before means on the day only: %v", only)
	}
}

func TestBirthdayEarlyReminderCrossesYearEnd(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	occ := NextOccurrence(2, 1, 0, day(2026, 12, 20))
	got := BirthdayRemindAt(occ, 5, kyiv, time.Date(2026, 12, 20, 12, 0, 0, 0, kyiv))
	if !got[0].Equal(time.Date(2026, 12, 28, 9, 0, 0, 0, kyiv)) {
		t.Fatalf("%v", got)
	}
}

func TestDigestDueOncePerDayAfterEight(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	before := time.Date(2026, 10, 5, 7, 59, 0, 0, kyiv)
	after := time.Date(2026, 10, 5, 8, 0, 0, 0, kyiv)
	if DigestDue(before, kyiv, "") {
		t.Fatal("not due before 08:00")
	}
	if !DigestDue(after, kyiv, "2026-10-04") {
		t.Fatal("due at 08:00 when not sent today")
	}
	if DigestDue(after, kyiv, "2026-10-05") {
		t.Fatal("already sent today")
	}
}

func TestSnooze(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	now := time.Date(2026, 10, 5, 22, 30, 0, 0, kyiv)
	if got, _ := SnoozeUntil(SnoozeOneHour, now, kyiv); !got.Equal(now.Add(time.Hour)) {
		t.Fatalf("one hour: %v", got)
	}
	if got, _ := SnoozeUntil(SnoozeTomorrowMorning, now, kyiv); !got.Equal(time.Date(2026, 10, 6, 9, 0, 0, 0, kyiv)) {
		t.Fatalf("tomorrow morning: %v", got)
	}
	late := time.Date(2026, 10, 5, 23, 30, 0, 0, time.UTC)
	if got, _ := SnoozeUntil(SnoozeTomorrowMorning, late, kyiv); !got.Equal(time.Date(2026, 10, 7, 9, 0, 0, 0, kyiv)) {
		t.Fatalf("23:30 UTC is already 6 Oct in Kyiv, so tomorrow is 7 Oct: %v", got)
	}
}

func TestSnoozeRejectsUnknownOptions(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	for _, o := range []Snooze{0, 99} {
		if _, err := SnoozeUntil(o, time.Now(), kyiv); !errors.Is(err, ErrBadSnooze) {
			t.Errorf("option %d: %v", o, err)
		}
	}
}

func TestBirthdayRemindersInThePastAreDropped(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	occ := NextOccurrence(7, 10, 0, day(2026, 10, 5))
	got := BirthdayRemindAt(occ, 5, kyiv, time.Date(2026, 10, 5, 12, 0, 0, 0, kyiv))
	if len(got) != 1 || !got[0].Equal(time.Date(2026, 10, 7, 9, 0, 0, 0, kyiv)) {
		t.Fatalf("the early reminder (2 Oct) is already past: %v", got)
	}
	today := NextOccurrence(5, 10, 0, day(2026, 10, 5))
	if left := BirthdayRemindAt(today, 0, kyiv, time.Date(2026, 10, 5, 10, 0, 0, 0, kyiv)); len(left) != 0 {
		t.Fatalf("09:00 today has passed: %v", left)
	}
}

func TestDigestNotRepeatedAfterMovingWest(t *testing.T) {
	la := mustLoc(t, "America/Los_Angeles")
	now := time.Date(2026, 10, 5, 7, 0, 0, 0, time.UTC)
	if DigestDue(now, la, "2026-10-05") {
		t.Fatal("already sent for 5 Oct in Kyiv; 4 Oct in LA must not get a second digest")
	}
	if !DigestDue(time.Date(2026, 10, 5, 16, 0, 0, 0, time.UTC), la, "2026-10-04") {
		t.Fatal("08:00+ on 5 Oct in LA, last sent 4 Oct: due")
	}
}

func TestReminderInsideMidnightGap(t *testing.T) {
	santiago := mustLoc(t, "America/Santiago")
	d, _ := ParseDeadline("2026-09-06", "00:30")
	at := d.At(santiago)
	if got := at.In(santiago).Format("2006-01-02"); got != "2026-09-06" {
		t.Fatalf("deadline moved to %s", got)
	}
	if remind := TaskRemindAt(d, santiago); !remind.Before(at) {
		t.Fatalf("reminder %v not before deadline %v", remind, at)
	}
}
