-- +goose Up
-- Add phone number column to users table
ALTER TABLE users ADD COLUMN phone_number VARCHAR(20);

-- Create index for phone number for faster lookups
CREATE INDEX idx_users_phone_number ON users(phone_number);

-- +goose Down
-- Remove the index and column
DROP INDEX IF EXISTS idx_users_phone_number;
ALTER TABLE users DROP COLUMN IF EXISTS phone_number;