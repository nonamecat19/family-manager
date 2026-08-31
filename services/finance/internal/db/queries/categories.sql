-- name: ListCategoryGroups :many
SELECT * FROM category_groups
WHERE family_id = $1
  AND (sqlc.narg('kind')::text IS NULL OR kind = @kind)
  AND (sqlc.arg('include_archived')::bool OR NOT archived)
ORDER BY sort_order, created_at;

-- name: GetCategoryGroup :one
SELECT * FROM category_groups
WHERE id = $1 AND family_id = $2;

-- name: CreateCategoryGroup :one
INSERT INTO category_groups (family_id, name, kind, icon, color_step, sort_order)
VALUES ($1, $2, $3, $4, $5,
    COALESCE((SELECT MAX(sort_order) + 1 FROM category_groups WHERE family_id = $1), 0))
RETURNING *;

-- name: UpdateCategoryGroup :one
UPDATE category_groups
SET name       = COALESCE(sqlc.narg('name')::text, name),
    icon       = COALESCE(sqlc.narg('icon')::text, icon),
    color_step = COALESCE(sqlc.narg('color_step')::int, color_step),
    archived   = COALESCE(sqlc.narg('archived')::bool, archived),
    updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteCategoryGroup :execrows
DELETE FROM category_groups
WHERE id = $1 AND family_id = $2;

-- name: ReorderCategoryGroup :exec
UPDATE category_groups
SET sort_order = $3, updated_at = NOW()
WHERE id = $1 AND family_id = $2;

-- name: CountCategoriesInGroup :one
SELECT COUNT(*) FROM categories
WHERE group_id = $1 AND family_id = $2;

-- MoveCategoriesToGroup is the reassignment DeleteCategoryGroup requires: a group holding
-- categories that hold transactions cannot silently vanish, so its categories are re-parented
-- first and the delete is refused if that did not happen.
-- name: MoveCategoriesToGroup :execrows
UPDATE categories
SET group_id = $3, updated_at = NOW()
WHERE group_id = $1 AND family_id = $2;

-- name: ListCategories :many
SELECT * FROM categories
WHERE family_id = $1
  AND (sqlc.narg('group_id')::uuid IS NULL OR group_id = @group_id)
  AND (sqlc.narg('kind')::text IS NULL OR kind = @kind)
  AND (sqlc.arg('include_archived')::bool OR NOT archived)
ORDER BY sort_order, created_at;

-- name: GetCategory :one
SELECT * FROM categories
WHERE id = $1 AND family_id = $2;

-- name: CreateCategory :one
INSERT INTO categories (family_id, group_id, name, kind, icon, sort_order)
VALUES ($1, $2, $3, $4, $5,
    COALESCE((SELECT MAX(sort_order) + 1 FROM categories WHERE group_id = $2), 0))
RETURNING *;

-- name: UpdateCategory :one
UPDATE categories
SET name       = COALESCE(sqlc.narg('name')::text, name),
    icon       = COALESCE(sqlc.narg('icon')::text, icon),
    archived   = COALESCE(sqlc.narg('archived')::bool, archived),
    updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: MoveCategory :one
UPDATE categories
SET group_id = $3, updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE id = $1 AND family_id = $2;

-- ReorderCategory renumbers one category inside the group the client named. The group is part
-- of the predicate rather than a thing the handler trusts the list to agree with: a batch that
-- mixed in an id from another group would otherwise renumber a grid nobody was looking at.
-- name: ReorderCategory :execrows
UPDATE categories
SET sort_order = $3, updated_at = NOW()
WHERE id = $1 AND family_id = $2 AND group_id = sqlc.arg('group_id')::uuid;

-- name: MoveTransactionsToCategory :execrows
UPDATE transactions
SET category_id = sqlc.narg('target_category_id')::uuid, updated_at = NOW()
WHERE category_id = $1 AND family_id = $2;

-- name: CountCategoryTransactions :one
SELECT COUNT(*) FROM transactions
WHERE category_id = $1 AND family_id = $2;
