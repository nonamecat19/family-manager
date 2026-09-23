package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/schedule"
)

func (h *Handler) ListMyDueReminders(
	ctx context.Context, req *connect.Request[tasksv1.ListMyDueRemindersRequest],
) (*connect.Response[tasksv1.ListMyDueRemindersResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	now := h.now()
	rows, err := h.q.ListDueRemindersForUser(ctx, db.ListDueRemindersForUserParams{
		FamilyID: c.familyID,
		UserID:   c.userID,
		Now:      pgTimestamptz(now),
		Channel:  "push",
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list due reminders")
	}

	reminders := make([]*tasksv1.Reminder, 0, len(rows))
	for _, r := range rows {
		rem, err := h.toProtoReminder(ctx, c, r, now)
		if err != nil {
			h.log.WarnContext(ctx, "convert reminder", slog.String("error", err.Error()))
			continue
		}
		reminders = append(reminders, rem)
	}

	return connect.NewResponse(&tasksv1.ListMyDueRemindersResponse{Reminders: reminders}), nil
}

func (h *Handler) AckReminder(
	ctx context.Context, req *connect.Request[tasksv1.AckReminderRequest],
) (*connect.Response[tasksv1.AckReminderResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	reminderID, err := requireUUID("reminder_id", req.Msg.GetReminderId())
	if err != nil {
		return nil, err
	}

	rows, err := h.q.AckReminder(ctx, db.AckReminderParams{
		ID: reminderID, FamilyID: c.familyID, UserID: c.userID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "ack reminder")
	}
	if rows == 0 {
		return nil, notFound("reminder")
	}

	return connect.NewResponse(&tasksv1.AckReminderResponse{}), nil
}

func (h *Handler) SnoozeReminder(
	ctx context.Context, req *connect.Request[tasksv1.SnoozeReminderRequest],
) (*connect.Response[tasksv1.SnoozeReminderResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	reminderID, err := requireUUID("reminder_id", req.Msg.GetReminderId())
	if err != nil {
		return nil, err
	}

	option := req.Msg.GetOption()
	var snoozeUntil time.Time
	now := h.now()
	switch option {
	case tasksv1.SnoozeOption_SNOOZE_OPTION_ONE_HOUR:
		snoozeUntil = now.Add(time.Hour)
	case tasksv1.SnoozeOption_SNOOZE_OPTION_TOMORROW_MORNING:
		tomorrow := now.AddDate(0, 0, 1)
		snoozeUntil = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, time.UTC)
	default:
		return nil, invalid("invalid snooze option")
	}

	reminder, err := h.q.SnoozeReminder(ctx, db.SnoozeReminderParams{
		ID: reminderID, FamilyID: c.familyID, UserID: c.userID, RemindAt: pgTimestamptz(snoozeUntil),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("reminder")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "snooze reminder")
	}

	return connect.NewResponse(&tasksv1.SnoozeReminderResponse{
		RemindAt: timestamppb.New(reminder.RemindAt.Time),
	}), nil
}

func (h *Handler) GetDigest(
	ctx context.Context, req *connect.Request[tasksv1.GetDigestRequest],
) (*connect.Response[tasksv1.GetDigestResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	now, loc := h.now(), hh.loc
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	dateStr := today.Format(schedule.DateLayout)

	_, err = h.q.GetDigestSent(ctx, db.GetDigestSentParams{
		FamilyID: c.familyID, UserID: c.userID, SentOn: pgDate(today),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "check digest sent")
	}

	tasksRows, err := h.q.ListTasks(ctx, db.ListTasksParams{
		FamilyID: c.familyID, Status: "open", Assignee: c.userID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list tasks for digest")
	}

	var todayTasks []*tasksv1.Task
	var overdueTasks []*tasksv1.Task
	assignees, err := assigneesOf(ctx, h.q, c, taskIDs(tasksRows))
	if err != nil {
		return nil, h.internal(ctx, err, "list assignees")
	}

	for _, t := range tasksRows {
		pb := toProtoTask(t, assignees[id(t.ID)], now, loc)
		dl := deadlineOf(t)
		if !dl.IsZero() && dl.Overdue(now, loc) {
			overdueTasks = append(overdueTasks, pb)
		} else if dl.On == dateStr {
			todayTasks = append(todayTasks, pb)
		}
	}

	birthdaysRows, err := h.q.ListBirthdays(ctx, c.familyID)
	if err != nil {
		return nil, h.internal(ctx, err, "list birthdays for digest")
	}

	var upcomingBirthdays []*tasksv1.Birthday
	for _, b := range birthdaysRows {
		occ := schedule.NextOccurrence(int(b.Day), int(b.Month), birthYear(b), now.In(loc))
		if occ.DaysUntil >= 0 && occ.DaysUntil <= 7 {
			upcomingBirthdays = append(upcomingBirthdays, toProtoBirthday(b, now, loc))
		}
	}

	due := len(todayTasks) > 0 || len(overdueTasks) > 0 || len(upcomingBirthdays) > 0

	return connect.NewResponse(&tasksv1.GetDigestResponse{
		Date:                dateStr,
		Due:                 due,
		Today:               todayTasks,
		Overdue:             overdueTasks,
		UpcomingBirthdays:   upcomingBirthdays,
	}), nil
}

func (h *Handler) MarkDigestSent(
	ctx context.Context, req *connect.Request[tasksv1.MarkDigestSentRequest],
) (*connect.Response[tasksv1.MarkDigestSentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	dateStr := req.Msg.GetDate()
	if dateStr == "" {
		return nil, invalid("date is required")
	}
	date, err := time.Parse(schedule.DateLayout, dateStr)
	if err != nil {
		return nil, invalid("invalid date format: %v", err)
	}

	if err := h.q.MarkDigestSent(ctx, db.MarkDigestSentParams{
		FamilyID: c.familyID, UserID: c.userID, SentOn: pgDate(date),
	}); err != nil {
		return nil, h.internal(ctx, err, "mark digest sent")
	}

	return connect.NewResponse(&tasksv1.MarkDigestSentResponse{}), nil
}

func (h *Handler) toProtoReminder(ctx context.Context, c caller, r db.TaskReminder, now time.Time) (*tasksv1.Reminder, error) {
	var kind tasksv1.ReminderKind
	switch r.Kind {
	case kindTaskDue:
		kind = tasksv1.ReminderKind_REMINDER_KIND_TASK_DUE
	case kindBirthday:
		kind = tasksv1.ReminderKind_REMINDER_KIND_BIRTHDAY
	default:
		return nil, errors.New("unknown reminder kind")
	}

	var task *tasksv1.Task
	var birthday *tasksv1.Birthday

	if r.Kind == kindTaskDue {
		t, err := h.q.GetTask(ctx, db.GetTaskParams{ID: r.ItemID, FamilyID: r.FamilyID})
		if err == nil {
			assignees, err := assigneesOf(ctx, h.q, c, []pgtype.UUID{r.ItemID})
			if err == nil {
				task = toProtoTask(t, assignees[id(r.ItemID)], now, h.familyLoc(ctx, c))
			}
		}
	} else if r.Kind == kindBirthday {
		b, err := h.q.GetBirthday(ctx, db.GetBirthdayParams{ID: r.ItemID, FamilyID: r.FamilyID})
		if err == nil {
			birthday = toProtoBirthday(b, now, h.familyLoc(ctx, c))
		}
	}

	return &tasksv1.Reminder{
		Id:        id(r.ID),
		Kind:      kind,
		Task:      task,
		Birthday:  birthday,
		RemindAt:  timestamppb.New(r.RemindAt.Time),
	}, nil
}