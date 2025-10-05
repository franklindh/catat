-- name: GetTransaction :one
SELECT id, user_id, category_id, amount, description, transaction_date, created_at
FROM transactions
WHERE id = $1 LIMIT 1;

-- name: GetTransactions :many
SELECT id, user_id, category_id, amount, description, transaction_date, created_at
FROM transactions
WHERE user_id = $1
ORDER BY transaction_date DESC
LIMIT $2 
OFFSET $3;

-- name: GetTransactionsByDateRange :many
SELECT id, user_id, category_id, amount, description, transaction_date, created_at
FROM transactions
WHERE user_id = $1 AND transaction_date >= $2 AND transaction_date <= $3
ORDER BY transaction_date DESC
LIMIT $4 
OFFSET $5;

-- name: CreateTransaction :one
INSERT INTO transactions (user_id, category_id, amount, description, transaction_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, category_id, amount, description, transaction_date, created_at;

-- name: UpdateTransaction :one
UPDATE transactions
SET 
  category_id = COALESCE($2, category_id), 
  amount = COALESCE($3, amount), 
  description = COALESCE($4, description), 
  transaction_date = COALESCE($5, transaction_date)
WHERE id = $1 AND user_id = $6
RETURNING id, user_id, category_id, amount, description, transaction_date, created_at;

-- name: DeleteTransaction :exec
DELETE FROM transactions
WHERE id = $1 AND user_id = $2;