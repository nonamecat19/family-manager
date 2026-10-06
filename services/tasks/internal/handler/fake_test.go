package handler

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/services/tasks/db"
)

type fakeStore struct {
	settings    map[string]db.FamilySetting
	members     map[string]db.KnownMember
	tasks       map[string]db.Task
	assignees   map[string]db.TaskAssignee
	reminders   map[string]db.TaskReminder
	deliveries  map[string]db.ReminderDelivery
	digestSends map[string]db.DigestSend

	failOn map[string]error
	seq    int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		settings:    map[string]db.FamilySetting{},
		members:     map[string]db.KnownMember{},
		tasks:       map[string]db.Task{},
		assignees:   map[string]db.TaskAssignee{},
		reminders:   map[string]db.TaskReminder{},
		deliveries:  map[string]db.ReminderDelivery{},
		digestSends: map[string]db.DigestSend{},
		failOn:      map[string]error{},
	}
}

func (s *fakeStore) fail(op string) error { return s.failOn[op] }

func (s *fakeStore) newUUID() pgtype.UUID {
	s.seq++
	return pgconv.MustUUID(fmt.Sprintf("%08x-0000-4000-8000-%012x", s.seq, s.seq))
}

func (s *fakeStore) InTx(_ context.Context, fn func(q db.Querier) error) error { return fn(s) }

func same(a, b pgtype.UUID) bool { return a.Valid && b.Valid && id(a) == id(b) }

func now() pgtype.Timestamptz { return pgtype.Timestamptz{Valid: true} }

// --- FamilySettings ---
func (s *fakeStore) GetFamilySettings(_ context.Context, familyID pgtype.UUID) (db.FamilySetting, error) {
	if err := s.fail("GetFamilySettings"); err != nil {
		return db.FamilySetting{}, err
	}
	row, ok := s.settings[id(familyID)]
	if !ok {
		return db.FamilySetting{}, pgx.ErrNoRows
	}
	return row, nil
}

func (s *fakeStore) InitFamilySettings(_ context.Context, arg db.InitFamilySettingsParams) error {
	if _, ok := s.settings[id(arg.FamilyID)]; ok {
		return nil
	}
	s.settings[id(arg.FamilyID)] = db.FamilySetting{
		FamilyID:   arg.FamilyID,
		Timezone:   arg.Timezone,
		DigestTime: "08:00",
		CreatedAt:  now(), UpdatedAt: now(),
	}
	return nil
}

func (s *fakeStore) SetFamilyTimezone(_ context.Context, arg db.SetFamilyTimezoneParams) (db.FamilySetting, error) {
	row, ok := s.settings[id(arg.FamilyID)]
	if !ok {
		return db.FamilySetting{}, pgx.ErrNoRows
	}
	row.Timezone = arg.Timezone
	row.UpdatedAt = now()
	s.settings[id(arg.FamilyID)] = row
	return row, nil
}

func (s *fakeStore) ListAllFamilySettings(_ context.Context) ([]db.FamilySetting, error) {
	var out []db.FamilySetting
	for _, v := range s.settings {
		out = append(out, v)
	}
	return out, nil
}

// --- KnownMember ---
func (s *fakeStore) UpsertKnownMember(_ context.Context, arg db.UpsertKnownMemberParams) error {
	if err := s.fail("UpsertKnownMember"); err != nil {
		return err
	}
	key := id(arg.FamilyID) + "|" + id(arg.UserID)
	s.members[key] = db.KnownMember{
		FamilyID:    arg.FamilyID,
		UserID:      arg.UserID,
		DisplayName: arg.DisplayName,
		Email:       arg.Email,
		SeenAt:      now(),
	}
	return nil
}

func (s *fakeStore) TouchKnownMember(_ context.Context, arg db.TouchKnownMemberParams) error {
	if err := s.fail("TouchKnownMember"); err != nil {
		return err
	}
	key := id(arg.FamilyID) + "|" + id(arg.UserID)
	m, ok := s.members[key]
	if !ok {
		m = db.KnownMember{FamilyID: arg.FamilyID, UserID: arg.UserID}
	}
	if arg.Email != "" {
		m.Email = arg.Email
	}
	m.SeenAt = now()
	s.members[key] = m
	return nil
}

func (s *fakeStore) ListKnownMembers(_ context.Context, familyID pgtype.UUID) ([]db.KnownMember, error) {
	var out []db.KnownMember
	for _, m := range s.members {
		if same(m.FamilyID, familyID) {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *fakeStore) DeleteKnownMember(_ context.Context, arg db.DeleteKnownMemberParams) error {
	key := id(arg.FamilyID) + "|" + id(arg.UserID)
	delete(s.members, key)
	return nil
}

func (s *fakeStore) DeleteKnownMembersExcept(_ context.Context, arg db.DeleteKnownMembersExceptParams) error {
	keep := make(map[string]bool)
	for _, u := range arg.Keep {
		keep[id(arg.FamilyID)+"|"+id(u)] = true
	}
	for k, m := range s.members {
		if same(m.FamilyID, arg.FamilyID) && !keep[k] {
			delete(s.members, k)
		}
	}
	return nil
}

// --- Task ---
func (s *fakeStore) CreateTask(_ context.Context, arg db.CreateTaskParams) (db.Task, error) {
	if err := s.fail("CreateTask"); err != nil {
		return db.Task{}, err
	}
	t := db.Task{
		ID:                s.newUUID(),
		FamilyID:          arg.FamilyID,
		Title:             arg.Title,
		Notes:             arg.Notes,
		Priority:          arg.Priority,
		Status:            "open",
		DueOn:             arg.DueOn,
		DueTime:           arg.DueTime,
		DueAt:             arg.DueAt,
		CreatedByUserID:   arg.CreatedByUserID,
		CompletedByUserID: pgtype.UUID{},
		CompletedAt:       pgtype.Timestamptz{},
		CreatedAt:         now(),
		UpdatedAt:         now(),
	}
	s.tasks[id(t.ID)] = t
	return t, nil
}

func (s *fakeStore) GetTask(_ context.Context, arg db.GetTaskParams) (db.Task, error) {
	t, ok := s.tasks[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) {
		return db.Task{}, pgx.ErrNoRows
	}
	return t, nil
}

func (s *fakeStore) GetTaskForUpdate(_ context.Context, arg db.GetTaskForUpdateParams) (db.Task, error) {
	return s.GetTask(context.Background(), db.GetTaskParams{ID: arg.ID, FamilyID: arg.FamilyID})
}

func (s *fakeStore) UpdateTask(_ context.Context, arg db.UpdateTaskParams) (db.Task, error) {
	t, ok := s.tasks[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) {
		return db.Task{}, pgx.ErrNoRows
	}
	t.Title = arg.Title
	t.Notes = arg.Notes
	t.Priority = arg.Priority
	t.DueOn = arg.DueOn
	t.DueTime = arg.DueTime
	t.DueAt = arg.DueAt
	t.UpdatedAt = now()
	s.tasks[id(arg.ID)] = t
	return t, nil
}

func (s *fakeStore) CompleteTask(_ context.Context, arg db.CompleteTaskParams) (db.Task, error) {
	t, ok := s.tasks[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) || t.Status != "open" {
		return db.Task{}, pgx.ErrNoRows
	}
	t.Status = "done"
	t.CompletedAt = arg.CompletedAt
	t.CompletedByUserID = arg.CompletedByUserID
	t.UpdatedAt = now()
	s.tasks[id(arg.ID)] = t
	return t, nil
}

func (s *fakeStore) ReopenTask(_ context.Context, arg db.ReopenTaskParams) (db.Task, error) {
	t, ok := s.tasks[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) || t.Status != "done" {
		return db.Task{}, pgx.ErrNoRows
	}
	t.Status = "open"
	t.CompletedAt = pgtype.Timestamptz{}
	t.CompletedByUserID = pgtype.UUID{}
	t.UpdatedAt = now()
	s.tasks[id(arg.ID)] = t
	return t, nil
}

func (s *fakeStore) DeleteTask(_ context.Context, arg db.DeleteTaskParams) (int64, error) {
	_, ok := s.tasks[id(arg.ID)]
	if !ok || !same(s.tasks[id(arg.ID)].FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.tasks, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) ListTasks(_ context.Context, arg db.ListTasksParams) ([]db.Task, error) {
	if err := s.fail("ListTasks"); err != nil {
		return nil, err
	}
	var out []db.Task
	for _, t := range s.tasks {
		if !same(t.FamilyID, arg.FamilyID) {
			continue
		}
		if arg.Status != "" && t.Status != arg.Status {
			continue
		}
		if arg.Assignee.Valid {
			found := false
			for _, a := range s.assignees {
				if same(a.TaskID, t.ID) && same(a.UserID, arg.Assignee) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		di := out[i].DueAt.Time
		dj := out[j].DueAt.Time
		if di.IsZero() && !dj.IsZero() {
			return false
		}
		if !di.IsZero() && dj.IsZero() {
			return true
		}
		if !di.IsZero() && !dj.IsZero() && !di.Equal(dj) {
			return di.Before(dj)
		}
		pi := priorityOrder(out[i].Priority)
		pj := priorityOrder(out[j].Priority)
		if pi != pj {
			return pi < pj
		}
		if !out[i].CreatedAt.Time.Equal(out[j].CreatedAt.Time) {
			return out[i].CreatedAt.Time.Before(out[j].CreatedAt.Time)
		}
		return id(out[i].ID) < id(out[j].ID)
	})
	return out, nil
}

func priorityOrder(p string) int {
	switch p {
	case "urgent":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	default:
		return 3
	}
}

// --- TaskAssignee ---
func (s *fakeStore) AddAssignee(_ context.Context, arg db.AddAssigneeParams) error {
	if err := s.fail("AddAssignee"); err != nil {
		return err
	}
	key := id(arg.TaskID) + "|" + id(arg.UserID)
	s.assignees[key] = db.TaskAssignee{
		TaskID:   arg.TaskID,
		FamilyID: arg.FamilyID,
		UserID:   arg.UserID,
	}
	return nil
}

func (s *fakeStore) DeleteAssignees(_ context.Context, arg db.DeleteAssigneesParams) error {
	for k := range s.assignees {
		if id(s.assignees[k].TaskID) == id(arg.TaskID) && same(s.assignees[k].FamilyID, arg.FamilyID) {
			delete(s.assignees, k)
		}
	}
	return nil
}

func (s *fakeStore) ListAssignees(_ context.Context, arg db.ListAssigneesParams) ([]db.ListAssigneesRow, error) {
	var out []db.ListAssigneesRow
	taskSet := make(map[string]bool)
	for _, tid := range arg.TaskIds {
		taskSet[id(tid)] = true
	}
	for _, a := range s.assignees {
		if !same(a.FamilyID, arg.FamilyID) {
			continue
		}
		if !taskSet[id(a.TaskID)] {
			continue
		}
		out = append(out, db.ListAssigneesRow{TaskID: a.TaskID, UserID: a.UserID})
	}
	sort.Slice(out, func(i, j int) bool {
		if id(out[i].TaskID) != id(out[j].TaskID) {
			return id(out[i].TaskID) < id(out[j].TaskID)
		}
		return id(out[i].UserID) < id(out[j].UserID)
	})
	return out, nil
}

// --- TaskReminder ---
func (s *fakeStore) UpsertReminder(_ context.Context, arg db.UpsertReminderParams) error {
	if err := s.fail("UpsertReminder"); err != nil {
		return err
	}
	key := id(arg.FamilyID) + "|" + id(arg.UserID) + "|" + arg.Kind + "|" + id(arg.ItemID) + "|" + arg.Occurrence
	if existing, ok := s.reminders[key]; ok {
		if existing.AckedAt.Valid || existing.SnoozedUntil.Valid {
			return nil
		}
	}
	s.reminders[key] = db.TaskReminder{
		ID:           s.newUUID(),
		FamilyID:     arg.FamilyID,
		UserID:       arg.UserID,
		Kind:         arg.Kind,
		ItemID:       arg.ItemID,
		Occurrence:   arg.Occurrence,
		RemindAt:     arg.RemindAt,
		SnoozedUntil: pgtype.Timestamptz{},
		AckedAt:      pgtype.Timestamptz{},
		CreatedAt:    now(),
	}
	return nil
}

func (s *fakeStore) DeletePendingRemindersForItem(_ context.Context, arg db.DeletePendingRemindersForItemParams) error {
	for k, r := range s.reminders {
		if same(r.FamilyID, arg.FamilyID) && r.Kind == arg.Kind && same(r.ItemID, arg.ItemID) && !r.AckedAt.Valid {
			delete(s.reminders, k)
		}
	}
	return nil
}

func (s *fakeStore) DeletePendingRemindersForItemExcept(_ context.Context, arg db.DeletePendingRemindersForItemExceptParams) error {
	keep := make(map[string]bool)
	for _, u := range arg.KeepUsers {
		keep[id(u)+"|"+arg.KeepOccurrence] = true
	}
	for k, r := range s.reminders {
		if same(r.FamilyID, arg.FamilyID) && r.Kind == arg.Kind && same(r.ItemID, arg.ItemID) && !r.AckedAt.Valid {
			key := id(r.UserID) + "|" + r.Occurrence
			if !keep[key] {
				delete(s.reminders, k)
			}
		}
	}
	return nil
}

func (s *fakeStore) GetReminder(_ context.Context, arg db.GetReminderParams) (db.TaskReminder, error) {
	for _, r := range s.reminders {
		if same(r.ID, arg.ID) && same(r.FamilyID, arg.FamilyID) && same(r.UserID, arg.UserID) {
			return r, nil
		}
	}
	return db.TaskReminder{}, pgx.ErrNoRows
}

func (s *fakeStore) ListDueReminders(_ context.Context, arg db.ListDueRemindersParams) ([]db.TaskReminder, error) {
	var out []db.TaskReminder
	for _, r := range s.reminders {
		if r.AckedAt.Valid {
			continue
		}
		if !r.RemindAt.Valid || r.RemindAt.Time.After(arg.Now.Time) {
			continue
		}
		delKey := id(r.ID) + "|" + arg.Channel
		if _, ok := s.deliveries[delKey]; ok {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].RemindAt.Time.Equal(out[j].RemindAt.Time) {
			return out[i].RemindAt.Time.Before(out[j].RemindAt.Time)
		}
		return id(out[i].ID) < id(out[j].ID)
	})
	if int32(len(out)) > arg.MaxRows {
		out = out[:arg.MaxRows]
	}
	return out, nil
}

func (s *fakeStore) ListDueRemindersForUser(_ context.Context, arg db.ListDueRemindersForUserParams) ([]db.TaskReminder, error) {
	var out []db.TaskReminder
	for _, r := range s.reminders {
		if !same(r.FamilyID, arg.FamilyID) || !same(r.UserID, arg.UserID) {
			continue
		}
		if r.AckedAt.Valid {
			continue
		}
		if !r.RemindAt.Valid || r.RemindAt.Time.After(arg.Now.Time) {
			continue
		}
		delKey := id(r.ID) + "|" + arg.Channel
		if _, ok := s.deliveries[delKey]; ok {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].RemindAt.Time.Equal(out[j].RemindAt.Time) {
			return out[i].RemindAt.Time.Before(out[j].RemindAt.Time)
		}
		return id(out[i].ID) < id(out[j].ID)
	})
	if len(out) > 50 {
		out = out[:50]
	}
	return out, nil
}

func (s *fakeStore) AckReminder(_ context.Context, arg db.AckReminderParams) (int64, error) {
	for k, r := range s.reminders {
		if same(r.ID, arg.ID) && same(r.FamilyID, arg.FamilyID) && same(r.UserID, arg.UserID) {
			if r.AckedAt.Valid {
				return 0, nil
			}
			r.AckedAt = pgtype.Timestamptz{Valid: true}
			s.reminders[k] = r
			s.deliveries[id(r.ID)+"|push"] = db.ReminderDelivery{ReminderID: r.ID, Channel: "push", DeliveredAt: now()}
			return 1, nil
		}
	}
	return 0, nil
}

func (s *fakeStore) SnoozeReminder(_ context.Context, arg db.SnoozeReminderParams) (db.TaskReminder, error) {
	for k, r := range s.reminders {
		if same(r.ID, arg.ID) && same(r.FamilyID, arg.FamilyID) && same(r.UserID, arg.UserID) {
			r.RemindAt = arg.RemindAt
			r.SnoozedUntil = arg.RemindAt
			r.AckedAt = pgtype.Timestamptz{}
			s.reminders[k] = r
			return r, nil
		}
	}
	return db.TaskReminder{}, pgx.ErrNoRows
}

func (s *fakeStore) ClearDeliveries(_ context.Context, arg db.ClearDeliveriesParams) error {
	for k := range s.deliveries {
		if len(k) > len(id(arg.ReminderID))+1 && k[:len(id(arg.ReminderID))+1] == id(arg.ReminderID)+"|" {
			delete(s.deliveries, k)
		}
	}
	return nil
}

func (s *fakeStore) RecordDelivery(_ context.Context, arg db.RecordDeliveryParams) error {
	s.deliveries[id(arg.ReminderID)+"|"+arg.Channel] = db.ReminderDelivery{
		ReminderID:  arg.ReminderID,
		FamilyID:    arg.FamilyID,
		Channel:     arg.Channel,
		DeliveredAt: now(),
	}
	return nil
}

func (s *fakeStore) GetDigestSent(_ context.Context, arg db.GetDigestSentParams) (bool, error) {
	key := id(arg.FamilyID) + "|" + id(arg.UserID) + "|" + arg.SentOn.Time.Format("2006-01-02")
	_, ok := s.digestSends[key]
	return ok, nil
}

func (s *fakeStore) MarkDigestSent(_ context.Context, arg db.MarkDigestSentParams) error {
	key := id(arg.FamilyID) + "|" + id(arg.UserID) + "|" + arg.SentOn.Time.Format("2006-01-02")
	s.digestSends[key] = db.DigestSend{
		FamilyID: arg.FamilyID,
		UserID:   arg.UserID,
		SentOn:   arg.SentOn,
		SentAt:   now(),
	}
	return nil
}

// --- Stub methods for Querier interface ---
func (s *fakeStore) CalendarLinkedInFamily(_ context.Context, arg db.CalendarLinkedInFamilyParams) (bool, error) {
	return false, nil
}
func (s *fakeStore) CreateBirthday(_ context.Context, arg db.CreateBirthdayParams) (db.Birthday, error) {
	return db.Birthday{}, nil
}
func (s *fakeStore) DeleteBirthday(_ context.Context, arg db.DeleteBirthdayParams) (int64, error) {
	return 0, nil
}
func (s *fakeStore) DeleteCalendarLink(_ context.Context, arg db.DeleteCalendarLinkParams) error {
	return nil
}
func (s *fakeStore) DeleteCalendarLinksForUser(_ context.Context, arg db.DeleteCalendarLinksForUserParams) error {
	return nil
}
func (s *fakeStore) DeleteGoogleConnection(_ context.Context, arg db.DeleteGoogleConnectionParams) (int64, error) {
	return 0, nil
}
func (s *fakeStore) GetBirthday(_ context.Context, arg db.GetBirthdayParams) (db.Birthday, error) {
	return db.Birthday{}, pgx.ErrNoRows
}
func (s *fakeStore) GetCalendarLink(_ context.Context, arg db.GetCalendarLinkParams) (db.CalendarLink, error) {
	return db.CalendarLink{}, pgx.ErrNoRows
}
func (s *fakeStore) GetCalendarLinkByEvent(_ context.Context, arg db.GetCalendarLinkByEventParams) (db.CalendarLink, error) {
	return db.CalendarLink{}, pgx.ErrNoRows
}
func (s *fakeStore) GetGoogleConnection(_ context.Context, arg db.GetGoogleConnectionParams) (db.GoogleConnection, error) {
	return db.GoogleConnection{}, pgx.ErrNoRows
}
func (s *fakeStore) ListAllBirthdays(_ context.Context) ([]db.Birthday, error) { return nil, nil }
func (s *fakeStore) ListBirthdays(_ context.Context, familyID pgtype.UUID) ([]db.Birthday, error) {
	return nil, nil
}
func (s *fakeStore) ListCalendarLinksForItem(_ context.Context, arg db.ListCalendarLinksForItemParams) ([]db.CalendarLink, error) {
	return nil, nil
}
func (s *fakeStore) ListFamilyGoogleConnections(_ context.Context, familyID pgtype.UUID) ([]db.GoogleConnection, error) {
	return nil, nil
}
func (s *fakeStore) ListOrphanCalendarLinks(_ context.Context, arg db.ListOrphanCalendarLinksParams) ([]db.CalendarLink, error) {
	return nil, nil
}
func (s *fakeStore) ListSyncableGoogleConnections(_ context.Context) ([]db.GoogleConnection, error) {
	return nil, nil
}
func (s *fakeStore) SetGoogleCalendar(_ context.Context, arg db.SetGoogleCalendarParams) (db.GoogleConnection, error) {
	return db.GoogleConnection{}, nil
}
func (s *fakeStore) SetGoogleError(_ context.Context, arg db.SetGoogleErrorParams) error { return nil }
func (s *fakeStore) SetGoogleSyncState(_ context.Context, arg db.SetGoogleSyncStateParams) error {
	return nil
}
func (s *fakeStore) UpsertCalendarLink(_ context.Context, arg db.UpsertCalendarLinkParams) error {
	return nil
}
func (s *fakeStore) UpsertGoogleConnection(_ context.Context, arg db.UpsertGoogleConnectionParams) (db.GoogleConnection, error) {
	return db.GoogleConnection{}, nil
}
func (s *fakeStore) UpdateBirthday(_ context.Context, arg db.UpdateBirthdayParams) (db.Birthday, error) {
	return db.Birthday{}, nil
}

type recorder struct {
	subjects []string
	messages []proto.Message
}

func (r *recorder) EnsureStream(_ context.Context, domain string) error { return nil }

func (r *recorder) Publish(_ context.Context, subject events.Subject, msg any) error {
	r.subjects = append(r.subjects, string(subject))
	if pm, ok := msg.(proto.Message); ok {
		r.messages = append(r.messages, pm)
	}
	return nil
}

func (r *recorder) sawSubject(want string) bool {
	for _, s := range r.subjects {
		if s == want {
			return true
		}
	}
	return false
}

var _ db.Querier = (*fakeStore)(nil)
