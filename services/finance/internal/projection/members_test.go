package projection

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

const (
	testFamily = "00000000-0000-4000-8000-000000000001"
	testUser   = "00000000-0000-4000-8000-000000000002"
)

// fakeQuerier implements the three member queries the projection uses and panics on the rest,
// so a query added to the projection without a test is a failure rather than a silent nil.
type fakeQuerier struct {
	db.Querier
	rows map[string]db.FinanceMember
}

func newFake() *fakeQuerier { return &fakeQuerier{rows: map[string]db.FinanceMember{}} }

func key(family, user pgtype.UUID) string {
	return pgconv.UUIDString(family) + "/" + pgconv.UUIDString(user)
}

func (f *fakeQuerier) GetMember(_ context.Context, arg db.GetMemberParams) (db.FinanceMember, error) {
	m, ok := f.rows[key(arg.FamilyID, arg.UserID)]
	if !ok {
		return db.FinanceMember{}, pgx.ErrNoRows
	}
	return m, nil
}

func (f *fakeQuerier) UpsertMember(_ context.Context, arg db.UpsertMemberParams) (db.FinanceMember, error) {
	m := db.FinanceMember{
		FamilyID: arg.FamilyID, UserID: arg.UserID, DisplayName: arg.DisplayName,
		Initial: arg.Initial, AvatarColorStep: arg.AvatarColorStep,
		Role: arg.Role, Status: arg.Status, Email: arg.Email,
	}
	f.rows[key(arg.FamilyID, arg.UserID)] = m
	return m, nil
}

func (f *fakeQuerier) DeleteMember(_ context.Context, arg db.DeleteMemberParams) (int64, error) {
	k := key(arg.FamilyID, arg.UserID)
	if _, ok := f.rows[k]; !ok {
		return 0, nil
	}
	delete(f.rows, k)
	return 1, nil
}

func (f *fakeQuerier) CountMembers(_ context.Context, _ pgtype.UUID) (int64, error) {
	return int64(len(f.rows)), nil
}

func joinedPayload(t *testing.T, role familyv1.Role) []byte {
	t.Helper()
	b, err := proto.Marshal(&familyv1.MemberJoinedEvent{
		FamilyId: testFamily, UserId: testUser, Role: role,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestJoinedCreatesTheMember(t *testing.T) {
	q := newFake()
	m := NewMembers(q, nil)

	if err := m.Handle(context.Background(), events.SubjectFamilyMemberJoined,
		joinedPayload(t, familyv1.Role_ROLE_ADMIN)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	row, ok := q.rows[key(pgconv.MustUUID(testFamily), pgconv.MustUUID(testUser))]
	if !ok {
		t.Fatal("no member row was written")
	}
	if row.Role != "owner" {
		t.Errorf("role = %q, want owner", row.Role)
	}
	if row.Status != "active" {
		t.Errorf("status = %q, want active", row.Status)
	}
}

// JetStream redelivers; a second copy of the same join must not blank the name the member
// filled in by signing in.
func TestJoinedTwiceKeepsTheName(t *testing.T) {
	q := newFake()
	m := NewMembers(q, nil)
	ctx := context.Background()
	payload := joinedPayload(t, familyv1.Role_ROLE_MEMBER)

	if err := m.Handle(ctx, events.SubjectFamilyMemberJoined, payload); err != nil {
		t.Fatalf("handle: %v", err)
	}
	k := key(pgconv.MustUUID(testFamily), pgconv.MustUUID(testUser))
	row := q.rows[k]
	row.DisplayName = "Олена"
	row.Initial = "О"
	q.rows[k] = row

	if err := m.Handle(ctx, events.SubjectFamilyMemberJoined, payload); err != nil {
		t.Fatalf("handle (again): %v", err)
	}
	if got := q.rows[k].DisplayName; got != "Олена" {
		t.Errorf("display name = %q, want it kept", got)
	}
}

func TestRemovedDeletesTheMember(t *testing.T) {
	q := newFake()
	m := NewMembers(q, nil)
	ctx := context.Background()

	if err := m.Handle(ctx, events.SubjectFamilyMemberJoined,
		joinedPayload(t, familyv1.Role_ROLE_MEMBER)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	payload, err := proto.Marshal(&familyv1.MemberRemovedEvent{
		FamilyId: testFamily, UserId: testUser,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := m.Handle(ctx, events.SubjectFamilyMemberRemoved, payload); err != nil {
		t.Fatalf("handle removed: %v", err)
	}
	if len(q.rows) != 0 {
		t.Errorf("rows = %v, want the member gone", q.rows)
	}
}

// A payload this service cannot parse will not parse on redelivery either: it is recorded and
// acked, never nacked into an endless loop.
func TestUnparseablePayloadIsAcked(t *testing.T) {
	q := newFake()
	m := NewMembers(q, nil)
	if err := m.Handle(context.Background(), events.SubjectFamilyMemberJoined,
		[]byte{0xff, 0xff, 0xff}); err != nil {
		t.Errorf("err = %v, want nil so the message is acked", err)
	}
	if len(q.rows) != 0 {
		t.Error("a bad payload wrote a row")
	}
}

// A database failure nacks, so JetStream redelivers rather than losing the member.
func TestWriteFailureNacks(t *testing.T) {
	q := &failingQuerier{}
	m := NewMembers(q, nil)
	err := m.Handle(context.Background(), events.SubjectFamilyMemberJoined,
		joinedPayload(t, familyv1.Role_ROLE_MEMBER))
	if err == nil {
		t.Fatal("err = nil, want the message nacked")
	}
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want it to wrap the store failure", err)
	}
}

var errBoom = errors.New("boom")

type failingQuerier struct{ db.Querier }

func (f *failingQuerier) GetMember(context.Context, db.GetMemberParams) (db.FinanceMember, error) {
	return db.FinanceMember{}, pgx.ErrNoRows
}
func (f *failingQuerier) CountMembers(context.Context, pgtype.UUID) (int64, error) { return 0, nil }
func (f *failingQuerier) UpsertMember(context.Context, db.UpsertMemberParams) (db.FinanceMember, error) {
	return db.FinanceMember{}, errBoom
}
