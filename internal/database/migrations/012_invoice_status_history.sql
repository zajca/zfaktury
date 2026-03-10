-- +goose Up
CREATE TABLE invoice_status_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_id INTEGER NOT NULL REFERENCES invoices(id),
    old_status TEXT NOT NULL DEFAULT '',
    new_status TEXT NOT NULL,
    changed_at TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_invoice_status_history_invoice_id ON invoice_status_history(invoice_id);

-- +goose Down
DROP TABLE IF EXISTS invoice_status_history;
