-- name: GetUser :one
SELECT id, google_id, email, name, balance, avatar_url, created_at, updated_at
FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByGoogleID :one
SELECT id, google_id, email, name, balance, avatar_url, created_at, updated_at
FROM users
WHERE google_id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, google_id, email, name, balance, avatar_url, created_at, updated_at
FROM users
WHERE email = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (google_id, email, name, balance, avatar_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, google_id, email, name, balance, avatar_url, created_at, updated_at;

-- name: UpdateUser :exec
UPDATE users
SET name = $2, avatar_url = $3, updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserBalance :exec
UPDATE users
SET balance = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;