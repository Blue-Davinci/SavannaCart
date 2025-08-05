-- Create categories table
CREATE TABLE
    categories (
        id SERIAL PRIMARY KEY,
        name VARCHAR(120) NOT NULL,
        parent_id INTEGER REFERENCES categories (id) ON DELETE SET NULL,
        version INTEGER NOT NULL DEFAULT 1,
        created_at TIMESTAMP(0)
        WITH
            TIME ZONE NOT NULL DEFAULT NOW (),
            updated_at TIMESTAMP(0)
        WITH
            TIME ZONE NOT NULL DEFAULT NOW ()
    );

-- Enforce unique name *within* the same parent branch (case‑insensitive)  
CREATE UNIQUE INDEX ux_categories_name_parent ON categories (LOWER(name), COALESCE(parent_id, 0));

-- Fast look‑ups for hierarchy traversal
CREATE INDEX idx_categories_parent ON categories (parent_id);
