// Package projection maintains finance's read-models of facts other services own.
//
// finance_members is the only one: the household's roster belongs to services/family, and
// every screen in the app draws it — the member switcher, the "who paid" avatar on a feed row,
// the split bar. Rather than call family on every read, finance keeps a projection and updates
// it when family says the roster changed.
//
// The projection is deliberately allowed to be thin. `family.member.*` carries the ids and the
// role but not the person's name, so a row created from an event has an empty display name
// until the member signs in and the handler fills it from their own token. A missing name
// renders as an empty chip; a missing ROW renders as a member who does not exist, which is the
// failure worth avoiding.
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

// Bus is the half of events.Bus this package uses, so a test can substitute it.
type Bus interface {
	Subscribe(ctx context.Context, subject events.Subject, durable string, h events.Handler) (func(), error)
}

// Members projects `family.member.*` onto finance_members.
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

// Subscribe attaches the two consumers and returns a function that stops both. A failure to
// attach is returned rather than logged away: a service that silently never projects looks
// exactly like a household with no members.
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

// Handle is the events.Handler for both subjects. Returning an error nacks the message, so a
// database outage means redelivery rather than a member the projection never learns about.
func (m *Members) Handle(ctx context.Context, subject events.Subject, payload []byte) error {
	switch subject {
	case events.SubjectFamilyMemberJoined:
		var ev familyv1.MemberJoinedEvent
		if err := proto.Unmarshal(payload, &ev); err != nil {
			// A payload this service cannot parse will not parse on redelivery either, so it
			// is acked and recorded instead of poisoning the consumer.
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

	// The existing row wins on the fields the event does not carry: a re-delivered join must
	// not blank a display name the member already filled in by signing in.
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
		// The colour is assigned once, from the member's position in the household, so the
		// same person keeps the same purple on every screen and across devices.
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

// removed deletes the row. finance keeps the member's TRANSACTIONS — the household's history
// is not rewritten because someone left — and those rows carry the member id, so the feed
// falls back to an unnamed avatar rather than losing the spending.
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

// avatarSteps is the accent ramp's width: the projection assigns a colour by position, and the
// design's ramp has eight steps.
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
