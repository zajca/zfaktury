-- Contacts: customers, vendors, and other business entities
CREATE TABLE contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL DEFAULT 'company' CHECK (type IN ('company', 'individual')),
    name TEXT NOT NULL,
    ico TEXT,
    dic TEXT,
    street TEXT,
    city TEXT,
    zip TEXT,
    country TEXT NOT NULL DEFAULT 'CZ',
    email TEXT,
    phone TEXT,
    web TEXT,
    bank_account TEXT,
    bank_code TEXT,
    iban TEXT,
    swift TEXT,
    payment_terms_days INTEGER NOT NULL DEFAULT 14,
    tags TEXT,
    notes TEXT,
    is_favorite INTEGER NOT NULL DEFAULT 0,
    vat_unreliable INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at TEXT
);

CREATE INDEX idx_contacts_type ON contacts(type);
CREATE INDEX idx_contacts_ico ON contacts(ico);
CREATE INDEX idx_contacts_name ON contacts(name);
CREATE INDEX idx_contacts_deleted_at ON contacts(deleted_at);

-- Invoice sequences: numbering patterns per year
CREATE TABLE invoice_sequences (
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

CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year);
CREATE INDEX idx_invoice_sequences_deleted_at ON invoice_sequences(deleted_at);

-- Invoices: issued and received invoices
CREATE TABLE invoices (
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
    deleted_at TEXT
);

CREATE INDEX idx_invoices_customer_id ON invoices(customer_id);
CREATE INDEX idx_invoices_sequence_id ON invoices(sequence_id);
CREATE INDEX idx_invoices_type ON invoices(type);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_issue_date ON invoices(issue_date);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_invoice_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_variable_symbol ON invoices(variable_symbol);
CREATE INDEX idx_invoices_deleted_at ON invoices(deleted_at);

-- Invoice items: line items on an invoice
CREATE TABLE invoice_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_id INTEGER NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 100,
    unit TEXT NOT NULL DEFAULT 'ks',
    unit_price INTEGER NOT NULL DEFAULT 0,
    vat_rate_percent INTEGER NOT NULL DEFAULT 0,
    vat_amount INTEGER NOT NULL DEFAULT 0,
    total_amount INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);

-- Expenses: business expenses and costs
CREATE TABLE expenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    vendor_id INTEGER REFERENCES contacts(id),
    expense_number TEXT,
    category TEXT,
    description TEXT NOT NULL,
    issue_date TEXT NOT NULL,
    amount INTEGER NOT NULL DEFAULT 0,
    currency_code TEXT NOT NULL DEFAULT 'CZK',
    exchange_rate INTEGER NOT NULL DEFAULT 100,
    vat_rate_percent INTEGER NOT NULL DEFAULT 0,
    vat_amount INTEGER NOT NULL DEFAULT 0,
    is_tax_deductible INTEGER NOT NULL DEFAULT 1,
    business_percent INTEGER NOT NULL DEFAULT 100,
    payment_method TEXT NOT NULL DEFAULT 'bank_transfer' CHECK (payment_method IN ('bank_transfer', 'cash', 'card', 'other')),
    document_path TEXT,
    notes TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at TEXT
);

CREATE INDEX idx_expenses_vendor_id ON expenses(vendor_id);
CREATE INDEX idx_expenses_category ON expenses(category);
CREATE INDEX idx_expenses_issue_date ON expenses(issue_date);
CREATE INDEX idx_expenses_deleted_at ON expenses(deleted_at);

-- Audit log: tracks changes to entities
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete')),
    old_values TEXT,
    new_values TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

-- Settings: key-value store for application settings
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
