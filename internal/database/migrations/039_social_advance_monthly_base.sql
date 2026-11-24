-- +goose Up

-- Monthly assessment base for the new advance (ř. 35 of the ČSSZ overview).
-- NULL means the row was calculated before this column existed, when the
-- advance base was the final annual base divided by the months of activity.
ALTER TABLE social_insurance_overviews ADD COLUMN new_monthly_base INTEGER;

-- +goose Down

ALTER TABLE social_insurance_overviews DROP COLUMN new_monthly_base;
