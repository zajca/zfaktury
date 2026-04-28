-- +goose Up
-- Distinguish between broker statements (subject to AI extraction) and raw
-- data exports (xlsx/csv/zip), which are merely attached to the DPFO bundle.
ALTER TABLE investment_documents ADD COLUMN kind TEXT NOT NULL DEFAULT 'statement';

-- +goose Down
ALTER TABLE investment_documents DROP COLUMN kind;
