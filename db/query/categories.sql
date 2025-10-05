-- name: GetCategory :one
SELECT id, user_id, name, icon_url, created_at, updated_at
FROM categories
WHERE id = $1 LIMIT 1;

-- name: GetCategoriesByUser :many
SELECT id, user_id, name, icon_url, created_at, updated_at
FROM categories
WHERE user_id = $1
ORDER BY name;

-- name: GetCategoryByName :one
SELECT id, user_id, name, icon_url, created_at, updated_at
FROM categories
WHERE user_id = $1 AND name = $2 LIMIT 1;

-- name: CreateCategory :one
INSERT INTO categories (user_id, name, icon_url)
VALUES ($1, $2, $3)
RETURNING id, user_id, name, icon_url, created_at, updated_at;

-- name: UpdateCategory :one
UPDATE categories
SET 
  name = COALESCE($2, name), 
  icon_url = COALESCE($3, icon_url), 
  updated_at = NOW()
WHERE id = $1 AND user_id = $4
RETURNING id, user_id, name, icon_url, created_at, updated_at; 

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = $1 AND user_id = $2;