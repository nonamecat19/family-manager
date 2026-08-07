package handler

import (
	"fmt"
	"time"
)

// budgetWindow is the concrete date range a recurring budget covers on a given day.
//
// The whole point of deriving it is that a budget never needs a row per month: "500 a month
// from the 15th" is stored once, and the window that contains today is computed on read.
type budgetWindow struct {
	from time.Time
	to   time.Time // inclusive
}

// windowFor returns the occurrence of a budget's period that contains asOf.
//
// asOf before the anchor yields the first window: a budget starting next month reads as a
// full, untouched one rather than as a negative or empty range.
func windowFor(period string, startOn, asOf time.Time) (budgetWindow, error) {
	startOn = dayOf(startOn)
	asOf = dayOf(asOf)

	switch period {
	case periodWeek:
		return weekWindow(startOn, asOf), nil
	case periodMonth:
		return monthWindow(startOn, asOf), nil
	case periodYear:
		return yearWindow(startOn, asOf), nil
	default:
		return budgetWindow{}, fmt.Errorf("unknown budget period %q", period)
	}
}

// dayOf strips the clock, so a window boundary is a calendar day and not an instant. Windows
// are computed in the same local frame the dates were parsed in.
func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func weekWindow(startOn, asOf time.Time) budgetWindow {
	if asOf.Before(startOn) {
		return budgetWindow{from: startOn, to: startOn.AddDate(0, 0, 6)}
	}
	// Whole weeks elapsed since the anchor; integer division floors, which is what "the
	// window containing today" means.
	elapsed := int(asOf.Sub(startOn).Hours() / 24)
	from := startOn.AddDate(0, 0, (elapsed/7)*7)
	return budgetWindow{from: from, to: from.AddDate(0, 0, 6)}
}

func monthWindow(startOn, asOf time.Time) budgetWindow {
	if asOf.Before(startOn) {
		return budgetWindow{from: startOn, to: addMonthClamped(startOn, 1).AddDate(0, 0, -1)}
	}

	months := (asOf.Year()-startOn.Year())*12 + int(asOf.Month()-startOn.Month())
	from := addMonthClamped(startOn, months)
	// The anchor day may not exist in every month (the 31st), so the computed start can land
	// after asOf; step back one occurrence when it does.
	if from.After(asOf) {
		months--
		from = addMonthClamped(startOn, months)
	}
	return budgetWindow{from: from, to: addMonthClamped(startOn, months+1).AddDate(0, 0, -1)}
}

func yearWindow(startOn, asOf time.Time) budgetWindow {
	if asOf.Before(startOn) {
		return budgetWindow{from: startOn, to: addYearClamped(startOn, 1).AddDate(0, 0, -1)}
	}

	years := asOf.Year() - startOn.Year()
	from := addYearClamped(startOn, years)
	if from.After(asOf) {
		years--
		from = addYearClamped(startOn, years)
	}
	return budgetWindow{from: from, to: addYearClamped(startOn, years+1).AddDate(0, 0, -1)}
}

// addMonthClamped adds months and clamps the day to the target month's length. Go's AddDate
// normalises instead: 31 January plus one month is 3 March, which would silently shift a
// budget's anchor forward every short month until it stopped matching what the user set.
func addMonthClamped(t time.Time, months int) time.Time {
	target := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, months, 0)
	day := t.Day()
	if last := daysIn(target.Year(), target.Month()); day > last {
		day = last
	}
	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, time.UTC)
}

// addYearClamped exists for the same reason as addMonthClamped: 29 February plus one year.
func addYearClamped(t time.Time, years int) time.Time {
	year := t.Year() + years
	day := t.Day()
	if last := daysIn(year, t.Month()); day > last {
		day = last
	}
	return time.Date(year, t.Month(), day, 0, 0, 0, 0, time.UTC)
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// daysRemaining counts asOf itself, so the last day of a window reads 1 rather than 0 — "you
// have one day left" is true on the final day. A window already past reads 0.
func (w budgetWindow) daysRemaining(asOf time.Time) int32 {
	asOf = dayOf(asOf)
	if asOf.After(w.to) {
		return 0
	}
	if asOf.Before(w.from) {
		asOf = w.from
	}
	return int32(w.to.Sub(asOf).Hours()/24) + 1
}
