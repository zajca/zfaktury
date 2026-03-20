-- Income Tax Returns (Danove priznani - DPFO)
CREATE TABLE income_tax_returns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    filing_type TEXT NOT NULL DEFAULT 'regular',

    total_revenue INTEGER NOT NULL DEFAULT 0,
    actual_expenses INTEGER NOT NULL DEFAULT 0,
    flat_rate_percent INTEGER NOT NULL DEFAULT 0,
    flat_rate_amount INTEGER NOT NULL DEFAULT 0,
    used_expenses INTEGER NOT NULL DEFAULT 0,

    tax_base INTEGER NOT NULL DEFAULT 0,
    tax_base_rounded INTEGER NOT NULL DEFAULT 0,

    tax_at_15 INTEGER NOT NULL DEFAULT 0,
    tax_at_23 INTEGER NOT NULL DEFAULT 0,
    total_tax INTEGER NOT NULL DEFAULT 0,

    credit_basic INTEGER NOT NULL DEFAULT 0,
    credit_spouse INTEGER NOT NULL DEFAULT 0,
    credit_disability INTEGER NOT NULL DEFAULT 0,
    credit_student INTEGER NOT NULL DEFAULT 0,
    total_credits INTEGER NOT NULL DEFAULT 0,
    tax_after_credits INTEGER NOT NULL DEFAULT 0,

    child_benefit INTEGER NOT NULL DEFAULT 0,
    tax_after_benefit INTEGER NOT NULL DEFAULT 0,

    prepayments INTEGER NOT NULL DEFAULT 0,
    tax_due INTEGER NOT NULL DEFAULT 0,

    xml_data BLOB,
    status TEXT NOT NULL DEFAULT 'draft',
    filed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,

    UNIQUE(year, filing_type)
);

-- Audit trail: which invoices are included in an income tax return
CREATE TABLE income_tax_return_invoices (
    income_tax_return_id INTEGER NOT NULL REFERENCES income_tax_returns(id) ON DELETE CASCADE,
    invoice_id INTEGER NOT NULL REFERENCES invoices(id),
    PRIMARY KEY (income_tax_return_id, invoice_id)
);

-- Audit trail: which expenses are included in an income tax return
CREATE TABLE income_tax_return_expenses (
    income_tax_return_id INTEGER NOT NULL REFERENCES income_tax_returns(id) ON DELETE CASCADE,
    expense_id INTEGER NOT NULL REFERENCES expenses(id),
    PRIMARY KEY (income_tax_return_id, expense_id)
);

-- Social Insurance Overviews (Prehled OSVC pro CSSZ)
CREATE TABLE social_insurance_overviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    filing_type TEXT NOT NULL DEFAULT 'regular',

    total_revenue INTEGER NOT NULL DEFAULT 0,
    total_expenses INTEGER NOT NULL DEFAULT 0,
    tax_base INTEGER NOT NULL DEFAULT 0,
    assessment_base INTEGER NOT NULL DEFAULT 0,
    min_assessment_base INTEGER NOT NULL DEFAULT 0,
    final_assessment_base INTEGER NOT NULL DEFAULT 0,

    insurance_rate INTEGER NOT NULL DEFAULT 292,
    total_insurance INTEGER NOT NULL DEFAULT 0,
    prepayments INTEGER NOT NULL DEFAULT 0,
    difference INTEGER NOT NULL DEFAULT 0,
    new_monthly_prepay INTEGER NOT NULL DEFAULT 0,

    xml_data BLOB,
    status TEXT NOT NULL DEFAULT 'draft',
    filed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,

    UNIQUE(year, filing_type)
);

-- Health Insurance Overviews (Prehled OSVC pro ZP)
CREATE TABLE health_insurance_overviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    filing_type TEXT NOT NULL DEFAULT 'regular',

    total_revenue INTEGER NOT NULL DEFAULT 0,
    total_expenses INTEGER NOT NULL DEFAULT 0,
    tax_base INTEGER NOT NULL DEFAULT 0,
    assessment_base INTEGER NOT NULL DEFAULT 0,
    min_assessment_base INTEGER NOT NULL DEFAULT 0,
    final_assessment_base INTEGER NOT NULL DEFAULT 0,

    insurance_rate INTEGER NOT NULL DEFAULT 135,
    total_insurance INTEGER NOT NULL DEFAULT 0,
    prepayments INTEGER NOT NULL DEFAULT 0,
    difference INTEGER NOT NULL DEFAULT 0,
    new_monthly_prepay INTEGER NOT NULL DEFAULT 0,

    xml_data BLOB,
    status TEXT NOT NULL DEFAULT 'draft',
    filed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,

    UNIQUE(year, filing_type)
);
