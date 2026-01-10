-- name: GetUser :one
SELECT id, email, name, avatar_url, google_auth_id, created_at, updated_at
FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, name, avatar_url, google_auth_id, created_at, updated_at
FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByGoogleAuthID :one
SELECT id, email, name, avatar_url, google_auth_id, created_at, updated_at
FROM users WHERE google_auth_id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (email, name, avatar_url, google_auth_id)
VALUES ($1, $2, $3, $4)
RETURNING id, email, name, avatar_url, google_auth_id, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET 
    name = $2, 
    avatar_url = $3
WHERE id = $1
RETURNING id, email, name, avatar_url, google_auth_id, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;