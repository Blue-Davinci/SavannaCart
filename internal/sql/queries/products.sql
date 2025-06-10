-- name: CreateNewProducts :one
INSERT INTO products (
    name,
    price_kes,
    category_id,
    description,
    stock_quantity
) VALUES ($1, $2, $3, $4, $5)
 RETURNING id, version, created_at, updated_at;

-- name: GetAllProductsWithCategory :many
SELECT 
    count(*) OVER() AS total_count,
    p.id,
    p.name,
    p.price_kes,
    p.category_id,
    p.description,
    p.stock_quantity,
    p.version,
    p.created_at,
    p.updated_at,
    -- Category details
    c.id as category_id_info,        -- Category's own ID
    c.name as category_name,
    c.parent_id as category_parent_id -- Category's parent ID (correct!)
FROM products p
LEFT JOIN categories c ON p.category_id = c.id
WHERE ($1 = '' OR to_tsvector('simple', p.name) @@ plainto_tsquery('simple', $1))
ORDER BY p.name
LIMIT $2 OFFSET $3;

-- name: GetProductById :one
SELECT
    id,
    name,
    price_kes,
    category_id,
    description,
    stock_quantity,
    version,
    created_at,
    updated_at
FROM products
WHERE id = $1 AND version = $2;

