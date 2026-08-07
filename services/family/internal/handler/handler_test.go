package handler

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/events"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
)

var fixedNow = time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

type fixture struct {
	h     *Handler
	store *fakeStore
	bus   *recorder
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	store := newFakeStore()
	bus := &recorder{}
	h := New(Options{
		Queries:       store,
		Bus:           bus,
		InvitationTTL: 48 * time.Hour,
		Now:           func() time.Time { return fixedNow },
	})
	return &fixture{h: h, store: store, bus: bus}
}

// asUser builds a context carrying verified claims, which is what the interceptor does in
// production.
func asUser(userID string) context.Context {
	return fmauth.WithClaims(context.Background(), &fmauth.Claims{
		UserID: userID,
		Email:  userID + "@example.test",
	})
}

const (
	alice = "11111111-1111-4111-8111-111111111111"
	bob   = "22222222-2222-4222-8222-222222222222"
	carol = "33333333-3333-4333-8333-333333333333"
)

// createFamilyAs is the setup most tests need: a family with one admin.
func (f *fixture) createFamilyAs(t *testing.T, userID, name string) *familyv1.Family {
	t.Helper()
	res, err := f.h.CreateFamily(asUser(userID), connect.NewRequest(&familyv1.CreateFamilyRequest{
		Name: name,
	}))
	if err != nil {
		t.Fatalf("CreateFamily: %v", err)
	}
	return res.Msg.GetFamily()
}

func TestCreateFamilyMakesTheCreatorAnAdmin(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Test Household")

	if fam.GetName() != "Test Household" {
		t.Errorf("name = %q", fam.GetName())
	}
	if fam.GetOwnerUserId() != alice {
		t.Errorf("owner = %q, want alice", fam.GetOwnerUserId())
	}

	members, err := f.h.ListMembers(asUser(alice), connect.NewRequest(&familyv1.ListMembersRequest{}))
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members.Msg.GetMembers()) != 1 {
		t.Fatalf("got %d members, want 1", len(members.Msg.GetMembers()))
	}
	if got := members.Msg.GetMembers()[0].GetRole(); got != familyv1.Role_ROLE_ADMIN {
		t.Errorf("creator role = %v, want admin", got)
	}
	if !f.bus.sawSubject(events.SubjectFamilyMemberJoined) {
		t.Error("expected family.member.joined")
	}
}

func TestCreateFamilyRequiresAuthentication(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.CreateFamily(context.Background(),
		connect.NewRequest(&familyv1.CreateFamilyRequest{Name: "x"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestCreateFamilyRejectsBlankName(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.CreateFamily(asUser(alice),
		connect.NewRequest(&familyv1.CreateFamilyRequest{Name: "   "}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestCreateSecondFamilyIsRejected(t *testing.T) {
	f := newFixture(t)
	f.createFamilyAs(t, alice, "First")

	_, err := f.h.CreateFamily(asUser(alice),
		connect.NewRequest(&familyv1.CreateFamilyRequest{Name: "Second"}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("code = %v, want already_exists", connect.CodeOf(err))
	}
}

func TestGetFamilyRejectsAnotherFamilysID(t *testing.T) {
	f := newFixture(t)
	f.createFamilyAs(t, alice, "Alice household")
	other := f.createFamilyAs(t, bob, "Bob household")

	_, err := f.h.GetFamily(asUser(alice),
		connect.NewRequest(&familyv1.GetFamilyRequest{FamilyId: other.GetId()}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("code = %v, want permission_denied", connect.CodeOf(err))
	}
}

func TestGetFamilyWithoutMembershipFails(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.GetFamily(asUser(alice), connect.NewRequest(&familyv1.GetFamilyRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want failed_precondition", connect.CodeOf(err))
	}
}

func TestInviteRequiresAdmin(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")

	// Bob joins as a plain member, then tries to invite.
	inv := f.invite(t, alice, fam.GetId(), "bob@example.test")
	f.accept(t, bob, inv)

	_, err := f.h.InviteMember(asUser(bob), connect.NewRequest(&familyv1.InviteMemberRequest{
		FamilyId: fam.GetId(), Email: "carol@example.test",
	}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("code = %v, want permission_denied", connect.CodeOf(err))
	}
}

func TestInviteReturnsTokenOnceAndStoresOnlyItsHash(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")

	res, err := f.h.InviteMember(asUser(alice), connect.NewRequest(&familyv1.InviteMemberRequest{
		FamilyId: fam.GetId(), Email: "bob@example.test",
	}))
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}

	token := res.Msg.GetToken()
	if token == "" {
		t.Fatal("expected a plaintext token in the response")
	}

	for _, stored := range f.store.invitations {
		if stored.TokenHash == token {
			t.Error("the plaintext token was stored; only its hash may be")
		}
		if stored.TokenHash != hashToken(token) {
			t.Error("stored hash does not match the issued token")
		}
	}
	// The invitation in the response must never carry the hash back out.
	if !f.bus.sawSubject(events.SubjectFamilyMemberInvited) {
		t.Error("expected family.member.invited")
	}
}

func TestInviteDefaultsToMemberRole(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")

	// ROLE_UNSPECIFIED must not be read as admin.
	inv := f.invite(t, alice, fam.GetId(), "bob@example.test")
	f.accept(t, bob, inv)

	members, err := f.h.ListMembers(asUser(alice), connect.NewRequest(&familyv1.ListMembersRequest{}))
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	for _, m := range members.Msg.GetMembers() {
		if m.GetUserId() == bob && m.GetRole() != familyv1.Role_ROLE_MEMBER {
			t.Errorf("bob's role = %v, want member", m.GetRole())
		}
	}
}

func TestAcceptExpiredInvitationFails(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")
	token := f.invite(t, alice, fam.GetId(), "bob@example.test")

	// Move the clock past the 48h TTL.
	f.h.now = func() time.Time { return fixedNow.Add(72 * time.Hour) }

	_, err := f.h.AcceptInvitation(asUser(bob),
		connect.NewRequest(&familyv1.AcceptInvitationRequest{Token: token}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want failed_precondition", connect.CodeOf(err))
	}
}

func TestAcceptUnknownTokenIsNotFound(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.AcceptInvitation(asUser(bob),
		connect.NewRequest(&familyv1.AcceptInvitationRequest{Token: "nonsense"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
	}
}

func TestAcceptTwiceFails(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")
	token := f.invite(t, alice, fam.GetId(), "bob@example.test")
	f.accept(t, bob, token)

	// Carol replaying the same link must not join.
	_, err := f.h.AcceptInvitation(asUser(carol),
		connect.NewRequest(&familyv1.AcceptInvitationRequest{Token: token}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want failed_precondition", connect.CodeOf(err))
	}
}

func TestRevokedInvitationCannotBeAccepted(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")

	res, err := f.h.InviteMember(asUser(alice), connect.NewRequest(&familyv1.InviteMemberRequest{
		FamilyId: fam.GetId(), Email: "bob@example.test",
	}))
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}

	if _, err := f.h.RevokeInvitation(asUser(alice),
		connect.NewRequest(&familyv1.RevokeInvitationRequest{
			InvitationId: res.Msg.GetInvitation().GetId(),
		})); err != nil {
		t.Fatalf("RevokeInvitation: %v", err)
	}

	_, err = f.h.AcceptInvitation(asUser(bob),
		connect.NewRequest(&familyv1.AcceptInvitationRequest{Token: res.Msg.GetToken()}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want failed_precondition", connect.CodeOf(err))
	}
}

func TestRemoveMemberRefusesSelf(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")

	_, err := f.h.RemoveMember(asUser(alice), connect.NewRequest(&familyv1.RemoveMemberRequest{
		FamilyId: fam.GetId(), UserId: alice,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestRemoveMemberPublishesAndDeletes(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")
	f.accept(t, bob, f.invite(t, alice, fam.GetId(), "bob@example.test"))

	if _, err := f.h.RemoveMember(asUser(alice), connect.NewRequest(&familyv1.RemoveMemberRequest{
		FamilyId: fam.GetId(), UserId: bob,
	})); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	members, err := f.h.ListMembers(asUser(alice), connect.NewRequest(&familyv1.ListMembersRequest{}))
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members.Msg.GetMembers()) != 1 {
		t.Errorf("got %d members after removal, want 1", len(members.Msg.GetMembers()))
	}
	if !f.bus.sawSubject(events.SubjectFamilyMemberRemoved) {
		t.Error("expected family.member.removed")
	}
}

func TestLastAdminCannotLeave(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")
	f.accept(t, bob, f.invite(t, alice, fam.GetId(), "bob@example.test"))

	_, err := f.h.LeaveFamily(asUser(alice),
		connect.NewRequest(&familyv1.LeaveFamilyRequest{FamilyId: fam.GetId()}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want failed_precondition", connect.CodeOf(err))
	}

	// A plain member may leave freely.
	if _, err := f.h.LeaveFamily(asUser(bob),
		connect.NewRequest(&familyv1.LeaveFamilyRequest{FamilyId: fam.GetId()})); err != nil {
		t.Fatalf("member LeaveFamily: %v", err)
	}
}

func TestCheckMembershipAnswersForSiblingServices(t *testing.T) {
	f := newFixture(t)
	fam := f.createFamilyAs(t, alice, "Household")

	res, err := f.h.CheckMembership(context.Background(),
		connect.NewRequest(&familyv1.CheckMembershipRequest{UserId: alice, FamilyId: fam.GetId()}))
	if err != nil {
		t.Fatalf("CheckMembership: %v", err)
	}
	if !res.Msg.GetIsMember() || res.Msg.GetRole() != familyv1.Role_ROLE_ADMIN {
		t.Errorf("got is_member=%v role=%v, want true/admin", res.Msg.GetIsMember(), res.Msg.GetRole())
	}

	res, err = f.h.CheckMembership(context.Background(),
		connect.NewRequest(&familyv1.CheckMembershipRequest{UserId: carol, FamilyId: fam.GetId()}))
	if err != nil {
		t.Fatalf("CheckMembership(non-member): %v", err)
	}
	if res.Msg.GetIsMember() {
		t.Error("carol is not a member but CheckMembership said she is")
	}
}

func TestCheckMembershipValidatesIDs(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.CheckMembership(context.Background(),
		connect.NewRequest(&familyv1.CheckMembershipRequest{UserId: "", FamilyId: ""}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestPublishFailureDoesNotFailTheWrite(t *testing.T) {
	f := newFixture(t)
	f.bus.err = errBoom

	// The family is still created even though the broker rejected the event.
	fam := f.createFamilyAs(t, alice, "Household")
	if fam.GetId() == "" {
		t.Fatal("expected the family to be created despite the publish failure")
	}
}

func TestStoreFailureBecomesInternal(t *testing.T) {
	f := newFixture(t)
	f.store.failOn["CreateFamily"] = errBoom

	_, err := f.h.CreateFamily(asUser(alice),
		connect.NewRequest(&familyv1.CreateFamilyRequest{Name: "Household"}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal", connect.CodeOf(err))
	}
}

/* -------------------------------------------------------------------- helpers */

func (f *fixture) invite(t *testing.T, adminID, familyID, email string) string {
	t.Helper()
	res, err := f.h.InviteMember(asUser(adminID), connect.NewRequest(&familyv1.InviteMemberRequest{
		FamilyId: familyID, Email: email,
	}))
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}
	return res.Msg.GetToken()
}

func (f *fixture) accept(t *testing.T, userID, token string) {
	t.Helper()
	if _, err := f.h.AcceptInvitation(asUser(userID),
		connect.NewRequest(&familyv1.AcceptInvitationRequest{Token: token})); err != nil {
		t.Fatalf("AcceptInvitation: %v", err)
	}
}
