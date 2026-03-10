-- +goose Up
ALTER TABLE invoices ADD COLUMN related_invoice_id INTEGER REFERENCES invoices(id);
ALTER TABLE invoices ADD COLUMN relation_type TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_invoices_related_invoice_id ON invoices(related_invoice_id);

-- +goose Down
-- SQLite doesn't support DROP COLUMN easily, so we only remove the index.
DROP INDEX IF EXISTS idx_invoices_related_invoice_id;
