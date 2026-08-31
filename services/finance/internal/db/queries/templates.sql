-- Templates are private to their owner: every read is keyed by owner_user_id, so asking for
-- another member's templates returns an empty list rather than an error. They are private,
-- not secret.

-- name: ListTemplates :many
SELECT * FROM quick_templates
WHERE family_id = $1 AND owner_user_id = $2
ORDER BY sort_order, created_at;

-- name: GetTemplate :one
SELECT * FROM quick_templates
WHERE id = $1 AND family_id = $2 AND owner_user_id = $3;

-- name: CreateTemplate :one
INSERT INTO quick_templates (family_id, owner_user_id, label, icon, amount_minor,
    currency_code, type, category_id, account_id, member_id, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    COALESCE((SELECT MAX(sort_order) + 1 FROM quick_templates
              WHERE family_id = $1 AND owner_user_id = $2), 0))
RETURNING *;

-- name: UpdateTemplate :one
UPDATE quick_templates
SET label        = COALESCE(sqlc.narg('label')::text, label),
    icon         = COALESCE(sqlc.narg('icon')::text, icon),
    amount_minor = COALESCE(sqlc.narg('amount_minor')::bigint, amount_minor),
    -- The currency follows the account: a template moved onto a USD card is a USD template,
    -- and without this column the row would keep a code its account no longer holds.
    currency_code = COALESCE(sqlc.narg('currency_code')::text, currency_code),
    category_id  = COALESCE(sqlc.narg('category_id')::uuid, category_id),
    account_id   = COALESCE(sqlc.narg('account_id')::uuid, account_id),
    member_id    = COALESCE(sqlc.narg('member_id')::uuid, member_id),
    updated_at   = NOW()
WHERE id = $1 AND family_id = $2 AND owner_user_id = $3
RETURNING *;

-- name: DeleteTemplate :execrows
DELETE FROM quick_templates
WHERE id = $1 AND family_id = $2 AND owner_user_id = $3;

-- name: ReorderTemplate :exec
UPDATE quick_templates
SET sort_order = $4, updated_at = NOW()
WHERE id = $1 AND family_id = $2 AND owner_user_id = $3;

-- name: RecordTemplateUse :one
UPDATE quick_templates
SET usage_count = usage_count + 1, last_used_at = NOW(), updated_at = NOW()
WHERE id = $1 AND family_id = $2 AND owner_user_id = $3
RETURNING *;
