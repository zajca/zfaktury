-- +goose Up

-- VAT Returns (Priznani k DPH)
CREATE TABLE vat_returns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL DEFAULT 0,
    quarter INTEGER NOT NULL DEFAULT 0,
    filing_type TEXT NOT NULL DEFAULT 'regular',

    output_vat_base_21 INTEGER NOT NULL DEFAULT 0,
    output_vat_amount_21 INTEGER NOT NULL DEFAULT 0,
    output_vat_base_12 INTEGER NOT NULL DEFAULT 0,
    output_vat_amount_12 INTEGER NOT NULL DEFAULT 0,
    output_vat_base_0 INTEGER NOT NULL DEFAULT 0,

    reverse_charge_base_21 INTEGER NOT NULL DEFAULT 0,
    reverse_charge_amount_21 INTEGER NOT NULL DEFAULT 0,
    reverse_charge_base_12 INTEGER NOT NULL DEFAULT 0,
    reverse_charge_amount_12 INTEGER NOT NULL DEFAULT 0,

    input_vat_base_21 INTEGER NOT NULL DEFAULT 0,
    input_vat_amount_21 INTEGER NOT NULL DEFAULT 0,
    input_vat_base_12 INTEGER NOT NULL DEFAULT 0,
    input_vat_amount_12 INTEGER NOT NULL DEFAULT 0,

    total_output_vat INTEGER NOT NULL DEFAULT 0,
    total_input_vat INTEGER NOT NULL DEFAULT 0,
    net_vat INTEGER NOT NULL DEFAULT 0,

    xml_data BLOB,
    status TEXT NOT NULL DEFAULT 'draft',
    filed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,

    UNIQUE(year, month, quarter, filing_type)
);

-- Audit trail: which invoices are included in a VAT return
CREATE TABLE vat_return_invoices (
    vat_return_id INTEGER NOT NULL REFERENCES vat_returns(id) ON DELETE CASCADE,
    invoice_id INTEGER NOT NULL REFERENCES invoices(id),
    PRIMARY KEY (vat_return_id, invoice_id)
);

-- Audit trail: which expenses are included in a VAT return
CREATE TABLE vat_return_expenses (
    vat_return_id INTEGER NOT NULL REFERENCES vat_returns(id) ON DELETE CASCADE,
    expense_id INTEGER NOT NULL REFERENCES expenses(id),
    PRIMARY KEY (vat_return_id, expense_id)
);

-- VAT Control Statements (Kontrolni hlaseni)
CREATE TABLE vat_control_statements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL,
    filing_type TEXT NOT NULL DEFAULT 'regular',

    xml_data BLOB,
    status TEXT NOT NULL DEFAULT 'draft',
    filed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,

    UNIQUE(year, month, filing_type)
);

-- Control statement lines (per-transaction detail)
CREATE TABLE vat_control_statement_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    control_statement_id INTEGER NOT NULL REFERENCES vat_control_statements(id) ON DELETE CASCADE,
    section TEXT NOT NULL,
    partner_dic TEXT NOT NULL,
    document_number TEXT NOT NULL,
    dppd TEXT NOT NULL,
    base INTEGER NOT NULL DEFAULT 0,
    vat INTEGER NOT NULL DEFAULT 0,
    vat_rate_percent INTEGER NOT NULL DEFAULT 0,
    invoice_id INTEGER REFERENCES invoices(id),
    expense_id INTEGER REFERENCES expenses(id)
);

CREATE INDEX idx_control_lines_statement ON vat_control_statement_lines(control_statement_id);

-- VIES Summaries (Souhrnne hlaseni)
CREATE TABLE vies_summaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    quarter INTEGER NOT NULL,
    filing_type TEXT NOT NULL DEFAULT 'regular',

    xml_data BLOB,
    status TEXT NOT NULL DEFAULT 'draft',
    filed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,

    UNIQUE(year, quarter, filing_type)
);

-- VIES summary lines (per-partner EU summary)
CREATE TABLE vies_summary_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    vies_summary_id INTEGER NOT NULL REFERENCES vies_summaries(id) ON DELETE CASCADE,
    partner_dic TEXT NOT NULL,
    country_code TEXT NOT NULL,
    total_amount INTEGER NOT NULL DEFAULT 0,
    service_code TEXT NOT NULL DEFAULT '3'
);

CREATE INDEX idx_vies_lines_summary ON vies_summary_lines(vies_summary_id);

-- +goose Down

DROP INDEX IF EXISTS idx_vies_lines_summary;
DROP TABLE IF EXISTS vies_summary_lines;
DROP TABLE IF EXISTS vies_summaries;
DROP INDEX IF EXISTS idx_control_lines_statement;
DROP TABLE IF EXISTS vat_control_statement_lines;
DROP TABLE IF EXISTS vat_control_statements;
DROP TABLE IF EXISTS vat_return_expenses;
DROP TABLE IF EXISTS vat_return_invoices;
DROP TABLE IF EXISTS vat_returns;
