-- +goose Up

-- Naskenovaná Potvrzení (PDF/JPG/PNG/WEBP)
CREATE TABLE employment_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    document_kind TEXT NOT NULL DEFAULT 'advance', -- advance | withholding | bonus
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    extraction_status TEXT NOT NULL DEFAULT 'pending', -- pending | extracted | failed
    extraction_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_employment_docs_year ON employment_documents(year);

-- Vyextrahovaný / ručně zadaný certifikát (1 plátce, 1 typ Potvrzení, 1 období)
CREATE TABLE employment_income_certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    document_id INTEGER REFERENCES employment_documents(id) ON DELETE SET NULL,
    certificate_type TEXT NOT NULL DEFAULT 'advance', -- advance | withholding
    employer_name TEXT NOT NULL DEFAULT '',
    employer_ico TEXT NOT NULL DEFAULT '',
    employer_address TEXT NOT NULL DEFAULT '',
    contract_type TEXT NOT NULL DEFAULT 'dpc', -- dpc | dpp | hpp | other
    period_from TEXT NOT NULL,
    period_to TEXT NOT NULL,
    -- Z Potvrzení 25 5460 vzor 33 (advance)
    gross_income INTEGER NOT NULL DEFAULT 0,                -- ř.2 + ř.4 Potvrzení -> ř.31 DAP
    income_without_advance INTEGER NOT NULL DEFAULT 0,      -- část bez záloh dle §38h (zahr. zastup. úřady, zahr. zaměstnavatelé) -> ř.35 DAP
    foreign_tax_paid INTEGER NOT NULL DEFAULT 0,            -- §6 odst.13 daň zaplacená v zahraničí -> ř.33 DAP
    advance_tax_withheld INTEGER NOT NULL DEFAULT 0,        -- ř.8 Potvrzení -> ř.84 DAP
    annual_settlement_refund INTEGER NOT NULL DEFAULT 0,    -- vrácený přeplatek z RZ (snižuje ř.84)
    monthly_bonus_paid INTEGER NOT NULL DEFAULT 0,          -- ř.5 + ř.13 Potvrzení -> ř.89 DAP (kc_vyplbonus)
    -- Z Potvrzení 25 5460/A vzor 12 (withholding)
    withheld_final_tax INTEGER NOT NULL DEFAULT 0,          -- §36/6/7 sražená daň -> ř.87 DAP
    include_withholding_in_dap INTEGER NOT NULL DEFAULT 0,  -- 1 = zahrnout do ř.31 a ř.87
    notes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',               -- draft | confirmed
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (year, employer_ico, certificate_type, period_from, period_to)
        ON CONFLICT REPLACE
);
CREATE INDEX idx_employment_certs_year ON employment_income_certificates(year);

-- §6 agregáty na income_tax_returns
ALTER TABLE income_tax_returns ADD COLUMN section6_gross_income INTEGER NOT NULL DEFAULT 0;             -- ř.31
ALTER TABLE income_tax_returns ADD COLUMN section6_income_without_advance INTEGER NOT NULL DEFAULT 0;   -- ř.35
ALTER TABLE income_tax_returns ADD COLUMN section6_foreign_tax INTEGER NOT NULL DEFAULT 0;              -- ř.33
ALTER TABLE income_tax_returns ADD COLUMN section6_tax_base INTEGER NOT NULL DEFAULT 0;                 -- ř.34/36
ALTER TABLE income_tax_returns ADD COLUMN section6_advance_withheld INTEGER NOT NULL DEFAULT 0;         -- ř.84
ALTER TABLE income_tax_returns ADD COLUMN section6_withholding_credited INTEGER NOT NULL DEFAULT 0;     -- ř.87
ALTER TABLE income_tax_returns ADD COLUMN section6_monthly_bonus_paid INTEGER NOT NULL DEFAULT 0;       -- ř.89 (kc_vyplbonus)
ALTER TABLE income_tax_returns ADD COLUMN section6_certs_advance INTEGER NOT NULL DEFAULT 0;            -- potv_zam count
ALTER TABLE income_tax_returns ADD COLUMN section6_certs_withholding INTEGER NOT NULL DEFAULT 0;        -- potv_36 count
ALTER TABLE income_tax_returns ADD COLUMN section6_certs_bonus INTEGER NOT NULL DEFAULT 0;              -- potv_dazvyh count

-- +goose Down
ALTER TABLE income_tax_returns DROP COLUMN section6_gross_income;
ALTER TABLE income_tax_returns DROP COLUMN section6_income_without_advance;
ALTER TABLE income_tax_returns DROP COLUMN section6_foreign_tax;
ALTER TABLE income_tax_returns DROP COLUMN section6_tax_base;
ALTER TABLE income_tax_returns DROP COLUMN section6_advance_withheld;
ALTER TABLE income_tax_returns DROP COLUMN section6_withholding_credited;
ALTER TABLE income_tax_returns DROP COLUMN section6_monthly_bonus_paid;
ALTER TABLE income_tax_returns DROP COLUMN section6_certs_advance;
ALTER TABLE income_tax_returns DROP COLUMN section6_certs_withholding;
ALTER TABLE income_tax_returns DROP COLUMN section6_certs_bonus;
DROP TABLE IF EXISTS employment_income_certificates;
DROP TABLE IF EXISTS employment_documents;
