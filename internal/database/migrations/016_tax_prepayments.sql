-- +goose Up

-- Per-year tax settings (flat rate)
CREATE TABLE tax_year_settings (
    year INTEGER NOT NULL PRIMARY KEY,
    flat_rate_percent INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Per-month prepayments for each year
CREATE TABLE tax_prepayments (
    year INTEGER NOT NULL,
    month INTEGER NOT NULL CHECK(month >= 1 AND month <= 12),
    tax_amount INTEGER NOT NULL DEFAULT 0,
    social_amount INTEGER NOT NULL DEFAULT 0,
    health_amount INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (year, month)
);

-- Migrate existing global settings to current year (best-effort)
INSERT OR IGNORE INTO tax_year_settings (year, flat_rate_percent)
SELECT 2025, CAST(COALESCE((SELECT value FROM settings WHERE key = 'flat_rate_percent'), '0') AS INTEGER);

-- Remove migrated keys from settings
DELETE FROM settings WHERE key IN ('tax_prepayments', 'social_prepayments', 'health_prepayments', 'flat_rate_percent');

-- +goose Down
INSERT OR IGNORE INTO settings (key, value)
SELECT 'flat_rate_percent', CAST(flat_rate_percent AS TEXT) FROM tax_year_settings WHERE year = 2025;
DROP TABLE IF EXISTS tax_prepayments;
DROP TABLE IF EXISTS tax_year_settings;
