package handler

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/services/tasks/db"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
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
		DueOn: pgtype.Date{Time: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		DueTime: pgtype.Time{Microseconds: 10*3600*1e6, Valid: true},
		DueAt: pgtype.Timestamptz{Time: time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC), Valid: true},
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

	// Check event published
	if !rec.sawSubject("tasks.task.assigned") {
		t.Errorf("subjects = %v, want tasks.task.assigned", rec.subjects)
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
		DueOn: pgtype.Date{Time: baseTime.AddDate(0, 0, 1), Valid: true},
		DueAt: pgtype.Timestamptz{Time: baseTime.AddDate(0, 0, 1), Valid: true},
		CreatedByUserID: t1.CreatedByUserID,
		CreatedAt: pgtype.Timestamptz{Time: baseTime, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: baseTime, Valid: true},
	}

	t2 := seedTask(store, "High priority, due later", sergiy)
	s.tasks[id(t2.ID)] = db.Task{
		ID: t2.ID, FamilyID: t2.FamilyID, Title: t2.Title, Status: "open", Priority: "high",
		DueOn: pgtype.Date{Time: baseTime.AddDate(0, 0, 5), Valid: true},
		DueAt: pgtype.Timestamptz{Time: baseTime.AddDate(0, 0, 5), Valid: true},
		CreatedByUserID: t2.CreatedByUserID,
		CreatedAt: pgtype.Timestamptz{Time: baseTime.Add(time.Hour), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: baseTime, Valid: true},
	}

	t3 := seedTask(store, "No deadline", sergiy)
	s.tasks[id(t3.ID)] = db.Task{
		ID: t3.ID, FamilyID: t3.FamilyID, Title: t3.Title, Status: "open", Priority: "medium",
		DueOn: pgtype.Date{}, DueAt: pgtype.Timestamptz{},
		CreatedByUserID: t3.CreatedByUserID,
		CreatedAt: pgtype.Timestamptz{Time: baseTime.Add(2*time.Hour), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: baseTime, Valid: true},
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