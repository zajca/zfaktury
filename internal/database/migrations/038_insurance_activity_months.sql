-- +goose Up

-- Bit mask of the calendar months with activity (bit 0 = January). Rows
-- before this migration were computed as a full year (all 12 bits).
ALTER TABLE social_insurance_overviews ADD COLUMN activity_months INTEGER NOT NULL DEFAULT 4095;
ALTER TABLE health_insurance_overviews ADD COLUMN activity_months INTEGER NOT NULL DEFAULT 4095;

-- +goose Down

ALTER TABLE health_insurance_overviews DROP COLUMN activity_months;
ALTER TABLE social_insurance_overviews DROP COLUMN activity_months;
