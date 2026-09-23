package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/calsync"
	"github.com/nnc/family-manager/services/tasks/internal/crypto"
	"github.com/nnc/family-manager/services/tasks/internal/gcal"
)

type calendarRecorder struct {
	tasks     []string
	birthdays []string
	restores  []bool
	fail      error
}

func (c *calendarRecorder) PushTask(_ context.Context, _, taskID pgtype.UUID) error {
	c.tasks = append(c.tasks, id(taskID))
	return c.fail
}

func (c *calendarRecorder) PushBirthday(_ context.Context, _, birthdayID pgtype.UUID) error {
	c.birthdays = append(c.birthdays, id(birthdayID))
	return c.fail
}

func (c *calendarRecorder) MutateTask(ctx context.Context, familyID, taskID pgtype.UUID, write func() (bool, error)) error {
	restore, err := write()
	if err != nil {
		return err
	}
	c.restores = append(c.restores, restore)
	_ = c.PushTask(ctx, familyID, taskID)
	return nil
}

func (c *calendarRecorder) MutateBirthday(ctx context.Context, familyID, birthdayID pgtype.UUID, write func() error) error {
	if err := write(); err != nil {
		return err
	}
	_ = c.PushBirthday(ctx, familyID, birthdayID)
	return nil
}

func (c *calendarRecorder) GoogleClient() gcal.Client {
	return nil
}

func (c *calendarRecorder) Box() *crypto.Box {
	return nil
}

func (c *calendarRecorder) SwitchCalendar(_ context.Context, _, _ pgtype.UUID) error {
	return nil
}

func TestTaskWritesDriveCalendarAndDetectRestoredDeadline(t *testing.T) {
	h, _, _ := newTestHandler(t)
	cal := &calendarRecorder{}
	h.calendar = cal
	task := createTask(t, h, &tasksv1.CreateTaskRequest{Title: "Pay rent", DueOn: "2026-09-03"})
	_, err := h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{TaskId: task.Id, DueOn: strPtr("")}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{TaskId: task.Id, DueOn: strPtr("2026-09-03")}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.CompleteTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CompleteTaskRequest{TaskId: task.Id}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.ReopenTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.ReopenTaskRequest{TaskId: task.Id}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.DeleteTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.DeleteTaskRequest{TaskId: task.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if len(cal.tasks) != 6 || len(cal.restores) != 5 || cal.restores[0] || !cal.restores[1] {
		t.Errorf("task calls = %v, restore flags = %v", cal.tasks, cal.restores)
	}
}

func TestBirthdayWritesDriveCalendar(t *testing.T) {
	h, _ := newBirthdayHandler(t, nil)
	cal := &calendarRecorder{}
	h.calendar = cal
	birthday := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "Nadia", Day: 10, Month: 9})
	_, err := h.UpdateBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateBirthdayRequest{BirthdayId: birthday.Id, Name: strPtr("Nadia S")}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.DeleteBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.DeleteBirthdayRequest{BirthdayId: birthday.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if len(cal.birthdays) != 3 {
		t.Errorf("birthday calls = %v, want create/update/delete", cal.birthdays)
	}
}

func TestCalendarFailureDoesNotFailTaskWrite(t *testing.T) {
	h, store, _ := newTestHandler(t)
	cal := &calendarRecorder{fail: errors.New("google unavailable")}
	h.calendar = cal
	resp, err := h.CreateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CreateTaskRequest{Title: "Pay rent", DueOn: "2026-09-03"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.tasks[resp.Msg.Task.Id]; !ok {
		t.Error("task write was lost after calendar failure")
	}
}

type calendarStore struct {
	*birthdayStore
	conn  db.GoogleConnection
	links map[string]db.CalendarLink
}

func (s *calendarStore) InTx(_ context.Context, fn func(db.Querier) error) error { return fn(s) }

func (s *calendarStore) GetGoogleConnection(_ context.Context, arg db.GetGoogleConnectionParams) (db.GoogleConnection, error) {
	if !same(arg.FamilyID, s.conn.FamilyID) || !same(arg.UserID, s.conn.UserID) {
		return db.GoogleConnection{}, pgx.ErrNoRows
	}
	return s.conn, nil
}

func (s *calendarStore) ListFamilyGoogleConnections(_ context.Context, familyID pgtype.UUID) ([]db.GoogleConnection, error) {
	if same(familyID, s.conn.FamilyID) {
		return []db.GoogleConnection{s.conn}, nil
	}
	return nil, nil
}

func (s *calendarStore) SetGoogleError(_ context.Context, arg db.SetGoogleErrorParams) error {
	s.conn.LastError = arg.LastError
	return nil
}

func (s *calendarStore) GetCalendarLink(_ context.Context, arg db.GetCalendarLinkParams) (db.CalendarLink, error) {
	l, ok := s.links[arg.Kind+"|"+id(arg.ItemID)]
	if !ok {
		return db.CalendarLink{}, pgx.ErrNoRows
	}
	return l, nil
}

func (s *calendarStore) ListCalendarLinksForItem(_ context.Context, arg db.ListCalendarLinksForItemParams) ([]db.CalendarLink, error) {
	l, ok := s.links[arg.Kind+"|"+id(arg.ItemID)]
	if !ok {
		return nil, nil
	}
	return []db.CalendarLink{l}, nil
}

func (s *calendarStore) ListCalendarLinksForUser(_ context.Context, arg db.ListCalendarLinksForUserParams) ([]db.CalendarLink, error) {
	var out []db.CalendarLink
	for _, l := range s.links {
		out = append(out, l)
	}
	return out, nil
}

func (s *calendarStore) ListOrphanCalendarLinks(_ context.Context, arg db.ListOrphanCalendarLinksParams) ([]db.CalendarLink, error) {
	return nil, nil
}

func (s *calendarStore) CalendarLinkedInFamily(_ context.Context, arg db.CalendarLinkedInFamilyParams) (bool, error) {
	return false, nil
}

func (s *calendarStore) UpsertCalendarLink(_ context.Context, arg db.UpsertCalendarLinkParams) error {
	s.links[arg.Kind+"|"+id(arg.ItemID)] = db.CalendarLink{
		FamilyID: arg.FamilyID, UserID: arg.UserID, Kind: arg.Kind, ItemID: arg.ItemID,
		CalendarID: arg.CalendarID, EventID: arg.EventID, Etag: arg.Etag,
	}
	return nil
}

func (s *calendarStore) DeleteCalendarLink(_ context.Context, arg db.DeleteCalendarLinkParams) error {
	delete(s.links, arg.Kind+"|"+id(arg.ItemID))
	return nil
}

func TestHandlerWritesReachFakeGoogleCalendar(t *testing.T) {
	box, err := crypto.NewBox(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	if err != nil {
		t.Fatal(err)
	}
	google := gcal.NewFake()
	google.SetNow(func() time.Time { return testNow })
	google.AddCode("code", "v", "sergiy@example.com")
	token, err := google.Exchange(context.Background(), "code", "v", "")
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal(token.RefreshToken, crypto.Owner(testFamily, sergiy))
	if err != nil {
		t.Fatal(err)
	}
	store := &calendarStore{
		birthdayStore: &birthdayStore{fakeStore: newFakeStore(), birthdays: map[string]db.Birthday{}},
		conn: db.GoogleConnection{
			FamilyID: pgconv.MustUUID(testFamily), UserID: pgconv.MustUUID(sergiy),
			RefreshToken: sealed, CalendarID: "sergiy@example.com",
		},
		links: map[string]db.CalendarLink{},
	}
	seedHousehold(store.fakeStore)
	pusher := calsync.NewPusher(calsync.Options{Store: store, Google: google, Box: box, Now: func() time.Time { return testNow }})
	h := New(Options{Queries: store, Tx: store, Calendar: pusher, Now: func() time.Time { return testNow }})
	task := createTask(t, h, &tasksv1.CreateTaskRequest{Title: "Pay rent", DueOn: "2026-09-03"})
	if n := len(google.Live(store.conn.CalendarID)); n != 1 {
		t.Fatalf("events after task create = %d, want 1", n)
	}
	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{TaskId: task.Id, Title: strPtr("Pay later")}))
	if err != nil {
		t.Fatal(err)
	}
	l := store.links[calsync.KindTask+"|"+task.Id]
	e, _ := google.Event(store.conn.CalendarID, l.EventID)
	if e.Summary != "Pay later" {
		t.Errorf("event title = %q, want Pay later", e.Summary)
	}
	_, err = h.DeleteTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.DeleteTaskRequest{TaskId: task.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(google.Live(store.conn.CalendarID)); n != 0 {
		t.Errorf("events after task delete = %d, want 0", n)
	}
	birthday := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "Nadia", Day: 10, Month: 9})
	if n := len(google.Live(store.conn.CalendarID)); n != 1 {
		t.Errorf("events after birthday create = %d, want 1", n)
	}
	_, err = h.DeleteBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.DeleteBirthdayRequest{BirthdayId: birthday.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(google.Live(store.conn.CalendarID)); n != 0 {
		t.Errorf("events after birthday delete = %d, want 0", n)
	}
}
