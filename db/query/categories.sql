-- name: GetCategory :one
SELECT id, user_id, name, type, icon_url, created_at, updated_at 
FROM categories 
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetCategoriesByUser :many
SELECT id, user_id, name, type, icon_url, created_at, updated_at 
FROM categories 
WHERE user_id = $1 AND type = $2 AND deleted_at IS NULL 
ORDER BY name;

-- name: CreateCategory :one
INSERT INTO categories (user_id, name, type, icon_url)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, name, type, icon_url, created_at, updated_at;

-- name: UpdateCategory :one
UPDATE categories
SET 
    name = $2,
    icon_url = $3,
    type = $4
WHERE id = $1 AND user_id = $5 AND deleted_at IS NULL
RETURNING id, user_id, name, type, icon_url, created_at, updated_at;

-- name: DeleteCategory :exec
UPDATE categories SET deleted_at = NOW()
WHERE id = $1 AND user_id = $2;