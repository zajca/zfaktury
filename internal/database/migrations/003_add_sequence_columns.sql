-- +goose Up
-- +goose NO TRANSACTION

-- Add updated_at and deleted_at columns to invoice_sequences for soft delete support.
CREATE TABLE invoice_sequences_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    prefix TEXT NOT NULL,
    next_number INTEGER NOT NULL DEFAULT 1,
    year INTEGER NOT NULL,
    format_pattern TEXT NOT NULL DEFAULT '{prefix}{year}{number:04d}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at TEXT,
    UNIQUE(prefix, year)
);

INSERT INTO invoice_sequences_new (id, prefix, next_number, year, format_pattern, created_at, updated_at)
SELECT id, prefix, next_number, year, format_pattern, created_at, created_at
FROM invoice_sequences;

DROP TABLE invoice_sequences;
ALTER TABLE invoice_sequences_new RENAME TO invoice_sequences;

CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year);
CREATE INDEX idx_invoice_sequences_deleted_at ON invoice_sequences(deleted_at);

-- +goose Down
-- +goose NO TRANSACTION

CREATE TABLE invoice_sequences_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    prefix TEXT NOT NULL,
    next_number INTEGER NOT NULL DEFAULT 1,
    year INTEGER NOT NULL,
    format_pattern TEXT NOT NULL DEFAULT '{prefix}{year}{number:04d}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(prefix, year)
);

INSERT INTO invoice_sequences_old (id, prefix, next_number, year, format_pattern, created_at)
SELECT id, prefix, next_number, year, format_pattern, created_at
FROM invoice_sequences;

DROP TABLE invoice_sequences;
ALTER TABLE invoice_sequences_old RENAME TO invoice_sequences;

CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year);
