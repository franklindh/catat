-- name: GetAccount :one
SELECT id, user_id, name, current_balance, is_main_account, created_at, updated_at
FROM accounts 
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetAccountsByUser :many
SELECT id, user_id, name, current_balance, is_main_account, created_at, updated_at
FROM accounts 
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY is_main_account DESC, name ASC;

-- name: CreateAccount :one
INSERT INTO accounts (user_id, name, current_balance, is_main_account)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, name, current_balance, is_main_account, created_at, updated_at;

-- name: UpdateAccount :one
UPDATE accounts
SET 
    name = $2,
    is_main_account = $3
WHERE id = $1 AND user_id = $4 AND deleted_at IS NULL
RETURNING id, user_id, name, current_balance, is_main_account, created_at, updated_at;

-- name: UpdateAccountBalance :exec
UPDATE accounts
SET current_balance = current_balance + $2
WHERE id = $1;

-- name: DeleteAccount :exec
UPDATE accounts SET deleted_at = NOW()
WHERE id = $1 AND user_id = $2;