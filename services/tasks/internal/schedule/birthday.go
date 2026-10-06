package schedule

import (
	"errors"
	"time"
)

var (
	ErrBadMonth      = errors.New("month must be 1-12")
	ErrBadDay        = errors.New("day does not exist in that month")
	ErrBadYear       = errors.New("birth date must be between 1900 and today")
	ErrNotLeapYear   = errors.New("29 February needs a leap birth year")
	ErrBadRemindDays = errors.New("remind days before must be 0-60")
)

const MaxRemindDays = 60

func IsLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func daysIn(month, year int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func civilDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func ValidateBirthday(day, month, year, remindDays int, today time.Time) error {
	today = civilDate(today)
	if month < 1 || month > 12 {
		return ErrBadMonth
	}
	if day < 1 || day > daysIn(month, 2000) {
		return ErrBadDay
	}
	if year != 0 {
		if month == 2 && day == 29 && !IsLeap(year) {
			return ErrNotLeapYear
		}
		born := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if year < 1900 || born.After(today) {
			return ErrBadYear
		}
	}
	if remindDays < 0 || remindDays > MaxRemindDays {
		return ErrBadRemindDays
	}
	return nil
}

type Occurrence struct {
	On         time.Time
	DaysUntil  int
	TurningAge int
	HasAge     bool
}

func observedOn(day, month, year int) time.Time {
	if month == 2 && day == 29 && !IsLeap(year) {
		day = 28
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func NextOccurrence(day, month, birthYear int, today time.Time) Occurrence {
	today = civilDate(today)
	y := today.Year()
	on := observedOn(day, month, y)
	if on.Before(today) {
		y++
		on = observedOn(day, month, y)
	}
	occ := Occurrence{On: on, DaysUntil: DaysBetween(today, on)}
	if birthYear > 0 {
		occ.TurningAge = y - birthYear
		occ.HasAge = true
	}
	return occ
}
