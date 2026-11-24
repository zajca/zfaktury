-- +goose Up

-- Filing method used at the last recalculation; the XML section H is written from it.
ALTER TABLE social_insurance_overviews ADD COLUMN filing_method TEXT NOT NULL DEFAULT ''
    CHECK (filing_method IN ('', 'standard', 'electronic', 'advisor'));
-- Month of the payment year (overview year + 1) the new advance starts; 0 = not
-- calculated since this column exists.
ALTER TABLE social_insurance_overviews ADD COLUMN advance_start_month INTEGER NOT NULL DEFAULT 0
    CHECK (advance_start_month BETWEEN 0 AND 12);
-- JSON array of {from_month, monthly_base, monthly_prepay} in halere; '' = none.
ALTER TABLE social_insurance_overviews ADD COLUMN advance_periods TEXT NOT NULL DEFAULT '';
-- Comma-separated warning tokens of the last calculation.
ALTER TABLE social_insurance_overviews ADD COLUMN warnings TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE social_insurance_overviews DROP COLUMN warnings;
ALTER TABLE social_insurance_overviews DROP COLUMN advance_periods;
ALTER TABLE social_insurance_overviews DROP COLUMN advance_start_month;
ALTER TABLE social_insurance_overviews DROP COLUMN filing_method;
