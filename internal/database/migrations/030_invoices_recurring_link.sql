-- +goose Up
-- Link an invoice back to the recurring template that generated it, so the
-- auto-send sweep can find a template's still-unsent (draft) invoices and email
-- them on the next scheduler run (decoupling sending from generation and giving
-- failed sends an automatic retry). NULL for manually created invoices.
ALTER TABLE invoices ADD COLUMN recurring_invoice_id INTEGER REFERENCES recurring_invoices(id);

CREATE INDEX idx_invoices_recurring_invoice_id ON invoices(recurring_invoice_id);

-- +goose Down
DROP INDEX idx_invoices_recurring_invoice_id;
ALTER TABLE invoices DROP COLUMN recurring_invoice_id;
