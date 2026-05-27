-- +goose Up
-- +goose StatementBegin

-- Replace the table-level UNIQUE(company_id, prefix, year) on
-- invoice_sequences with a partial unique index that only covers active
-- (non-soft-deleted) rows. Without this, deleting a sequence leaves the
-- (company_id, prefix, year) slot locked forever — the next attempt to
-- create one with the same prefix and year hits the table constraint even
-- though the service-layer uniqueness check (which filters by
-- deleted_at IS NULL) lets the request through.
--
-- SQLite cannot drop a table-level constraint in place, so we rebuild the
-- table the same way migration 025 did, and then add the partial unique
-- index. We do NOT rename the original table out of the way first, because
-- modern SQLite auto-updates FK references in invoices.sequence_id on
-- RENAME and a stale rename would leave that reference dangling.

CREATE TABLE invoice_sequences__new (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id     INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id),
	prefix         TEXT    NOT NULL,
	next_number    INTEGER NOT NULL DEFAULT 1,
	year           INTEGER NOT NULL,
	format_pattern TEXT    NOT NULL DEFAULT '{prefix}{year}{number:04d}',
	created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	updated_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	deleted_at     TEXT
);

INSERT INTO invoice_sequences__new (id, company_id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at)
SELECT id, company_id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at FROM invoice_sequences;

DROP TABLE invoice_sequences;
ALTER TABLE invoice_sequences__new RENAME TO invoice_sequences;

CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year);
CREATE INDEX idx_invoice_sequences_deleted_at ON invoice_sequences(deleted_at);
CREATE INDEX idx_invoice_sequences_company ON invoice_sequences(company_id);

CREATE UNIQUE INDEX idx_invoice_sequences_company_prefix_year_active
	ON invoice_sequences(company_id, prefix, year)
	WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse: restore the table-level UNIQUE constraint. This fails if soft-
-- deleted rows overlap an active row on (company_id, prefix, year). That is
-- acceptable; Down migrations are best-effort.

DROP INDEX IF EXISTS idx_invoice_sequences_company_prefix_year_active;
DROP INDEX IF EXISTS idx_invoice_sequences_company;
DROP INDEX IF EXISTS idx_invoice_sequences_deleted_at;
DROP INDEX IF EXISTS idx_invoice_sequences_year;

CREATE TABLE invoice_sequences__new (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id     INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id),
	prefix         TEXT    NOT NULL,
	next_number    INTEGER NOT NULL DEFAULT 1,
	year           INTEGER NOT NULL,
	format_pattern TEXT    NOT NULL DEFAULT '{prefix}{year}{number:04d}',
	created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	updated_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	deleted_at     TEXT,
	UNIQUE(company_id, prefix, year)
);

INSERT INTO invoice_sequences__new (id, company_id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at)
SELECT id, company_id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at FROM invoice_sequences;

DROP TABLE invoice_sequences;
ALTER TABLE invoice_sequences__new RENAME TO invoice_sequences;

CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year);
CREATE INDEX idx_invoice_sequences_deleted_at ON invoice_sequences(deleted_at);
CREATE INDEX idx_invoice_sequences_company ON invoice_sequences(company_id);

-- +goose StatementEnd
