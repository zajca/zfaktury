-- +goose Up
ALTER TABLE income_tax_returns ADD COLUMN deduction_mortgage INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN deduction_life_insurance INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN deduction_pension INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN deduction_donation INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN deduction_union_dues INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE income_tax_returns DROP COLUMN deduction_mortgage;
ALTER TABLE income_tax_returns DROP COLUMN deduction_life_insurance;
ALTER TABLE income_tax_returns DROP COLUMN deduction_pension;
ALTER TABLE income_tax_returns DROP COLUMN deduction_donation;
ALTER TABLE income_tax_returns DROP COLUMN deduction_union_dues;
