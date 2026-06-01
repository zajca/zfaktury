-- +goose Up
-- Let a recurring invoice template pick which invoice-number sequence its
-- generated invoices use. NULL / 0 keeps the previous behaviour (auto-assign the
-- company's "FV" sequence), so existing templates are unaffected until edited.
ALTER TABLE recurring_invoices ADD COLUMN sequence_id INTEGER REFERENCES invoice_sequences(id);

-- +goose Down
ALTER TABLE recurring_invoices DROP COLUMN sequence_id;
