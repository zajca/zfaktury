CREATE TABLE recurring_expenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    vendor_id INTEGER REFERENCES contacts(id),
    category TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL,
    amount INTEGER NOT NULL,
    currency_code TEXT NOT NULL DEFAULT 'CZK',
    exchange_rate INTEGER NOT NULL DEFAULT 0,
    vat_rate_percent INTEGER NOT NULL DEFAULT 0,
    vat_amount INTEGER NOT NULL DEFAULT 0,
    is_tax_deductible INTEGER NOT NULL DEFAULT 1,
    business_percent INTEGER NOT NULL DEFAULT 100,
    payment_method TEXT NOT NULL DEFAULT 'bank_transfer',
    notes TEXT NOT NULL DEFAULT '',
    frequency TEXT NOT NULL DEFAULT 'monthly',
    next_issue_date TEXT NOT NULL,
    end_date TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at TEXT
);

ALTER TABLE expenses ADD COLUMN recurring_expense_id INTEGER REFERENCES recurring_expenses(id);
