-- name: CreateCategory :one
INSERT INTO "categories" (
  user_id,
  name,
  type,
  parent_id
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetCategory :one
SELECT * FROM "categories"
WHERE id = $1 AND user_id = $2;

-- name: ListCategories :many
SELECT * FROM "categories"
WHERE user_id = $1 AND parent_id IS NULL
ORDER BY name;

-- name: ListSubCategories :many
SELECT * FROM "categories"
WHERE user_id = $1 AND parent_id = $2
ORDER BY name;

-- name: UpdateCategory :one
UPDATE "categories"
SET
  name = $2,
  type = $3,
  parent_id = $4,
  updated_at = now()
WHERE
  id = $1 AND user_id = $5
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM "categories"
WHERE id = $1 AND user_id = $2;