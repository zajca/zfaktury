-- +goose Up
ALTER TABLE expenses ADD COLUMN tax_reviewed_at TEXT;

-- +goose Down
-- SQLite doesn't support DROP COLUMN in older versions
-- This is a no-op for safety
