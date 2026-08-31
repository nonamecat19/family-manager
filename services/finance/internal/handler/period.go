package handler

import (
	"fmt"
	"time"

	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
)

// dateLayout is the only date format that crosses this service's wire. Calendar dates, not
// timestamps: "what did we spend in August" is a calendar question, and an instant would move
// the answer with the reader's offset.
const dateLayout = "2006-01-02"

// dayRange is an inclusive calendar window, resolved once per request so that the screen, the
// budget bar and the widget all read the same edges.
type dayRange struct {
	from time.Time
	to   time.Time
}

func (r dayRange) fromString() string { return r.from.Format(dateLayout) }
func (r dayRange) toString() string   { return r.to.Format(dateLayout) }

func (r dayRange) proto() *financev1.DateRange {
	return &financev1.DateRange{From: r.fromString(), To: r.toString()}
}

// contains is inclusive on both ends, matching the SQL the same range is passed to.
func (r dayRange) contains(d time.Time) bool {
	return !d.Before(r.from) && !d.After(r.to)
}

// parseDay reads a YYYY-MM-DD in loc. An empty string is not an error here — every caller
// decides its own default (today, the budget's anchor) — so callers check ok.
func parseDay(s string, loc *time.Location) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(dateLayout, s, loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// startOfDay drops the clock, which is what makes two dates comparable as calendar days
// regardless of which instant produced them.
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// weekdayIndex maps the stored week_starts_on to Go's Weekday. Monday is the fallback: the
// design's week starts on Monday and a household that never chose should get that, not the
// device locale's answer.
func weekdayIndex(name string) time.Weekday {
	switch name {
	case "sunday":
		return time.Sunday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	default:
		return time.Monday
	}
}

// startOfWeek walks back to the household's chosen first weekday.
func startOfWeek(t time.Time, first time.Weekday) time.Time {
	d := startOfDay(t)
	for d.Weekday() != first {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// addMonthsClamped is AddDate with the clamp Go's normalisation does not do: 31 January plus
// one month is 28 February, not 3 March. A budget anchored on the 31st must not skip a month.
func addMonthsClamped(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m, 1, 0, 0, 0, 0, t.Location()).AddDate(0, n, 0)
	last := first.AddDate(0, 1, -1).Day()
	if d > last {
		d = last
	}
	return time.Date(first.Year(), first.Month(), d, 0, 0, 0, 0, t.Location())
}

// resolvePeriod turns the app's Period into the window the queries use. An unset Period is
// the current month, which is what every screen opens on.
func resolvePeriod(p *financev1.Period, now time.Time, loc *time.Location, weekStart string) (dayRange, error) {
	today := startOfDay(now.In(loc))

	anchor := today
	if a, ok := parseDay(p.GetAnchor(), loc); ok {
		anchor = a
	}

	switch p.GetGranularity() {
	case financev1.PeriodGranularity_PERIOD_GRANULARITY_DAY:
		return dayRange{from: anchor, to: anchor}, nil

	case financev1.PeriodGranularity_PERIOD_GRANULARITY_WEEK:
		from := startOfWeek(anchor, weekdayIndex(weekStart))
		return dayRange{from: from, to: from.AddDate(0, 0, 6)}, nil

	case financev1.PeriodGranularity_PERIOD_GRANULARITY_YEAR:
		from := time.Date(anchor.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return dayRange{from: from, to: from.AddDate(1, 0, -1)}, nil

	case financev1.PeriodGranularity_PERIOD_GRANULARITY_CUSTOM:
		from, okFrom := parseDay(p.GetRange().GetFrom(), loc)
		to, okTo := parseDay(p.GetRange().GetTo(), loc)
		if !okFrom || !okTo {
			return dayRange{}, fmt.Errorf("custom period needs range.from and range.to as YYYY-MM-DD")
		}
		if to.Before(from) {
			return dayRange{}, fmt.Errorf("period range ends before it starts")
		}
		return dayRange{from: from, to: to}, nil

	default:
		// UNSPECIFIED and MONTH are the same answer: the calendar month around the anchor.
		from := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, loc)
		return dayRange{from: from, to: from.AddDate(0, 1, -1)}, nil
	}
}

// budgetWindow is the rolling window a budget is evaluated in: the one containing asOf,
// counted from the budget's own anchor rather than from the calendar, so a household that
// budgets from the 5th is not forced onto calendar months.
//
// asOf before the anchor reports the first window: a budget starting next month is not
// "0 spent of an undefined window", it is the window that has not begun yet.
func budgetWindow(period string, start, asOf time.Time) dayRange {
	start = startOfDay(start)
	asOf = startOfDay(asOf)

	switch period {
	case budgetPeriodWeek:
		n := int(asOf.Sub(start).Hours()/24) / 7
		if asOf.Before(start) {
			n = 0
		}
		from := start.AddDate(0, 0, n*7)
		return dayRange{from: from, to: from.AddDate(0, 0, 6)}

	case budgetPeriodYear:
		n := asOf.Year() - start.Year()
		if n < 0 {
			n = 0
		}
		from := addMonthsClamped(start, n*12)
		if from.After(asOf) && n > 0 {
			n--
			from = addMonthsClamped(start, n*12)
		}
		return dayRange{from: from, to: addMonthsClamped(start, (n+1)*12).AddDate(0, 0, -1)}

	default: // month
		n := (asOf.Year()-start.Year())*12 + int(asOf.Month()) - int(start.Month())
		if n < 0 {
			n = 0
		}
		from := addMonthsClamped(start, n)
		if from.After(asOf) && n > 0 {
			n--
			from = addMonthsClamped(start, n)
		}
		return dayRange{from: from, to: addMonthsClamped(start, n+1).AddDate(0, 0, -1)}
	}
}

// seriesBuckets is the chart's x-axis: count buckets ending with the one containing now,
// oldest first, so the current bucket is always the last and the chart can outline it without
// a second request.
func seriesBuckets(g financev1.PeriodGranularity, count int, now time.Time, loc *time.Location, weekStart string) []dayRange {
	if count <= 0 {
		count = 7
	}
	today := startOfDay(now.In(loc))

	out := make([]dayRange, 0, count)
	for i := count - 1; i >= 0; i-- {
		switch g {
		case financev1.PeriodGranularity_PERIOD_GRANULARITY_DAY:
			d := today.AddDate(0, 0, -i)
			out = append(out, dayRange{from: d, to: d})
		case financev1.PeriodGranularity_PERIOD_GRANULARITY_WEEK:
			from := startOfWeek(today, weekdayIndex(weekStart)).AddDate(0, 0, -7*i)
			out = append(out, dayRange{from: from, to: from.AddDate(0, 0, 6)})
		case financev1.PeriodGranularity_PERIOD_GRANULARITY_YEAR:
			from := time.Date(today.Year()-i, time.January, 1, 0, 0, 0, 0, loc)
			out = append(out, dayRange{from: from, to: from.AddDate(1, 0, -1)})
		default: // month
			from := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -i, 0)
			out = append(out, dayRange{from: from, to: from.AddDate(0, 1, -1)})
		}
	}
	return out
}

// bucketLabel is the axis label. Numeric and locale-free on purpose: the app draws "Лют" from
// the start date, and a server-side Ukrainian month name would have to be translated twice.
func bucketLabel(g financev1.PeriodGranularity, r dayRange) string {
	switch g {
	case financev1.PeriodGranularity_PERIOD_GRANULARITY_DAY:
		return r.from.Format("2006-01-02")
	case financev1.PeriodGranularity_PERIOD_GRANULARITY_WEEK:
		return r.from.Format("2006-01-02")
	case financev1.PeriodGranularity_PERIOD_GRANULARITY_YEAR:
		return r.from.Format("2006")
	default:
		return r.from.Format("2006-01")
	}
}
