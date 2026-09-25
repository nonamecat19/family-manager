package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/sdk/go/family/v1/familyv1connect"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/family"
)

type birthdayStore struct {
	*fakeStore
	birthdays map[string]db.Birthday
}

func (s *birthdayStore) InTx(_ context.Context, fn func(q db.Querier) error) error { return fn(s) }

func (s *birthdayStore) CreateBirthday(_ context.Context, arg db.CreateBirthdayParams) (db.Birthday, error) {
	b := db.Birthday{
		ID: s.newUUID(), FamilyID: arg.FamilyID, Name: arg.Name, Day: arg.Day, Month: arg.Month,
		Year: arg.Year, RemindDaysBefore: arg.RemindDaysBefore, CreatedByUserID: arg.CreatedByUserID,
		CreatedAt: now(), UpdatedAt: now(),
	}
	s.birthdays[id(b.ID)] = b
	return b, nil
}

func (s *birthdayStore) GetBirthday(_ context.Context, arg db.GetBirthdayParams) (db.Birthday, error) {
	b, ok := s.birthdays[id(arg.ID)]
	if !ok || !same(b.FamilyID, arg.FamilyID) {
		return db.Birthday{}, pgx.ErrNoRows
	}
	return b, nil
}

func (s *birthdayStore) UpdateBirthday(_ context.Context, arg db.UpdateBirthdayParams) (db.Birthday, error) {
	b, ok := s.birthdays[id(arg.ID)]
	if !ok || !same(b.FamilyID, arg.FamilyID) {
		return db.Birthday{}, pgx.ErrNoRows
	}
	b.Name, b.Day, b.Month, b.Year, b.RemindDaysBefore = arg.Name, arg.Day, arg.Month, arg.Year, arg.RemindDaysBefore
	s.birthdays[id(b.ID)] = b
	return b, nil
}

func (s *birthdayStore) DeleteBirthday(_ context.Context, arg db.DeleteBirthdayParams) (int64, error) {
	b, ok := s.birthdays[id(arg.ID)]
	if !ok || !same(b.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.birthdays, id(arg.ID))
	return 1, nil
}

func (s *birthdayStore) ListBirthdays(_ context.Context, familyID pgtype.UUID) ([]db.Birthday, error) {
	var out []db.Birthday
	for _, b := range s.birthdays {
		if same(b.FamilyID, familyID) {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return id(out[i].ID) < id(out[j].ID) })
	return out, nil
}

func newBirthdayHandler(t *testing.T, fam *family.Client) (*Handler, *birthdayStore) {
	t.Helper()
	store := &birthdayStore{fakeStore: newFakeStore(), birthdays: map[string]db.Birthday{}}
	seedHousehold(store.fakeStore)
	h := New(Options{
		Queries: store, Tx: store, FamilyPublic: fam,
		Now: func() time.Time { return testNow },
	})
	return h, store
}

func createBirthday(t *testing.T, h *Handler, req *tasksv1.CreateBirthdayRequest) *tasksv1.Birthday {
	t.Helper()
	resp, err := h.CreateBirthday(ctxOf(sergiy), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("CreateBirthday: %v", err)
	}
	return resp.Msg.Birthday
}

func birthdayReminders(s *birthdayStore, birthdayID string) []db.TaskReminder {
	var out []db.TaskReminder
	for _, r := range s.reminders {
		if r.Kind == kindBirthday && id(r.ItemID) == birthdayID {
			out = append(out, r)
		}
	}
	return out
}

func TestCreateBirthdayReturnsNextOccurrenceAndAge(t *testing.T) {
	h, _ := newBirthdayHandler(t, nil)

	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{
		Name: "  Олена  ", Day: 15, Month: 9, Year: 1990, RemindDaysBefore: 7,
	})

	if b.Name != "Олена" {
		t.Errorf("name = %q, want trimmed", b.Name)
	}
	if b.Day != 15 || b.Month != 9 || b.Year != 1990 || b.RemindDaysBefore != 7 {
		t.Errorf("fields = %d/%d/%d remind %d", b.Day, b.Month, b.Year, b.RemindDaysBefore)
	}
	if b.NextOn != "2026-09-15" || b.DaysUntil != 16 || b.TurningAge != 36 {
		t.Errorf("next = %s in %d days turning %d, want 2026-09-15 in 16 turning 36", b.NextOn, b.DaysUntil, b.TurningAge)
	}
}

func TestCreateBirthdayWithoutYearHasNoAge(t *testing.T) {
	h, _ := newBirthdayHandler(t, nil)

	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "Бабуся", Day: 30, Month: 8})

	if b.Year != 0 || b.TurningAge != 0 {
		t.Errorf("year %d turning %d, want 0/0", b.Year, b.TurningAge)
	}
	if b.NextOn != "2026-08-30" || b.DaysUntil != 0 {
		t.Errorf("next = %s in %d, want today", b.NextOn, b.DaysUntil)
	}
}

func TestLeapDayBirthdayObservedOn28February(t *testing.T) {
	h, _ := newBirthdayHandler(t, nil)

	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "Leap", Day: 29, Month: 2, Year: 2020})

	if b.NextOn != "2027-02-28" || b.TurningAge != 7 {
		t.Errorf("next = %s turning %d, want 2027-02-28 turning 7", b.NextOn, b.TurningAge)
	}
}

func TestCreateBirthdayValidation(t *testing.T) {
	h, _ := newBirthdayHandler(t, nil)
	cases := map[string]*tasksv1.CreateBirthdayRequest{
		"blank name":        {Name: "  ", Day: 1, Month: 1},
		"long name":         {Name: strings.Repeat("я", maxTitleRunes+1), Day: 1, Month: 1},
		"month 13":          {Name: "x", Day: 1, Month: 13},
		"31 April":          {Name: "x", Day: 31, Month: 4},
		"29 Feb non-leap":   {Name: "x", Day: 29, Month: 2, Year: 2023},
		"future birth date": {Name: "x", Day: 31, Month: 8, Year: 2026},
		"remind days > 60":  {Name: "x", Day: 1, Month: 1, RemindDaysBefore: 61},
		"negative remind":   {Name: "x", Day: 1, Month: 1, RemindDaysBefore: -1},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := h.CreateBirthday(ctxOf(sergiy), connect.NewRequest(req))
			if got := codeOf(t, err); got != connect.CodeInvalidArgument {
				t.Errorf("code = %v, want InvalidArgument", got)
			}
		})
	}
}

func TestCreateBirthdaySchedulesRemindersForEveryKnownMember(t *testing.T) {
	h, store := newBirthdayHandler(t, nil)

	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "Олена", Day: 15, Month: 9, RemindDaysBefore: 3})

	got := map[string]time.Time{}
	for _, r := range birthdayReminders(store, b.Id) {
		got[id(r.UserID)+" "+r.Occurrence] = r.RemindAt.Time
	}
	early := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	dayOf := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	want := map[string]time.Time{
		sergiy + " 2026-09-15-3d": early, sergiy + " 2026-09-15": dayOf,
		olena + " 2026-09-15-3d": early, olena + " 2026-09-15": dayOf,
	}
	if len(got) != len(want) {
		t.Fatalf("reminders = %v, want %v", got, want)
	}
	for k, at := range want {
		if !got[k].Equal(at) {
			t.Errorf("reminder %s at %v, want %v", k, got[k], at)
		}
	}
}

func TestBirthdayRemindersUseFamilyTimezone(t *testing.T) {
	h, store := newBirthdayHandler(t, nil)
	s := store.settings[testFamily]
	s.Timezone = "Europe/Kyiv"
	store.settings[testFamily] = s

	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "x", Day: 15, Month: 9})

	for _, r := range birthdayReminders(store, b.Id) {
		if want := time.Date(2026, 9, 15, 6, 0, 0, 0, time.UTC); !r.RemindAt.Time.Equal(want) {
			t.Errorf("remind_at = %v, want 09:00 Kyiv (%v)", r.RemindAt.Time, want)
		}
	}
}

func TestBirthdayRemindersInThePastAreSkipped(t *testing.T) {
	h, store := newBirthdayHandler(t, nil)

	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "x", Day: 31, Month: 8, RemindDaysBefore: 5})

	for _, r := range birthdayReminders(store, b.Id) {
		if r.RemindAt.Time.Before(testNow) {
			t.Errorf("reminder %s scheduled in the past: %v", r.Occurrence, r.RemindAt.Time)
		}
	}
	if n := len(birthdayReminders(store, b.Id)); n != 2 {
		t.Errorf("reminders = %d, want 2 (day-of for each member)", n)
	}
}

func TestUpdateBirthdayPartialKeepsOtherFields(t *testing.T) {
	h, _ := newBirthdayHandler(t, nil)
	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "Old", Day: 15, Month: 9, Year: 1990, RemindDaysBefore: 7})

	name := "New"
	resp, err := h.UpdateBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateBirthdayRequest{
		BirthdayId: b.Id, Name: &name,
	}))
	if err != nil {
		t.Fatalf("UpdateBirthday: %v", err)
	}
	got := resp.Msg.Birthday
	if got.Name != "New" || got.Day != 15 || got.Month != 9 || got.Year != 1990 || got.RemindDaysBefore != 7 {
		t.Errorf("got %+v", got)
	}
}

func TestUpdateBirthdayClearsYearAndReschedules(t *testing.T) {
	h, store := newBirthdayHandler(t, nil)
	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "x", Day: 15, Month: 9, Year: 1990, RemindDaysBefore: 7})

	year, day, remind := int32(0), int32(20), int32(0)
	resp, err := h.UpdateBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateBirthdayRequest{
		BirthdayId: b.Id, Year: &year, Day: &day, RemindDaysBefore: &remind,
	}))
	if err != nil {
		t.Fatalf("UpdateBirthday: %v", err)
	}
	if got := resp.Msg.Birthday; got.Year != 0 || got.TurningAge != 0 || got.NextOn != "2026-09-20" {
		t.Errorf("got year %d age %d next %s", got.Year, got.TurningAge, got.NextOn)
	}
	for _, r := range birthdayReminders(store, b.Id) {
		if r.Occurrence != "2026-09-20" {
			t.Errorf("stale reminder %s survived the update", r.Occurrence)
		}
	}
	if n := len(birthdayReminders(store, b.Id)); n != 2 {
		t.Errorf("reminders = %d, want 2", n)
	}
}

func TestUpdateBirthdayValidatesMergedDate(t *testing.T) {
	h, _ := newBirthdayHandler(t, nil)
	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "x", Day: 31, Month: 1})

	month := int32(4)
	_, err := h.UpdateBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateBirthdayRequest{
		BirthdayId: b.Id, Month: &month,
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("31 + April: code = %v, want InvalidArgument", got)
	}
}

func TestDeleteBirthdayRemovesPendingReminders(t *testing.T) {
	h, store := newBirthdayHandler(t, nil)
	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "x", Day: 15, Month: 9, RemindDaysBefore: 3})

	if _, err := h.DeleteBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.DeleteBirthdayRequest{BirthdayId: b.Id})); err != nil {
		t.Fatalf("DeleteBirthday: %v", err)
	}
	if _, ok := store.birthdays[b.Id]; ok {
		t.Error("birthday still stored")
	}
	if n := len(birthdayReminders(store, b.Id)); n != 0 {
		t.Errorf("pending reminders = %d, want 0", n)
	}

	_, err := h.DeleteBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.DeleteBirthdayRequest{BirthdayId: b.Id}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("second delete: code = %v, want NotFound", got)
	}
}

func TestBirthdaysAreFamilyScoped(t *testing.T) {
	h, store := newBirthdayHandler(t, nil)
	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "ours", Day: 15, Month: 9})
	outsider := withClaims(context.Background(), olena, otherFam)
	store.settings[otherFam] = db.FamilySetting{FamilyID: pgconv.MustUUID(otherFam), Timezone: "UTC"}

	list, err := h.ListBirthdays(outsider, connect.NewRequest(&tasksv1.ListBirthdaysRequest{}))
	if err != nil {
		t.Fatalf("ListBirthdays: %v", err)
	}
	if n := len(list.Msg.Birthdays); n != 0 {
		t.Errorf("other family sees %d birthdays, want 0", n)
	}
	list, err = h.ListBirthdays(ctxOf(olena), connect.NewRequest(&tasksv1.ListBirthdaysRequest{}))
	if err != nil {
		t.Fatalf("ListBirthdays: %v", err)
	}
	if n := len(list.Msg.Birthdays); n != 1 || list.Msg.Birthdays[0].Id != b.Id {
		t.Errorf("own family sees %v, want [%s]", list.Msg.Birthdays, b.Id)
	}

	name := "theirs"
	_, err = h.UpdateBirthday(outsider, connect.NewRequest(&tasksv1.UpdateBirthdayRequest{BirthdayId: b.Id, Name: &name}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("update from other family: code = %v, want NotFound", got)
	}
	_, err = h.DeleteBirthday(outsider, connect.NewRequest(&tasksv1.DeleteBirthdayRequest{BirthdayId: b.Id}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("delete from other family: code = %v, want NotFound", got)
	}
	if store.birthdays[b.Id].Name != "ours" {
		t.Error("other family changed our birthday")
	}
}

func TestBirthdayRPCsRequireClaims(t *testing.T) {
	h, _ := newBirthdayHandler(t, nil)

	_, err := h.CreateBirthday(context.Background(), connect.NewRequest(&tasksv1.CreateBirthdayRequest{Name: "x", Day: 1, Month: 1}))
	if got := codeOf(t, err); got != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want Unauthenticated", got)
	}
}

type membersFamily struct {
	familyv1connect.UnimplementedFamilyServiceHandler
	bearer string
}

func (f *membersFamily) ListMembers(
	_ context.Context, req *connect.Request[familyv1.ListMembersRequest],
) (*connect.Response[familyv1.ListMembersResponse], error) {
	f.bearer = fmauth.BearerToken(req.Header().Get("Authorization"))
	return connect.NewResponse(&familyv1.ListMembersResponse{Members: []*familyv1.Member{
		{UserId: sergiy, DisplayName: "Сергій", Email: "s@example.com"},
		{UserId: newbie, DisplayName: "Новенька", Email: "n@example.com"},
	}}), nil
}

const newbie = "00000000-0000-4000-8000-000000000004"

func TestCreateBirthdayRefreshesKnownMembersWithCallerBearer(t *testing.T) {
	fam := &membersFamily{}
	mux := http.NewServeMux()
	mux.Handle(familyv1connect.NewFamilyServiceHandler(fam))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	h, store := newBirthdayHandler(t, family.New(srv.URL, srv.URL, 0))

	req := connect.NewRequest(&tasksv1.CreateBirthdayRequest{Name: "x", Day: 15, Month: 9})
	req.Header().Set("Authorization", "Bearer caller-token")
	resp, err := h.CreateBirthday(ctxOf(sergiy), req)
	if err != nil {
		t.Fatalf("CreateBirthday: %v", err)
	}

	if fam.bearer != "caller-token" {
		t.Errorf("ListMembers bearer = %q, want the caller's token", fam.bearer)
	}
	if _, ok := store.members[testFamily+"|"+olena]; ok {
		t.Error("olena left the family but is still a known member")
	}
	if m, ok := store.members[testFamily+"|"+newbie]; !ok || m.DisplayName != "Новенька" {
		t.Errorf("newbie known member = %+v, %v", m, ok)
	}
	users := map[string]bool{}
	for _, r := range birthdayReminders(store, resp.Msg.Birthday.Id) {
		users[id(r.UserID)] = true
	}
	if !users[sergiy] || !users[newbie] || users[olena] {
		t.Errorf("reminder recipients = %v, want sergiy and newbie", users)
	}
}

func TestBirthdayWritesNeedFamilyTimezone(t *testing.T) {
	h, store := newBirthdayHandler(t, nil)
	b := createBirthday(t, h, &tasksv1.CreateBirthdayRequest{Name: "x", Day: 15, Month: 9})
	delete(store.settings, testFamily)

	_, err := h.CreateBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.CreateBirthdayRequest{Name: "y", Day: 1, Month: 10}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("create without settings: code = %v, want FailedPrecondition", got)
	}
	name := "z"
	_, err = h.UpdateBirthday(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateBirthdayRequest{BirthdayId: b.Id, Name: &name}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("update without settings: code = %v, want FailedPrecondition", got)
	}
}

type failingFamily struct {
	familyv1connect.UnimplementedFamilyServiceHandler
}

func (failingFamily) ListMembers(
	context.Context, *connect.Request[familyv1.ListMembersRequest],
) (*connect.Response[familyv1.ListMembersResponse], error) {
	return nil, connect.NewError(connect.CodeUnavailable, errors.New("family down"))
}

func TestFamilyOutageKeepsKnownMembers(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(familyv1connect.NewFamilyServiceHandler(failingFamily{}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	h, store := newBirthdayHandler(t, family.New(srv.URL, srv.URL, 0))

	req := connect.NewRequest(&tasksv1.CreateBirthdayRequest{Name: "x", Day: 15, Month: 9})
	req.Header().Set("Authorization", "Bearer caller-token")
	resp, err := h.CreateBirthday(ctxOf(sergiy), req)
	if err != nil {
		t.Fatalf("CreateBirthday: %v", err)
	}

	for _, u := range []string{sergiy, olena} {
		if _, ok := store.members[testFamily+"|"+u]; !ok {
			t.Errorf("known member %s dropped during a family outage", u)
		}
	}
	if n := len(birthdayReminders(store, resp.Msg.Birthday.Id)); n != 2 {
		t.Errorf("reminders = %d, want 2", n)
	}
}
