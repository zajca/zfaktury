-- +goose Up

-- Spouse credit: months at whose start a dependent child under 3 lived in the
-- common household (§ 35bb odst. 2, applies from 2024). Existing records keep 0,
-- which caps the credit at 0 for 2024+ until the months are filled in.
ALTER TABLE tax_spouse_credits ADD COLUMN child_under3_months INTEGER NOT NULL DEFAULT 0;

-- Disability credits can be claimed for part of the year (§ 35ba odst. 3).
-- Existing disability data keeps the full year.
ALTER TABLE tax_personal_credits ADD COLUMN disability_months INTEGER NOT NULL DEFAULT 12;
ALTER TABLE tax_personal_credits ADD COLUMN ztpp_months INTEGER NOT NULL DEFAULT 12;

-- +goose Down

ALTER TABLE tax_personal_credits DROP COLUMN ztpp_months;
ALTER TABLE tax_personal_credits DROP COLUMN disability_months;
ALTER TABLE tax_spouse_credits DROP COLUMN child_under3_months;
