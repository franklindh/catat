-- name: GetTransaction :one
SELECT id, user_id, category_id, amount, balance_after, description, transaction_date, created_at
FROM transactions
WHERE id = $1 LIMIT 1;

-- name: GetTransactionsByUser :many
SELECT id, user_id, category_id, amount, balance_after, description, transaction_date, created_at
FROM transactions
WHERE user_id = $1
ORDER BY transaction_date DESC;

-- name: GetTransactionsByUserAndCategory :many
SELECT id, user_id, category_id, amount, balance_after, description, transaction_date, created_at
FROM transactions
WHERE user_id = $1 AND category_id = $2
ORDER BY transaction_date DESC;

-- name: GetTransactionsByUserInDateRange :many
SELECT id, user_id, category_id, amount, balance_after, description, transaction_date, created_at
FROM transactions
WHERE user_id = $1 AND transaction_date >= $2 AND transaction_date <= $3
ORDER BY transaction_date DESC;

-- name: CreateTransaction :one
INSERT INTO transactions (user_id, category_id, amount, balance_after, description, transaction_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, category_id, amount, balance_after, description, transaction_date, created_at;

-- name: UpdateTransaction :exec
UPDATE transactions
SET category_id = $2, amount = $3, balance_after = $4, description = $5, transaction_date = $6
WHERE id = $1 AND user_id = $7;

-- name: DeleteTransaction :exec
DELETE FROM transactions
WHERE id = $1 AND user_id = $2;