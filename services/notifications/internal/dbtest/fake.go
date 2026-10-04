package dbtest

import (
	"context"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/services/notifications/db"
)

type Event struct {
	Subject     string
	Status      string
	ClaimedAt   time.Time
	ProcessedAt time.Time
}

type Fake struct {
	mu sync.Mutex

	Tokens  map[string]db.PushToken
	Mutes   map[string]map[string]bool
	Events  map[string]Event
	Tickets map[string]db.PushTicket

	Now    func() time.Time
	FailOn map[string]error

	seq int
}

func New() *Fake {
	return &Fake{
		Tokens:  map[string]db.PushToken{},
		Mutes:   map[string]map[string]bool{},
		Events:  map[string]Event{},
		Tickets: map[string]db.PushTicket{},
		Now:     time.Now,
		FailOn:  map[string]error{},
	}
}

func (f *Fake) fail(op string) error { return f.FailOn[op] }

func (f *Fake) InTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(f)
}

func (f *Fake) UpsertPushToken(_ context.Context, arg db.UpsertPushTokenParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("UpsertPushToken"); err != nil {
		return 0, err
	}
	t, ok := f.Tokens[arg.Token]
	if ok && t.UserID != arg.UserID && (t.DeviceID == "" || t.DeviceID != arg.DeviceID) {
		return 0, nil
	}
	if !ok {
		f.seq++
		t.CreatedAt = pgtype.Timestamptz{Time: time.Unix(int64(f.seq), 0), Valid: true}
	}
	t.UpdatedAt = pgtype.Timestamptz{Time: f.Now(), Valid: true}
	t.Token, t.UserID, t.FamilyID = arg.Token, arg.UserID, arg.FamilyID
	t.Platform, t.App, t.DeviceID = arg.Platform, arg.App, arg.DeviceID
	f.Tokens[arg.Token] = t
	return 1, nil
}

func (f *Fake) DeleteUserPushToken(_ context.Context, arg db.DeleteUserPushTokenParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.Tokens[arg.Token]
	if !ok || t.UserID != arg.UserID {
		return 0, nil
	}
	delete(f.Tokens, arg.Token)
	return 1, nil
}

func (f *Fake) DeleteDeadPushTokens(_ context.Context, arg db.DeleteDeadPushTokensParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("DeleteDeadPushTokens"); err != nil {
		return 0, err
	}
	var n int64
	for i, token := range arg.Tokens {
		t, ok := f.Tokens[token]
		if ok && t.UpdatedAt.Time.Before(arg.SentAt[i].Time) {
			delete(f.Tokens, token)
			n++
		}
	}
	return n, nil
}

func (f *Fake) sorted(keep func(db.PushToken) bool) []db.PushToken {
	var out []db.PushToken
	for _, t := range f.Tokens {
		if keep(t) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		ui, uj := pgconv.UUIDString(out[i].UserID), pgconv.UUIDString(out[j].UserID)
		if ui != uj {
			return ui < uj
		}
		return out[i].CreatedAt.Time.Before(out[j].CreatedAt.Time)
	})
	return out
}

func (f *Fake) ListFamilyPushTokens(_ context.Context, familyID pgtype.UUID) ([]db.PushToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("ListFamilyPushTokens"); err != nil {
		return nil, err
	}
	return f.sorted(func(t db.PushToken) bool { return t.FamilyID.Valid && t.FamilyID == familyID }), nil
}

func (f *Fake) ListUserPushTokens(_ context.Context, userID pgtype.UUID) ([]db.PushToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sorted(func(t db.PushToken) bool { return t.UserID == userID }), nil
}

func (f *Fake) SetUserFamily(_ context.Context, arg db.SetUserFamilyParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, t := range f.Tokens {
		if t.UserID == arg.UserID {
			t.FamilyID = arg.FamilyID
			f.Tokens[k] = t
		}
	}
	return nil
}

func (f *Fake) ClearUserFamily(_ context.Context, arg db.ClearUserFamilyParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, t := range f.Tokens {
		if t.UserID == arg.UserID && t.FamilyID == arg.FamilyID {
			t.FamilyID = pgtype.UUID{}
			f.Tokens[k] = t
		}
	}
	return nil
}

func (f *Fake) ListMutes(_ context.Context, userID pgtype.UUID) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("ListMutes"); err != nil {
		return nil, err
	}
	var out []string
	for topic := range f.Mutes[pgconv.UUIDString(userID)] {
		out = append(out, topic)
	}
	slices.Sort(out)
	return out, nil
}

func (f *Fake) ListMutesForUsers(_ context.Context, userIDs []pgtype.UUID) ([]db.ListMutesForUsersRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []db.ListMutesForUsersRow
	for _, u := range userIDs {
		for topic := range f.Mutes[pgconv.UUIDString(u)] {
			out = append(out, db.ListMutesForUsersRow{UserID: u, Topic: topic})
		}
	}
	return out, nil
}

func (f *Fake) DeleteMutes(_ context.Context, userID pgtype.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("DeleteMutes"); err != nil {
		return err
	}
	delete(f.Mutes, pgconv.UUIDString(userID))
	return nil
}

func (f *Fake) InsertMute(_ context.Context, arg db.InsertMuteParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := pgconv.UUIDString(arg.UserID)
	if f.Mutes[key] == nil {
		f.Mutes[key] = map[string]bool{}
	}
	f.Mutes[key][arg.Topic] = true
	return nil
}

func (f *Fake) ClaimEvent(_ context.Context, arg db.ClaimEventParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("ClaimEvent"); err != nil {
		return 0, err
	}
	e, ok := f.Events[arg.EventID]
	staleBefore := f.Now().Add(-time.Duration(arg.StaleAfterSeconds * float64(time.Second)))
	if ok && (e.Status != "claimed" || !e.ClaimedAt.Before(staleBefore)) {
		return 0, nil
	}
	if !ok {
		e = Event{Subject: arg.Subject, ProcessedAt: f.Now()}
	}
	e.Status, e.ClaimedAt = "claimed", f.Now()
	f.Events[arg.EventID] = e
	return 1, nil
}

func (f *Fake) CurrentTime(context.Context) (pgtype.Timestamptz, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return pgtype.Timestamptz{Time: f.Now(), Valid: true}, nil
}

func (f *Fake) EventStatus(_ context.Context, eventID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.Events[eventID]
	if !ok {
		return "", pgx.ErrNoRows
	}
	return e.Status, nil
}

func (f *Fake) CompleteEvent(_ context.Context, eventID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("CompleteEvent"); err != nil {
		return err
	}
	if e, ok := f.Events[eventID]; ok {
		e.Status, e.ProcessedAt = "done", f.Now()
		f.Events[eventID] = e
	}
	return nil
}

func (f *Fake) ReleaseEvent(_ context.Context, eventID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e, ok := f.Events[eventID]; ok && e.Status == "claimed" {
		delete(f.Events, eventID)
	}
	return nil
}

func (f *Fake) Processed(eventID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Events[eventID].Status == "done"
}

func (f *Fake) PruneProcessedEvents(_ context.Context, before pgtype.Timestamptz) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for id, e := range f.Events {
		if e.ProcessedAt.Before(before.Time) {
			delete(f.Events, id)
			n++
		}
	}
	return n, nil
}

func (f *Fake) InsertPushTicket(_ context.Context, arg db.InsertPushTicketParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.Tickets[arg.ID]; !ok {
		f.Tickets[arg.ID] = db.PushTicket{
			ID: arg.ID, Token: arg.Token,
			CreatedAt: pgtype.Timestamptz{Time: f.Now(), Valid: true},
		}
	}
	return nil
}

func (f *Fake) ListDuePushTickets(_ context.Context, arg db.ListDuePushTicketsParams) ([]db.PushTicket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []db.PushTicket
	for _, t := range f.Tickets {
		if t.CreatedAt.Time.Before(arg.CreatedAt.Time) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Time.Before(out[j].CreatedAt.Time) })
	if len(out) > int(arg.Limit) {
		out = out[:arg.Limit]
	}
	return out, nil
}

func (f *Fake) DeletePushTickets(_ context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, id := range ids {
		delete(f.Tickets, id)
	}
	return nil
}

var _ db.Querier = (*Fake)(nil)
