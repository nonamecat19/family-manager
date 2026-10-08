-- name: CreateBirthday :one
INSERT INTO birthdays (family_id, name, day, month, year, remind_days_before, created_by_user_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetBirthday :one
SELECT * FROM birthdays
WHERE id = $1 AND family_id = $2;

-- name: UpdateBirthday :one
UPDATE birthdays
SET name = $3, day = $4, month = $5, year = $6, remind_days_before = $7, updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteBirthday :execrows
DELETE FROM birthdays
WHERE id = $1 AND family_id = $2;

-- name: ListBirthdays :many
SELECT * FROM birthdays
WHERE family_id = $1
ORDER BY month, day, name, id;

-- name: ListAllBirthdays :many
SELECT * FROM birthdays
ORDER BY family_id, month, day, id;
