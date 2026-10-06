package schedule

import (
	"errors"
	"time"
)

const (
	DateLayout = "2006-01-02"
	TimeLayout = "15:04"
)

var (
	ErrBadDate         = errors.New("deadline date must be YYYY-MM-DD")
	ErrBadTime         = errors.New("deadline time must be HH:MM or HH:MM:SS")
	ErrTimeWithoutDate = errors.New("a deadline time needs a date")
)

type Deadline struct {
	On   string
	Time string
}

func parseClock(s string) (time.Time, error) {
	for _, layout := range []string{TimeLayout, "15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, ErrBadTime
}

func ParseDeadline(on, hhmm string) (Deadline, error) {
	if on == "" {
		if hhmm != "" {
			return Deadline{}, ErrTimeWithoutDate
		}
		return Deadline{}, nil
	}
	if _, err := time.Parse(DateLayout, on); err != nil {
		return Deadline{}, ErrBadDate
	}
	if hhmm == "" {
		return Deadline{On: on}, nil
	}
	clock, err := parseClock(hhmm)
	if err != nil {
		return Deadline{}, ErrBadTime
	}
	return Deadline{On: on, Time: clock.Format(TimeLayout)}, nil
}

func (d Deadline) IsZero() bool { return d.On == "" }

func (d Deadline) HasTime() bool { return d.Time != "" }

func wall(t time.Time) int64 {
	y, m, day := t.Date()
	h, mi, s := t.Clock()
	return time.Date(y, m, day, h, mi, s, 0, time.UTC).Unix()
}

func LocalTime(y int, m time.Month, day, h, mi, s int, loc *time.Location) time.Time {
	want := time.Date(y, m, day, h, mi, s, 0, time.UTC).Unix()
	t := time.Date(y, m, day, h, mi, s, 0, loc)
	for i := 0; i < 3; i++ {
		got := wall(t.In(loc))
		if got >= want {
			break
		}
		t = t.Add(time.Duration(want-got) * time.Second)
	}
	if wall(t.In(loc)) != want {
		return t
	}
	for _, back := range []time.Duration{3 * time.Hour, 2 * time.Hour, time.Hour, 30 * time.Minute} {
		if earlier := t.Add(-back); wall(earlier.In(loc)) == want {
			return earlier
		}
	}
	return t
}

func (d Deadline) At(loc *time.Location) time.Time {
	if d.IsZero() {
		return time.Time{}
	}
	day, err := time.Parse(DateLayout, d.On)
	if err != nil {
		return time.Time{}
	}
	if !d.HasTime() {
		next := day.AddDate(0, 0, 1)
		return LocalTime(next.Year(), next.Month(), next.Day(), 0, 0, 0, loc).Add(-time.Second)
	}
	clock, err := parseClock(d.Time)
	if err != nil {
		return time.Time{}
	}
	return LocalTime(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, loc)
}

func (d Deadline) Overdue(now time.Time, loc *time.Location) bool {
	at := d.At(loc)
	return !at.IsZero() && now.After(at)
}

func Civil(now time.Time, loc *time.Location) time.Time {
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

func Today(now time.Time, loc *time.Location) string {
	return Civil(now, loc).Format(DateLayout)
}

func DaysBetween(from, to time.Time) int {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}
