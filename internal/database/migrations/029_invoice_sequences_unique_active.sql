-- +goose Up

-- The table-level UNIQUE(prefix, year) also counted soft-deleted rows, so a
-- deleted sequence could never be recreated. Uniqueness is enforced only among
-- live rows by a partial unique index instead.
-- SQLite cannot drop a table constraint in place, so the table is rebuilt.
-- Foreign keys are disabled around migrations (see database.Migrate), so
-- invoices.sequence_id keeps pointing at the same ids after the rename.

CREATE TABLE invoice_sequences_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    prefix TEXT NOT NULL,
    next_number INTEGER NOT NULL DEFAULT 1,
    year INTEGER NOT NULL,
    format_pattern TEXT NOT NULL DEFAULT '{prefix}{year}{number:04d}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at TEXT
);

INSERT INTO invoice_sequences_new (id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at)
    SELECT id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at FROM invoice_sequences;

DROP TABLE invoice_sequences;
ALTER TABLE invoice_sequences_new RENAME TO invoice_sequences;

CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year);
CREATE INDEX idx_invoice_sequences_deleted_at ON invoice_sequences(deleted_at);
CREATE UNIQUE INDEX idx_invoice_sequences_prefix_year_active ON invoice_sequences(prefix, year) WHERE deleted_at IS NULL;

-- +goose Down

-- Restores the table-level UNIQUE(prefix, year). This fails when a live row and
-- a soft-deleted row share the same prefix and year; remove one of them first.

CREATE TABLE invoice_sequences_old (
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

INSERT INTO invoice_sequences_old (id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at)
    SELECT id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at FROM invoice_sequences;

DROP TABLE invoice_sequences;
ALTER TABLE invoice_sequences_old RENAME TO invoice_sequences;

CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year);
CREATE INDEX idx_invoice_sequences_deleted_at ON invoice_sequences(deleted_at);
