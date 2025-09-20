-- name: CreateUser :one
INSERT INTO users (
  email, 
  name,
  password
) VALUES (
  $1, $2, $3
)
RETURNING id, email, name, created_at;

-- name: GetUserByEmail :one
SELECT id, email, name, created_at, updated_at FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, name, created_at, updated_at FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET
  email = COALESCE(sqlc.narg(email), email),
  name = COALESCE(sqlc.narg(name), name),
  password = COALESCE(sqlc.narg(password), password),
  password_changed_at = COALESCE(sqlc.narg(password_changed_at), password_changed_at),
  updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING id, email, name, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, name, created_at, updated_at FROM users
ORDER BY id
LIMIT $1 OFFSET $2;