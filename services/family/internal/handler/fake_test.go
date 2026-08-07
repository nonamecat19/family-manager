package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/services/family/db"
)

// fakeStore is an in-memory db.Querier. The handler's rules (who may invite, who may leave)
// are what these tests exercise; Postgres itself is not under test here.
type fakeStore struct {
	families    map[string]db.Family
	members     map[string]db.FamilyMember // keyed family|user
	invitations map[string]db.FamilyInvitation

	failOn map[string]error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		families:    map[string]db.Family{},
		members:     map[string]db.FamilyMember{},
		invitations: map[string]db.FamilyInvitation{},
		failOn:      map[string]error{},
	}
}

func memberKey(familyID, userID pgtype.UUID) string {
	return pgconv.UUIDString(familyID) + "|" + pgconv.UUIDString(userID)
}

func (s *fakeStore) fail(op string) error { return s.failOn[op] }

func (s *fakeStore) CreateFamily(_ context.Context, arg db.CreateFamilyParams) (db.Family, error) {
	if err := s.fail("CreateFamily"); err != nil {
		return db.Family{}, err
	}
	fam := db.Family{
		ID:          pgconv.MustUUID(newUUID()),
		Name:        arg.Name,
		OwnerUserID: arg.OwnerUserID,
		CreatedAt:   pgtype.Timestamptz{Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Valid: true},
	}
	s.families[pgconv.UUIDString(fam.ID)] = fam
	return fam, nil
}

func (s *fakeStore) GetFamily(_ context.Context, id pgtype.UUID) (db.Family, error) {
	fam, ok := s.families[pgconv.UUIDString(id)]
	if !ok {
		return db.Family{}, pgx.ErrNoRows
	}
	return fam, nil
}

func (s *fakeStore) UpdateFamily(_ context.Context, arg db.UpdateFamilyParams) (db.Family, error) {
	fam, ok := s.families[pgconv.UUIDString(arg.ID)]
	if !ok {
		return db.Family{}, pgx.ErrNoRows
	}
	fam.Name = arg.Name
	s.families[pgconv.UUIDString(arg.ID)] = fam
	return fam, nil
}

func (s *fakeStore) DeleteFamily(_ context.Context, id pgtype.UUID) error {
	delete(s.families, pgconv.UUIDString(id))
	return nil
}

func (s *fakeStore) AddMember(_ context.Context, arg db.AddMemberParams) (db.FamilyMember, error) {
	if err := s.fail("AddMember"); err != nil {
		return db.FamilyMember{}, err
	}
	m := db.FamilyMember{
		FamilyID:    arg.FamilyID,
		UserID:      arg.UserID,
		DisplayName: arg.DisplayName,
		Email:       arg.Email,
		Role:        arg.Role,
		JoinedAt:    pgtype.Timestamptz{Valid: true},
	}
	s.members[memberKey(arg.FamilyID, arg.UserID)] = m
	return m, nil
}

func (s *fakeStore) ListMembers(_ context.Context, familyID pgtype.UUID) ([]db.FamilyMember, error) {
	var out []db.FamilyMember
	for _, m := range s.members {
		if pgconv.UUIDString(m.FamilyID) == pgconv.UUIDString(familyID) {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *fakeStore) GetMembership(_ context.Context, userID pgtype.UUID) (db.FamilyMember, error) {
	for _, m := range s.members {
		if pgconv.UUIDString(m.UserID) == pgconv.UUIDString(userID) {
			return m, nil
		}
	}
	return db.FamilyMember{}, pgx.ErrNoRows
}

func (s *fakeStore) GetMember(_ context.Context, arg db.GetMemberParams) (db.FamilyMember, error) {
	m, ok := s.members[memberKey(arg.FamilyID, arg.UserID)]
	if !ok {
		return db.FamilyMember{}, pgx.ErrNoRows
	}
	return m, nil
}

func (s *fakeStore) RemoveMember(_ context.Context, arg db.RemoveMemberParams) (int64, error) {
	key := memberKey(arg.FamilyID, arg.UserID)
	if _, ok := s.members[key]; !ok {
		return 0, nil
	}
	delete(s.members, key)
	return 1, nil
}

func (s *fakeStore) CountAdmins(_ context.Context, familyID pgtype.UUID) (int64, error) {
	var n int64
	for _, m := range s.members {
		if pgconv.UUIDString(m.FamilyID) == pgconv.UUIDString(familyID) && m.Role == roleAdmin {
			n++
		}
	}
	return n, nil
}

func (s *fakeStore) CreateInvitation(
	_ context.Context, arg db.CreateInvitationParams,
) (db.FamilyInvitation, error) {
	if err := s.fail("CreateInvitation"); err != nil {
		return db.FamilyInvitation{}, err
	}
	inv := db.FamilyInvitation{
		ID:            pgconv.MustUUID(newUUID()),
		FamilyID:      arg.FamilyID,
		InviterUserID: arg.InviterUserID,
		Email:         arg.Email,
		Role:          arg.Role,
		TokenHash:     arg.TokenHash,
		Status:        invitationPending,
		ExpiresAt:     arg.ExpiresAt,
		CreatedAt:     pgtype.Timestamptz{Valid: true},
	}
	s.invitations[pgconv.UUIDString(inv.ID)] = inv
	return inv, nil
}

func (s *fakeStore) GetInvitation(_ context.Context, id pgtype.UUID) (db.FamilyInvitation, error) {
	inv, ok := s.invitations[pgconv.UUIDString(id)]
	if !ok {
		return db.FamilyInvitation{}, pgx.ErrNoRows
	}
	return inv, nil
}

func (s *fakeStore) GetInvitationByTokenHash(
	_ context.Context, hash string,
) (db.FamilyInvitation, error) {
	for _, inv := range s.invitations {
		if inv.TokenHash == hash {
			return inv, nil
		}
	}
	return db.FamilyInvitation{}, pgx.ErrNoRows
}

func (s *fakeStore) ListInvitations(
	_ context.Context, familyID pgtype.UUID,
) ([]db.FamilyInvitation, error) {
	var out []db.FamilyInvitation
	for _, inv := range s.invitations {
		if pgconv.UUIDString(inv.FamilyID) == pgconv.UUIDString(familyID) {
			out = append(out, inv)
		}
	}
	return out, nil
}

func (s *fakeStore) MarkInvitationAccepted(
	_ context.Context, arg db.MarkInvitationAcceptedParams,
) (db.FamilyInvitation, error) {
	inv, ok := s.invitations[pgconv.UUIDString(arg.ID)]
	if !ok || inv.Status != invitationPending {
		return db.FamilyInvitation{}, pgx.ErrNoRows
	}
	inv.Status = "accepted"
	inv.AcceptedBy = arg.AcceptedBy
	s.invitations[pgconv.UUIDString(arg.ID)] = inv
	return inv, nil
}

func (s *fakeStore) MarkInvitationRevoked(_ context.Context, id pgtype.UUID) (int64, error) {
	inv, ok := s.invitations[pgconv.UUIDString(id)]
	if !ok || inv.Status != invitationPending {
		return 0, nil
	}
	inv.Status = "revoked"
	s.invitations[pgconv.UUIDString(id)] = inv
	return 1, nil
}

func (s *fakeStore) ExpireStaleInvitations(context.Context) (int64, error) { return 0, nil }

// recorder captures published events so tests can assert on them without NATS.
type recorder struct {
	published []events.Subject
	err       error
}

func (r *recorder) Publish(_ context.Context, subject events.Subject, _ proto.Message) error {
	if r.err != nil {
		return r.err
	}
	r.published = append(r.published, subject)
	return nil
}

func (r *recorder) sawSubject(s events.Subject) bool {
	for _, got := range r.published {
		if got == s {
			return true
		}
	}
	return false
}

var errBoom = errors.New("boom")

// newUUID hands out deterministic, valid v4-shaped ids so failures are reproducible.
var uuidCounter int

func newUUID() string {
	uuidCounter++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", uuidCounter)
}
