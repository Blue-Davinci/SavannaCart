-- name: CreateCategory :one
INSERT INTO categories (
    name,
    parent_id
) VALUES ($1, $2)
RETURNING id, name, parent_id, version, created_at, updated_at;

-- name: GetAllCategories :many
SELECT count(*) OVER() AS total_count,
    id,
    name,
    parent_id,
    version,
    created_at,
    updated_at
FROM categories
WHERE ($1 = '' OR to_tsvector('simple', name) @@ plainto_tsquery('simple', $1))
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: GetCategoryById :one
SELECT
    id,
    name,
    parent_id,
    version,
    created_at,
    updated_at
FROM categories
WHERE id = $1 AND version = $2;

-- name: UpdateCategory :one
UPDATE categories
SET
    name = $2,
    parent_id = $3,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1 AND version = $4
RETURNING name, parent_id, version, updated_at;

-- name: DeleteCategory :one
DELETE FROM categories
WHERE id = $1
RETURNING id;