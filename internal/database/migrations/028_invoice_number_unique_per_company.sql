-- Make invoice_number unique PER COMPANY instead of globally.
--
-- Migration 001 declared `invoice_number TEXT NOT NULL UNIQUE`, a global
-- constraint. In a multi-company setup two distinct issuers (companies) may
-- legitimately produce the same invoice-number string; the global UNIQUE wrongly
-- blocks that and breaks automatic generation across companies. SQLite cannot
-- drop an inline column constraint, so the invoices table is rebuilt without the
-- global UNIQUE and with a composite UNIQUE(company_id, invoice_number) instead.
--
-- Runs without goose's transaction wrapper so foreign_keys can be toggled (it is
-- a no-op inside a transaction); the rebuild itself is wrapped in an explicit
-- transaction.

-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE invoices_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sequence_id INTEGER REFERENCES invoice_sequences(id),
    invoice_number TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'regular' CHECK (type IN ('regular', 'proforma', 'credit_note')),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'sent', 'paid', 'overdue', 'cancelled')),
    issue_date TEXT NOT NULL,
    due_date TEXT NOT NULL,
    delivery_date TEXT,
    variable_symbol TEXT,
    constant_symbol TEXT,
    customer_id INTEGER NOT NULL REFERENCES contacts(id),
    currency_code TEXT NOT NULL DEFAULT 'CZK',
    exchange_rate INTEGER NOT NULL DEFAULT 100,
    payment_method TEXT NOT NULL DEFAULT 'bank_transfer' CHECK (payment_method IN ('bank_transfer', 'cash', 'card', 'other')),
    bank_account TEXT,
    bank_code TEXT,
    iban TEXT,
    swift TEXT,
    subtotal_amount INTEGER NOT NULL DEFAULT 0,
    vat_amount INTEGER NOT NULL DEFAULT 0,
    total_amount INTEGER NOT NULL DEFAULT 0,
    paid_amount INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    internal_notes TEXT,
    sent_at TEXT,
    paid_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at TEXT,
    related_invoice_id INTEGER REFERENCES invoices(id),
    relation_type TEXT NOT NULL DEFAULT '',
    company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id),
    UNIQUE (company_id, invoice_number)
);
INSERT INTO invoices_new (
    id, sequence_id, invoice_number, type, status, issue_date, due_date, delivery_date,
    variable_symbol, constant_symbol, customer_id, currency_code, exchange_rate, payment_method,
    bank_account, bank_code, iban, swift, subtotal_amount, vat_amount, total_amount, paid_amount,
    notes, internal_notes, sent_at, paid_at, created_at, updated_at, deleted_at,
    related_invoice_id, relation_type, company_id
)
SELECT
    id, sequence_id, invoice_number, type, status, issue_date, due_date, delivery_date,
    variable_symbol, constant_symbol, customer_id, currency_code, exchange_rate, payment_method,
    bank_account, bank_code, iban, swift, subtotal_amount, vat_amount, total_amount, paid_amount,
    notes, internal_notes, sent_at, paid_at, created_at, updated_at, deleted_at,
    related_invoice_id, relation_type, company_id
FROM invoices;
DROP TABLE invoices;
ALTER TABLE invoices_new RENAME TO invoices;
CREATE INDEX idx_invoices_customer_id ON invoices(customer_id);
CREATE INDEX idx_invoices_sequence_id ON invoices(sequence_id);
CREATE INDEX idx_invoices_type ON invoices(type);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_issue_date ON invoices(issue_date);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_invoice_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_variable_symbol ON invoices(variable_symbol);
CREATE INDEX idx_invoices_deleted_at ON invoices(deleted_at);
CREATE INDEX idx_invoices_related_invoice_id ON invoices(related_invoice_id);
CREATE INDEX idx_invoices_company ON invoices(company_id);
CREATE UNIQUE INDEX idx_invoices_company_id ON invoices(company_id, id);
COMMIT;
PRAGMA foreign_keys=ON;

-- +goose Down
PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE invoices_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sequence_id INTEGER REFERENCES invoice_sequences(id),
    invoice_number TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL DEFAULT 'regular' CHECK (type IN ('regular', 'proforma', 'credit_note')),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'sent', 'paid', 'overdue', 'cancelled')),
    issue_date TEXT NOT NULL,
    due_date TEXT NOT NULL,
    delivery_date TEXT,
    variable_symbol TEXT,
    constant_symbol TEXT,
    customer_id INTEGER NOT NULL REFERENCES contacts(id),
    currency_code TEXT NOT NULL DEFAULT 'CZK',
    exchange_rate INTEGER NOT NULL DEFAULT 100,
    payment_method TEXT NOT NULL DEFAULT 'bank_transfer' CHECK (payment_method IN ('bank_transfer', 'cash', 'card', 'other')),
    bank_account TEXT,
    bank_code TEXT,
    iban TEXT,
    swift TEXT,
    subtotal_amount INTEGER NOT NULL DEFAULT 0,
    vat_amount INTEGER NOT NULL DEFAULT 0,
    total_amount INTEGER NOT NULL DEFAULT 0,
    paid_amount INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    internal_notes TEXT,
    sent_at TEXT,
    paid_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at TEXT,
    related_invoice_id INTEGER REFERENCES invoices(id),
    relation_type TEXT NOT NULL DEFAULT '',
    company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id)
);
INSERT INTO invoices_old (
    id, sequence_id, invoice_number, type, status, issue_date, due_date, delivery_date,
    variable_symbol, constant_symbol, customer_id, currency_code, exchange_rate, payment_method,
    bank_account, bank_code, iban, swift, subtotal_amount, vat_amount, total_amount, paid_amount,
    notes, internal_notes, sent_at, paid_at, created_at, updated_at, deleted_at,
    related_invoice_id, relation_type, company_id
)
SELECT
    id, sequence_id, invoice_number, type, status, issue_date, due_date, delivery_date,
    variable_symbol, constant_symbol, customer_id, currency_code, exchange_rate, payment_method,
    bank_account, bank_code, iban, swift, subtotal_amount, vat_amount, total_amount, paid_amount,
    notes, internal_notes, sent_at, paid_at, created_at, updated_at, deleted_at,
    related_invoice_id, relation_type, company_id
FROM invoices;
DROP TABLE invoices;
ALTER TABLE invoices_old RENAME TO invoices;
CREATE INDEX idx_invoices_customer_id ON invoices(customer_id);
CREATE INDEX idx_invoices_sequence_id ON invoices(sequence_id);
CREATE INDEX idx_invoices_type ON invoices(type);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_issue_date ON invoices(issue_date);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_invoice_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_variable_symbol ON invoices(variable_symbol);
CREATE INDEX idx_invoices_deleted_at ON invoices(deleted_at);
CREATE INDEX idx_invoices_related_invoice_id ON invoices(related_invoice_id);
CREATE INDEX idx_invoices_company ON invoices(company_id);
CREATE UNIQUE INDEX idx_invoices_company_id ON invoices(company_id, id);
COMMIT;
PRAGMA foreign_keys=ON;
