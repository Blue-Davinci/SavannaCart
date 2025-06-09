-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,                -- Unique user ID
    first_name VARCHAR(50) NOT NULL,         -- First name 
    last_name VARCHAR(50) NOT NULL,          -- Last name
    email CITEXT UNIQUE NOT NULL,            -- Case-insensitive email, must be unique
    profile_avatar_url TEXT NOT NULL,        -- URL to user's profile picture
    password BYTEA NOT NULL,                 -- Securely stored password hash (room for expansion in case of direct login rather than OAuth)
    oidc_sub TEXT UNIQUE NOT NULL,        -- OpenID Connect subject identifier, unique for each user
    role_level TEXT NOT NULL DEFAULT 'regular', -- User role (e.g., admin, user, moderator)
    activated BOOLEAN DEFAULT FALSE NOT NULL, -- Account activation status (email confirmation)
    version INTEGER DEFAULT 1 NOT NULL,      -- Record versioning for optimistic locking
    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(), -- Timestamp of account creation
    updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(), -- Timestamp for last update (e.g., profile changes)
    last_login TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW() -- Track the user's last login time
);

-- +goose StatementBegin
-- Create the reusable trigger function for updated_at
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER 
LANGUAGE plpgsql
AS $$
BEGIN
  -- Update the updated_at field before any row update
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- Attach the trigger to the `users` table
CREATE TRIGGER trigger_update_users_timestamp
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

-- Create indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_last_login ON users(last_login);
CREATE INDEX idx_users_role_level ON users(role_level);

-- +goose Down
-- Drop the trigger for the `users` table
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trigger_update_users_timestamp ON users;
DROP FUNCTION IF EXISTS update_timestamp();  -- Only drop if no other table depends on it!
-- +goose StatementEnd

DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_phone_number;
DROP INDEX IF EXISTS idx_users_last_login;


DROP TABLE users;

