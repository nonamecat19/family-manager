// Package handler implements family.v1.FamilyService over Connect (and gRPC, from the same
// type). Authorization lives here: the caller's identity comes from the verified access
// token, never from the request body.
package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5"
	"github.com/nnc/family-manager/libs/go/rpc"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/services/family/db"
)

// Handler serves family.v1.FamilyService.
type Handler struct {
	q             db.Querier
	tx            Tx
	bus           EventBus
	log           *slog.Logger
	invitationTTL time.Duration
	now           func() time.Time
}

// Tx is the transaction boundary. Implemented by internal/store over a pgx pool, and by the
// fake store in tests.
type Tx interface {
	InTx(ctx context.Context, fn func(q db.Querier) error) error
}

// Options configures a Handler. Only Queries is required.
type Options struct {
	Queries db.Querier
	// Tx groups the writes that must not half-apply. Nil runs each on its own.
	Tx            Tx
	Bus           EventBus
	Log           *slog.Logger
	InvitationTTL time.Duration
	// Now is injected by tests so expiry is deterministic.
	Now func() time.Time
}

func New(opts Options) *Handler {
	h := &Handler{
		q:             opts.Queries,
		tx:            opts.Tx,
		bus:           opts.Bus,
		log:           opts.Log,
		invitationTTL: opts.InvitationTTL,
		now:           opts.Now,
	}
	if h.log == nil {
		h.log = slog.Default()
	}
	if h.invitationTTL == 0 {
		h.invitationTTL = 7 * 24 * time.Hour
	}
	if h.now == nil {
		h.now = time.Now
	}
	if h.tx == nil {
		h.tx = withoutTx{h.q}
	}
	if h.bus == nil {
		h.bus = noopBus{}
	}
	return h
}

func (h *Handler) CreateFamily(
	ctx context.Context, req *connect.Request[familyv1.CreateFamilyRequest],
) (*connect.Response[familyv1.CreateFamilyResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}
	name, err := requiredName(req.Msg.GetName())
	if err != nil {
		return nil, err
	}

	// One family per user: creating a second would orphan the first family's ledger.
	if _, err := h.q.GetMembership(ctx, pgconv.MustUUID(claims.UserID)); err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("already in a family"))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, h.internal(ctx, err, "check membership")
	}

	ownerID, err := pgconv.UUID(claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad user id: %w", err))
	}

	// A household with no members is not a household. Written separately, a failure on the
	// second statement left the family row behind with nobody in it: invisible to its owner,
	// who has no membership to find it through, and permanent, because nothing deletes it.
	var (
		fam    db.Family
		member db.FamilyMember
	)
	if err := h.tx.InTx(ctx, func(q db.Querier) error {
		var err error
		fam, err = q.CreateFamily(ctx, db.CreateFamilyParams{Name: name, OwnerUserID: ownerID})
		if err != nil {
			return h.internal(ctx, err, "create family")
		}
		member, err = q.AddMember(ctx, db.AddMemberParams{
			FamilyID:    fam.ID,
			UserID:      ownerID,
			DisplayName: claims.Email,
			Email:       claims.Email,
			Role:        roleAdmin,
		})
		if err != nil {
			return h.internal(ctx, err, "add owner as member")
		}
		return nil
	}); err != nil {
		return nil, err
	}

	h.publishJoined(ctx, fam, member)

	return connect.NewResponse(&familyv1.CreateFamilyResponse{Family: toProtoFamily(fam)}), nil
}

func (h *Handler) GetFamily(
	ctx context.Context, req *connect.Request[familyv1.GetFamilyRequest],
) (*connect.Response[familyv1.GetFamilyResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}

	// An empty family_id means "mine" — the common case, and the only one a client should use.
	familyID := req.Msg.GetFamilyId()
	membership, err := h.membershipOf(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if familyID != "" && familyID != pgconv.UUIDString(membership.FamilyID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("not your family"))
	}

	fam, err := h.q.GetFamily(ctx, membership.FamilyID)
	if err != nil {
		return nil, h.internal(ctx, err, "get family")
	}
	members, err := h.q.ListMembers(ctx, membership.FamilyID)
	if err != nil {
		return nil, h.internal(ctx, err, "list members")
	}

	return connect.NewResponse(&familyv1.GetFamilyResponse{
		Family:  toProtoFamily(fam),
		Members: toProtoMembers(members),
	}), nil
}

func (h *Handler) UpdateFamily(
	ctx context.Context, req *connect.Request[familyv1.UpdateFamilyRequest],
) (*connect.Response[familyv1.UpdateFamilyResponse], error) {
	membership, err := h.requireAdmin(ctx, req.Msg.GetFamilyId())
	if err != nil {
		return nil, err
	}
	name, err := requiredName(req.Msg.GetName())
	if err != nil {
		return nil, err
	}

	fam, err := h.q.UpdateFamily(ctx, db.UpdateFamilyParams{ID: membership.FamilyID, Name: name})
	if err != nil {
		return nil, h.internal(ctx, err, "update family")
	}
	return connect.NewResponse(&familyv1.UpdateFamilyResponse{Family: toProtoFamily(fam)}), nil
}

func (h *Handler) ListMembers(
	ctx context.Context, req *connect.Request[familyv1.ListMembersRequest],
) (*connect.Response[familyv1.ListMembersResponse], error) {
	membership, err := h.requireMember(ctx, req.Msg.GetFamilyId())
	if err != nil {
		return nil, err
	}
	members, err := h.q.ListMembers(ctx, membership.FamilyID)
	if err != nil {
		return nil, h.internal(ctx, err, "list members")
	}
	return connect.NewResponse(&familyv1.ListMembersResponse{Members: toProtoMembers(members)}), nil
}

func (h *Handler) RemoveMember(
	ctx context.Context, req *connect.Request[familyv1.RemoveMemberRequest],
) (*connect.Response[familyv1.RemoveMemberResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}
	membership, err := h.requireAdmin(ctx, req.Msg.GetFamilyId())
	if err != nil {
		return nil, err
	}

	target := req.Msg.GetUserId()
	if target == claims.UserID {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("use LeaveFamily to remove yourself"))
	}

	targetID, err := pgconv.UUID(target)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad user id: %w", err))
	}

	rows, err := h.q.RemoveMember(ctx, db.RemoveMemberParams{
		FamilyID: membership.FamilyID, UserID: targetID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "remove member")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("member not found"))
	}

	h.publish(ctx, events.SubjectFamilyMemberRemoved, &familyv1.MemberRemovedEvent{
		FamilyId:        pgconv.UUIDString(membership.FamilyID),
		UserId:          target,
		RemovedByUserId: claims.UserID,
		OccurredAt:      h.timestamp(),
	})

	return connect.NewResponse(&familyv1.RemoveMemberResponse{}), nil
}

func (h *Handler) LeaveFamily(
	ctx context.Context, req *connect.Request[familyv1.LeaveFamilyRequest],
) (*connect.Response[familyv1.LeaveFamilyResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}
	membership, err := h.requireMember(ctx, req.Msg.GetFamilyId())
	if err != nil {
		return nil, err
	}

	// The last admin cannot walk out and leave the family unadministrable.
	if membership.Role == roleAdmin {
		admins, err := h.q.CountAdmins(ctx, membership.FamilyID)
		if err != nil {
			return nil, h.internal(ctx, err, "count admins")
		}
		if admins <= 1 {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				errors.New("promote another admin before leaving"))
		}
	}

	if _, err := h.q.RemoveMember(ctx, db.RemoveMemberParams{
		FamilyID: membership.FamilyID, UserID: membership.UserID,
	}); err != nil {
		return nil, h.internal(ctx, err, "leave family")
	}

	h.publish(ctx, events.SubjectFamilyMemberRemoved, &familyv1.MemberRemovedEvent{
		FamilyId:        pgconv.UUIDString(membership.FamilyID),
		UserId:          claims.UserID,
		RemovedByUserId: claims.UserID,
		OccurredAt:      h.timestamp(),
	})

	return connect.NewResponse(&familyv1.LeaveFamilyResponse{}), nil
}

func (h *Handler) InviteMember(
	ctx context.Context, req *connect.Request[familyv1.InviteMemberRequest],
) (*connect.Response[familyv1.InviteMemberResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}
	membership, err := h.requireAdmin(ctx, req.Msg.GetFamilyId())
	if err != nil {
		return nil, err
	}

	// Normalised, not merely trimmed. The address stored here is compared — by a person
	// reading the pending-invitation list, and by anyone who later wires this to email — with
	// what services/auth stored at registration, which is lowercased. An invitation to
	// "Ada@Example.com" for an account registered as "ada@example.com" is an invitation to
	// nobody, and it fails silently: the row exists, the list shows it, and it is never matched.
	email := fmauth.NormalizeEmail(req.Msg.GetEmail())
	if !fmauth.LooksLikeEmail(email) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("a valid email is required"))
	}

	token, hash, err := newInvitationToken()
	if err != nil {
		return nil, h.internal(ctx, err, "generate token")
	}

	inv, err := h.q.CreateInvitation(ctx, db.CreateInvitationParams{
		FamilyID:      membership.FamilyID,
		InviterUserID: membership.UserID,
		Email:         email,
		Role:          roleFromProto(req.Msg.GetRole()),
		TokenHash:     hash,
		ExpiresAt:     pgconv.TimestampFrom(h.now().Add(h.invitationTTL)),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create invitation")
	}

	h.publish(ctx, events.SubjectFamilyMemberInvited, &familyv1.MemberInvitedEvent{
		FamilyId:      pgconv.UUIDString(inv.FamilyID),
		InvitationId:  pgconv.UUIDString(inv.ID),
		InviterUserId: claims.UserID,
		Email:         email,
		OccurredAt:    h.timestamp(),
	})

	return connect.NewResponse(&familyv1.InviteMemberResponse{
		Invitation: toProtoInvitation(inv),
		Token:      token,
	}), nil
}

func (h *Handler) AcceptInvitation(
	ctx context.Context, req *connect.Request[familyv1.AcceptInvitationRequest],
) (*connect.Response[familyv1.AcceptInvitationResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}
	token := trimmed(req.Msg.GetToken())
	if token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("token is required"))
	}

	inv, err := h.q.GetInvitationByTokenHash(ctx, hashToken(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("invitation not found"))
		}
		return nil, h.internal(ctx, err, "get invitation")
	}
	if inv.Status != invitationPending {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("invitation is %s", inv.Status))
	}
	if inv.ExpiresAt.Valid && !inv.ExpiresAt.Time.After(h.now()) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("invitation expired"))
	}

	if _, err := h.q.GetMembership(ctx, pgconv.MustUUID(claims.UserID)); err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("already in a family"))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, h.internal(ctx, err, "check membership")
	}

	userID, err := pgconv.UUID(claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad user id: %w", err))
	}

	// Joining and spending the invitation are one act. Separately, a failure on the second
	// left the person in the household with the invitation still pending — a token that has
	// already been used and can be used again, which is precisely what marking it accepted
	// is for.
	var member db.FamilyMember
	if err := h.tx.InTx(ctx, func(q db.Querier) error {
		var err error
		member, err = q.AddMember(ctx, db.AddMemberParams{
			FamilyID:    inv.FamilyID,
			UserID:      userID,
			DisplayName: claims.Email,
			Email:       claims.Email,
			Role:        inv.Role,
		})
		if err != nil {
			return h.internal(ctx, err, "add member")
		}
		if _, err := q.MarkInvitationAccepted(ctx, db.MarkInvitationAcceptedParams{
			ID: inv.ID, AcceptedBy: userID,
		}); err != nil {
			return h.internal(ctx, err, "mark invitation accepted")
		}
		return nil
	}); err != nil {
		return nil, err
	}

	fam, err := h.q.GetFamily(ctx, inv.FamilyID)
	if err != nil {
		return nil, h.internal(ctx, err, "get family")
	}

	h.publishJoined(ctx, fam, member)

	return connect.NewResponse(&familyv1.AcceptInvitationResponse{
		Family: toProtoFamily(fam),
		Member: toProtoMember(member),
	}), nil
}

func (h *Handler) RevokeInvitation(
	ctx context.Context, req *connect.Request[familyv1.RevokeInvitationRequest],
) (*connect.Response[familyv1.RevokeInvitationResponse], error) {
	invID, err := pgconv.UUID(req.Msg.GetInvitationId())
	if err != nil || !invID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invitation_id is required"))
	}

	inv, err := h.q.GetInvitation(ctx, invID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("invitation not found"))
		}
		return nil, h.internal(ctx, err, "get invitation")
	}

	if _, err := h.requireAdmin(ctx, pgconv.UUIDString(inv.FamilyID)); err != nil {
		return nil, err
	}

	rows, err := h.q.MarkInvitationRevoked(ctx, invID)
	if err != nil {
		return nil, h.internal(ctx, err, "revoke invitation")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("invitation is not pending"))
	}
	return connect.NewResponse(&familyv1.RevokeInvitationResponse{}), nil
}

func (h *Handler) ListInvitations(
	ctx context.Context, req *connect.Request[familyv1.ListInvitationsRequest],
) (*connect.Response[familyv1.ListInvitationsResponse], error) {
	membership, err := h.requireAdmin(ctx, req.Msg.GetFamilyId())
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListInvitations(ctx, membership.FamilyID)
	if err != nil {
		return nil, h.internal(ctx, err, "list invitations")
	}

	out := make([]*familyv1.Invitation, 0, len(rows))
	for _, inv := range rows {
		out = append(out, toProtoInvitation(inv))
	}
	return connect.NewResponse(&familyv1.ListInvitationsResponse{Invitations: out}), nil
}

// CheckMembership is called by sibling services over gRPC before they touch
// family-scoped rows. It answers about any user, so it is not restricted to the caller.
func (h *Handler) CheckMembership(
	ctx context.Context, req *connect.Request[familyv1.CheckMembershipRequest],
) (*connect.Response[familyv1.CheckMembershipResponse], error) {
	userID, err := pgconv.UUID(req.Msg.GetUserId())
	if err != nil || !userID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user_id is required"))
	}
	familyID, err := pgconv.UUID(req.Msg.GetFamilyId())
	if err != nil || !familyID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("family_id is required"))
	}

	member, err := h.q.GetMember(ctx, db.GetMemberParams{FamilyID: familyID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return connect.NewResponse(&familyv1.CheckMembershipResponse{IsMember: false}), nil
		}
		return nil, h.internal(ctx, err, "get member")
	}

	return connect.NewResponse(&familyv1.CheckMembershipResponse{
		IsMember: true,
		Role:     roleToProto(member.Role),
	}), nil
}

// GetUserMembership resolves a user's family. services/auth calls it when minting a token,
// so the family_id claim every other service scopes by comes from the service that owns
// membership rather than from anything a client can set.
//
// It takes the user id as an argument rather than from a token, which is why it is served on
// the internal listener only — see cmd/server/main.go.
func (h *Handler) GetUserMembership(
	ctx context.Context, req *connect.Request[familyv1.GetUserMembershipRequest],
) (*connect.Response[familyv1.GetUserMembershipResponse], error) {
	userID, err := pgconv.UUID(req.Msg.GetUserId())
	if err != nil || !userID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user_id is required"))
	}

	member, err := h.q.GetMembership(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Not in a family is a normal first-launch state, not an error: the token is
			// minted without a family_id and the app shows onboarding.
			return connect.NewResponse(&familyv1.GetUserMembershipResponse{InFamily: false}), nil
		}
		return nil, h.internal(ctx, err, "get membership")
	}

	return connect.NewResponse(&familyv1.GetUserMembershipResponse{
		InFamily: true,
		FamilyId: pgconv.UUIDString(member.FamilyID),
		Role:     roleToProto(member.Role),
	}), nil
}

/* ------------------------------------------------------------------ internals */

// membershipOf resolves the caller's single membership row.
func (h *Handler) membershipOf(ctx context.Context, userID string) (db.FamilyMember, error) {
	id, err := pgconv.UUID(userID)
	if err != nil || !id.Valid {
		return db.FamilyMember{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token has no usable subject"))
	}

	member, err := h.q.GetMembership(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.FamilyMember{}, connect.NewError(connect.CodeFailedPrecondition,
				errors.New("caller belongs to no family"))
		}
		return db.FamilyMember{}, h.internal(ctx, err, "get membership")
	}
	return member, nil
}

// requireMember checks the caller is in the family they name (or in any, when they name none).
func (h *Handler) requireMember(ctx context.Context, familyID string) (db.FamilyMember, error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return db.FamilyMember{}, err
	}
	member, err := h.membershipOf(ctx, claims.UserID)
	if err != nil {
		return db.FamilyMember{}, err
	}
	if familyID != "" && familyID != pgconv.UUIDString(member.FamilyID) {
		return db.FamilyMember{}, connect.NewError(connect.CodePermissionDenied,
			errors.New("not your family"))
	}
	return member, nil
}

func (h *Handler) requireAdmin(ctx context.Context, familyID string) (db.FamilyMember, error) {
	member, err := h.requireMember(ctx, familyID)
	if err != nil {
		return db.FamilyMember{}, err
	}
	if member.Role != roleAdmin {
		return db.FamilyMember{}, connect.NewError(connect.CodePermissionDenied,
			errors.New("admin role required"))
	}
	return member, nil
}

func (h *Handler) publishJoined(ctx context.Context, fam db.Family, member db.FamilyMember) {
	h.publish(ctx, events.SubjectFamilyMemberJoined, &familyv1.MemberJoinedEvent{
		FamilyId:   pgconv.UUIDString(fam.ID),
		UserId:     pgconv.UUIDString(member.UserID),
		Role:       roleToProto(member.Role),
		OccurredAt: h.timestamp(),
	})
}

// internal hides driver detail from clients while keeping the cause in the log.
// internal hands the cause to the log and an opaque reference to the caller, so a pgx error
// never becomes part of a response body.
func (h *Handler) internal(ctx context.Context, err error, what string) error {
	return rpc.Internal(ctx, h.log, err, what)
}

// withoutTx is the fallback when no Tx is supplied: each statement commits on its own. It
// keeps a zero-valued Options working and does not pretend to be a transaction.
type withoutTx struct{ q db.Querier }

func (w withoutTx) InTx(_ context.Context, fn func(db.Querier) error) error { return fn(w.q) }

// maxNameRunes bounds a household name. It is a label on a screen — "The Lovelaces", "Home" —
// written by an authenticated member into a TEXT column with no width, and the ceiling that
// matters is the one a header renders under. Runes, not bytes, so a Ukrainian household name
// is not worth half an English one.
const maxNameRunes = 80

func requiredName(raw string) (string, error) {
	name := trimmed(raw)
	if name == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if len([]rune(name)) > maxNameRunes {
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("name must be at most %d characters", maxNameRunes))
	}
	return name, nil
}

// newInvitationToken returns the plaintext token (shown once) and the hash that is stored.
func newInvitationToken() (token, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
