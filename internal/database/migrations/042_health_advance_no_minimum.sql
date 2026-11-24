-- +goose Up

-- Filing method used at the last recalculation (decides the advance start month).
ALTER TABLE health_insurance_overviews ADD COLUMN filing_method TEXT NOT NULL DEFAULT ''
    CHECK (filing_method IN ('', 'standard', 'electronic', 'advisor'));
-- Month of the payment year (overview year + 1) the new advance starts; 0 = not calculated.
ALTER TABLE health_insurance_overviews ADD COLUMN advance_start_month INTEGER NOT NULL DEFAULT 0
    CHECK (advance_start_month BETWEEN 0 AND 12);
-- Comma-separated warning tokens of the last calculation.
ALTER TABLE health_insurance_overviews ADD COLUMN warnings TEXT NOT NULL DEFAULT '';
-- Months of activity in which the minimum base does not apply (§ 3a odst. 3 zákona
-- č. 592/1992 Sb.), bit 0 = January; 0 = none.
ALTER TABLE health_insurance_overviews ADD COLUMN no_minimum_months INTEGER NOT NULL DEFAULT 0
    CHECK (no_minimum_months BETWEEN 0 AND 4095);
ALTER TABLE health_insurance_overviews ADD COLUMN no_minimum_reason TEXT NOT NULL DEFAULT ''
    CHECK (no_minimum_reason IN ('', 'employee', 'state_insured', 'ztp', 'retirement_age'));

-- +goose Down

ALTER TABLE health_insurance_overviews DROP COLUMN no_minimum_reason;
ALTER TABLE health_insurance_overviews DROP COLUMN no_minimum_months;
ALTER TABLE health_insurance_overviews DROP COLUMN warnings;
ALTER TABLE health_insurance_overviews DROP COLUMN advance_start_month;
ALTER TABLE health_insurance_overviews DROP COLUMN filing_method;
