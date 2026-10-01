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

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/schedule"
)

func (h *Handler) ListTasks(
	ctx context.Context, req *connect.Request[tasksv1.ListTasksRequest],
) (*connect.Response[tasksv1.ListTasksResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	filter := req.Msg.GetFilter()
	var status string
	var assignee pgtype.UUID
	switch filter {
	case tasksv1.TaskFilter_TASK_FILTER_OPEN:
		status = "open"
	case tasksv1.TaskFilter_TASK_FILTER_MINE:
		assignee = c.userID
	case tasksv1.TaskFilter_TASK_FILTER_DONE:
		status = "done"
	case tasksv1.TaskFilter_TASK_FILTER_ALL:
	default:
		return nil, invalid("unknown task filter: %v", filter)
	}

	rows, err := h.q.ListTasks(ctx, db.ListTasksParams{
		FamilyID: c.familyID,
		Status:   status,
		Assignee: assignee,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list tasks")
	}

	now := h.now()
	loc, _ := time.LoadLocation("UTC")
	if hh, err := h.household(ctx, c); err == nil {
		loc = hh.loc
	}

	tasks := make([]*tasksv1.Task, 0, len(rows))
	for _, t := range rows {
		tasks = append(tasks, h.toProtoTask(ctx, t, now, loc))
	}
	return connect.NewResponse(&tasksv1.ListTasksResponse{Tasks: tasks}), nil
}

func (h *Handler) GetTask(
	ctx context.Context, req *connect.Request[tasksv1.GetTaskRequest],
) (*connect.Response[tasksv1.GetTaskResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	taskID, err := requireUUID("task_id", req.Msg.GetTaskId())
	if err != nil {
		return nil, err
	}

	task, err := h.q.GetTask(ctx, db.GetTaskParams{ID: taskID, FamilyID: c.familyID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("task")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get task")
	}

	now := h.now()
	loc, _ := time.LoadLocation("UTC")
	if hh, err := h.household(ctx, c); err == nil {
		loc = hh.loc
	}

	return connect.NewResponse(&tasksv1.GetTaskResponse{Task: h.toProtoTask(ctx, task, now, loc)}), nil
}

func (h *Handler) CreateTask(
	ctx context.Context, req *connect.Request[tasksv1.CreateTaskRequest],
) (*connect.Response[tasksv1.CreateTaskResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	msg := req.Msg
	if err := checkText("title", msg.GetTitle(), maxTitleRunes); err != nil {
		return nil, err
	}
	if err := checkText("notes", msg.GetNotes(), maxNotesRunes); err != nil {
		return nil, err
	}

	priority := protoPriority(msg.GetPriority())
	if priority == "" {
		priority = "medium"
	}

	dl, err := schedule.ParseDeadline(msg.GetDueOn(), msg.GetDueTime())
	if err != nil {
		return nil, invalid("%v", err)
	}

	var dueOn pgtype.Date
	var dueTime pgtype.Time
	var dueAt pgtype.Timestamptz
	if !dl.IsZero() {
		dueOn = pgtype.Date{Time: parseDate(dl.On), Valid: true}
		if dl.HasTime() {
			dueTime = pgTime(parseTime(dl.Time))
		}
	}

	var task db.Task
	now := h.now()
	loc, _ := time.LoadLocation("UTC")
	if hh, err := h.household(ctx, c); err == nil {
		loc = hh.loc
	}
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		var innerErr error
		task, innerErr = q.CreateTask(ctx, db.CreateTaskParams{
			FamilyID:        c.familyID,
			Title:           msg.GetTitle(),
			Notes:           msg.GetNotes(),
			Priority:        priority,
			DueOn:           dueOn,
			DueTime:         dueTime,
			DueAt:           dueAt,
			CreatedByUserID: c.userID,
		})
		if innerErr != nil {
			return innerErr
		}

		assignees, innerErr := uuidList("assignee_user_ids", msg.GetAssigneeUserIds())
		if innerErr != nil {
			return innerErr
		}
		if len(assignees) == 0 {
			assignees = []pgtype.UUID{c.userID}
		}
		for _, a := range assignees {
			if innerErr = q.AddAssignee(ctx, db.AddAssigneeParams{
				TaskID: task.ID, FamilyID: c.familyID, UserID: a,
			}); innerErr != nil {
				return innerErr
			}
		}

		if !dueAt.Time.IsZero() {
			for _, a := range assignees {
				remindAt := schedule.TaskRemindAt(dl, loc)
				if !remindAt.IsZero() {
					occ := dueAt.Time.Format(time.RFC3339)
					if innerErr = q.UpsertReminder(ctx, db.UpsertReminderParams{
						FamilyID:   c.familyID,
						UserID:     a,
						Kind:       "task_due",
						ItemID:     task.ID,
						Occurrence: occ,
						RemindAt:   pgTimestamptz(remindAt),
					}); innerErr != nil {
						return innerErr
					}
				}
			}
		}

		if h.bus != nil {
			assignedBy := c.user
			for _, a := range assignees {
				if a == c.userID {
					continue
				}
				if innerErr = h.bus.Publish(ctx, events.SubjectTasksTaskAssigned, &tasksv1.TaskAssignedEvent{
					FamilyId:           c.family,
					TaskId:             pgconv.UUIDString(task.ID),
					Title:              task.Title,
					AssigneeUserIds:    []string{pgconv.UUIDString(a)},
					AssignedByUserId:   assignedBy,
					OccurredAt:         timestamppb.Now(),
				}); innerErr != nil {
					h.log.WarnContext(ctx, "publish task assigned", slog.String("error", innerErr.Error()))
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create task")
	}

	return connect.NewResponse(&tasksv1.CreateTaskResponse{Task: h.toProtoTask(ctx, task, now, loc)}), nil
}

func (h *Handler) UpdateTask(
	ctx context.Context, req *connect.Request[tasksv1.UpdateTaskRequest],
) (*connect.Response[tasksv1.UpdateTaskResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	msg := req.Msg
	taskID, err := requireUUID("task_id", msg.GetTaskId())
	if err != nil {
		return nil, err
	}

	var task db.Task
	now := h.now()
	loc, _ := time.LoadLocation("UTC")
	if hh, err := h.household(ctx, c); err == nil {
		loc = hh.loc
	}
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		existing, innerErr := q.GetTaskForUpdate(ctx, db.GetTaskForUpdateParams{
			ID: taskID, FamilyID: c.familyID,
		})
		if errors.Is(innerErr, pgx.ErrNoRows) {
			return notFound("task")
		}
		if innerErr != nil {
			return h.internal(ctx, innerErr, "get task for update")
		}

		title := existing.Title
		if msg.Title != nil {
			if err := checkText("title", msg.GetTitle(), maxTitleRunes); err != nil {
				return err
			}
			title = msg.GetTitle()
		}
		notes := existing.Notes
		if msg.Notes != nil {
			if err := checkText("notes", msg.GetNotes(), maxNotesRunes); err != nil {
				return err
			}
			notes = msg.GetNotes()
		}
		priority := existing.Priority
		if msg.Priority != nil {
			priority = protoPriority(msg.GetPriority())
			if priority == "" {
				return invalid("priority is required")
			}
		}

		dl := schedule.Deadline{On: existing.DueOn.Time.Format(schedule.DateLayout)}
		if existing.DueTime.Valid {
			dl.Time = timeFromPgTime(existing.DueTime).Format(schedule.TimeLayout)
		}
		if msg.DueOn != nil {
			dl.On = msg.GetDueOn()
		}
		if msg.DueTime != nil {
			dl.Time = msg.GetDueTime()
		}
		if dl.On == "" && dl.Time != "" {
			return invalid("due_time requires due_on")
		}
		parsedDL, innerErr := schedule.ParseDeadline(dl.On, dl.Time)
		if innerErr != nil {
			return invalid("%v", innerErr)
		}

		var dueOn pgtype.Date
		var dueTime pgtype.Time
		var dueAt pgtype.Timestamptz
		if !parsedDL.IsZero() {
			dueOn = pgtype.Date{Time: parseDate(parsedDL.On), Valid: true}
			if parsedDL.HasTime() {
				dueTime = pgTime(parseTime(parsedDL.Time))
			}
		}

		assigneesChanged := false
		var newAssignees []pgtype.UUID
		if msg.Assignees != nil && msg.Assignees.GetUserIds() != nil {
			assigneesChanged = true
			newAssignees, innerErr = uuidList("assignee_user_ids", msg.GetAssignees().GetUserIds())
			if innerErr != nil {
				return innerErr
			}
		}

		task, innerErr = q.UpdateTask(ctx, db.UpdateTaskParams{
			ID:       taskID,
			FamilyID: c.familyID,
			Title:    title,
			Notes:    notes,
			Priority: priority,
			DueOn:    dueOn,
			DueTime:  dueTime,
			DueAt:    dueAt,
		})
		if innerErr != nil {
			return innerErr
		}

		if assigneesChanged {
			if innerErr = q.DeleteAssignees(ctx, db.DeleteAssigneesParams{
				TaskID: taskID, FamilyID: c.familyID,
			}); innerErr != nil {
				return innerErr
			}
			for _, a := range newAssignees {
				if innerErr = q.AddAssignee(ctx, db.AddAssigneeParams{
					TaskID: taskID, FamilyID: c.familyID, UserID: a,
				}); innerErr != nil {
					return innerErr
				}
			}
		}

		occ := dueAt.Time.Format(time.RFC3339)
		if !dueAt.Time.IsZero() {
			targetAssignees := newAssignees
			if !assigneesChanged {
				rows, innerErr := q.ListAssignees(ctx, db.ListAssigneesParams{
					FamilyID: c.familyID, TaskIds: []pgtype.UUID{taskID},
				})
				if innerErr != nil {
					return innerErr
				}
				targetAssignees = make([]pgtype.UUID, len(rows))
				for i, r := range rows {
					targetAssignees[i] = r.UserID
				}
			}
			for _, a := range targetAssignees {
				remindAt := schedule.TaskRemindAt(parsedDL, loc)
				if !remindAt.IsZero() {
					if innerErr = q.UpsertReminder(ctx, db.UpsertReminderParams{
						FamilyID:   c.familyID,
						UserID:     a,
						Kind:       "task_due",
						ItemID:     taskID,
						Occurrence: occ,
						RemindAt:   pgTimestamptz(remindAt),
					}); innerErr != nil {
						return innerErr
					}
				}
			}
		} else {
			if innerErr = q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
				FamilyID: c.familyID, Kind: "task_due", ItemID: taskID,
			}); innerErr != nil {
				return innerErr
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.UpdateTaskResponse{Task: h.toProtoTask(ctx, task, now, loc)}), nil
}

func (h *Handler) CompleteTask(
	ctx context.Context, req *connect.Request[tasksv1.CompleteTaskRequest],
) (*connect.Response[tasksv1.CompleteTaskResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	taskID, err := requireUUID("task_id", req.Msg.GetTaskId())
	if err != nil {
		return nil, err
	}

	var task db.Task
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		completedAt := h.now()
		t, innerErr := q.CompleteTask(ctx, db.CompleteTaskParams{
			CompletedAt:       pgTimestamptz(completedAt),
			CompletedByUserID: c.userID,
			ID:                taskID,
			FamilyID:          c.familyID,
		})
		if errors.Is(innerErr, pgx.ErrNoRows) {
			return notFound("task")
		}
		if innerErr != nil {
			return h.internal(ctx, innerErr, "complete task")
		}
		task = t

		if innerErr = q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
			FamilyID: c.familyID, Kind: "task_due", ItemID: taskID,
		}); innerErr != nil {
			return innerErr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	now := h.now()
	loc, _ := time.LoadLocation("UTC")
	if hh, err := h.household(ctx, c); err == nil {
		loc = hh.loc
	}

	return connect.NewResponse(&tasksv1.CompleteTaskResponse{Task: h.toProtoTask(ctx, task, now, loc)}), nil
}

func (h *Handler) ReopenTask(
	ctx context.Context, req *connect.Request[tasksv1.ReopenTaskRequest],
) (*connect.Response[tasksv1.ReopenTaskResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	taskID, err := requireUUID("task_id", req.Msg.GetTaskId())
	if err != nil {
		return nil, err
	}

	var task db.Task
	now := h.now()
	loc, _ := time.LoadLocation("UTC")
	if hh, err := h.household(ctx, c); err == nil {
		loc = hh.loc
	}
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		t, innerErr := q.ReopenTask(ctx, db.ReopenTaskParams{
			ID: taskID, FamilyID: c.familyID,
		})
		if errors.Is(innerErr, pgx.ErrNoRows) {
			return notFound("task")
		}
		if innerErr != nil {
			return h.internal(ctx, innerErr, "reopen task")
		}
		task = t

		dl := schedule.Deadline{On: task.DueOn.Time.Format(schedule.DateLayout)}
		if task.DueTime.Valid {
			dl.Time = timeFromPgTime(task.DueTime).Format(schedule.TimeLayout)
		}
		if !dl.IsZero() {
			rows, innerErr := q.ListAssignees(ctx, db.ListAssigneesParams{
				FamilyID: c.familyID, TaskIds: []pgtype.UUID{taskID},
			})
			if innerErr != nil {
				return innerErr
			}
			occ := task.DueAt.Time.Format(time.RFC3339)
			for _, a := range rows {
				remindAt := schedule.TaskRemindAt(dl, loc)
				if !remindAt.IsZero() {
					if innerErr = q.UpsertReminder(ctx, db.UpsertReminderParams{
						FamilyID:   c.familyID,
						UserID:     a.UserID,
						Kind:       "task_due",
						ItemID:     taskID,
						Occurrence: occ,
						RemindAt:   pgTimestamptz(remindAt),
					}); innerErr != nil {
						return innerErr
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.ReopenTaskResponse{Task: h.toProtoTask(ctx, task, now, loc)}), nil
}

func (h *Handler) DeleteTask(
	ctx context.Context, req *connect.Request[tasksv1.DeleteTaskRequest],
) (*connect.Response[tasksv1.DeleteTaskResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	taskID, err := requireUUID("task_id", req.Msg.GetTaskId())
	if err != nil {
		return nil, err
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		rows, innerErr := q.DeleteTask(ctx, db.DeleteTaskParams{
			ID: taskID, FamilyID: c.familyID,
		})
		if innerErr != nil {
			return h.internal(ctx, innerErr, "delete task")
		}
		if rows == 0 {
			return notFound("task")
		}

		if innerErr = q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
			FamilyID: c.familyID, Kind: "task_due", ItemID: taskID,
		}); innerErr != nil {
			return innerErr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.DeleteTaskResponse{}), nil
}

func (h *Handler) toProtoTask(
	ctx context.Context, t db.Task, now time.Time, loc *time.Location,
) *tasksv1.Task {
	dl := schedule.Deadline{On: t.DueOn.Time.Format(schedule.DateLayout)}
	if t.DueTime.Valid {
		dl.Time = timeFromPgTime(t.DueTime).Format(schedule.TimeLayout)
	}
	overdue := dl.Overdue(now, loc)

	assignees := []string{}
	rows, err := h.q.ListAssignees(ctx, db.ListAssigneesParams{
		FamilyID: t.FamilyID, TaskIds: []pgtype.UUID{t.ID},
	})
	if err == nil {
		for _, r := range rows {
			assignees = append(assignees, pgconv.UUIDString(r.UserID))
		}
	}

	return &tasksv1.Task{
		Id:                  pgconv.UUIDString(t.ID),
		FamilyId:            pgconv.UUIDString(t.FamilyID),
		Title:               t.Title,
		Notes:               t.Notes,
		Priority:            protoPriorityFromString(t.Priority),
		Status:              protoStatusFromString(t.Status),
		DueOn:               t.DueOn.Time.Format(schedule.DateLayout),
		DueTime:             timeFromPgTime(t.DueTime).Format(schedule.TimeLayout),
		AssigneeUserIds:     assignees,
		CreatedByUserId:     pgconv.UUIDString(t.CreatedByUserID),
		CompletedByUserId:   pgconv.UUIDString(t.CompletedByUserID),
		DueAt:               timestamppb.New(t.DueAt.Time),
		CompletedAt:         timestamppb.New(t.CompletedAt.Time),
		CreatedAt:           timestamppb.New(t.CreatedAt.Time),
		UpdatedAt:           timestamppb.New(t.UpdatedAt.Time),
		Overdue:             overdue,
	}
}

func protoPriority(p tasksv1.Priority) string {
	switch p {
	case tasksv1.Priority_PRIORITY_LOW:
		return "low"
	case tasksv1.Priority_PRIORITY_MEDIUM:
		return "medium"
	case tasksv1.Priority_PRIORITY_HIGH:
		return "high"
	case tasksv1.Priority_PRIORITY_URGENT:
		return "urgent"
	default:
		return ""
	}
}

func protoPriorityFromString(s string) tasksv1.Priority {
	switch s {
	case "low":
		return tasksv1.Priority_PRIORITY_LOW
	case "medium":
		return tasksv1.Priority_PRIORITY_MEDIUM
	case "high":
		return tasksv1.Priority_PRIORITY_HIGH
	case "urgent":
		return tasksv1.Priority_PRIORITY_URGENT
	default:
		return tasksv1.Priority_PRIORITY_UNSPECIFIED
	}
}

func protoStatusFromString(s string) tasksv1.TaskStatus {
	switch s {
	case "open":
		return tasksv1.TaskStatus_TASK_STATUS_OPEN
	case "done":
		return tasksv1.TaskStatus_TASK_STATUS_DONE
	default:
		return tasksv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

func parseDate(s string) time.Time {
	t, _ := time.Parse(schedule.DateLayout, s)
	return t
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(schedule.TimeLayout, s)
	return t
}

type household struct {
	settings db.FamilySetting
	loc      *time.Location
}

func (h *Handler) household(ctx context.Context, c caller) (household, error) {
	row, err := h.q.GetFamilySettings(ctx, c.familyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return household{}, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("family settings not initialized"))
	}
	if err != nil {
		return household{}, h.internal(ctx, err, "load family settings")
	}
	loc, lerr := time.LoadLocation(row.Timezone)
	if lerr != nil {
		h.log.WarnContext(ctx, "unknown family timezone",
			slog.String("timezone", row.Timezone), slog.String("error", lerr.Error()))
		loc = time.UTC
	}
	return household{settings: row, loc: loc}, nil
}