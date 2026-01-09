-- name: CreateTransaction :one
INSERT INTO transactions (
    account_id, category_id, amount, description, transaction_date, type, related_transfer_account_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, account_id, category_id, amount, description, transaction_date, type, created_at;

-- name: GetTransaction :one
SELECT id, account_id, category_id, amount, description, transaction_date, type, created_at
FROM transactions
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListTransactions :many
SELECT id, account_id, category_id, amount, description, transaction_date, type
FROM transactions
WHERE account_id = $1 AND deleted_at IS NULL
ORDER BY transaction_date DESC
LIMIT $2 OFFSET $3;

-- name: UpdateTransaction :one
UPDATE transactions
SET 
    category_id = $2,
    amount = $3,
    description = $4,
    transaction_date = $5,
    type = $6
WHERE id = $1 AND account_id = $7 AND deleted_at IS NULL
RETURNING id, account_id, category_id, amount, description, transaction_date, type;

-- name: DeleteTransaction :exec
UPDATE transactions SET deleted_at = NOW()
WHERE id = $1 AND account_id = $2;