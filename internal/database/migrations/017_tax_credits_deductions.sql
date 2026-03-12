-- +goose Up

-- Spouse tax credit (sleva na manzela/ku), at most one per year.
CREATE TABLE tax_spouse_credits (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    year             INTEGER NOT NULL,
    spouse_name      TEXT    NOT NULL DEFAULT '',
    spouse_birth_number TEXT NOT NULL DEFAULT '',
    spouse_income    INTEGER NOT NULL DEFAULT 0,
    spouse_ztp       INTEGER NOT NULL DEFAULT 0,
    credit_amount    INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT    NOT NULL,
    updated_at       TEXT    NOT NULL,
    UNIQUE(year)
);

-- Child tax benefit (danove zvyhodneni na dite), 0-N per year.
CREATE TABLE tax_child_credits (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    year           INTEGER NOT NULL,
    child_name     TEXT    NOT NULL DEFAULT '',
    birth_number   TEXT    NOT NULL DEFAULT '',
    child_order    INTEGER NOT NULL DEFAULT 1,
    months_claimed INTEGER NOT NULL DEFAULT 12,
    ztp            INTEGER NOT NULL DEFAULT 0,
    credit_amount  INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT    NOT NULL,
    updated_at     TEXT    NOT NULL
);
CREATE INDEX idx_tax_child_credits_year ON tax_child_credits(year);

-- Personal tax credits (student, disability), at most one per year.
CREATE TABLE tax_personal_credits (
    year              INTEGER PRIMARY KEY,
    is_student        INTEGER NOT NULL DEFAULT 0,
    student_months    INTEGER NOT NULL DEFAULT 0,
    disability_level  INTEGER NOT NULL DEFAULT 0,
    credit_student    INTEGER NOT NULL DEFAULT 0,
    credit_disability INTEGER NOT NULL DEFAULT 0,
    created_at        TEXT    NOT NULL,
    updated_at        TEXT    NOT NULL
);

-- Tax deductions (nezdanitelne casti zakladu dane).
CREATE TABLE tax_deductions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    year           INTEGER NOT NULL,
    category       TEXT    NOT NULL,
    description    TEXT    NOT NULL DEFAULT '',
    claimed_amount INTEGER NOT NULL DEFAULT 0,
    max_amount     INTEGER NOT NULL DEFAULT 0,
    allowed_amount INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT    NOT NULL,
    updated_at     TEXT    NOT NULL
);
CREATE INDEX idx_tax_deductions_year ON tax_deductions(year);

-- Proof documents for tax deductions.
CREATE TABLE tax_deduction_documents (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    tax_deduction_id  INTEGER NOT NULL REFERENCES tax_deductions(id) ON DELETE CASCADE,
    filename          TEXT    NOT NULL,
    content_type      TEXT    NOT NULL,
    storage_path      TEXT    NOT NULL,
    size              INTEGER NOT NULL DEFAULT 0,
    extracted_amount  INTEGER NOT NULL DEFAULT 0,
    confidence        REAL    NOT NULL DEFAULT 0.0,
    created_at        TEXT    NOT NULL,
    deleted_at        TEXT
);
CREATE INDEX idx_tax_deduction_documents_deduction ON tax_deduction_documents(tax_deduction_id);

-- Add total_deductions column to income_tax_returns.
ALTER TABLE income_tax_returns ADD COLUMN total_deductions INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE income_tax_returns DROP COLUMN total_deductions;
DROP TABLE IF EXISTS tax_deduction_documents;
DROP TABLE IF EXISTS tax_deductions;
DROP TABLE IF EXISTS tax_personal_credits;
DROP TABLE IF EXISTS tax_child_credits;
DROP TABLE IF EXISTS tax_spouse_credits;
