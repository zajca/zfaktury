-- +goose Up
CREATE TABLE invoice_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_id INTEGER NOT NULL REFERENCES invoices(id),
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    deleted_at TEXT
);
CREATE INDEX idx_invoice_documents_invoice_id ON invoice_documents(invoice_id) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_invoice_documents_invoice_id;
DROP TABLE IF EXISTS invoice_documents;
