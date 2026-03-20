-- Convert vat_unreliable (INTEGER 0/1) to vat_unreliable_at (TEXT nullable timestamp).
-- Existing rows with vat_unreliable=1 get current timestamp; 0 becomes NULL.
ALTER TABLE contacts ADD COLUMN vat_unreliable_at TEXT;
UPDATE contacts SET vat_unreliable_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE vat_unreliable = 1;
