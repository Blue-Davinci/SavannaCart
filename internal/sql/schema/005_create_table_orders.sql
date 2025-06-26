-- +goose Up
CREATE TABLE orders (
    id            SERIAL PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_kes     NUMERIC(14,2) NOT NULL CHECK (total_kes >= 0),
    status        VARCHAR(30) NOT NULL DEFAULT 'PLACED',
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Update indexes accordingly
CREATE INDEX idx_orders_user          ON orders(user_id);
CREATE INDEX idx_orders_status_created    ON orders(status, created_at DESC);



-- +goose Down
DROP TABLE IF EXISTS orders  CASCADE;
