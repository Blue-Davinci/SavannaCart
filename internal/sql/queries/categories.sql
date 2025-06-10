-- name: CreateCategory :one
INSERT INTO categories (
    name,
    parent_id
) VALUES ($1, $2)
RETURNING id, name, parent_id, version, created_at, updated_at;

-- name: GetAllCategories :many
SELECT
    id,
    name,
    parent_id,
    version,
    created_at,
    updated_at
FROM categories
ORDER BY name;

-- name: UpdateCategory :one
UPDATE categories
SET
    name = $2,
    parent_id = $3,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1 AND version = $4
RETURNING name, parent_id, version, updated_at;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = $1;