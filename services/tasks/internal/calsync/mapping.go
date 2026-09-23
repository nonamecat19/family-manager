package calsync

import (
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/gcal"
	"github.com/nnc/family-manager/services/tasks/internal/schedule"
)

const (
	KindTask     = "task"
	KindBirthday = "birthday"

	DonePrefix     = "✓ "
	BirthdayPrefix = "🎂 "

	TimedEventLength = 30 * time.Minute
)

func TaskEvent(t db.Task, loc *time.Location) (gcal.Event, bool) {
	if !t.DueOn.Valid {
		return gcal.Event{}, false
	}
	dl := schedule.Deadline{On: t.DueOn.Time.Format(schedule.DateLayout)}
	if t.DueTime.Valid {
		dl.Time = clock(t.DueTime)
	}

	e := gcal.Event{
		Summary:     t.Title,
		Description: t.Notes,
		Private:     privateProps(KindTask, t.ID, t.FamilyID),
	}
	if t.Status == "done" {
		e.Summary = DonePrefix + t.Title
	}
	if dl.HasTime() {
		end := dl.At(loc)
		e.Start, e.End, e.TimeZone = end.Add(-TimedEventLength), end, loc.String()
		return e, true
	}
	e.AllDay = true
	e.StartDate = dl.On
	e.EndDate = t.DueOn.Time.AddDate(0, 0, 1).Format(schedule.DateLayout)
	return e, true
}

func BirthdayEvent(b db.Birthday, now time.Time, loc *time.Location) gcal.Event {
	year := 0
	if b.Year != nil {
		year = int(*b.Year)
	}
	occ := schedule.NextOccurrence(int(b.Day), int(b.Month), year, now.In(loc))
	rule := "RRULE:FREQ=YEARLY"
	if b.Day == 29 && b.Month == 2 {
		rule = "RRULE:FREQ=YEARLY;BYMONTH=2;BYMONTHDAY=-1"
	}
	return gcal.Event{
		Summary:    BirthdayPrefix + b.Name,
		AllDay:     true,
		StartDate:  occ.On.Format(schedule.DateLayout),
		EndDate:    occ.On.AddDate(0, 0, 1).Format(schedule.DateLayout),
		Recurrence: []string{rule},
		Private:    privateProps(KindBirthday, b.ID, b.FamilyID),
	}
}

func TaskTitle(summary string) string {
	return strings.TrimPrefix(summary, DonePrefix)
}

func BirthdayName(summary string) string {
	return strings.TrimPrefix(summary, BirthdayPrefix)
}

func privateProps(kind string, itemID, familyID pgtype.UUID) map[string]string {
	return map[string]string{
		gcal.PropKind:   kind,
		gcal.PropItemID: pgconv.UUIDString(itemID),
		gcal.PropFamily: pgconv.UUIDString(familyID),
	}
}

func clock(t pgtype.Time) string {
	minutes := t.Microseconds / int64(time.Minute/time.Microsecond)
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}
