package projection

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"

	"github.com/nnc/family-manager/libs/go/events"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

type Bus interface {
	Subscribe(ctx context.Context, subject events.Subject, durable string, h events.Handler) (func(), error)
}

type Members struct {
	q   db.Querier
	log *slog.Logger
}

func NewMembers(q db.Querier, log *slog.Logger) *Members {
	if log == nil {
		log = slog.Default()
	}
	return &Members{q: q, log: log}
}

func (m *Members) Subscribe(ctx context.Context, bus Bus) (func(), error) {
	joined, err := bus.Subscribe(ctx, events.SubjectFamilyMemberJoined, "finance-members-joined", m.Handle)
	if err != nil {
		return nil, err
	}
	removed, err := bus.Subscribe(ctx, events.SubjectFamilyMemberRemoved, "finance-members-removed", m.Handle)
	if err != nil {
		joined()
		return nil, err
	}
	return func() {
		joined()
		removed()
	}, nil
}

func (m *Members) Handle(ctx context.Context, subject events.Subject, payload []byte) error {
	switch subject {
	case events.SubjectFamilyMemberJoined:
		var ev familyv1.MemberJoinedEvent
		if err := proto.Unmarshal(payload, &ev); err != nil {
			m.log.ErrorContext(ctx, "member joined: bad payload", slog.String("error", err.Error()))
			return nil
		}
		return m.joined(ctx, &ev)
	case events.SubjectFamilyMemberRemoved:
		var ev familyv1.MemberRemovedEvent
		if err := proto.Unmarshal(payload, &ev); err != nil {
			m.log.ErrorContext(ctx, "member removed: bad payload", slog.String("error", err.Error()))
			return nil
		}
		return m.removed(ctx, &ev)
	default:
		return nil
	}
}

func (m *Members) joined(ctx context.Context, ev *familyv1.MemberJoinedEvent) error {
	familyID, userID, ok := ids(ev.GetFamilyId(), ev.GetUserId())
	if !ok {
		m.log.ErrorContext(ctx, "member joined: ids are not uuids",
			slog.String("family_id", ev.GetFamilyId()), slog.String("user_id", ev.GetUserId()))
		return nil
	}

	existing, err := m.q.GetMember(ctx, db.GetMemberParams{FamilyID: familyID, UserID: userID})
	params := db.UpsertMemberParams{
		FamilyID: familyID, UserID: userID,
		Role:   roleFromProto(ev.GetRole()),
		Status: "active",
	}
	if err == nil {
		params.DisplayName = existing.DisplayName
		params.Initial = existing.Initial
		params.AvatarColorStep = existing.AvatarColorStep
		params.Email = existing.Email
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("get member: %w", err)
	} else {
		count, cerr := m.q.CountMembers(ctx, familyID)
		if cerr != nil {
			return fmt.Errorf("count members: %w", cerr)
		}
		params.AvatarColorStep = int32(count % avatarSteps)
	}

	if _, err := m.q.UpsertMember(ctx, params); err != nil {
		return fmt.Errorf("upsert member: %w", err)
	}
	m.log.InfoContext(ctx, "member projected", slog.String("user_id", ev.GetUserId()))
	return nil
}

func (m *Members) removed(ctx context.Context, ev *familyv1.MemberRemovedEvent) error {
	familyID, userID, ok := ids(ev.GetFamilyId(), ev.GetUserId())
	if !ok {
		m.log.ErrorContext(ctx, "member removed: ids are not uuids",
			slog.String("family_id", ev.GetFamilyId()), slog.String("user_id", ev.GetUserId()))
		return nil
	}
	if _, err := m.q.DeleteMember(ctx, db.DeleteMemberParams{FamilyID: familyID, UserID: userID}); err != nil {
		return fmt.Errorf("delete member: %w", err)
	}
	return nil
}

func roleFromProto(role familyv1.Role) string {
	if role == familyv1.Role_ROLE_ADMIN {
		return "owner"
	}
	return "member"
}

const avatarSteps = 8

func ids(familyID, userID string) (pgtype.UUID, pgtype.UUID, bool) {
	f, err := pgconv.UUID(familyID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, false
	}
	u, err := pgconv.UUID(userID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, false
	}
	return f, u, true
}
