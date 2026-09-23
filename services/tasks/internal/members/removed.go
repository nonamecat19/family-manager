package members

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/services/tasks/db"
)

const (
	claimTimeout = 60 * time.Second
)

var (
	staleAfter   = float64(claimTimeout.Seconds())
	errInFlight  = errors.New("event is being handled by another delivery")
	statusDone   = "done"
)

type Bus interface {
	SubscribeWith(ctx context.Context, subject events.Subject, durable string, h events.Handler, opts events.SubscribeOptions) (func(), error)
}

type Querier interface {
	ClaimEvent(ctx context.Context, arg db.ClaimEventParams) (int64, error)
	EventStatus(ctx context.Context, eventID string) (string, error)
	CompleteEvent(ctx context.Context, eventID string) error
	ReleaseEvent(ctx context.Context, eventID string) error
	DeleteKnownMember(ctx context.Context, arg db.DeleteKnownMemberParams) error
	DeleteTaskAssigneesForUser(ctx context.Context, arg db.DeleteTaskAssigneesForUserParams) error
	DeletePendingRemindersForUser(ctx context.Context, arg db.DeletePendingRemindersForUserParams) error
}

type MemberRemoved struct {
	q   Querier
	log *slog.Logger
}

func New(q Querier, log *slog.Logger) *MemberRemoved {
	return &MemberRemoved{q: q, log: log}
}

func (m *MemberRemoved) Subscribe(ctx context.Context, bus Bus) (func(), error) {
	stop, err := bus.SubscribeWith(ctx, events.SubjectFamilyMemberRemoved, "tasks-member-removed", m.Handle,
		events.SubscribeOptions{Heartbeat: 10 * time.Second})
	if err != nil {
		return nil, err
	}
	return stop, nil
}

func (m *MemberRemoved) Handle(ctx context.Context, subject events.Subject, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var ev familyv1.MemberRemovedEvent
	if err := proto.Unmarshal(payload, &ev); err != nil {
		m.log.ErrorContext(ctx, "bad event payload",
			slog.String("subject", string(subject)), slog.String("error", err.Error()))
		return nil
	}

	familyID, userID, ok := uuids(ev.GetFamilyId(), ev.GetUserId())
	if !ok {
		m.log.ErrorContext(ctx, "member removed: ids are not uuids")
		return nil
	}

	id := eventID(subject, payload)
	c, err := m.claim(ctx, id)
	if err != nil || c == nil {
		return err
	}

	err = m.q.DeleteKnownMember(ctx, db.DeleteKnownMemberParams{FamilyID: familyID, UserID: userID})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		c.release(ctx)
		return fmt.Errorf("delete known member: %w", err)
	}

	err = m.q.DeleteTaskAssigneesForUser(ctx, db.DeleteTaskAssigneesForUserParams{FamilyID: familyID, UserID: userID})
	if err != nil {
		c.release(ctx)
		return fmt.Errorf("delete task assignees: %w", err)
	}

	err = m.q.DeletePendingRemindersForUser(ctx, db.DeletePendingRemindersForUserParams{FamilyID: familyID, UserID: userID})
	if err != nil {
		c.release(ctx)
		return fmt.Errorf("delete pending reminders: %w", err)
	}

	return c.complete(ctx)
}

func (m *MemberRemoved) claim(ctx context.Context, id string) (*claim, error) {
	rows, err := m.q.ClaimEvent(ctx, db.ClaimEventParams{
		EventID:           id,
		Subject:           string(events.SubjectFamilyMemberRemoved),
		StaleAfterSeconds: staleAfter,
	})
	if err != nil {
		return nil, fmt.Errorf("claim event: %w", err)
	}
	if rows > 0 {
		return &claim{m: m, id: id}, nil
	}
	status, err := m.q.EventStatus(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errInFlight
	}
	if err != nil {
		return nil, fmt.Errorf("event status: %w", err)
	}
	if status == statusDone {
		return nil, nil
	}
	return nil, errInFlight
}

type claim struct {
	m        *MemberRemoved
	id       string
	accepted bool
	completed bool
}

func (c *claim) accept(ctx context.Context) {
	c.accepted = true
	if err := c.complete(ctx); err != nil {
		c.m.log.ErrorContext(ctx, "could not record an accepted delivery",
			slog.String("event_id", c.id), slog.String("error", err.Error()))
	}
}

func (c *claim) complete(ctx context.Context) error {
	if c.completed {
		return nil
	}
	if err := c.m.q.CompleteEvent(context.WithoutCancel(ctx), c.id); err != nil {
		return fmt.Errorf("complete event: %w", err)
	}
	c.completed = true
	return nil
}

func (c *claim) release(ctx context.Context) {
	if err := c.m.q.ReleaseEvent(context.WithoutCancel(ctx), c.id); err != nil {
		c.m.log.WarnContext(ctx, "could not release an event claim; it expires on its own",
			slog.String("event_id", c.id), slog.String("error", err.Error()))
	}
}

func eventID(subject events.Subject, payload []byte) string {
	sum := sha256.New()
	sum.Write([]byte(subject))
	sum.Write([]byte{0})
	sum.Write(payload)
	return hex.EncodeToString(sum.Sum(nil))
}

func uuids(familyID, userID string) (pgtype.UUID, pgtype.UUID, bool) {
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