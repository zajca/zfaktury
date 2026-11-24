-- +goose Up

-- Months of secondary activity (vedlejší SVČ), a subset of activity_months; 0 = none.
ALTER TABLE social_insurance_overviews ADD COLUMN secondary_months INTEGER NOT NULL DEFAULT 0
    CHECK (secondary_months BETWEEN 0 AND 4095);
ALTER TABLE social_insurance_overviews ADD COLUMN secondary_reason TEXT NOT NULL DEFAULT ''
    CHECK (secondary_reason IN ('', 'employment', 'pension', 'child_care', 'maternity', 'care', 'student'));
-- Employment assessment base entered by the user (halere); NULL = not entered.
ALTER TABLE social_insurance_overviews ADD COLUMN employment_assessment_base INTEGER;
-- Rows of the ČSSZ overview stored by the calculation so the XML reports one calculation.
ALTER TABLE social_insurance_overviews ADD COLUMN reported_employment_base INTEGER; -- ř. 29, NULL = row empty
ALTER TABLE social_insurance_overviews ADD COLUMN main_tax_base_share INTEGER NOT NULL DEFAULT 0;      -- ř. 24 h
ALTER TABLE social_insurance_overviews ADD COLUMN secondary_tax_base_share INTEGER NOT NULL DEFAULT 0; -- ř. 24 v
ALTER TABLE social_insurance_overviews ADD COLUMN secondary_assessment_base INTEGER NOT NULL DEFAULT 0; -- ř. 25 v
ALTER TABLE social_insurance_overviews ADD COLUMN main_partial_base INTEGER NOT NULL DEFAULT 0;        -- ř. 26 h
ALTER TABLE social_insurance_overviews ADD COLUMN secondary_partial_base INTEGER NOT NULL DEFAULT 0;   -- ř. 26 v
ALTER TABLE social_insurance_overviews ADD COLUMN secondary_decisive_amount INTEGER NOT NULL DEFAULT 0; -- prorated rozhodná částka
ALTER TABLE social_insurance_overviews ADD COLUMN sum_base INTEGER NOT NULL DEFAULT 0;     -- ř. 30
ALTER TABLE social_insurance_overviews ADD COLUMN insured_base INTEGER NOT NULL DEFAULT 0; -- ř. 31
-- Rows before this migration had no employment base, so ř. 30 and ř. 31 equal ř. 28.
UPDATE social_insurance_overviews SET sum_base = final_assessment_base, insured_base = final_assessment_base;

-- +goose Down

ALTER TABLE social_insurance_overviews DROP COLUMN insured_base;
ALTER TABLE social_insurance_overviews DROP COLUMN sum_base;
ALTER TABLE social_insurance_overviews DROP COLUMN secondary_decisive_amount;
ALTER TABLE social_insurance_overviews DROP COLUMN secondary_partial_base;
ALTER TABLE social_insurance_overviews DROP COLUMN main_partial_base;
ALTER TABLE social_insurance_overviews DROP COLUMN secondary_assessment_base;
ALTER TABLE social_insurance_overviews DROP COLUMN secondary_tax_base_share;
ALTER TABLE social_insurance_overviews DROP COLUMN main_tax_base_share;
ALTER TABLE social_insurance_overviews DROP COLUMN reported_employment_base;
ALTER TABLE social_insurance_overviews DROP COLUMN employment_assessment_base;
ALTER TABLE social_insurance_overviews DROP COLUMN secondary_reason;
ALTER TABLE social_insurance_overviews DROP COLUMN secondary_months;
