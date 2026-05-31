-- +goose Up

ALTER TABLE income_tax_returns ADD COLUMN tax_ruleset_id TEXT NOT NULL DEFAULT '';
ALTER TABLE income_tax_returns ADD COLUMN tax_ruleset_status TEXT NOT NULL DEFAULT '';
ALTER TABLE income_tax_returns ADD COLUMN tax_ruleset_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE income_tax_returns ADD COLUMN calculated_at TEXT;

-- +goose Down

ALTER TABLE income_tax_returns DROP COLUMN calculated_at;
ALTER TABLE income_tax_returns DROP COLUMN tax_ruleset_hash;
ALTER TABLE income_tax_returns DROP COLUMN tax_ruleset_status;
ALTER TABLE income_tax_returns DROP COLUMN tax_ruleset_id;
