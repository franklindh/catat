-- name: CreateTransaction :one
INSERT INTO transactions (
    user_id, category_id, amount, description, transaction_date, type
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, user_id, category_id, amount, description, transaction_date, type, created_at;

-- name: GetTransaction :one
SELECT id, user_id, category_id, amount, description, transaction_date, type
FROM transactions
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL LIMIT 1;

-- name: ListTransactions :many
SELECT id, user_id, category_id, amount, description, transaction_date, type
FROM transactions
WHERE user_id = $1 AND deleted_at IS NULL
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
WHERE id = $1 AND user_id = $7 AND deleted_at IS NULL
RETURNING id, user_id, category_id, amount, description, transaction_date, type;

-- name: DeleteTransaction :exec
UPDATE transactions SET deleted_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: GetUserBalance :one
SELECT 
    COALESCE(SUM(CASE WHEN type = 'INCOME' THEN amount ELSE 0 END), 0) - 
    COALESCE(SUM(CASE WHEN type = 'EXPENSE' THEN amount ELSE 0 END), 0) 
    AS current_balance
FROM transactions
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetExpenseByDateRange :one
SELECT COALESCE(SUM(amount), 0) as total_expense
FROM transactions
WHERE user_id = $1 
  AND type = 'EXPENSE' 
  AND transaction_date BETWEEN $2 AND $3
  AND deleted_at IS NULL;