-- name: GetUser :one
SELECT id, email, name, avatar_url, google_auth_id, role, created_at, updated_at
FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, name, avatar_url, google_auth_id, role, created_at, updated_at
FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByGoogleAuthID :one
SELECT id, email, name, avatar_url, google_auth_id, role, created_at, updated_at
FROM users WHERE google_auth_id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (email, name, avatar_url, google_auth_id)
VALUES ($1, $2, $3, $4)
RETURNING id, email, name, avatar_url, google_auth_id, role, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET 
    name = $2, 
    avatar_url = $3
WHERE id = $1
RETURNING id, email, name, avatar_url, google_auth_id, role, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, name, avatar_url, google_auth_id, role, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: UpdateUserRole :one
UPDATE users
SET role = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, email, name, avatar_url, google_auth_id, role, created_at, updated_at;

-- name: SearchUsers :many
SELECT id, email, name, avatar_url, google_auth_id, role, created_at, updated_at
FROM users
WHERE 
    email ILIKE '%' || $1 || '%' OR
    name ILIKE '%' || $1 || '%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;