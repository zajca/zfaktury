-- +goose Up

-- Uploaded broker statements
CREATE TABLE investment_documents (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    year               INTEGER NOT NULL,
    platform           TEXT NOT NULL DEFAULT 'other',
    filename           TEXT NOT NULL,
    content_type       TEXT NOT NULL,
    storage_path       TEXT NOT NULL,
    size               INTEGER NOT NULL DEFAULT 0,
    extraction_status  TEXT NOT NULL DEFAULT 'pending',
    extraction_error   TEXT NOT NULL DEFAULT '',
    created_at         TEXT NOT NULL,
    updated_at         TEXT NOT NULL
);

-- §8 Capital income entries
CREATE TABLE capital_income_entries (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    year                 INTEGER NOT NULL,
    document_id          INTEGER REFERENCES investment_documents(id),
    category             TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    income_date          TEXT NOT NULL,
    gross_amount         INTEGER NOT NULL DEFAULT 0,
    withheld_tax_cz      INTEGER NOT NULL DEFAULT 0,
    withheld_tax_foreign INTEGER NOT NULL DEFAULT 0,
    country_code         TEXT NOT NULL DEFAULT '',
    needs_declaring      INTEGER NOT NULL DEFAULT 0,
    net_amount           INTEGER NOT NULL DEFAULT 0,
    created_at           TEXT NOT NULL,
    updated_at           TEXT NOT NULL
);

-- §10 Security/crypto transactions
CREATE TABLE security_transactions (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    year              INTEGER NOT NULL DEFAULT 0,
    document_id       INTEGER REFERENCES investment_documents(id),
    asset_type        TEXT NOT NULL,
    asset_name        TEXT NOT NULL,
    isin              TEXT NOT NULL DEFAULT '',
    transaction_type  TEXT NOT NULL,
    transaction_date  TEXT NOT NULL,
    quantity          INTEGER NOT NULL DEFAULT 0,
    unit_price        INTEGER NOT NULL DEFAULT 0,
    total_amount      INTEGER NOT NULL DEFAULT 0,
    fees              INTEGER NOT NULL DEFAULT 0,
    currency_code     TEXT NOT NULL DEFAULT 'CZK',
    exchange_rate     INTEGER NOT NULL DEFAULT 10000,
    cost_basis        INTEGER NOT NULL DEFAULT 0,
    computed_gain     INTEGER NOT NULL DEFAULT 0,
    time_test_exempt  INTEGER NOT NULL DEFAULT 0,
    exempt_amount     INTEGER NOT NULL DEFAULT 0,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
);

-- Indexes
CREATE INDEX idx_capital_income_year ON capital_income_entries(year);
CREATE INDEX idx_security_tx_year ON security_transactions(year);
CREATE INDEX idx_security_tx_asset ON security_transactions(asset_name, asset_type);
CREATE INDEX idx_investment_docs_year ON investment_documents(year);

-- Extend income_tax_returns with investment income fields
ALTER TABLE income_tax_returns ADD COLUMN capital_income_gross INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN capital_income_tax INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN capital_income_net INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN other_income_gross INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN other_income_expenses INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN other_income_exempt INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN other_income_net INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE income_tax_returns DROP COLUMN capital_income_gross;
ALTER TABLE income_tax_returns DROP COLUMN capital_income_tax;
ALTER TABLE income_tax_returns DROP COLUMN capital_income_net;
ALTER TABLE income_tax_returns DROP COLUMN other_income_gross;
ALTER TABLE income_tax_returns DROP COLUMN other_income_expenses;
ALTER TABLE income_tax_returns DROP COLUMN other_income_exempt;
ALTER TABLE income_tax_returns DROP COLUMN other_income_net;
DROP TABLE IF EXISTS security_transactions;
DROP TABLE IF EXISTS capital_income_entries;
DROP TABLE IF EXISTS investment_documents;
