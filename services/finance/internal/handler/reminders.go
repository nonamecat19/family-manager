package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// Reminders are per-user, not per-household: the settings row that creates one is on a
// person's device, and a household-wide reminder would notify people who never asked.

func (h *Handler) ListReminders(
	ctx context.Context, req *connect.Request[financev1.ListRemindersRequest],
) (*connect.Response[financev1.ListRemindersResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListReminders(ctx, db.ListRemindersParams{
		FamilyID: c.familyID, UserID: c.userID,
		IncludeDisabled: req.Msg.GetIncludeDisabled(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list reminders")
	}
	out := make([]*financev1.Reminder, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProtoReminder(r))
	}
	return connect.NewResponse(&financev1.ListRemindersResponse{Reminders: out}), nil
}

// UpsertReminder rather than create+update: a reminder is edited far more often than it is
// created, and the app's editor has no meaningful distinction between the two.
func (h *Handler) UpsertReminder(
	ctx context.Context, req *connect.Request[financev1.UpsertReminderRequest],
) (*connect.Response[financev1.UpsertReminderResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	if err := checkText("title", msg.GetTitle(), maxTitleRunes); err != nil {
		return nil, err
	}

	var dueAt pgtype.Timestamptz
	if msg.GetDueAt() != nil {
		dueAt = pgtype.Timestamptz{Time: msg.GetDueAt().AsTime(), Valid: true}
	}
	repeat := msg.GetRepeat()
	unit := recurrenceUnitFromProto(repeat.GetUnit())
	interval := repeat.GetInterval()
	if interval < 0 {
		return nil, invalid("repeat.interval must not be negative")
	}
	// A repeat needs both halves or neither: an interval with no unit is not a schedule, and
	// the CHECK constraint would reject the pair anyway.
	if unit == "" {
		interval = 0
	}

	if id := trimmed(msg.GetReminderId()); id != "" {
		reminderID, err := requireUUID("reminder_id", id)
		if err != nil {
			return nil, err
		}
		row, err := h.q.UpdateReminder(ctx, db.UpdateReminderParams{
			ID: reminderID, FamilyID: c.familyID, UserID: c.userID,
			Kind: reminderKindFromProto(msg.GetKind()), Title: trimmed(msg.GetTitle()),
			DueAt: dueAt, RepeatInterval: interval, RepeatUnit: unit, Enabled: msg.GetEnabled(),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound("reminder")
		}
		if err != nil {
			return nil, h.internal(ctx, err, "update reminder")
		}
		return connect.NewResponse(&financev1.UpsertReminderResponse{
			Reminder: toProtoReminder(row),
		}), nil
	}

	row, err := h.q.CreateReminder(ctx, db.CreateReminderParams{
		FamilyID: c.familyID, UserID: c.userID,
		Kind: reminderKindFromProto(msg.GetKind()), Title: trimmed(msg.GetTitle()),
		DueAt: dueAt, RepeatInterval: interval, RepeatUnit: unit, Enabled: msg.GetEnabled(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create reminder")
	}
	return connect.NewResponse(&financev1.UpsertReminderResponse{
		Reminder: toProtoReminder(row),
	}), nil
}

func (h *Handler) DeleteReminder(
	ctx context.Context, req *connect.Request[financev1.DeleteReminderRequest],
) (*connect.Response[financev1.DeleteReminderResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("reminder_id", req.Msg.GetReminderId())
	if err != nil {
		return nil, err
	}
	rows, err := h.q.DeleteReminder(ctx, db.DeleteReminderParams{
		ID: id, FamilyID: c.familyID, UserID: c.userID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "delete reminder")
	}
	if rows == 0 {
		return nil, notFound("reminder")
	}
	return connect.NewResponse(&financev1.DeleteReminderResponse{}), nil
}
