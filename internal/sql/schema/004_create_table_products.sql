-- +goose Up
CREATE TABLE
    products (
        id SERIAL PRIMARY KEY,
        name VARCHAR(160) NOT NULL,
        price_kes NUMERIC(12, 2) NOT NULL CHECK (price_kes >= 0),
        category_id INTEGER NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,
        description TEXT,
        stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
        version INTEGER NOT NULL DEFAULT 1,
        created_at TIMESTAMP(0)
        WITH
            TIME ZONE NOT NULL DEFAULT NOW (),
            updated_at TIMESTAMP(0)
        WITH
            TIME ZONE NOT NULL DEFAULT NOW ()
    );

-- Ensure no duplicate product names within the same category  
CREATE UNIQUE INDEX ux_products_name_cat ON products (LOWER(name), category_id);

-- Speed up frequent filters
CREATE INDEX idx_products_category ON products (category_id);

CREATE INDEX idx_products_price ON products (price_kes);

-- +goose Down
DROP TABLE IF EXISTS products CASCADE;