-- name: CreateUser :one
INSERT INTO "users" (
  email, 
  name,
  password
) VALUES (
  $1, $2, $3
)
RETURNING id, email, name, password, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, name, password, created_at, updated_at FROM "users"
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, name, password, created_at, updated_at FROM "users"
WHERE id = $1;

-- name: UpdateUser :one
UPDATE "users"
SET
  email = $2,
  name = $3,
  updated_at = now()
WHERE id = $1
RETURNING id, email, name, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM "users"
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, name, created_at, updated_at FROM "users"
ORDER BY id
LIMIT $1 OFFSET $2;