package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/sdk/go/family/v1/familyv1connect"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/family"
)

func withClaims(ctx context.Context, userID, familyID string) context.Context {
	return fmauth.WithClaims(ctx, &fmauth.Claims{UserID: userID, FamilyID: familyID})
}

const (
	testFamily = "00000000-0000-4000-8000-000000000001"
	sergiy     = "00000000-0000-4000-8000-000000000002"
	olena      = "00000000-0000-4000-8000-000000000003"
	otherFam   = "00000000-0000-4000-8000-000000000009"
)

var testNow = time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)

func newTestHandler(t *testing.T) (*Handler, *fakeStore, *recorder) {
	t.Helper()
	store := newFakeStore()
	rec := &recorder{}
	h := New(Options{
		Queries: store, Tx: store, Bus: rec,
		Now: func() time.Time { return testNow },
	})
	seedHousehold(store)
	return h, store, rec
}

func seedHousehold(s *fakeStore) {
	family := pgconv.MustUUID(testFamily)
	s.settings[testFamily] = db.FamilySetting{
		FamilyID: family, Timezone: "UTC",
	}
	for _, m := range []struct {
		id, name string
	}{{sergiy, "Сергій"}, {olena, "Олена"}} {
		uid := pgconv.MustUUID(m.id)
		s.members[id(family)+"|"+id(uid)] = db.KnownMember{
			FamilyID:    family,
			UserID:      uid,
			DisplayName: m.name,
			Email:       m.name + "@example.com",
			SeenAt:      now(),
		}
	}
}

func ctxOf(user string) context.Context {
	return withClaims(context.Background(), user, testFamily)
}

func seedTask(s *fakeStore, title string, assignees ...string) db.Task {
	t := db.Task{
		ID:                s.newUUID(),
		FamilyID:          pgconv.MustUUID(testFamily),
		Title:             title,
		Notes:             "",
		Priority:          "medium",
		Status:            "open",
		DueOn:             pgtype.Date{},
		DueTime:           pgtype.Time{},
		DueAt:             pgtype.Timestamptz{},
		CreatedByUserID:   pgconv.MustUUID(sergiy),
		CompletedByUserID: pgtype.UUID{},
		CompletedAt:       pgtype.Timestamptz{},
		CreatedAt:         now(),
		UpdatedAt:         now(),
	}
	s.tasks[id(t.ID)] = t
	for _, a := range assignees {
		aid := pgconv.MustUUID(a)
		s.assignees[id(t.ID)+"|"+id(aid)] = db.TaskAssignee{
			TaskID: t.ID, FamilyID: pgconv.MustUUID(testFamily), UserID: aid,
		}
	}
	return t
}

func codeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	return connect.CodeOf(err)
}

func TestListTasksEmpty(t *testing.T) {
	h, _, _ := newTestHandler(t)

	resp, err := h.ListTasks(ctxOf(sergiy), connect.NewRequest(&tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_OPEN,
	}))
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(resp.Msg.Tasks) != 0 {
		t.Errorf("want 0 tasks, got %d", len(resp.Msg.Tasks))
	}
}

func TestListTasksFilters(t *testing.T) {
	h, store, _ := newTestHandler(t)
	t1 := seedTask(store, "Task 1", sergiy)
	seedTask(store, "Task 2", olena)
	seedTask(store, "Task 3", sergiy)
	// Complete t1
	s := store
	s.tasks[id(t1.ID)] = db.Task{
		ID: t1.ID, FamilyID: t1.FamilyID, Title: t1.Title, Status: "done",
		CreatedByUserID: t1.CreatedByUserID, CreatedAt: t1.CreatedAt, UpdatedAt: now(),
	}

	// OPEN filter (shows all open tasks regardless of assignee)
	resp, err := h.ListTasks(ctxOf(sergiy), connect.NewRequest(&tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_OPEN,
	}))
	if err != nil {
		t.Fatalf("ListTasks OPEN: %v", err)
	}
	if len(resp.Msg.Tasks) != 2 {
		t.Errorf("OPEN filter: want 2 tasks, got %d", len(resp.Msg.Tasks))
	}

	// MINE filter (shows all tasks assigned to the user, open and done)
	resp, err = h.ListTasks(ctxOf(sergiy), connect.NewRequest(&tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_MINE,
	}))
	if err != nil {
		t.Fatalf("ListTasks MINE: %v", err)
	}
	if len(resp.Msg.Tasks) != 2 {
		t.Errorf("MINE filter: want 2 tasks (Task 1 done + Task 3 open), got %d", len(resp.Msg.Tasks))
		for _, task := range resp.Msg.Tasks {
			t.Logf("  task: %s status=%s", task.Title, task.Status)
		}
	}

	// DONE filter
	resp, err = h.ListTasks(ctxOf(sergiy), connect.NewRequest(&tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_DONE,
	}))
	if err != nil {
		t.Fatalf("ListTasks DONE: %v", err)
	}
	if len(resp.Msg.Tasks) != 1 {
		t.Errorf("DONE filter: want 1 task, got %d", len(resp.Msg.Tasks))
	}
	if resp.Msg.Tasks[0].Id != id(t1.ID) {
		t.Errorf("DONE filter: wrong task %s", resp.Msg.Tasks[0].Id)
	}

	// ALL filter
	resp, err = h.ListTasks(ctxOf(sergiy), connect.NewRequest(&tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_ALL,
	}))
	if err != nil {
		t.Fatalf("ListTasks ALL: %v", err)
	}
	if len(resp.Msg.Tasks) != 3 {
		t.Errorf("ALL filter: want 3 tasks, got %d", len(resp.Msg.Tasks))
	}
}

func TestUpdateTaskRemindersRecreated(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := seedTask(store, "With deadline", sergiy)
	// Add a deadline
	s := store
	s.tasks[id(task.ID)] = db.Task{
		ID: task.ID, FamilyID: task.FamilyID, Title: task.Title, Status: "open",
		DueOn:           pgtype.Date{Time: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		DueTime:         pgtype.Time{Microseconds: 10 * 3600 * 1e6, Valid: true},
		DueAt:           pgtype.Timestamptz{Time: time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC), Valid: true},
		CreatedByUserID: task.CreatedByUserID, CreatedAt: task.CreatedAt, UpdatedAt: now(),
	}

	title := "Updated deadline"
	resp, err := h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: id(task.ID),
		Title:  &title,
	}))
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if resp.Msg.Task.DueOn != "2026-09-01" {
		t.Errorf("due_on preserved = %q, want 2026-09-01", resp.Msg.Task.DueOn)
	}
}

func TestCreateTask(t *testing.T) {
	h, store, rec := newTestHandler(t)

	resp, err := h.CreateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CreateTaskRequest{
		Title:           "Test Task",
		Notes:           "Some notes",
		Priority:        tasksv1.Priority_PRIORITY_HIGH,
		DueOn:           "2026-09-01",
		DueTime:         "10:00",
		AssigneeUserIds: []string{sergiy, olena},
	}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	task := resp.Msg.Task
	if task.Title != "Test Task" {
		t.Errorf("title = %q, want %q", task.Title, "Test Task")
	}
	if task.Priority != tasksv1.Priority_PRIORITY_HIGH {
		t.Errorf("priority = %v, want HIGH", task.Priority)
	}
	if task.DueOn != "2026-09-01" {
		t.Errorf("due_on = %q, want 2026-09-01", task.DueOn)
	}
	if task.DueTime != "10:00" {
		t.Errorf("due_time = %q, want 10:00", task.DueTime)
	}
	if len(task.AssigneeUserIds) != 2 {
		t.Errorf("assignees = %v, want 2", task.AssigneeUserIds)
	}
	if task.CreatedByUserId != sergiy {
		t.Errorf("created_by_user_id = %q, want %q", task.CreatedByUserId, sergiy)
	}
	if task.Status != tasksv1.TaskStatus_TASK_STATUS_OPEN {
		t.Errorf("status = %v, want OPEN", task.Status)
	}

	if len(rec.subjects) != 1 || rec.subjects[0] != "tasks.task.assigned" {
		t.Errorf("published %v, want one task assignment", rec.subjects)
	}
	if len(rec.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(rec.messages))
	}
	assigned := rec.messages[0].(*tasksv1.TaskAssignedEvent)
	if len(assigned.AssigneeUserIds) != 2 || assigned.TaskId != task.Id || assigned.AssignedByUserId != sergiy {
		t.Errorf("assignment = %+v", assigned)
	}

	// Check assignees stored
	if len(store.assignees) != 2 {
		t.Errorf("assignees stored = %d, want 2", len(store.assignees))
	}
}

func TestCreateTaskDefaultsToSelfAssignee(t *testing.T) {
	h, store, _ := newTestHandler(t)

	resp, err := h.CreateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CreateTaskRequest{
		Title: "Self-assigned",
	}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	task := resp.Msg.Task
	if len(task.AssigneeUserIds) != 1 || task.AssigneeUserIds[0] != sergiy {
		t.Errorf("assignees = %v, want [%s]", task.AssigneeUserIds, sergiy)
	}
	if len(store.assignees) != 1 {
		t.Errorf("assignees stored = %d, want 1", len(store.assignees))
	}
}

func TestCreateTaskValidation(t *testing.T) {
	h, _, _ := newTestHandler(t)

	// Title too long
	_, err := h.CreateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CreateTaskRequest{
		Title: string(make([]rune, 201)),
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("title too long: code = %v, want InvalidArgument", got)
	}

	// Notes too long
	_, err = h.CreateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CreateTaskRequest{
		Title: "OK",
		Notes: string(make([]rune, 4001)),
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("notes too long: code = %v, want InvalidArgument", got)
	}

	// Due time without date
	_, err = h.CreateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CreateTaskRequest{
		Title:   "OK",
		DueTime: "10:00",
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("due_time without date: code = %v, want InvalidArgument", got)
	}
}

func TestUpdateTask(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := seedTask(store, "Original", sergiy)

	title := "Updated"
	notes := "New notes"
	priority := tasksv1.Priority_PRIORITY_URGENT
	dueOn := "2026-09-15"
	dueTime := "14:30"
	resp, err := h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId:   id(task.ID),
		Title:    &title,
		Notes:    &notes,
		Priority: &priority,
		DueOn:    &dueOn,
		DueTime:  &dueTime,
	}))
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	if resp.Msg.Task.Title != "Updated" {
		t.Errorf("title = %q, want Updated", resp.Msg.Task.Title)
	}
	if resp.Msg.Task.Notes != "New notes" {
		t.Errorf("notes = %q, want New notes", resp.Msg.Task.Notes)
	}
	if resp.Msg.Task.Priority != tasksv1.Priority_PRIORITY_URGENT {
		t.Errorf("priority = %v, want URGENT", resp.Msg.Task.Priority)
	}
	if resp.Msg.Task.DueOn != "2026-09-15" {
		t.Errorf("due_on = %q, want 2026-09-15", resp.Msg.Task.DueOn)
	}
	if resp.Msg.Task.DueTime != "14:30" {
		t.Errorf("due_time = %q, want 14:30", resp.Msg.Task.DueTime)
	}
}

func TestUpdateTaskClearsAssignees(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := seedTask(store, "With assignees", sergiy, olena)

	resp, err := h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: id(task.ID),
		Assignees: &tasksv1.AssigneeList{
			UserIds: []string{},
		},
	}))
	if err != nil {
		t.Fatalf("UpdateTask clear assignees: %v", err)
	}

	if len(resp.Msg.Task.AssigneeUserIds) != 0 {
		t.Errorf("assignees = %v, want empty", resp.Msg.Task.AssigneeUserIds)
	}
	if len(store.assignees) != 0 {
		t.Errorf("assignees stored = %d, want 0", len(store.assignees))
	}
}

func TestUpdateTaskPublishesOnlyNewAssignees(t *testing.T) {
	h, store, rec := newTestHandler(t)
	task := seedTask(store, "Shared", sergiy)
	_, err := h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: id(task.ID), Assignees: &tasksv1.AssigneeList{UserIds: []string{sergiy, olena}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.messages) != 1 {
		t.Fatalf("events = %d, want 1", len(rec.messages))
	}
	assigned := rec.messages[0].(*tasksv1.TaskAssignedEvent)
	if len(assigned.AssigneeUserIds) != 1 || assigned.AssigneeUserIds[0] != olena {
		t.Errorf("new assignees = %v, want [%s]", assigned.AssigneeUserIds, olena)
	}
	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: id(task.ID), Assignees: &tasksv1.AssigneeList{UserIds: []string{sergiy, olena}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.messages) != 1 {
		t.Errorf("reassigning same users published %d events, want 1", len(rec.messages))
	}
}

func TestCompleteTask(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := seedTask(store, "To complete", sergiy)

	resp, err := h.CompleteTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CompleteTaskRequest{
		TaskId: id(task.ID),
	}))
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	if resp.Msg.Task.Status != tasksv1.TaskStatus_TASK_STATUS_DONE {
		t.Errorf("status = %v, want DONE", resp.Msg.Task.Status)
	}
	if resp.Msg.Task.CompletedByUserId != sergiy {
		t.Errorf("completed_by_user_id = %q, want %q", resp.Msg.Task.CompletedByUserId, sergiy)
	}

	// Check reminders deleted
	found := false
	for k := range store.reminders {
		if len(k) > len(id(task.ID)) && k[:len(id(task.ID))] == id(task.ID) {
			found = true
			break
		}
	}
	if found {
		t.Error("reminders for completed task should be deleted")
	}
}

func TestReopenTask(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := seedTask(store, "To reopen", sergiy)
	// Complete it first
	s := store
	s.tasks[id(task.ID)] = db.Task{
		ID: task.ID, FamilyID: task.FamilyID, Title: task.Title, Status: "done",
		CompletedAt: now(), CompletedByUserID: pgconv.MustUUID(sergiy),
		CreatedByUserID: task.CreatedByUserID, CreatedAt: task.CreatedAt, UpdatedAt: now(),
	}

	resp, err := h.ReopenTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.ReopenTaskRequest{
		TaskId: id(task.ID),
	}))
	if err != nil {
		t.Fatalf("ReopenTask: %v", err)
	}

	if resp.Msg.Task.Status != tasksv1.TaskStatus_TASK_STATUS_OPEN {
		t.Errorf("status = %v, want OPEN", resp.Msg.Task.Status)
	}
	if resp.Msg.Task.CompletedByUserId != "" {
		t.Errorf("completed_by_user_id = %q, want empty", resp.Msg.Task.CompletedByUserId)
	}
}

func TestDeleteTask(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := seedTask(store, "To delete", sergiy)

	resp, err := h.DeleteTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.DeleteTaskRequest{
		TaskId: id(task.ID),
	}))
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if resp.Msg == nil {
		t.Error("expected empty response")
	}

	// Task should be gone
	if _, ok := store.tasks[id(task.ID)]; ok {
		t.Error("task should be deleted")
	}
}

func TestCrossFamilyAccess(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := seedTask(store, "Other family task", sergiy)

	ctx := withClaims(context.Background(), sergiy, otherFam)
	_, err := h.GetTask(ctx, connect.NewRequest(&tasksv1.GetTaskRequest{
		TaskId: id(task.ID),
	}))
	if got := codeOf(t, err); got != connect.CodeNotFound && got != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want NotFound or FailedPrecondition", got)
	}
}

func TestCallerWithoutFamily(t *testing.T) {
	h, _, _ := newTestHandler(t)
	ctx := withClaims(context.Background(), sergiy, "")

	_, err := h.ListTasks(ctx, connect.NewRequest(&tasksv1.ListTasksRequest{}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want FailedPrecondition", got)
	}
}

func TestUnauthenticatedCaller(t *testing.T) {
	h, _, _ := newTestHandler(t)

	_, err := h.ListTasks(context.Background(), connect.NewRequest(&tasksv1.ListTasksRequest{}))
	if got := codeOf(t, err); got != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want Unauthenticated", got)
	}
}

func TestFakeStoreListTasksAssigneeFilter(t *testing.T) {
	s := newFakeStore()
	familyID := pgconv.MustUUID(testFamily)
	sergiyID := pgconv.MustUUID(sergiy)
	olenaID := pgconv.MustUUID(olena)

	// Create tasks
	t1 := db.Task{
		ID: s.newUUID(), FamilyID: familyID, Title: "Task 1", Status: "open", Priority: "medium",
		CreatedByUserID: sergiyID, CreatedAt: now(), UpdatedAt: now(),
	}
	t2 := db.Task{
		ID: s.newUUID(), FamilyID: familyID, Title: "Task 2", Status: "open", Priority: "medium",
		CreatedByUserID: sergiyID, CreatedAt: now(), UpdatedAt: now(),
	}
	t3 := db.Task{
		ID: s.newUUID(), FamilyID: familyID, Title: "Task 3", Status: "open", Priority: "medium",
		CreatedByUserID: sergiyID, CreatedAt: now(), UpdatedAt: now(),
	}
	s.tasks[id(t1.ID)] = t1
	s.tasks[id(t2.ID)] = t2
	s.tasks[id(t3.ID)] = t3

	// Add assignees
	s.assignees[id(t1.ID)+"|"+id(sergiyID)] = db.TaskAssignee{TaskID: t1.ID, FamilyID: familyID, UserID: sergiyID}
	s.assignees[id(t2.ID)+"|"+id(olenaID)] = db.TaskAssignee{TaskID: t2.ID, FamilyID: familyID, UserID: olenaID}
	s.assignees[id(t3.ID)+"|"+id(sergiyID)] = db.TaskAssignee{TaskID: t3.ID, FamilyID: familyID, UserID: sergiyID}

	// Test MINE filter for sergiy
	tasks, err := s.ListTasks(context.Background(), db.ListTasksParams{
		FamilyID: familyID,
		Status:   "open",
		Assignee: sergiyID,
	})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("want 2 tasks for sergiy, got %d", len(tasks))
		for _, task := range tasks {
			t.Logf("  task: %s", task.Title)
		}
	}

	// Test MINE filter for olena
	tasks, err = s.ListTasks(context.Background(), db.ListTasksParams{
		FamilyID: familyID,
		Status:   "open",
		Assignee: olenaID,
	})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("want 1 task for olena, got %d", len(tasks))
		for _, task := range tasks {
			t.Logf("  task: %s", task.Title)
		}
	}
}

func TestCallerExtractsUserID(t *testing.T) {
	h, _, _ := newTestHandler(t)

	// Create a context with claims
	ctx := ctxOf(sergiy)
	c, err := h.caller(ctx)
	if err != nil {
		t.Fatalf("caller: %v", err)
	}
	if c.user != sergiy {
		t.Errorf("user = %q, want %q", c.user, sergiy)
	}
	if !c.userID.Valid {
		t.Error("userID should be valid")
	}
	if id(c.userID) != sergiy {
		t.Errorf("userID = %q, want %q", id(c.userID), sergiy)
	}
	if c.family != testFamily {
		t.Errorf("family = %q, want %q", c.family, testFamily)
	}
	if !c.familyID.Valid {
		t.Error("familyID should be valid")
	}
	if id(c.familyID) != testFamily {
		t.Errorf("familyID = %q, want %q", id(c.familyID), testFamily)
	}
}

func TestTaskSorting(t *testing.T) {
	h, store, _ := newTestHandler(t)
	baseTime := testNow

	// Create tasks with different priorities and due dates
	t1 := seedTask(store, "Low priority, due soon", sergiy)
	s := store
	s.tasks[id(t1.ID)] = db.Task{
		ID: t1.ID, FamilyID: t1.FamilyID, Title: t1.Title, Status: "open", Priority: "low",
		DueOn:           pgtype.Date{Time: baseTime.AddDate(0, 0, 1), Valid: true},
		DueAt:           pgtype.Timestamptz{Time: baseTime.AddDate(0, 0, 1), Valid: true},
		CreatedByUserID: t1.CreatedByUserID,
		CreatedAt:       pgtype.Timestamptz{Time: baseTime, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: baseTime, Valid: true},
	}

	t2 := seedTask(store, "High priority, due later", sergiy)
	s.tasks[id(t2.ID)] = db.Task{
		ID: t2.ID, FamilyID: t2.FamilyID, Title: t2.Title, Status: "open", Priority: "high",
		DueOn:           pgtype.Date{Time: baseTime.AddDate(0, 0, 5), Valid: true},
		DueAt:           pgtype.Timestamptz{Time: baseTime.AddDate(0, 0, 5), Valid: true},
		CreatedByUserID: t2.CreatedByUserID,
		CreatedAt:       pgtype.Timestamptz{Time: baseTime.Add(time.Hour), Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: baseTime, Valid: true},
	}

	t3 := seedTask(store, "No deadline", sergiy)
	s.tasks[id(t3.ID)] = db.Task{
		ID: t3.ID, FamilyID: t3.FamilyID, Title: t3.Title, Status: "open", Priority: "medium",
		DueOn: pgtype.Date{}, DueAt: pgtype.Timestamptz{},
		CreatedByUserID: t3.CreatedByUserID,
		CreatedAt:       pgtype.Timestamptz{Time: baseTime.Add(2 * time.Hour), Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: baseTime, Valid: true},
	}

	resp, err := h.ListTasks(ctxOf(sergiy), connect.NewRequest(&tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_OPEN,
	}))
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}

	// Should be sorted by due_at (soonest first), then priority, then created_at
	if len(resp.Msg.Tasks) != 3 {
		t.Fatalf("want 3 tasks, got %d", len(resp.Msg.Tasks))
	}
	if resp.Msg.Tasks[0].Title != "Low priority, due soon" {
		t.Errorf("first = %q, want 'Low priority, due soon'", resp.Msg.Tasks[0].Title)
	}
	if resp.Msg.Tasks[1].Title != "High priority, due later" {
		t.Errorf("second = %q, want 'High priority, due later'", resp.Msg.Tasks[1].Title)
	}
	if resp.Msg.Tasks[2].Title != "No deadline" {
		t.Errorf("third = %q, want 'No deadline'", resp.Msg.Tasks[2].Title)
	}
}

const (
	stranger = "00000000-0000-4000-8000-00000000000b"
)

func seedOtherFamily(s *fakeStore) {
	family := pgconv.MustUUID(otherFam)
	s.settings[otherFam] = db.FamilySetting{FamilyID: family, Timezone: "UTC"}
	uid := pgconv.MustUUID(stranger)
	s.members[otherFam+"|"+stranger] = db.KnownMember{
		FamilyID: family, UserID: uid, DisplayName: "Чужий", SeenAt: now(),
	}
}

func useTimezone(s *fakeStore, tz string) {
	row := s.settings[testFamily]
	row.Timezone = tz
	s.settings[testFamily] = row
}

func taskReminders(s *fakeStore, taskID string) map[string]db.TaskReminder {
	out := map[string]db.TaskReminder{}
	for _, r := range s.reminders {
		if r.Kind == kindTaskDue && id(r.ItemID) == taskID {
			out[id(r.UserID)+" "+r.Occurrence] = r
		}
	}
	return out
}

func createTask(t *testing.T, h *Handler, req *tasksv1.CreateTaskRequest) *tasksv1.Task {
	t.Helper()
	resp, err := h.CreateTask(ctxOf(sergiy), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	return resp.Msg.Task
}

func strPtr(s string) *string { return &s }

func TestTasksAreFamilyScoped(t *testing.T) {
	h, store, _ := newTestHandler(t)
	seedOtherFamily(store)
	task := createTask(t, h, &tasksv1.CreateTaskRequest{Title: "A only", DueOn: "2026-09-01"})
	before := len(taskReminders(store, task.Id))
	b := withClaims(context.Background(), stranger, otherFam)

	list, err := h.ListTasks(b, connect.NewRequest(&tasksv1.ListTasksRequest{Filter: tasksv1.TaskFilter_TASK_FILTER_ALL}))
	if err != nil {
		t.Fatalf("ListTasks B: %v", err)
	}
	if len(list.Msg.Tasks) != 0 {
		t.Errorf("family B sees %d tasks, want 0", len(list.Msg.Tasks))
	}
	_, err = h.GetTask(b, connect.NewRequest(&tasksv1.GetTaskRequest{TaskId: task.Id}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("get: code = %v, want NotFound", got)
	}
	_, err = h.UpdateTask(b, connect.NewRequest(&tasksv1.UpdateTaskRequest{TaskId: task.Id, Title: strPtr("hijack")}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("update: code = %v, want NotFound", got)
	}
	_, err = h.CompleteTask(b, connect.NewRequest(&tasksv1.CompleteTaskRequest{TaskId: task.Id}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("complete: code = %v, want NotFound", got)
	}
	_, err = h.ReopenTask(b, connect.NewRequest(&tasksv1.ReopenTaskRequest{TaskId: task.Id}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("reopen: code = %v, want NotFound", got)
	}
	_, err = h.DeleteTask(b, connect.NewRequest(&tasksv1.DeleteTaskRequest{TaskId: task.Id}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("delete: code = %v, want NotFound", got)
	}

	got := store.tasks[task.Id]
	if got.Title != "A only" || got.Status != "open" {
		t.Errorf("task after family B calls = %q/%s", got.Title, got.Status)
	}
	if n := len(taskReminders(store, task.Id)); n != before {
		t.Errorf("reminders = %d, want %d", n, before)
	}
}

func TestAssigneesMustBeFamilyMembers(t *testing.T) {
	h, store, _ := newTestHandler(t)
	seedOtherFamily(store)

	for name, ids := range map[string][]string{
		"unknown user":        {sergiy, newbie},
		"other family's user": {stranger},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := h.CreateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CreateTaskRequest{
				Title: "x", AssigneeUserIds: ids,
			}))
			if got := codeOf(t, err); got != connect.CodeInvalidArgument {
				t.Errorf("create: code = %v, want InvalidArgument", got)
			}
		})
	}
	if len(store.tasks) != 0 {
		t.Errorf("tasks stored = %d, want 0", len(store.tasks))
	}

	task := createTask(t, h, &tasksv1.CreateTaskRequest{Title: "x", AssigneeUserIds: []string{olena}})
	_, err := h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, Assignees: &tasksv1.AssigneeList{UserIds: []string{olena, newbie}},
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("update: code = %v, want InvalidArgument", got)
	}
	resp, err := h.GetTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.GetTaskRequest{TaskId: task.Id}))
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if a := resp.Msg.Task.AssigneeUserIds; len(a) != 1 || a[0] != olena {
		t.Errorf("assignees after rejected update = %v, want [olena]", a)
	}
}

func TestAssigneeCheckUsesRefreshedMembers(t *testing.T) {
	fam := &membersFamily{}
	mux := http.NewServeMux()
	mux.Handle(familyv1connect.NewFamilyServiceHandler(fam))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	store := newFakeStore()
	seedHousehold(store)
	h := New(Options{
		Queries: store, Tx: store, FamilyPublic: family.New(srv.URL, srv.URL, 0),
		Now: func() time.Time { return testNow },
	})

	req := connect.NewRequest(&tasksv1.CreateTaskRequest{Title: "x", AssigneeUserIds: []string{newbie}})
	req.Header().Set("Authorization", "Bearer caller-token")
	resp, err := h.CreateTask(ctxOf(sergiy), req)
	if err != nil {
		t.Fatalf("CreateTask for a freshly joined member: %v", err)
	}
	if fam.bearer != "caller-token" {
		t.Errorf("ListMembers bearer = %q, want the caller's token", fam.bearer)
	}
	if a := resp.Msg.Task.AssigneeUserIds; len(a) != 1 || a[0] != newbie {
		t.Errorf("assignees = %v, want [newbie]", a)
	}

	req = connect.NewRequest(&tasksv1.CreateTaskRequest{Title: "y", AssigneeUserIds: []string{olena}})
	req.Header().Set("Authorization", "Bearer caller-token")
	_, err = h.CreateTask(ctxOf(sergiy), req)
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("assigning a member who left: code = %v, want InvalidArgument", got)
	}
}

func TestCreateTaskSetsDueAtInFamilyTimezone(t *testing.T) {
	h, store, _ := newTestHandler(t)
	useTimezone(store, "Europe/Kyiv")

	cases := map[string]struct {
		dueTime       string
		dueAt, remind time.Time
	}{
		"date only": {"", time.Date(2026, 9, 1, 20, 59, 59, 0, time.UTC), time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)},
		"timed":     {"10:00", time.Date(2026, 9, 1, 7, 0, 0, 0, time.UTC), time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			task := createTask(t, h, &tasksv1.CreateTaskRequest{
				Title: name, DueOn: "2026-09-01", DueTime: tc.dueTime, AssigneeUserIds: []string{sergiy, olena},
			})
			stored := store.tasks[task.Id]
			if !stored.DueAt.Valid || !stored.DueAt.Time.Equal(tc.dueAt) {
				t.Fatalf("due_at = %v, want %v", stored.DueAt, tc.dueAt)
			}
			if !task.DueAt.AsTime().Equal(tc.dueAt) {
				t.Errorf("response due_at = %v, want %v", task.DueAt.AsTime(), tc.dueAt)
			}
			occ := tc.dueAt.Format(time.RFC3339)
			got := taskReminders(store, task.Id)
			if len(got) != 2 {
				t.Fatalf("reminders = %v, want one per assignee", got)
			}
			for _, u := range []string{sergiy, olena} {
				r, ok := got[u+" "+occ]
				if !ok {
					t.Fatalf("no reminder for %s at occurrence %s in %v", u, occ, got)
				}
				if !r.RemindAt.Time.Equal(tc.remind) {
					t.Errorf("remind_at %s = %v, want %v", u, r.RemindAt.Time, tc.remind)
				}
			}
		})
	}
}

func TestCreateTaskWithoutDeadlineHasNoReminders(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := createTask(t, h, &tasksv1.CreateTaskRequest{Title: "someday"})
	if store.tasks[task.Id].DueAt.Valid || task.DueAt != nil || task.DueOn != "" || task.DueTime != "" {
		t.Errorf("deadline = %q %q %v, want none", task.DueOn, task.DueTime, task.DueAt)
	}
	if n := len(taskReminders(store, task.Id)); n != 0 {
		t.Errorf("reminders = %d, want 0", n)
	}
}

func TestUpdateTaskReplacesRemindersOnReassignment(t *testing.T) {
	h, store, _ := newTestHandler(t)
	useTimezone(store, "Europe/Kyiv")
	task := createTask(t, h, &tasksv1.CreateTaskRequest{
		Title: "x", DueOn: "2026-09-01", DueTime: "10:00", AssigneeUserIds: []string{sergiy, olena},
	})

	_, err := h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, Assignees: &tasksv1.AssigneeList{UserIds: []string{olena}},
	}))
	if err != nil {
		t.Fatalf("reassign: %v", err)
	}
	occ := "2026-09-01T07:00:00Z"
	got := taskReminders(store, task.Id)
	if _, ok := got[olena+" "+occ]; !ok || len(got) != 1 {
		t.Errorf("reminders after reassignment = %v, want only olena at %s", got, occ)
	}

	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, DueOn: strPtr("2026-09-03"),
	}))
	if err != nil {
		t.Fatalf("move deadline: %v", err)
	}
	occ = "2026-09-03T07:00:00Z"
	got = taskReminders(store, task.Id)
	r, ok := got[olena+" "+occ]
	if !ok || len(got) != 1 {
		t.Fatalf("reminders after moving the deadline = %v, want only olena at %s", got, occ)
	}
	if want := time.Date(2026, 9, 3, 6, 0, 0, 0, time.UTC); !r.RemindAt.Time.Equal(want) {
		t.Errorf("remind_at = %v, want %v", r.RemindAt.Time, want)
	}

	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, Assignees: &tasksv1.AssigneeList{},
	}))
	if err != nil {
		t.Fatalf("clear assignees: %v", err)
	}
	if got := taskReminders(store, task.Id); len(got) != 0 {
		t.Errorf("reminders with no assignees = %v, want none", got)
	}
}

func TestUpdateTaskDeadlineRules(t *testing.T) {
	h, store, _ := newTestHandler(t)
	useTimezone(store, "Europe/Kyiv")
	task := createTask(t, h, &tasksv1.CreateTaskRequest{Title: "x", DueOn: "2026-09-01", DueTime: "10:00"})

	resp, err := h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, DueTime: strPtr(""),
	}))
	if err != nil {
		t.Fatalf("clear time: %v", err)
	}
	if want := time.Date(2026, 9, 1, 20, 59, 59, 0, time.UTC); resp.Msg.Task.DueTime != "" ||
		!store.tasks[task.Id].DueAt.Time.Equal(want) {
		t.Errorf("after clearing time: due_time %q due_at %v, want end of local day %v",
			resp.Msg.Task.DueTime, store.tasks[task.Id].DueAt.Time, want)
	}

	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, DueTime: strPtr("18:30"),
	}))
	if err != nil {
		t.Fatalf("set time: %v", err)
	}
	resp, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, DueOn: strPtr(""),
	}))
	if err != nil {
		t.Fatalf("clear date: %v", err)
	}
	stored := store.tasks[task.Id]
	if resp.Msg.Task.DueOn != "" || resp.Msg.Task.DueTime != "" || stored.DueOn.Valid || stored.DueTime.Valid || stored.DueAt.Valid {
		t.Errorf("due_on \"\" left a deadline: %+v", stored)
	}
	if got := taskReminders(store, task.Id); len(got) != 0 {
		t.Errorf("reminders without a deadline = %v", got)
	}

	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, DueTime: strPtr("09:00"),
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("due_time without a date: code = %v, want InvalidArgument", got)
	}
	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, DueOn: strPtr(""), DueTime: strPtr("09:00"),
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("due_on cleared with a due_time: code = %v, want InvalidArgument", got)
	}
	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, DueOn: strPtr("01.09.2026"),
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("bad date: code = %v, want InvalidArgument", got)
	}
}

func TestUpdateTaskWithoutAssigneeListKeepsAssignees(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := createTask(t, h, &tasksv1.CreateTaskRequest{
		Title: "x", DueOn: "2026-09-01", AssigneeUserIds: []string{sergiy, olena},
	})

	resp, err := h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{
		TaskId: task.Id, Title: strPtr("renamed"),
	}))
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if len(resp.Msg.Task.AssigneeUserIds) != 2 {
		t.Errorf("assignees = %v, want both kept", resp.Msg.Task.AssigneeUserIds)
	}
	if n := len(taskReminders(store, task.Id)); n != 2 {
		t.Errorf("reminders = %d, want 2", n)
	}
}

func TestCompleteReopenDeleteManageReminders(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := createTask(t, h, &tasksv1.CreateTaskRequest{
		Title: "x", DueOn: "2026-09-01", AssigneeUserIds: []string{sergiy, olena},
	})

	if _, err := h.CompleteTask(ctxOf(olena), connect.NewRequest(&tasksv1.CompleteTaskRequest{TaskId: task.Id})); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	if got := taskReminders(store, task.Id); len(got) != 0 {
		t.Errorf("reminders after complete = %v, want none", got)
	}

	resp, err := h.ReopenTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.ReopenTaskRequest{TaskId: task.Id}))
	if err != nil {
		t.Fatalf("ReopenTask: %v", err)
	}
	if resp.Msg.Task.CompletedByUserId != "" || resp.Msg.Task.CompletedAt != nil {
		t.Errorf("reopened task keeps completion: %q %v", resp.Msg.Task.CompletedByUserId, resp.Msg.Task.CompletedAt)
	}
	occ := "2026-09-01T23:59:59Z"
	got := taskReminders(store, task.Id)
	if _, ok := got[sergiy+" "+occ]; !ok || len(got) != 2 {
		t.Errorf("reminders after reopen = %v, want both assignees at %s", got, occ)
	}

	if _, err := h.DeleteTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.DeleteTaskRequest{TaskId: task.Id})); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if got := taskReminders(store, task.Id); len(got) != 0 {
		t.Errorf("reminders after delete = %v, want none", got)
	}
}

func TestTaskWritesNeedFamilyTimezone(t *testing.T) {
	h, store, _ := newTestHandler(t)
	task := createTask(t, h, &tasksv1.CreateTaskRequest{Title: "x", DueOn: "2026-09-01"})
	if _, err := h.CompleteTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CompleteTaskRequest{TaskId: task.Id})); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	delete(store.settings, testFamily)

	_, err := h.CreateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.CreateTaskRequest{Title: "y", DueOn: "2026-09-02"}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("create: code = %v, want FailedPrecondition", got)
	}
	_, err = h.UpdateTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.UpdateTaskRequest{TaskId: task.Id, DueOn: strPtr("2026-09-05")}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("update: code = %v, want FailedPrecondition", got)
	}
	_, err = h.ReopenTask(ctxOf(sergiy), connect.NewRequest(&tasksv1.ReopenTaskRequest{TaskId: task.Id}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("reopen: code = %v, want FailedPrecondition", got)
	}
	if len(store.tasks) != 1 || store.tasks[task.Id].Status != "done" {
		t.Errorf("writes without a timezone changed the store: %+v", store.tasks)
	}
}

func TestTouchKnownMemberKeepsDisplayName(t *testing.T) {
	h, store, _ := newTestHandler(t)
	key := testFamily + "|" + sergiy

	ctx := fmauth.WithClaims(context.Background(), &fmauth.Claims{
		UserID: sergiy, FamilyID: testFamily, Email: "new@example.com",
	})
	if _, err := h.ListTasks(ctx, connect.NewRequest(&tasksv1.ListTasksRequest{Filter: tasksv1.TaskFilter_TASK_FILTER_ALL})); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if m := store.members[key]; m.DisplayName != "Сергій" || m.Email != "new@example.com" {
		t.Errorf("after touch with email: %q %q", m.DisplayName, m.Email)
	}

	if _, err := h.ListTasks(ctxOf(sergiy), connect.NewRequest(&tasksv1.ListTasksRequest{Filter: tasksv1.TaskFilter_TASK_FILTER_ALL})); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if m := store.members[key]; m.DisplayName != "Сергій" || m.Email != "new@example.com" {
		t.Errorf("after touch without email: %q %q", m.DisplayName, m.Email)
	}

	delete(store.members, testFamily+"|"+olena)
	if _, err := h.ListTasks(ctxOf(olena), connect.NewRequest(&tasksv1.ListTasksRequest{Filter: tasksv1.TaskFilter_TASK_FILTER_ALL})); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if _, ok := store.members[testFamily+"|"+olena]; !ok {
		t.Error("touch did not record a caller missing from known_members")
	}
}

func TestDoneFilterSortsByCompletionNewestFirst(t *testing.T) {
	h, store, _ := newTestHandler(t)
	for i, title := range []string{"first", "second", "third"} {
		task := seedTask(store, title, sergiy)
		task.Status = "done"
		task.CompletedByUserID = pgconv.MustUUID(sergiy)
		task.CompletedAt = pgtype.Timestamptz{Time: testNow.Add(time.Duration(i) * time.Hour), Valid: true}
		store.tasks[id(task.ID)] = task
	}

	resp, err := h.ListTasks(ctxOf(sergiy), connect.NewRequest(&tasksv1.ListTasksRequest{Filter: tasksv1.TaskFilter_TASK_FILTER_DONE}))
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	var got []string
	for _, task := range resp.Msg.Tasks {
		got = append(got, task.Title)
	}
	if len(got) != 3 || got[0] != "third" || got[1] != "second" || got[2] != "first" {
		t.Errorf("done order = %v, want third, second, first", got)
	}
}
