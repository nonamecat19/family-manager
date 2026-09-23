package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/libs/go/rpc"
	"github.com/nnc/family-manager/sdk/go/tasks/v1/tasksv1connect"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/crypto"
	"github.com/nnc/family-manager/services/tasks/internal/family"
	"github.com/nnc/family-manager/services/tasks/internal/gcal"
)

type Tx interface {
	InTx(ctx context.Context, fn func(q db.Querier) error) error
}

type EventBus interface {
	EnsureStream(ctx context.Context, domain string) error
	Publish(ctx context.Context, subject events.Subject, msg proto.Message) error
}

type CalendarSync interface {
	PushTask(context.Context, pgtype.UUID, pgtype.UUID) error
	PushBirthday(context.Context, pgtype.UUID, pgtype.UUID) error
	MutateTask(context.Context, pgtype.UUID, pgtype.UUID, func() (bool, error)) error
	MutateBirthday(context.Context, pgtype.UUID, pgtype.UUID, func() error) error
	GoogleClient() gcal.Client
	Box() *crypto.Box
	SwitchCalendar(ctx context.Context, familyID, userID pgtype.UUID) error
}

type Handler struct {
	tasksv1connect.UnimplementedTasksServiceHandler
	q         db.Querier
	tx        Tx
	bus       EventBus
	log       *slog.Logger
	now       func() time.Time
	family    *family.Client
	familyPub *family.Client
	calendar  CalendarSync
}

type Options struct {
	Queries      db.Querier
	Tx           Tx
	Bus          EventBus
	Family       *family.Client
	FamilyPublic *family.Client
	Calendar     CalendarSync
	Log          *slog.Logger
	Now          func() time.Time
}

func New(opts Options) *Handler {
	h := &Handler{
		q:         opts.Queries,
		tx:        opts.Tx,
		bus:       opts.Bus,
		log:       opts.Log,
		now:       opts.Now,
		family:    opts.Family,
		familyPub: opts.FamilyPublic,
		calendar:  opts.Calendar,
	}
	if h.log == nil {
		h.log = slog.Default()
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

func (h *Handler) pushTask(ctx context.Context, familyID, taskID pgtype.UUID) {
	if h.calendar == nil {
		return
	}
	if err := h.calendar.PushTask(ctx, familyID, taskID); err != nil {
		h.log.WarnContext(ctx, "push task to calendar", slog.String("error", err.Error()))
	}
}

func (h *Handler) pushBirthday(ctx context.Context, familyID, birthdayID pgtype.UUID) {
	if h.calendar == nil {
		return
	}
	if err := h.calendar.PushBirthday(ctx, familyID, birthdayID); err != nil {
		h.log.WarnContext(ctx, "push birthday to calendar", slog.String("error", err.Error()))
	}
}

func (h *Handler) mutateTask(ctx context.Context, familyID, taskID pgtype.UUID, write func() (bool, error)) error {
	if h.calendar != nil {
		return h.calendar.MutateTask(ctx, familyID, taskID, write)
	}
	_, err := write()
	return err
}

func (h *Handler) mutateBirthday(ctx context.Context, familyID, birthdayID pgtype.UUID, write func() error) error {
	if h.calendar != nil {
		return h.calendar.MutateBirthday(ctx, familyID, birthdayID, write)
	}
	return write()
}

type withoutTx struct{ q db.Querier }

func (w withoutTx) InTx(_ context.Context, fn func(db.Querier) error) error { return fn(w.q) }

type noopBus struct{}

func (noopBus) EnsureStream(ctx context.Context, domain string) error { return nil }
func (noopBus) Publish(ctx context.Context, subject events.Subject, msg proto.Message) error {
	return nil
}

func (h *Handler) internal(ctx context.Context, err error, what string) error {
	return rpc.Internal(ctx, h.log, err, what)
}

type caller struct {
	family   string
	user     string
	email    string
	familyID pgtype.UUID
	userID   pgtype.UUID
}

func (c caller) memberID() pgtype.UUID { return c.userID }

func (h *Handler) caller(ctx context.Context) (caller, error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return caller{}, err
	}
	if claims.UserID == "" {
		return caller{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token has no subject"))
	}
	if claims.FamilyID == "" {
		return caller{}, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("caller belongs to no family"))
	}
	familyID, err := pgconv.UUID(claims.FamilyID)
	if err != nil {
		return caller{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token family_id is not a uuid"))
	}
	userID, err := pgconv.UUID(claims.UserID)
	if err != nil {
		return caller{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token subject is not a uuid"))
	}
	return caller{
		family: claims.FamilyID, user: claims.UserID, email: claims.Email,
		familyID: familyID, userID: userID,
	}, nil
}

func (h *Handler) touchKnownMember(ctx context.Context, c caller) error {
	return h.tx.InTx(ctx, func(q db.Querier) error {
		return q.TouchKnownMember(ctx, db.TouchKnownMemberParams{
			FamilyID: c.familyID,
			UserID:   c.userID,
			Email:    c.email,
		})
	})
}

func invalid(format string, args ...any) error {
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(format, args...))
}

func notFound(what string) error {
	return connect.NewError(connect.CodeNotFound, errors.New(what+" not found"))
}

func requireUUID(field, s string) (pgtype.UUID, error) {
	if strings.TrimSpace(s) == "" {
		return pgtype.UUID{}, invalid("%s is required", field)
	}
	u, err := pgconv.UUID(strings.TrimSpace(s))
	if err != nil {
		return pgtype.UUID{}, invalid("%s is not a uuid: %v", field, err)
	}
	return u, nil
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: startOfDay(t), Valid: true}
}

func pgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

const (
	maxTitleRunes = 200
	maxNotesRunes = 4000
	maxBatchIDs   = 500
)

func checkText(field, value string, max int) error {
	if len([]rune(value)) > max {
		return invalid("%s must be at most %d characters", field, max)
	}
	return nil
}

func uuidList(field string, ids []string) ([]pgtype.UUID, error) {
	out := make([]pgtype.UUID, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, raw := range ids {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		u, err := pgconv.UUID(strings.TrimSpace(raw))
		if err != nil {
			return nil, invalid("%s contains a value that is not a uuid: %v", field, err)
		}
		if seen[pgconv.UUIDString(u)] {
			continue
		}
		seen[pgconv.UUIDString(u)] = true
		out = append(out, u)
	}
	return out, nil
}

func pgTime(t time.Time) pgtype.Time {
	return pgtype.Time{
		Microseconds: int64(t.Hour())*3600*1e6 + int64(t.Minute())*60*1e6 + int64(t.Second())*1e6,
		Valid:        true,
	}
}

func timeFromPgTime(t pgtype.Time) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	us := t.Microseconds
	return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(us) * time.Microsecond)
}
