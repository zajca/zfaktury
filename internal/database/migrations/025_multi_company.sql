-- +goose Up
-- +goose StatementBegin

-- 1. Companies table — the new home for what was 17 settings keys.
CREATE TABLE companies (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	name            TEXT    NOT NULL,
	legal_name      TEXT    NOT NULL,
	ico             TEXT    NOT NULL,
	dic             TEXT,
	vat_registered  INTEGER NOT NULL DEFAULT 0,
	street          TEXT, house_number TEXT, city TEXT, zip TEXT,
	email           TEXT, phone TEXT,
	first_name      TEXT, last_name TEXT,
	bank_account    TEXT, bank_code TEXT, iban TEXT, swift TEXT,
	logo_path       TEXT, accent_color TEXT,
	created_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL,
	deleted_at      TEXT
);
CREATE UNIQUE INDEX idx_companies_ico_active ON companies(ico) WHERE deleted_at IS NULL;

-- 2. Seed the default company from existing settings.
-- The WHERE EXISTS guard requires an actual ICO in settings; fresh installs
-- and partially-populated settings without ICO are no-ops (the frontend
-- empty-state will route these users to /companies/new).
INSERT INTO companies (
	id, name, legal_name, ico, dic, vat_registered,
	street, house_number, city, zip,
	email, phone,
	first_name, last_name,
	bank_account, bank_code, iban, swift,
	logo_path, accent_color,
	created_at, updated_at
)
SELECT
	1,
	COALESCE((SELECT value FROM settings WHERE key='company_name'), 'My Company'),  -- name
	COALESCE((SELECT value FROM settings WHERE key='company_name'), 'My Company'),  -- legal_name: no legacy key; mirrors name
	COALESCE((SELECT value FROM settings WHERE key='ico'), ''),
	NULLIF((SELECT value FROM settings WHERE key='dic'), ''),
	CASE WHEN COALESCE((SELECT value FROM settings WHERE key='vat_registered'), '0') = '1' THEN 1 ELSE 0 END,
	NULLIF((SELECT value FROM settings WHERE key='street'), ''),
	NULLIF((SELECT value FROM settings WHERE key='house_number'), ''),
	NULLIF((SELECT value FROM settings WHERE key='city'), ''),
	NULLIF((SELECT value FROM settings WHERE key='zip'), ''),
	NULLIF((SELECT value FROM settings WHERE key='email'), ''),
	NULLIF((SELECT value FROM settings WHERE key='phone'), ''),
	NULLIF((SELECT value FROM settings WHERE key='first_name'), ''),
	NULLIF((SELECT value FROM settings WHERE key='last_name'), ''),
	NULLIF((SELECT value FROM settings WHERE key='bank_account'), ''),
	NULLIF((SELECT value FROM settings WHERE key='bank_code'), ''),
	NULLIF((SELECT value FROM settings WHERE key='iban'), ''),
	NULLIF((SELECT value FROM settings WHERE key='swift'), ''),
	NULLIF((SELECT value FROM settings WHERE key='logo_path'), ''),
	NULLIF((SELECT value FROM settings WHERE key='accent_color'), ''),
	strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
	strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE EXISTS (SELECT 1 FROM settings WHERE key = 'ico' AND value IS NOT NULL AND value != '');

-- 3. Strip the 17 identity keys from settings (now lifted into companies).
DELETE FROM settings WHERE key IN (
	'company_name', 'ico', 'dic', 'vat_registered',
	'street', 'house_number', 'city', 'zip',
	'email', 'phone',
	'first_name', 'last_name',
	'bank_account', 'bank_code', 'iban', 'swift',
	'logo_path', 'accent_color'
);

-- Partition: contacts
ALTER TABLE contacts ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_contacts_company ON contacts(company_id);

-- Partition: leaf-style per-company tables (no composite FK needed).
-- Existing rows backfill to default company id=1 via the DEFAULT clause.
ALTER TABLE expense_categories ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_expense_categories_company ON expense_categories(company_id);

ALTER TABLE expense_documents ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_expense_documents_company ON expense_documents(company_id);

ALTER TABLE invoice_documents ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_invoice_documents_company ON invoice_documents(company_id);

ALTER TABLE invoice_status_history ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_invoice_status_history_company ON invoice_status_history(company_id);

ALTER TABLE recurring_expenses ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_recurring_expenses_company ON recurring_expenses(company_id);

ALTER TABLE payment_reminders ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_payment_reminders_company ON payment_reminders(company_id);

ALTER TABLE fakturoid_import_log ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_fakturoid_import_log_company ON fakturoid_import_log(company_id);

ALTER TABLE tax_year_settings ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_tax_year_settings_company ON tax_year_settings(company_id);

ALTER TABLE tax_prepayments ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_tax_prepayments_company ON tax_prepayments(company_id);

ALTER TABLE tax_spouse_credits ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_tax_spouse_credits_company ON tax_spouse_credits(company_id);

ALTER TABLE tax_child_credits ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_tax_child_credits_company ON tax_child_credits(company_id);

ALTER TABLE tax_personal_credits ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_tax_personal_credits_company ON tax_personal_credits(company_id);

ALTER TABLE tax_deductions ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_tax_deductions_company ON tax_deductions(company_id);

ALTER TABLE tax_deduction_documents ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_tax_deduction_documents_company ON tax_deduction_documents(company_id);

ALTER TABLE social_insurance_overviews ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_social_insurance_overviews_company ON social_insurance_overviews(company_id);

ALTER TABLE health_insurance_overviews ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_health_insurance_overviews_company ON health_insurance_overviews(company_id);

ALTER TABLE investment_documents ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_investment_documents_company ON investment_documents(company_id);

ALTER TABLE capital_income_entries ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_capital_income_entries_company ON capital_income_entries(company_id);

ALTER TABLE security_transactions ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_security_transactions_company ON security_transactions(company_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore the 17 identity keys from company id=1 (best-effort; multi-company
-- users will lose everything but the first company on downgrade).
INSERT INTO settings (key, value)
SELECT 'company_name', name FROM companies WHERE id = 1
UNION ALL SELECT 'ico', ico FROM companies WHERE id = 1
UNION ALL SELECT 'dic', COALESCE(dic, '') FROM companies WHERE id = 1
UNION ALL SELECT 'vat_registered', CASE WHEN vat_registered = 1 THEN '1' ELSE '0' END FROM companies WHERE id = 1
UNION ALL SELECT 'street', COALESCE(street, '') FROM companies WHERE id = 1
UNION ALL SELECT 'house_number', COALESCE(house_number, '') FROM companies WHERE id = 1
UNION ALL SELECT 'city', COALESCE(city, '') FROM companies WHERE id = 1
UNION ALL SELECT 'zip', COALESCE(zip, '') FROM companies WHERE id = 1
UNION ALL SELECT 'email', COALESCE(email, '') FROM companies WHERE id = 1
UNION ALL SELECT 'phone', COALESCE(phone, '') FROM companies WHERE id = 1
UNION ALL SELECT 'first_name', COALESCE(first_name, '') FROM companies WHERE id = 1
UNION ALL SELECT 'last_name', COALESCE(last_name, '') FROM companies WHERE id = 1
UNION ALL SELECT 'bank_account', COALESCE(bank_account, '') FROM companies WHERE id = 1
UNION ALL SELECT 'bank_code', COALESCE(bank_code, '') FROM companies WHERE id = 1
UNION ALL SELECT 'iban', COALESCE(iban, '') FROM companies WHERE id = 1
UNION ALL SELECT 'swift', COALESCE(swift, '') FROM companies WHERE id = 1
UNION ALL SELECT 'logo_path', COALESCE(logo_path, '') FROM companies WHERE id = 1
UNION ALL SELECT 'accent_color', COALESCE(accent_color, '') FROM companies WHERE id = 1;

-- Reverse: leaf-style per-company tables (reverse order of Up).
DROP INDEX IF EXISTS idx_security_transactions_company;
ALTER TABLE security_transactions DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_capital_income_entries_company;
ALTER TABLE capital_income_entries DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_investment_documents_company;
ALTER TABLE investment_documents DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_health_insurance_overviews_company;
ALTER TABLE health_insurance_overviews DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_social_insurance_overviews_company;
ALTER TABLE social_insurance_overviews DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_tax_deduction_documents_company;
ALTER TABLE tax_deduction_documents DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_tax_deductions_company;
ALTER TABLE tax_deductions DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_tax_personal_credits_company;
ALTER TABLE tax_personal_credits DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_tax_child_credits_company;
ALTER TABLE tax_child_credits DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_tax_spouse_credits_company;
ALTER TABLE tax_spouse_credits DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_tax_prepayments_company;
ALTER TABLE tax_prepayments DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_tax_year_settings_company;
ALTER TABLE tax_year_settings DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_fakturoid_import_log_company;
ALTER TABLE fakturoid_import_log DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_payment_reminders_company;
ALTER TABLE payment_reminders DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_recurring_expenses_company;
ALTER TABLE recurring_expenses DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_invoice_status_history_company;
ALTER TABLE invoice_status_history DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_invoice_documents_company;
ALTER TABLE invoice_documents DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_expense_documents_company;
ALTER TABLE expense_documents DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_expense_categories_company;
ALTER TABLE expense_categories DROP COLUMN company_id;

-- Reverse: contacts partition
DROP INDEX IF EXISTS idx_contacts_company;
ALTER TABLE contacts DROP COLUMN company_id;

DROP INDEX IF EXISTS idx_companies_ico_active;
DROP TABLE IF EXISTS companies;

-- +goose StatementEnd
