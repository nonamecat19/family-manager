-- Widgets are per-user placements, not household state: two members place the same type and
-- each sees their own scope, so every read is keyed by user_id as well as family_id.

-- name: ListWidgets :many
SELECT * FROM widget_instances
WHERE family_id = $1 AND user_id = $2
ORDER BY sort_order, created_at;

-- name: ListWidgetsByIDs :many
SELECT * FROM widget_instances
WHERE family_id = $1 AND user_id = $2 AND id = ANY(sqlc.arg('widget_ids')::uuid[])
ORDER BY sort_order, created_at;

-- name: GetWidget :one
SELECT * FROM widget_instances
WHERE id = $1 AND family_id = $2 AND user_id = $3;

-- name: CreateWidget :one
INSERT INTO widget_instances (family_id, user_id, type, size, scope_kind, scope_member_id,
    scope_account_id, target_ref, target_account_ids, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
    COALESCE((SELECT MAX(sort_order) + 1 FROM widget_instances
              WHERE family_id = $1 AND user_id = $2), 0))
RETURNING *;

-- name: UpdateWidget :one
UPDATE widget_instances
SET size             = COALESCE(sqlc.narg('size')::text, size),
    scope_kind       = COALESCE(sqlc.narg('scope_kind')::text, scope_kind),
    scope_member_id  = COALESCE(sqlc.narg('scope_member_id')::uuid, scope_member_id),
    scope_account_id = COALESCE(sqlc.narg('scope_account_id')::uuid, scope_account_id),
    target_ref       = COALESCE(sqlc.narg('target_ref')::uuid, target_ref),
    target_account_ids = COALESCE(sqlc.narg('target_account_ids')::uuid[], target_account_ids),
    updated_at       = NOW()
WHERE id = $1 AND family_id = $2 AND user_id = $3
RETURNING *;

-- name: DeleteWidget :execrows
DELETE FROM widget_instances
WHERE id = $1 AND family_id = $2 AND user_id = $3;
