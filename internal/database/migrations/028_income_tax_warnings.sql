-- +goose Up
-- Stored as a comma-separated list of short warning tokens
-- (e.g. "progressive_rate_review,withholding_partial_include"). Comma-
-- separated chosen over JSON for simplicity — values are short fixed-set
-- identifiers without nested structure. Empty string ("") = no warnings.
ALTER TABLE income_tax_returns ADD COLUMN warnings TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE income_tax_returns DROP COLUMN warnings;
