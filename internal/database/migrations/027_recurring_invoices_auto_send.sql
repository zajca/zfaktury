-- +goose Up
-- Add auto-send capability to recurring invoices. When auto_send is enabled the
-- daily scheduler emails the generated invoice to auto_send_recipient (or, when
-- blank, to the customer's contact email).
ALTER TABLE recurring_invoices ADD COLUMN auto_send INTEGER NOT NULL DEFAULT 0;
ALTER TABLE recurring_invoices ADD COLUMN auto_send_recipient TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE recurring_invoices DROP COLUMN auto_send_recipient;
ALTER TABLE recurring_invoices DROP COLUMN auto_send;
