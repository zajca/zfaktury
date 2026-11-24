-- +goose Up

-- How the income-tax return for the year is filed; it decides the overview
-- deadline (§ 136 daňového řádu). '' = not set, which ČSSZ treats as filed by 1 April.
ALTER TABLE tax_year_settings ADD COLUMN filing_method TEXT NOT NULL DEFAULT ''
    CHECK (filing_method IN ('', 'standard', 'electronic', 'advisor'));

-- +goose Down

ALTER TABLE tax_year_settings DROP COLUMN filing_method;
