package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/family"
	"github.com/nnc/family-manager/services/tasks/internal/schedule"
)

const kindTaskDue = "task_due"

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
	if filter == tasksv1.TaskFilter_TASK_FILTER_DONE {
		sort.SliceStable(rows, func(i, j int) bool {
			return rows[i].CompletedAt.Time.After(rows[j].CompletedAt.Time)
		})
	}

	assignees, err := assigneesOf(ctx, h.q, c, taskIDs(rows))
	if err != nil {
		return nil, h.internal(ctx, err, "list assignees")
	}

	now, loc := h.now(), h.familyLoc(ctx, c)
	tasks := make([]*tasksv1.Task, 0, len(rows))
	for _, t := range rows {
		tasks = append(tasks, toProtoTask(t, assignees[id(t.ID)], now, loc))
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
	assignees, err := assigneesOf(ctx, h.q, c, []pgtype.UUID{taskID})
	if err != nil {
		return nil, h.internal(ctx, err, "list assignees")
	}

	now, loc := h.now(), h.familyLoc(ctx, c)
	return connect.NewResponse(&tasksv1.GetTaskResponse{
		Task: toProtoTask(task, assignees[id(taskID)], now, loc),
	}), nil
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
	assignees, err := uuidList("assignee_user_ids", msg.GetAssigneeUserIds())
	if err != nil {
		return nil, err
	}
	if len(assignees) == 0 {
		assignees = []pgtype.UUID{c.userID}
	}

	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	now, loc := h.now(), hh.loc
	dueOn, dueTime, dueAt := deadlineColumns(dl, loc)
	members := h.fetchMembers(ctx, c, req.Header())

	var task db.Task
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if innerErr := refreshKnownMembers(ctx, q, c, members); innerErr != nil {
			return h.internal(ctx, innerErr, "refresh known members")
		}
		if innerErr := h.requireMembers(ctx, q, c, assignees); innerErr != nil {
			return innerErr
		}
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
			return h.internal(ctx, innerErr, "create task")
		}
		for _, a := range assignees {
			if innerErr = q.AddAssignee(ctx, db.AddAssigneeParams{
				TaskID: task.ID, FamilyID: c.familyID, UserID: a,
			}); innerErr != nil {
				return h.internal(ctx, innerErr, "add assignee")
			}
		}
		if innerErr = syncTaskReminders(ctx, q, c, task, assignees, loc); innerErr != nil {
			return h.internal(ctx, innerErr, "schedule task reminders")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.CreateTaskResponse{
		Task: toProtoTask(task, uuidStrings(assignees), now, loc),
	}), nil
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
	if msg.Title != nil {
		if err := checkText("title", msg.GetTitle(), maxTitleRunes); err != nil {
			return nil, err
		}
	}
	if msg.Notes != nil {
		if err := checkText("notes", msg.GetNotes(), maxNotesRunes); err != nil {
			return nil, err
		}
	}
	var newPriority string
	if msg.Priority != nil {
		if newPriority = protoPriority(msg.GetPriority()); newPriority == "" {
			return nil, invalid("priority is required")
		}
	}
	replaceAssignees := msg.Assignees != nil
	var newAssignees []pgtype.UUID
	if replaceAssignees {
		if newAssignees, err = uuidList("assignee_user_ids", msg.GetAssignees().GetUserIds()); err != nil {
			return nil, err
		}
	}

	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	now, loc := h.now(), hh.loc
	var members []family.Member
	if replaceAssignees {
		members = h.fetchMembers(ctx, c, req.Header())
	}

	var task db.Task
	var assignees []pgtype.UUID
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		existing, innerErr := q.GetTaskForUpdate(ctx, db.GetTaskForUpdateParams{ID: taskID, FamilyID: c.familyID})
		if errors.Is(innerErr, pgx.ErrNoRows) {
			return notFound("task")
		}
		if innerErr != nil {
			return h.internal(ctx, innerErr, "get task for update")
		}

		title := existing.Title
		if msg.Title != nil {
			title = msg.GetTitle()
		}
		notes := existing.Notes
		if msg.Notes != nil {
			notes = msg.GetNotes()
		}
		priority := existing.Priority
		if msg.Priority != nil {
			priority = newPriority
		}
		dl, innerErr := mergeDeadline(deadlineOf(existing), msg.DueOn, msg.DueTime)
		if innerErr != nil {
			return innerErr
		}
		dueOn, dueTime, dueAt := deadlineColumns(dl, loc)

		if replaceAssignees {
			if innerErr = refreshKnownMembers(ctx, q, c, members); innerErr != nil {
				return h.internal(ctx, innerErr, "refresh known members")
			}
			if innerErr = h.requireMembers(ctx, q, c, newAssignees); innerErr != nil {
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
			return h.internal(ctx, innerErr, "update task")
		}

		if replaceAssignees {
			if innerErr = q.DeleteAssignees(ctx, db.DeleteAssigneesParams{TaskID: taskID, FamilyID: c.familyID}); innerErr != nil {
				return h.internal(ctx, innerErr, "delete assignees")
			}
			for _, a := range newAssignees {
				if innerErr = q.AddAssignee(ctx, db.AddAssigneeParams{
					TaskID: taskID, FamilyID: c.familyID, UserID: a,
				}); innerErr != nil {
					return h.internal(ctx, innerErr, "add assignee")
				}
			}
			assignees = newAssignees
		} else {
			current, innerErr := assigneesOf(ctx, q, c, []pgtype.UUID{taskID})
			if innerErr != nil {
				return h.internal(ctx, innerErr, "list assignees")
			}
			if assignees, innerErr = uuidList("assignee_user_ids", current[id(taskID)]); innerErr != nil {
				return h.internal(ctx, innerErr, "parse assignees")
			}
		}

		if task.Status != "open" {
			return nil
		}
		if innerErr = syncTaskReminders(ctx, q, c, task, assignees, loc); innerErr != nil {
			return h.internal(ctx, innerErr, "schedule task reminders")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.UpdateTaskResponse{
		Task: toProtoTask(task, uuidStrings(assignees), now, loc),
	}), nil
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

	now := h.now()
	var task db.Task
	var assignees map[string][]string
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		var innerErr error
		task, innerErr = q.CompleteTask(ctx, db.CompleteTaskParams{
			CompletedAt:       pgTimestamptz(now),
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
		if innerErr = q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
			FamilyID: c.familyID, Kind: kindTaskDue, ItemID: taskID,
		}); innerErr != nil {
			return h.internal(ctx, innerErr, "delete task reminders")
		}
		if assignees, innerErr = assigneesOf(ctx, q, c, []pgtype.UUID{taskID}); innerErr != nil {
			return h.internal(ctx, innerErr, "list assignees")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.CompleteTaskResponse{
		Task: toProtoTask(task, assignees[id(taskID)], now, h.familyLoc(ctx, c)),
	}), nil
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

	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	now, loc := h.now(), hh.loc

	var task db.Task
	var assignees []pgtype.UUID
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		var innerErr error
		task, innerErr = q.ReopenTask(ctx, db.ReopenTaskParams{ID: taskID, FamilyID: c.familyID})
		if errors.Is(innerErr, pgx.ErrNoRows) {
			return notFound("task")
		}
		if innerErr != nil {
			return h.internal(ctx, innerErr, "reopen task")
		}
		current, innerErr := assigneesOf(ctx, q, c, []pgtype.UUID{taskID})
		if innerErr != nil {
			return h.internal(ctx, innerErr, "list assignees")
		}
		if assignees, innerErr = uuidList("assignee_user_ids", current[id(taskID)]); innerErr != nil {
			return h.internal(ctx, innerErr, "parse assignees")
		}
		if innerErr = syncTaskReminders(ctx, q, c, task, assignees, loc); innerErr != nil {
			return h.internal(ctx, innerErr, "schedule task reminders")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.ReopenTaskResponse{
		Task: toProtoTask(task, uuidStrings(assignees), now, loc),
	}), nil
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
		rows, innerErr := q.DeleteTask(ctx, db.DeleteTaskParams{ID: taskID, FamilyID: c.familyID})
		if innerErr != nil {
			return h.internal(ctx, innerErr, "delete task")
		}
		if rows == 0 {
			return notFound("task")
		}
		if innerErr = q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
			FamilyID: c.familyID, Kind: kindTaskDue, ItemID: taskID,
		}); innerErr != nil {
			return h.internal(ctx, innerErr, "delete task reminders")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.DeleteTaskResponse{}), nil
}

func (h *Handler) requireMembers(ctx context.Context, q db.Querier, c caller, users []pgtype.UUID) error {
	if len(users) == 0 {
		return nil
	}
	known, err := q.ListKnownMembers(ctx, c.familyID)
	if err != nil {
		return h.internal(ctx, err, "list known members")
	}
	members := make(map[string]bool, len(known))
	for _, m := range known {
		members[id(m.UserID)] = true
	}
	for _, u := range users {
		if !members[id(u)] {
			return invalid("assignee %s is not a member of the family", id(u))
		}
	}
	return nil
}

func syncTaskReminders(
	ctx context.Context, q db.Querier, c caller, t db.Task, assignees []pgtype.UUID, loc *time.Location,
) error {
	dl := deadlineOf(t)
	if dl.IsZero() || !t.DueAt.Valid {
		if err := q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
			FamilyID: c.familyID, Kind: kindTaskDue, ItemID: t.ID,
		}); err != nil {
			return fmt.Errorf("delete task reminders: %w", err)
		}
		return nil
	}
	occurrence := t.DueAt.Time.UTC().Format(time.RFC3339)
	remindAt := schedule.TaskRemindAt(dl, loc)
	for _, a := range assignees {
		if err := q.UpsertReminder(ctx, db.UpsertReminderParams{
			FamilyID:   c.familyID,
			UserID:     a,
			Kind:       kindTaskDue,
			ItemID:     t.ID,
			Occurrence: occurrence,
			RemindAt:   pgTimestamptz(remindAt),
		}); err != nil {
			return fmt.Errorf("upsert task reminder: %w", err)
		}
	}
	if err := q.DeletePendingRemindersForItemExcept(ctx, db.DeletePendingRemindersForItemExceptParams{
		FamilyID:       c.familyID,
		Kind:           kindTaskDue,
		ItemID:         t.ID,
		KeepUsers:      assignees,
		KeepOccurrence: occurrence,
	}); err != nil {
		return fmt.Errorf("delete stale task reminders: %w", err)
	}
	return nil
}

func assigneesOf(ctx context.Context, q db.Querier, c caller, ids []pgtype.UUID) (map[string][]string, error) {
	out := make(map[string][]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.ListAssignees(ctx, db.ListAssigneesParams{FamilyID: c.familyID, TaskIds: ids})
	if err != nil {
		return nil, fmt.Errorf("list assignees: %w", err)
	}
	for _, r := range rows {
		out[id(r.TaskID)] = append(out[id(r.TaskID)], id(r.UserID))
	}
	return out, nil
}

func mergeDeadline(current schedule.Deadline, dueOn, dueTime *string) (schedule.Deadline, error) {
	on, clock := current.On, current.Time
	if dueOn != nil {
		on = *dueOn
		if on == "" {
			clock = ""
		}
	}
	if dueTime != nil {
		clock = *dueTime
	}
	if on == "" && clock != "" {
		return schedule.Deadline{}, invalid("due_time needs a due_on date")
	}
	dl, err := schedule.ParseDeadline(on, clock)
	if err != nil {
		return schedule.Deadline{}, invalid("%v", err)
	}
	return dl, nil
}

func deadlineColumns(dl schedule.Deadline, loc *time.Location) (pgtype.Date, pgtype.Time, pgtype.Timestamptz) {
	if dl.IsZero() {
		return pgtype.Date{}, pgtype.Time{}, pgtype.Timestamptz{}
	}
	on := pgtype.Date{Time: parseDate(dl.On), Valid: true}
	var clock pgtype.Time
	if dl.HasTime() {
		clock = pgTime(parseTime(dl.Time))
	}
	return on, clock, pgTimestamptz(dl.At(loc))
}

func deadlineOf(t db.Task) schedule.Deadline {
	if !t.DueOn.Valid {
		return schedule.Deadline{}
	}
	dl := schedule.Deadline{On: t.DueOn.Time.Format(schedule.DateLayout)}
	if t.DueTime.Valid {
		dl.Time = timeFromPgTime(t.DueTime).Format(schedule.TimeLayout)
	}
	return dl
}

func taskIDs(tasks []db.Task) []pgtype.UUID {
	out := make([]pgtype.UUID, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, t.ID)
	}
	return out
}

func uuidStrings(ids []pgtype.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, u := range ids {
		out = append(out, id(u))
	}
	return out
}

func id(u pgtype.UUID) string { return pgconv.UUIDString(u) }

func optionalTimestamp(t pgtype.Timestamptz) *timestamppb.Timestamp {
	if !t.Valid {
		return nil
	}
	return timestamppb.New(t.Time)
}

func toProtoTask(t db.Task, assignees []string, now time.Time, loc *time.Location) *tasksv1.Task {
	if assignees == nil {
		assignees = []string{}
	}
	dl := deadlineOf(t)
	completedBy := ""
	if t.CompletedByUserID.Valid {
		completedBy = id(t.CompletedByUserID)
	}
	return &tasksv1.Task{
		Id:                id(t.ID),
		FamilyId:          id(t.FamilyID),
		Title:             t.Title,
		Notes:             t.Notes,
		Priority:          protoPriorityFromString(t.Priority),
		Status:            protoStatusFromString(t.Status),
		DueOn:             dl.On,
		DueTime:           dl.Time,
		AssigneeUserIds:   assignees,
		CreatedByUserId:   id(t.CreatedByUserID),
		CompletedByUserId: completedBy,
		DueAt:             optionalTimestamp(t.DueAt),
		CompletedAt:       optionalTimestamp(t.CompletedAt),
		CreatedAt:         timestamppb.New(t.CreatedAt.Time),
		UpdatedAt:         timestamppb.New(t.UpdatedAt.Time),
		Overdue:           t.Status == "open" && dl.Overdue(now, loc),
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
