package dbtest

import (
	"context"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/services/notifications/db"
)

type Fake struct {
	mu sync.Mutex

	Tokens    map[string]db.PushToken
	Mutes     map[string]map[string]bool
	Processed map[string]time.Time
	Tickets   map[string]db.PushTicket

	Now    func() time.Time
	FailOn map[string]error

	seq int
}

func New() *Fake {
	return &Fake{
		Tokens:    map[string]db.PushToken{},
		Mutes:     map[string]map[string]bool{},
		Processed: map[string]time.Time{},
		Tickets:   map[string]db.PushTicket{},
		Now:       time.Now,
		FailOn:    map[string]error{},
	}
}

func (f *Fake) fail(op string) error { return f.FailOn[op] }

func (f *Fake) InTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(f)
}

func (f *Fake) UpsertPushToken(_ context.Context, arg db.UpsertPushTokenParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("UpsertPushToken"); err != nil {
		return err
	}
	t, ok := f.Tokens[arg.Token]
	if !ok {
		f.seq++
		t.CreatedAt = pgtype.Timestamptz{Time: time.Unix(int64(f.seq), 0), Valid: true}
	}
	t.Token, t.UserID, t.FamilyID = arg.Token, arg.UserID, arg.FamilyID
	t.Platform, t.App, t.DeviceID = arg.Platform, arg.App, arg.DeviceID
	f.Tokens[arg.Token] = t
	return nil
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

func (f *Fake) DeletePushTokens(_ context.Context, tokens []string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, t := range tokens {
		if _, ok := f.Tokens[t]; ok {
			delete(f.Tokens, t)
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

func (f *Fake) IsEventProcessed(_ context.Context, eventID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.Processed[eventID]
	return ok, nil
}

func (f *Fake) MarkEventProcessed(_ context.Context, arg db.MarkEventProcessedParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.Processed[arg.EventID]; !ok {
		f.Processed[arg.EventID] = f.Now()
	}
	return nil
}

func (f *Fake) PruneProcessedEvents(_ context.Context, before pgtype.Timestamptz) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for id, at := range f.Processed {
		if at.Before(before.Time) {
			delete(f.Processed, id)
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
