-- +goose Up
-- Convert vat_unreliable (INTEGER 0/1) to vat_unreliable_at (TEXT nullable timestamp).
-- Existing rows with vat_unreliable=1 get current timestamp; 0 becomes NULL.
ALTER TABLE contacts ADD COLUMN vat_unreliable_at TEXT;
UPDATE contacts SET vat_unreliable_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE vat_unreliable = 1;

-- SQLite cannot drop columns before 3.35.0, so we keep vat_unreliable but stop using it.
-- New code reads/writes only vat_unreliable_at.

-- +goose Down
UPDATE contacts SET vat_unreliable = 1 WHERE vat_unreliable_at IS NOT NULL;
UPDATE contacts SET vat_unreliable = 0 WHERE vat_unreliable_at IS NULL;
ALTER TABLE contacts DROP COLUMN vat_unreliable_at;
