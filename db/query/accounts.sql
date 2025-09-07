-- name: CreateAccount :one
INSERT INTO "accounts" (
  user_id,
  name,
  type,
  balance
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM "accounts"
WHERE id = $1 AND user_id = $2;

-- name: ListAccounts :many
SELECT * FROM "accounts"
WHERE user_id = $1
ORDER BY name;

-- name: UpdateAccountBalance :one
UPDATE "accounts"
SET balance = balance + sqlc.arg(amount)
WHERE id = $1
RETURNING *;

-- name: UpdateAccount :one
UPDATE "accounts"
SET
  name = $2,
  type = $3,
  updated_at = now()
WHERE
  id = $1 AND user_id = $4
RETURNING *;

-- name: DeleteAccount :exec
DELETE FROM "accounts"
WHERE id = $1 AND user_id = $2;