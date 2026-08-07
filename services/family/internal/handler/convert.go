package handler

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/services/family/db"
)

// Role and status values as stored. The CHECK constraints in 000001_init.up.sql are the
// other half of this contract.
const (
	roleAdmin  = "admin"
	roleMember = "member"

	invitationPending = "pending"
)

// EventBus is the slice of libs/go/events this service uses. Narrow on purpose: tests pass a
// recorder instead of standing up NATS.
type EventBus interface {
	Publish(ctx context.Context, subject events.Subject, msg proto.Message) error
}

// noopBus lets the service run (and tests pass) with no broker attached.
type noopBus struct{}

func (noopBus) Publish(context.Context, events.Subject, proto.Message) error { return nil }

// publish is fire-and-forget by design: a family change that succeeded must not be reported
// as failed because the broker hiccuped. The failure is logged, not returned.
func (h *Handler) publish(ctx context.Context, subject events.Subject, msg proto.Message) {
	if err := h.bus.Publish(ctx, subject, msg); err != nil {
		h.log.WarnContext(ctx, "publish failed", slog.String("subject", string(subject)),
			slog.String("error", err.Error()))
	}
}

func (h *Handler) timestamp() *timestamppb.Timestamp {
	return timestamppb.New(h.now())
}

func trimmed(s string) string { return strings.TrimSpace(s) }

func roleToProto(role string) familyv1.Role {
	switch role {
	case roleAdmin:
		return familyv1.Role_ROLE_ADMIN
	case roleMember:
		return familyv1.Role_ROLE_MEMBER
	default:
		return familyv1.Role_ROLE_UNSPECIFIED
	}
}

// roleFromProto defaults to member: an unspecified role must never silently become admin.
func roleFromProto(role familyv1.Role) string {
	if role == familyv1.Role_ROLE_ADMIN {
		return roleAdmin
	}
	return roleMember
}

func statusToProto(status string) familyv1.InvitationStatus {
	switch status {
	case "pending":
		return familyv1.InvitationStatus_INVITATION_STATUS_PENDING
	case "accepted":
		return familyv1.InvitationStatus_INVITATION_STATUS_ACCEPTED
	case "revoked":
		return familyv1.InvitationStatus_INVITATION_STATUS_REVOKED
	case "expired":
		return familyv1.InvitationStatus_INVITATION_STATUS_EXPIRED
	default:
		return familyv1.InvitationStatus_INVITATION_STATUS_UNSPECIFIED
	}
}

func toProtoFamily(f db.Family) *familyv1.Family {
	return &familyv1.Family{
		Id:          pgconv.UUIDString(f.ID),
		Name:        f.Name,
		OwnerUserId: pgconv.UUIDString(f.OwnerUserID),
		CreatedAt:   pgconv.Timestamp(f.CreatedAt),
		UpdatedAt:   pgconv.Timestamp(f.UpdatedAt),
	}
}

func toProtoMember(m db.FamilyMember) *familyv1.Member {
	return &familyv1.Member{
		UserId:      pgconv.UUIDString(m.UserID),
		FamilyId:    pgconv.UUIDString(m.FamilyID),
		DisplayName: m.DisplayName,
		Email:       m.Email,
		Role:        roleToProto(m.Role),
		JoinedAt:    pgconv.Timestamp(m.JoinedAt),
	}
}

func toProtoMembers(rows []db.FamilyMember) []*familyv1.Member {
	out := make([]*familyv1.Member, 0, len(rows))
	for _, m := range rows {
		out = append(out, toProtoMember(m))
	}
	return out
}

// toProtoInvitation deliberately omits token_hash: the hash never leaves the database.
func toProtoInvitation(inv db.FamilyInvitation) *familyv1.Invitation {
	return &familyv1.Invitation{
		Id:            pgconv.UUIDString(inv.ID),
		FamilyId:      pgconv.UUIDString(inv.FamilyID),
		InviterUserId: pgconv.UUIDString(inv.InviterUserID),
		Email:         inv.Email,
		Status:        statusToProto(inv.Status),
		ExpiresAt:     pgconv.Timestamp(inv.ExpiresAt),
		CreatedAt:     pgconv.Timestamp(inv.CreatedAt),
	}
}
