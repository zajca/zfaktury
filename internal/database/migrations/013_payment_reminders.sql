-- +goose Up
CREATE TABLE payment_reminders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_id INTEGER NOT NULL REFERENCES invoices(id),
    reminder_number INTEGER NOT NULL DEFAULT 1,
    sent_at TEXT NOT NULL,
    sent_to TEXT NOT NULL,
    subject TEXT NOT NULL,
    body_preview TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX idx_payment_reminders_invoice_id ON payment_reminders(invoice_id);

-- +goose Down
DROP TABLE IF EXISTS payment_reminders;
