-- +goose Up
ALTER TABLE tax_spouse_credits ADD COLUMN months_claimed INTEGER NOT NULL DEFAULT 12;

-- +goose Down
ALTER TABLE tax_spouse_credits DROP COLUMN months_claimed;
