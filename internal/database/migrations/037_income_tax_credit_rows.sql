-- +goose Up
ALTER TABLE tax_personal_credits ADD COLUMN ztpp INTEGER NOT NULL DEFAULT 0;
-- Legacy disability level 3 is the ZTP/P flag.
UPDATE tax_personal_credits SET ztpp = 1, disability_level = 0 WHERE disability_level = 3;
ALTER TABLE income_tax_returns ADD COLUMN credit_disability_12 INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN credit_disability_3 INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN credit_disability_ztpp INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE income_tax_returns DROP COLUMN credit_disability_ztpp;
ALTER TABLE income_tax_returns DROP COLUMN credit_disability_3;
ALTER TABLE income_tax_returns DROP COLUMN credit_disability_12;
-- Lossy: a degree combined with ZTP/P keeps only the degree.
UPDATE tax_personal_credits SET disability_level = 3 WHERE ztpp = 1 AND disability_level = 0;
ALTER TABLE tax_personal_credits DROP COLUMN ztpp;
