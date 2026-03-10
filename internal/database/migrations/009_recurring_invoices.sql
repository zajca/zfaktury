-- +goose Up
CREATE TABLE recurring_invoices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    customer_id INTEGER NOT NULL REFERENCES contacts(id),
    frequency TEXT NOT NULL DEFAULT 'monthly',
    next_issue_date TEXT NOT NULL,
    end_date TEXT,
    currency_code TEXT NOT NULL DEFAULT 'CZK',
    exchange_rate INTEGER NOT NULL DEFAULT 0,
    payment_method TEXT NOT NULL DEFAULT 'bank_transfer',
    bank_account TEXT NOT NULL DEFAULT '',
    bank_code TEXT NOT NULL DEFAULT '',
    iban TEXT NOT NULL DEFAULT '',
    swift TEXT NOT NULL DEFAULT '',
    constant_symbol TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at TEXT
);

CREATE TABLE recurring_invoice_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recurring_invoice_id INTEGER NOT NULL REFERENCES recurring_invoices(id),
    description TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    unit TEXT NOT NULL DEFAULT 'ks',
    unit_price INTEGER NOT NULL,
    vat_rate_percent INTEGER NOT NULL DEFAULT 21,
    sort_order INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS recurring_invoice_items;
DROP TABLE IF EXISTS recurring_invoices;
