-- name: CreateTransaction :one
INSERT INTO "transactions" (
  user_id,
  account_id,
  category_id,
  amount,
  description,
  transaction_date
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM "transactions"
WHERE id = $1 AND user_id = $2;

-- name: ListTransactions :many
SELECT * FROM "transactions"
WHERE user_id = $1
ORDER BY transaction_date DESC
LIMIT $2
OFFSET $3;

-- name: ListTransactionsByAccount :many
SELECT * FROM "transactions"
WHERE user_id = $1 AND account_id = $2
ORDER BY transaction_date DESC
LIMIT $3
OFFSET $4;

-- name: ListTransactionsByDateRange :many
SELECT * FROM "transactions"
WHERE
  user_id = $1 AND
  transaction_date >= $2 AND
  transaction_date <= $3
ORDER BY transaction_date DESC;

-- name: UpdateTransaction :one
UPDATE "transactions"
SET
  account_id = $2,
  category_id = $3,
  amount = $4,
  description = $5,
  transaction_date = $6
WHERE
  id = $1 AND user_id = $7
RETURNING *;

-- name: DeleteTransaction :exec
DELETE FROM "transactions"
WHERE id = $1 AND user_id = $2;