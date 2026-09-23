package calsync

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/crypto"
	"github.com/nnc/family-manager/services/tasks/internal/gcal"
)

const (
	family = "00000000-0000-4000-8000-000000000001"
	ann    = "00000000-0000-4000-8000-000000000002"
	bob    = "00000000-0000-4000-8000-000000000003"
	taskA  = "00000000-0000-4000-8000-0000000000a1"
	bdayB  = "00000000-0000-4000-8000-0000000000b1"

	annCal    = "ann@example.com"
	sharedCal = "family@group.calendar.google.com"
)

var testNow = time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)

type store struct {
	timezone        string
	tasks           map[string]db.Task
	birthdays       map[string]db.Birthday
	conns           map[string]db.GoogleConnection
	links           map[string]db.CalendarLink
	failLink        error
	slowLink        time.Duration
	afterOrphanList func()
}

func s(u pgtype.UUID) string { return pgconv.UUIDString(u) }

func linkKey(user pgtype.UUID, kind string, item pgtype.UUID) string {
	return s(user) + "|" + kind + "|" + s(item)
}

func (st *store) GetTask(_ context.Context, arg db.GetTaskParams) (db.Task, error) {
	t, ok := st.tasks[s(arg.ID)]
	if !ok || s(t.FamilyID) != s(arg.FamilyID) {
		return db.Task{}, pgx.ErrNoRows
	}
	return t, nil
}

func (st *store) GetBirthday(_ context.Context, arg db.GetBirthdayParams) (db.Birthday, error) {
	b, ok := st.birthdays[s(arg.ID)]
	if !ok || s(b.FamilyID) != s(arg.FamilyID) {
		return db.Birthday{}, pgx.ErrNoRows
	}
	return b, nil
}

func (st *store) GetFamilySettings(_ context.Context, familyID pgtype.UUID) (db.FamilySetting, error) {
	return db.FamilySetting{FamilyID: familyID, Timezone: st.timezone}, nil
}

func (st *store) GetGoogleConnection(_ context.Context, arg db.GetGoogleConnectionParams) (db.GoogleConnection, error) {
	c, ok := st.conns[s(arg.UserID)]
	if !ok {
		return db.GoogleConnection{}, pgx.ErrNoRows
	}
	return c, nil
}

func (st *store) ListFamilyGoogleConnections(_ context.Context, _ pgtype.UUID) ([]db.GoogleConnection, error) {
	var out []db.GoogleConnection
	for _, u := range []string{ann, bob} {
		if c, ok := st.conns[u]; ok && c.CalendarID != "" {
			out = append(out, c)
		}
	}
	return out, nil
}

func (st *store) SetGoogleError(_ context.Context, arg db.SetGoogleErrorParams) error {
	c := st.conns[s(arg.UserID)]
	c.LastError = arg.LastError
	st.conns[s(arg.UserID)] = c
	return nil
}

func (st *store) GetCalendarLink(_ context.Context, arg db.GetCalendarLinkParams) (db.CalendarLink, error) {
	time.Sleep(st.slowLink)
	l, ok := st.links[linkKey(arg.UserID, arg.Kind, arg.ItemID)]
	if !ok {
		return db.CalendarLink{}, pgx.ErrNoRows
	}
	return l, nil
}

func (st *store) ListCalendarLinksForItem(_ context.Context, arg db.ListCalendarLinksForItemParams) ([]db.CalendarLink, error) {
	var out []db.CalendarLink
	for _, l := range st.links {
		if l.Kind == arg.Kind && s(l.ItemID) == s(arg.ItemID) {
			out = append(out, l)
		}
	}
	return out, nil
}

func (st *store) ListCalendarLinksForUser(_ context.Context, arg db.ListCalendarLinksForUserParams) ([]db.CalendarLink, error) {
	var out []db.CalendarLink
	for _, l := range st.links {
		if s(l.FamilyID) == s(arg.FamilyID) && s(l.UserID) == s(arg.UserID) {
			out = append(out, l)
		}
	}
	return out, nil
}

func (st *store) ListOrphanCalendarLinks(_ context.Context, arg db.ListOrphanCalendarLinksParams) ([]db.CalendarLink, error) {
	var out []db.CalendarLink
	for _, l := range st.links {
		if s(l.UserID) != s(arg.UserID) {
			continue
		}
		t, task := st.tasks[s(l.ItemID)]
		_, bday := st.birthdays[s(l.ItemID)]
		if (l.Kind == KindTask && (!task || !t.DueOn.Valid)) || (l.Kind == KindBirthday && !bday) {
			out = append(out, l)
		}
	}
	if st.afterOrphanList != nil {
		st.afterOrphanList()
	}
	return out, nil
}

func (st *store) CalendarLinkedInFamily(_ context.Context, arg db.CalendarLinkedInFamilyParams) (bool, error) {
	for _, l := range st.links {
		if l.Kind == arg.Kind && s(l.ItemID) == s(arg.ItemID) && l.CalendarID == arg.CalendarID && s(l.UserID) != s(arg.UserID) {
			return true, nil
		}
	}
	return false, nil
}

func (st *store) UpsertCalendarLink(_ context.Context, arg db.UpsertCalendarLinkParams) error {
	if st.failLink != nil {
		return st.failLink
	}
	st.links[linkKey(arg.UserID, arg.Kind, arg.ItemID)] = db.CalendarLink{
		FamilyID: arg.FamilyID, UserID: arg.UserID, Kind: arg.Kind, ItemID: arg.ItemID,
		CalendarID: arg.CalendarID, EventID: arg.EventID, Etag: arg.Etag,
	}
	return nil
}

func (st *store) DeleteCalendarLink(_ context.Context, arg db.DeleteCalendarLinkParams) error {
	delete(st.links, linkKey(arg.UserID, arg.Kind, arg.ItemID))
	return nil
}

type harness struct {
	st     *store
	google *gcal.Fake
	box    *crypto.Box
	p      *Pusher
}

type blockingDeleteClient struct {
	gcal.Client
	started chan struct{}
	release chan struct{}
}

func (c *blockingDeleteClient) DeleteEvent(ctx context.Context, accessToken, calendarID, eventID string) error {
	close(c.started)
	<-c.release
	return c.Client.DeleteEvent(ctx, accessToken, calendarID, eventID)
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	box, err := crypto.NewBox(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	if err != nil {
		t.Fatal(err)
	}
	google := gcal.NewFake()
	google.SetNow(func() time.Time { return testNow })
	h := &harness{
		st: &store{
			timezone: "UTC", tasks: map[string]db.Task{}, birthdays: map[string]db.Birthday{},
			conns: map[string]db.GoogleConnection{}, links: map[string]db.CalendarLink{},
		},
		google: google,
		box:    box,
	}
	h.p = NewPusher(Options{Store: h.st, Google: google, Box: box, Now: func() time.Time { return testNow }})
	return h
}

func (h *harness) connect(t *testing.T, user, calendarID string) {
	t.Helper()
	code := "code-" + user
	h.google.AddCode(code, "v", user+"@example.com")
	tok, err := h.google.Exchange(context.Background(), code, "v", "")
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := h.box.Seal(tok.RefreshToken, crypto.Owner(family, user))
	if err != nil {
		t.Fatal(err)
	}
	h.st.conns[user] = db.GoogleConnection{
		FamilyID: pgconv.MustUUID(family), UserID: pgconv.MustUUID(user),
		Email: user + "@example.com", RefreshToken: sealed, CalendarID: calendarID,
	}
}

func (h *harness) putTask(t db.Task) {
	t.ID, t.FamilyID = pgconv.MustUUID(taskA), pgconv.MustUUID(family)
	if t.Status == "" {
		t.Status = "open"
	}
	h.st.tasks[taskA] = t
}

func (h *harness) pushTask(t *testing.T) {
	t.Helper()
	if err := h.p.PushTask(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(taskA)); err != nil {
		t.Fatalf("PushTask: %v", err)
	}
}

func (h *harness) link(user, kind, item string) (db.CalendarLink, bool) {
	l, ok := h.st.links[linkKey(pgconv.MustUUID(user), kind, pgconv.MustUUID(item))]
	return l, ok
}

func (h *harness) event(t *testing.T, user, kind, item string) gcal.Event {
	t.Helper()
	l, ok := h.link(user, kind, item)
	if !ok {
		t.Fatalf("no calendar link for %s %s", user, kind)
	}
	e, ok := h.google.Event(l.CalendarID, l.EventID)
	if !ok {
		t.Fatalf("event %s missing in %s", l.EventID, l.CalendarID)
	}
	return e
}

func date(s string) pgtype.Date {
	d, _ := time.Parse("2006-01-02", s)
	return pgtype.Date{Time: d, Valid: true}
}

func clockAt(h, m int) pgtype.Time {
	return pgtype.Time{Microseconds: int64(h*3600+m*60) * 1e6, Valid: true}
}

func TestDateOnlyTaskBecomesAllDayEvent(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", Notes: "by card", DueOn: date("2026-09-03")})

	h.pushTask(t)

	e := h.event(t, ann, KindTask, taskA)
	if !e.AllDay || e.StartDate != "2026-09-03" || e.EndDate != "2026-09-04" {
		t.Errorf("event dates = %v %s..%s, want all-day 2026-09-03..2026-09-04", e.AllDay, e.StartDate, e.EndDate)
	}
	if e.Summary != "Pay rent" || e.Description != "by card" {
		t.Errorf("summary/description = %q/%q", e.Summary, e.Description)
	}
	if e.Private[gcal.PropKind] != KindTask || e.Private[gcal.PropItemID] != taskA || e.Private[gcal.PropFamily] != family {
		t.Errorf("private props = %v", e.Private)
	}
}

func TestTimedTaskIsThirtyMinutesEndingAtDeadlineInFamilyTimezone(t *testing.T) {
	h := newHarness(t)
	h.st.timezone = "Europe/Kyiv"
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Call", DueOn: date("2026-09-03"), DueTime: clockAt(18, 30)})

	h.pushTask(t)

	e := h.event(t, ann, KindTask, taskA)
	wantEnd := time.Date(2026, 9, 3, 15, 30, 0, 0, time.UTC)
	if e.AllDay || !e.End.Equal(wantEnd) || !e.Start.Equal(wantEnd.Add(-30*time.Minute)) {
		t.Errorf("event = %v..%v allDay %v, want 30 minutes ending %v", e.Start, e.End, e.AllDay, wantEnd)
	}
	if e.TimeZone != "Europe/Kyiv" {
		t.Errorf("timezone = %q", e.TimeZone)
	}
}

func TestEditUpdatesTheSameEvent(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	first, _ := h.link(ann, KindTask, taskA)

	h.putTask(db.Task{Title: "Pay rent and water", DueOn: date("2026-09-05")})
	h.pushTask(t)

	after, _ := h.link(ann, KindTask, taskA)
	if after.EventID != first.EventID {
		t.Errorf("event id changed %s -> %s, want an update", first.EventID, after.EventID)
	}
	if after.Etag == first.Etag {
		t.Error("etag not refreshed after update")
	}
	if e := h.event(t, ann, KindTask, taskA); e.Summary != "Pay rent and water" || e.StartDate != "2026-09-05" {
		t.Errorf("event = %q on %s", e.Summary, e.StartDate)
	}
	if n := len(h.google.Live(annCal)); n != 1 {
		t.Errorf("live events = %d, want 1", n)
	}
}

func TestCompletedTaskKeepsEventWithCheckMark(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)

	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03"), Status: "done"})
	h.pushTask(t)

	if e := h.event(t, ann, KindTask, taskA); e.Summary != "✓ Pay rent" || e.Cancelled() {
		t.Errorf("event = %q cancelled %v, want check-marked and live", e.Summary, e.Cancelled())
	}
	if got := TaskTitle("✓ Pay rent"); got != "Pay rent" {
		t.Errorf("TaskTitle = %q", got)
	}
}

func TestDeletedTaskDeletesEventAndLink(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	l, _ := h.link(ann, KindTask, taskA)

	delete(h.st.tasks, taskA)
	h.pushTask(t)

	if e, _ := h.google.Event(annCal, l.EventID); !e.Cancelled() {
		t.Error("event still live after the task was deleted")
	}
	if _, ok := h.link(ann, KindTask, taskA); ok {
		t.Error("link kept after a successful Google delete")
	}
}

func TestRemovingDeadlineDeletesEvent(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)

	h.putTask(db.Task{Title: "Pay rent"})
	h.pushTask(t)

	if n := len(h.google.Live(annCal)); n != 0 {
		t.Errorf("live events = %d, want 0", n)
	}
}

func TestLinkKeptWhenGoogleDeleteFails(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	delete(h.st.tasks, taskA)

	h.google.FailNext = gcal.ErrRateLimited
	if err := h.p.PushTask(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(taskA)); !errors.Is(err, gcal.ErrRateLimited) {
		t.Fatalf("err = %v, want rate limited", err)
	}
	if _, ok := h.link(ann, KindTask, taskA); !ok {
		t.Fatal("link dropped before Google confirmed the delete")
	}

	if err := h.p.SweepOrphans(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(ann)); err != nil {
		t.Fatalf("SweepOrphans: %v", err)
	}
	if _, ok := h.link(ann, KindTask, taskA); ok {
		t.Error("orphan link not swept")
	}
	if n := len(h.google.Live(annCal)); n != 0 {
		t.Errorf("live events after sweep = %d, want 0", n)
	}
}

func TestEventDeletedInGoogleIsNotResurrected(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	l, _ := h.link(ann, KindTask, taskA)

	h.google.DeleteInGoogle(annCal, l.EventID)
	h.putTask(db.Task{Title: "Pay rent later", DueOn: date("2026-09-10")})
	h.pushTask(t)

	if e, _ := h.google.Event(annCal, l.EventID); !e.Cancelled() {
		t.Error("push resurrected an event deleted in Google")
	}
	if n := len(h.google.Live(annCal)); n != 0 {
		t.Errorf("live events = %d, want 0 (the pull deletes the task)", n)
	}
}

func TestEveryConnectedMemberGetsTheEventButSharedCalendarOnlyOnce(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, sharedCal)
	h.connect(t, bob, sharedCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})

	h.pushTask(t)
	h.pushTask(t)

	if n := len(h.google.Live(sharedCal)); n != 1 {
		t.Errorf("live events in shared calendar = %d, want 1", n)
	}

	h.connect(t, bob, "bob@example.com")
	h.pushTask(t)
	if n := len(h.google.Live("bob@example.com")); n != 1 {
		t.Errorf("live events in bob's own calendar = %d, want 1", n)
	}
}

func TestForbiddenCalendarAsksToPickAnother(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})

	if _, err := h.p.accessToken(context.Background(), h.st.conns[ann]); err != nil {
		t.Fatal(err)
	}
	h.google.FailNext = gcal.ErrForbidden
	err := h.p.PushTask(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(taskA))

	if !errors.Is(err, gcal.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
	if got := h.st.conns[ann].LastError; got != ErrTextReadOnly {
		t.Errorf("last_error = %q, want %q", got, ErrTextReadOnly)
	}
}

func TestRevokedGrantAsksToReconnect(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.google.RevokeAll()

	err := h.p.PushTask(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(taskA))

	if !errors.Is(err, gcal.ErrUnauthorized) {
		t.Fatalf("err = %v, want unauthorized", err)
	}
	if got := h.st.conns[ann].LastError; got != ErrTextReconnect {
		t.Errorf("last_error = %q, want %q", got, ErrTextReconnect)
	}
}

func TestAccessTokenIsReusedUntilExpiry(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.google.Calls = nil

	h.pushTask(t)
	h.putTask(db.Task{Title: "Pay rent!", DueOn: date("2026-09-03")})
	h.pushTask(t)

	refreshes := 0
	for _, c := range h.google.Calls {
		if c == "Refresh" {
			refreshes++
		}
	}
	if refreshes != 1 {
		t.Errorf("Refresh calls = %d, want 1 (calls %v)", refreshes, h.google.Calls)
	}
}

func TestBirthdayIsYearlyAllDayEvent(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	year := int32(1990)
	h.st.birthdays[bdayB] = db.Birthday{
		ID: pgconv.MustUUID(bdayB), FamilyID: pgconv.MustUUID(family), Name: "Олена", Day: 15, Month: 9, Year: &year,
	}

	if err := h.p.PushBirthday(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(bdayB)); err != nil {
		t.Fatalf("PushBirthday: %v", err)
	}

	e := h.event(t, ann, KindBirthday, bdayB)
	if !e.AllDay || e.StartDate != "2026-09-15" || e.EndDate != "2026-09-16" {
		t.Errorf("dates = %s..%s allDay %v", e.StartDate, e.EndDate, e.AllDay)
	}
	if len(e.Recurrence) != 1 || e.Recurrence[0] != "RRULE:FREQ=YEARLY" {
		t.Errorf("recurrence = %v", e.Recurrence)
	}
	if e.Summary != "🎂 Олена" || BirthdayName(e.Summary) != "Олена" {
		t.Errorf("summary = %q", e.Summary)
	}

	delete(h.st.birthdays, bdayB)
	if err := h.p.PushBirthday(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(bdayB)); err != nil {
		t.Fatalf("PushBirthday after delete: %v", err)
	}
	if n := len(h.google.Live(annCal)); n != 0 {
		t.Errorf("live events = %d, want 0", n)
	}
}

func TestLeapDayBirthdayRecursOnLastDayOfFebruary(t *testing.T) {
	e := BirthdayEvent(db.Birthday{Name: "Leap", Day: 29, Month: 2}, testNow, time.UTC)

	if e.StartDate != "2027-02-28" {
		t.Errorf("start = %s, want 2027-02-28", e.StartDate)
	}
	if len(e.Recurrence) != 1 || e.Recurrence[0] != "RRULE:FREQ=YEARLY;BYMONTH=2;BYMONTHDAY=-1" {
		t.Errorf("recurrence = %v", e.Recurrence)
	}
}

func TestNoConnectionsMeansNoGoogleCalls(t *testing.T) {
	h := newHarness(t)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.google.Calls = nil

	h.pushTask(t)

	if len(h.google.Calls) != 0 {
		t.Errorf("google calls = %v, want none", h.google.Calls)
	}
}

func TestSwitchingCalendarMovesTheEvent(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	old, _ := h.link(ann, KindTask, taskA)
	if err := h.p.SwitchCalendar(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(ann)); err != nil {
		t.Fatal(err)
	}

	c := h.st.conns[ann]
	c.CalendarID = "work@example.com"
	h.st.conns[ann] = c
	h.pushTask(t)

	if e, _ := h.google.Event(annCal, old.EventID); !e.Cancelled() {
		t.Error("event left live in the old calendar")
	}
	if n := len(h.google.Live("work@example.com")); n != 1 {
		t.Errorf("live events in new calendar = %d, want 1", n)
	}
	if l, _ := h.link(ann, KindTask, taskA); l.CalendarID != "work@example.com" {
		t.Errorf("link calendar = %s", l.CalendarID)
	}
}

func TestSwitchingAwayFromSharedCalendarLeavesOneCopy(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, sharedCal)
	h.connect(t, bob, sharedCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	if err := h.p.SwitchCalendar(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(ann)); err != nil {
		t.Fatal(err)
	}

	c := h.st.conns[ann]
	c.CalendarID = annCal
	h.st.conns[ann] = c
	h.pushTask(t)
	h.pushTask(t)

	if n := len(h.google.Live(sharedCal)); n != 1 {
		t.Errorf("live events in shared calendar = %d, want 1", n)
	}
	if n := len(h.google.Live(annCal)); n != 1 {
		t.Errorf("live events in ann's calendar = %d, want 1", n)
	}
}

func TestBothMembersSwitchSharedCalendarWithoutLeavingUntrackedEvent(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, sharedCal)
	h.connect(t, bob, sharedCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	if err := h.p.SwitchCalendar(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(ann)); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.link(bob, KindTask, taskA); !ok {
		t.Fatal("shared event was not transferred to bob")
	}
	c := h.st.conns[ann]
	c.CalendarID = annCal
	h.st.conns[ann] = c
	if err := h.p.SwitchCalendar(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(bob)); err != nil {
		t.Fatal(err)
	}
	if n := len(h.google.Live(sharedCal)); n != 0 {
		t.Errorf("old shared calendar has %d live events, want 0", n)
	}
	if _, ok := h.link(bob, KindTask, taskA); ok {
		t.Error("old shared calendar link remains")
	}
}

func TestStaleCalendarLinkIsDroppedWithoutDeletingOldEvent(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	old, _ := h.link(ann, KindTask, taskA)
	c := h.st.conns[ann]
	c.CalendarID = "work@example.com"
	h.st.conns[ann] = c
	h.pushTask(t)
	if e, _ := h.google.Event(annCal, old.EventID); e.Cancelled() {
		t.Error("stale link deleted an event in the old calendar")
	}
	if n := len(h.google.Live("work@example.com")); n != 1 {
		t.Errorf("new calendar events = %d, want 1", n)
	}
}

func TestLostLinkUsesDeterministicEventID(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	old, _ := h.link(ann, KindTask, taskA)
	delete(h.st.links, linkKey(pgconv.MustUUID(ann), KindTask, pgconv.MustUUID(taskA)))
	h.pushTask(t)
	l, ok := h.link(ann, KindTask, taskA)
	if !ok || l.EventID != old.EventID {
		t.Errorf("restored link = %+v, want event %s", l, old.EventID)
	}
	if n := len(h.google.Live(annCal)); n != 1 {
		t.Errorf("live events = %d, want 1", n)
	}
}

func TestCancelledEventWithLostLinkStaysCancelled(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	l, _ := h.link(ann, KindTask, taskA)
	h.google.DeleteInGoogle(annCal, l.EventID)
	delete(h.st.links, linkKey(pgconv.MustUUID(ann), KindTask, pgconv.MustUUID(taskA)))
	h.pushTask(t)
	if n := len(h.google.Live(annCal)); n != 0 {
		t.Errorf("live events = %d, want 0", n)
	}
}

func TestSweepOrphansDeletesTaskWithoutDeadline(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	task := h.st.tasks[taskA]
	task.DueOn = pgtype.Date{}
	h.st.tasks[taskA] = task
	if err := h.p.SweepOrphans(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(ann)); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.link(ann, KindTask, taskA); ok {
		t.Error("orphan link remains")
	}
	if n := len(h.google.Live(annCal)); n != 0 {
		t.Errorf("live events = %d, want 0", n)
	}
}

func TestSweepOrphansPreservesTaskWhoseDeadlineWasRestored(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	task := h.st.tasks[taskA]
	task.DueOn = pgtype.Date{}
	h.st.tasks[taskA] = task
	h.st.afterOrphanList = func() {
		task.DueOn = date("2026-09-03")
		h.st.tasks[taskA] = task
	}
	if err := h.p.SweepOrphans(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(ann)); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.link(ann, KindTask, taskA); !ok {
		t.Error("restored task link was deleted")
	}
	if n := len(h.google.Live(annCal)); n != 1 {
		t.Errorf("live events = %d, want 1", n)
	}
}

func TestDeadlineRestoredDuringSweepDeletionIsRecreated(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	task := h.st.tasks[taskA]
	task.DueOn = pgtype.Date{}
	h.st.tasks[taskA] = task
	client := &blockingDeleteClient{Client: h.google, started: make(chan struct{}), release: make(chan struct{})}
	h.p.google = client
	swept := make(chan error, 1)
	go func() {
		swept <- h.p.SweepOrphans(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(ann))
	}()
	<-client.started
	mutated := make(chan error, 1)
	go func() {
		mutated <- h.p.MutateTask(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(taskA), func() (bool, error) {
			task.DueOn = date("2026-09-03")
			h.st.tasks[taskA] = task
			return true, nil
		})
	}()
	close(client.release)
	if err := <-swept; err != nil {
		t.Fatal(err)
	}
	if err := <-mutated; err != nil {
		t.Fatal(err)
	}
	if n := len(h.google.Live(annCal)); n != 1 {
		t.Errorf("live events = %d, want 1", n)
	}
	if _, ok := h.link(ann, KindTask, taskA); !ok {
		t.Error("restored event has no link")
	}
}

func TestSwitchCalendarAbandonsEventAfterWriterAccessLost(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	h.google.FailNext = gcal.ErrForbidden
	if err := h.p.SwitchCalendar(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(ann)); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.link(ann, KindTask, taskA); ok {
		t.Error("link remains after losing writer access")
	}
}

func TestOldEventKeptLinkedWhenItsDeleteFails(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	old, _ := h.link(ann, KindTask, taskA)

	h.google.FailNext = gcal.ErrRateLimited
	if err := h.p.SwitchCalendar(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(ann)); err == nil {
		t.Fatal("want an error when the old event cannot be deleted")
	}

	if l, _ := h.link(ann, KindTask, taskA); l.EventID != old.EventID || l.CalendarID != annCal {
		t.Errorf("link = %+v, want the old event still tracked", l)
	}
	if n := len(h.google.Live("work@example.com")); n != 0 {
		t.Errorf("inserted into the new calendar before the old event was gone")
	}
}

func TestCancelledEventWithExpiredSyncTokenIsNotResurrected(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.pushTask(t)
	l, _ := h.link(ann, KindTask, taskA)
	c := h.st.conns[ann]
	c.SyncToken = "sync-1"
	h.st.conns[ann] = c

	h.google.DeleteInGoogle(annCal, l.EventID)
	h.google.ExpireSyncTokens()
	h.putTask(db.Task{Title: "Pay rent later", DueOn: date("2026-09-10")})
	h.pushTask(t)

	if n := len(h.google.Live(annCal)); n != 0 {
		t.Errorf("live events = %d, want 0", n)
	}
}

func TestEventDeletedWhenItsLinkCannotBeSaved(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.st.failLink = errors.New("db down")

	if err := h.p.PushTask(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(taskA)); err == nil {
		t.Fatal("want an error")
	}
	if n := len(h.google.Live(annCal)); n != 1 {
		t.Errorf("live events = %d, want 1 for retry", n)
	}
	h.st.failLink = nil
	h.pushTask(t)
	if n := len(h.google.Live(annCal)); n != 1 {
		t.Errorf("live events after retry = %d, want 1", n)
	}
}

func TestConcurrentPushesCreateOneEvent(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	h.st.slowLink = 5 * time.Millisecond

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = h.p.PushTask(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(taskA))
		}()
	}
	wg.Wait()

	if n := len(h.google.Live(annCal)); n != 1 {
		t.Errorf("live events = %d, want 1", n)
	}
}

func TestLostScopeAsksToReconnect(t *testing.T) {
	h := newHarness(t)
	h.connect(t, ann, annCal)
	h.putTask(db.Task{Title: "Pay rent", DueOn: date("2026-09-03")})
	if _, err := h.p.accessToken(context.Background(), h.st.conns[ann]); err != nil {
		t.Fatal(err)
	}
	h.google.FailNext = gcal.ErrInsufficientScope

	_ = h.p.PushTask(context.Background(), pgconv.MustUUID(family), pgconv.MustUUID(taskA))

	if got := h.st.conns[ann].LastError; got != ErrTextReconnect {
		t.Errorf("last_error = %q, want %q", got, ErrTextReconnect)
	}
}
