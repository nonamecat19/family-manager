package schedule

import (
	"errors"
	"time"
)

const (
	DigestHour        = 8
	MorningHour       = 9
	TimedDeadlineLead = time.Hour
)

var ErrBadSnooze = errors.New("unknown snooze option")

type Snooze int

const (
	SnoozeOneHour Snooze = iota + 1
	SnoozeTomorrowMorning
)

func atLocal(day time.Time, hour int, loc *time.Location) time.Time {
	return LocalTime(day.Year(), day.Month(), day.Day(), hour, 0, 0, loc)
}

func TaskRemindAt(d Deadline, loc *time.Location) time.Time {
	if d.IsZero() {
		return time.Time{}
	}
	if d.HasTime() {
		at := d.At(loc)
		if at.IsZero() {
			return time.Time{}
		}
		return at.Add(-TimedDeadlineLead)
	}
	day, err := time.Parse(DateLayout, d.On)
	if err != nil {
		return time.Time{}
	}
	return atLocal(day, MorningHour, loc)
}

func BirthdayRemindAt(occ Occurrence, remindDaysBefore int, loc *time.Location, now time.Time) []time.Time {
	candidates := []time.Time{atLocal(occ.On, MorningHour, loc)}
	if remindDaysBefore > 0 {
		early := atLocal(occ.On.AddDate(0, 0, -remindDaysBefore), MorningHour, loc)
		candidates = append([]time.Time{early}, candidates...)
	}
	out := candidates[:0]
	for _, c := range candidates {
		if !c.Before(now) {
			out = append(out, c)
		}
	}
	return out
}

func DigestAt(now time.Time, loc *time.Location) time.Time {
	return atLocal(Civil(now, loc), DigestHour, loc)
}

func DigestDue(now time.Time, loc *time.Location, lastSentOn string) bool {
	return !now.Before(DigestAt(now, loc)) && lastSentOn < Today(now, loc)
}

func SnoozeUntil(option Snooze, now time.Time, loc *time.Location) (time.Time, error) {
	switch option {
	case SnoozeOneHour:
		return now.Add(time.Hour), nil
	case SnoozeTomorrowMorning:
		return atLocal(Civil(now, loc).AddDate(0, 0, 1), MorningHour, loc), nil
	}
	return time.Time{}, ErrBadSnooze
}
