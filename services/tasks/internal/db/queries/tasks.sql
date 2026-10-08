-- name: CreateTask :one
INSERT INTO tasks (family_id, title, notes, priority, due_on, due_time, due_at, created_by_user_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = $1 AND family_id = $2;

-- name: GetTaskForUpdate :one
SELECT * FROM tasks
WHERE id = $1 AND family_id = $2
FOR UPDATE;

-- name: UpdateTask :one
UPDATE tasks
SET title = $3, notes = $4, priority = $5, due_on = $6, due_time = $7, due_at = $8, updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: CompleteTask :one
UPDATE tasks
SET status = 'done', completed_at = sqlc.arg(completed_at), completed_by_user_id = sqlc.arg(completed_by_user_id), updated_at = NOW()
WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND status = 'open'
RETURNING *;

-- name: ReopenTask :one
UPDATE tasks
SET status = 'open', completed_at = NULL, completed_by_user_id = NULL, updated_at = NOW()
WHERE id = $1 AND family_id = $2 AND status = 'done'
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks
WHERE id = $1 AND family_id = $2;

-- name: ListTasks :many
SELECT t.* FROM tasks t
WHERE t.family_id = sqlc.arg(family_id)
  AND (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status)::text)
  AND (sqlc.narg(assignee)::uuid IS NULL OR EXISTS (
        SELECT 1 FROM task_assignees a
        WHERE a.task_id = t.id AND a.family_id = t.family_id AND a.user_id = sqlc.narg(assignee)::uuid))
ORDER BY t.due_at NULLS LAST,
         CASE t.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
         t.created_at, t.id;

-- name: ListAssignees :many
SELECT task_id, user_id FROM task_assignees
WHERE family_id = sqlc.arg(family_id) AND task_id = ANY(sqlc.arg(task_ids)::uuid[])
ORDER BY task_id, user_id;

-- name: AddAssignee :exec
INSERT INTO task_assignees (task_id, family_id, user_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: DeleteAssignees :exec
DELETE FROM task_assignees
WHERE task_id = $1 AND family_id = $2;
