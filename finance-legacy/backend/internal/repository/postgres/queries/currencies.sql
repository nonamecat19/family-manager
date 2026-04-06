-- name: GetCurrencyByCode :one
SELECT id, code, type, exchange_rate, name, created_at, updated_at
FROM currencies
WHERE code = $1;

-- name: GetCurrencyByID :one
SELECT id, code, type, exchange_rate, name, created_at, updated_at
FROM currencies
WHERE id = $1;

-- name: GetAllCurrencies :many
SELECT id, code, type, exchange_rate, name, created_at, updated_at
FROM currencies
ORDER BY type, code;

-- name: GetCurrenciesByType :many
SELECT id, code, type, exchange_rate, name, created_at, updated_at
FROM currencies
WHERE type = $1
ORDER BY code;

-- name: UpdateCurrencyExchangeRate :exec
UPDATE currencies
SET exchange_rate = $1, updated_at = NOW()
WHERE code = $2;



