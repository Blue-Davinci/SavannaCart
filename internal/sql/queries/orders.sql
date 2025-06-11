
-- name: CreateOrder :one
INSERT INTO orders (
    user_id,
    total_kes,
    status
) VALUES ($1, $2, $3)
RETURNING id, user_id, total_kes, status, version, created_at, updated_at;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    order_id,
    product_id,
    quantity,
    unit_price_kes
) VALUES ($1, $2, $3, $4)
RETURNING id, order_id, product_id, quantity, unit_price_kes, created_at;

-- name: GetAllOrdersWithItems :many
SELECT 
    o.id as order_id,
    o.user_id,
    u.first_name as user_first_name,
    u.last_name as user_last_name,
    u.email as user_email,
    o.total_kes,
    o.status,
    o.version,
    o.created_at as order_created_at,
    o.updated_at as order_updated_at,
    oi.id as order_item_id,
    oi.product_id,
    p.name as product_name,
    oi.quantity,
    oi.unit_price_kes,
    oi.created_at as item_created_at,
    count(*) OVER() AS total_count
FROM orders o
INNER JOIN users u ON o.user_id = u.id
LEFT JOIN order_items oi ON o.id = oi.order_id
LEFT JOIN products p ON oi.product_id = p.id
WHERE ($1 = '' OR o.status = $1)
ORDER BY o.created_at DESC, oi.id ASC
LIMIT $2 OFFSET $3;

-- name: GetUserOrdersWithItems :many
SELECT 
    o.id as order_id,
    o.user_id,
    o.total_kes,
    o.status,
    o.version,
    o.created_at as order_created_at,
    o.updated_at as order_updated_at,
    oi.id as order_item_id,
    oi.product_id,
    p.name as product_name,
    oi.quantity,
    oi.unit_price_kes,
    oi.created_at as item_created_at,
    count(*) OVER() AS total_count
FROM orders o
LEFT JOIN order_items oi ON o.id = oi.order_id
LEFT JOIN products p ON oi.product_id = p.id
WHERE o.user_id = $1
ORDER BY o.created_at DESC, oi.id ASC
LIMIT $2 OFFSET $3;

-- name: GetOrderById :one
SELECT 
    id,
    user_id,
    total_kes,
    status,
    version,
    created_at,
    updated_at
FROM orders
WHERE id = $1;

-- name: GetOrderByIdWithItems :many
SELECT 
    o.id as order_id,
    o.user_id,
    o.total_kes,
    o.status,
    o.version,
    o.created_at as order_created_at,
    o.updated_at as order_updated_at,
    oi.id as order_item_id,
    oi.product_id,
    p.name as product_name,
    p.price_kes as current_price,
    oi.quantity,
    oi.unit_price_kes,
    oi.created_at as item_created_at
FROM orders o
LEFT JOIN order_items oi ON o.id = oi.order_id
LEFT JOIN products p ON oi.product_id = p.id
WHERE o.id = $1
ORDER BY oi.id ASC;

-- name: UpdateOrderStatus :one
UPDATE orders
SET 
    status = $2,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1 AND version = $3
RETURNING id, user_id, total_kes, status, version, created_at, updated_at;

-- name: GetOrderStatistics :one
SELECT 
    COUNT(*) as total_orders,
    COUNT(CASE WHEN status = 'PLACED' THEN 1 END) as placed_orders,
    COUNT(CASE WHEN status = 'PROCESSING' THEN 1 END) as processing_orders,
    COUNT(CASE WHEN status = 'SHIPPED' THEN 1 END) as shipped_orders,
    COUNT(CASE WHEN status = 'DELIVERED' THEN 1 END) as delivered_orders,
    COUNT(CASE WHEN status = 'CANCELLED' THEN 1 END) as cancelled_orders,
    COALESCE(SUM(total_kes), 0)::text as total_revenue,
    COALESCE(AVG(total_kes), 0)::text as average_order_value
FROM orders
WHERE created_at >= $1 AND created_at <= $2;

-- name: CheckProductAvailability :one
SELECT 
    id,
    name,
    stock_quantity,
    CASE 
        WHEN stock_quantity >= $2 THEN true
        ELSE false
    END as is_available
FROM products
WHERE id = $1;
