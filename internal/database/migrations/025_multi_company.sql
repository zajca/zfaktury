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

-- Rebuild: invoice_sequences gains company_id in its UNIQUE constraint.
-- SQLite cannot alter a UNIQUE constraint in place, so we build a fresh table
-- under a temp name, copy rows, drop the original, then rename the new one.
-- Important: we do NOT rename the ORIGINAL table out of the way, because
-- doing so would silently rewrite the FK reference in invoices.sequence_id
-- (modern SQLite auto-updates FK references on RENAME) and after the temp
-- table was later dropped that reference would point at a vanished name.
CREATE TABLE invoice_sequences__new (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id     INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id),
	prefix         TEXT    NOT NULL,
	next_number    INTEGER NOT NULL DEFAULT 1,
	year           INTEGER NOT NULL,
	format_pattern TEXT    NOT NULL DEFAULT '{prefix}{year}{number:04d}',
	created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	updated_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	deleted_at     TEXT,
	UNIQUE(company_id, prefix, year)
);

INSERT INTO invoice_sequences__new (id, company_id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at)
SELECT id, 1, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at FROM invoice_sequences;

DROP TABLE invoice_sequences;
ALTER TABLE invoice_sequences__new RENAME TO invoice_sequences;

CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year);
CREATE INDEX idx_invoice_sequences_deleted_at ON invoice_sequences(deleted_at);
CREATE INDEX idx_invoice_sequences_company ON invoice_sequences(company_id);

-- Partition: invoices + recurring_invoices (parents).
-- Adding the column is enough on the parent side; the UNIQUE(company_id, id)
-- index makes the parent a valid composite-FK target for the rebuilt children.
ALTER TABLE invoices            ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE recurring_invoices  ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX        idx_invoices_company              ON invoices(company_id);
CREATE INDEX        idx_recurring_invoices_company    ON recurring_invoices(company_id);
CREATE UNIQUE INDEX idx_invoices_company_id           ON invoices(company_id, id);
CREATE UNIQUE INDEX idx_recurring_invoices_company_id ON recurring_invoices(company_id, id);

-- Rebuild: invoice_items gains company_id and a composite FK
-- (company_id, invoice_id) -> invoices(company_id, id) so an item can never
-- reference an invoice owned by a different company. ON DELETE CASCADE keeps
-- the original deletion semantics from migration 001.
ALTER TABLE invoice_items RENAME TO invoice_items__old;
CREATE TABLE invoice_items (
	id               INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id       INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id),
	invoice_id       INTEGER NOT NULL,
	description      TEXT    NOT NULL,
	quantity         INTEGER NOT NULL DEFAULT 100,
	unit             TEXT    NOT NULL DEFAULT 'ks',
	unit_price       INTEGER NOT NULL DEFAULT 0,
	vat_rate_percent INTEGER NOT NULL DEFAULT 0,
	vat_amount       INTEGER NOT NULL DEFAULT 0,
	total_amount     INTEGER NOT NULL DEFAULT 0,
	sort_order       INTEGER NOT NULL DEFAULT 0,
	created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	FOREIGN KEY (company_id, invoice_id) REFERENCES invoices(company_id, id) ON DELETE CASCADE
);
INSERT INTO invoice_items (
	id, company_id, invoice_id, description, quantity, unit, unit_price,
	vat_rate_percent, vat_amount, total_amount, sort_order, created_at
)
SELECT
	id, 1, invoice_id, description, quantity, unit, unit_price,
	vat_rate_percent, vat_amount, total_amount, sort_order, created_at
FROM invoice_items__old;
DROP TABLE invoice_items__old;
CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);
CREATE INDEX idx_invoice_items_company    ON invoice_items(company_id);

-- Rebuild: recurring_invoice_items gains company_id and a composite FK to
-- recurring_invoices(company_id, id). Original schema had no ON DELETE
-- CASCADE on the single-column FK; the composite FK keeps the same default
-- (NO ACTION) so behaviour is preserved.
ALTER TABLE recurring_invoice_items RENAME TO recurring_invoice_items__old;
CREATE TABLE recurring_invoice_items (
	id                   INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id           INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id),
	recurring_invoice_id INTEGER NOT NULL,
	description          TEXT    NOT NULL,
	quantity             INTEGER NOT NULL,
	unit                 TEXT    NOT NULL DEFAULT 'ks',
	unit_price           INTEGER NOT NULL,
	vat_rate_percent     INTEGER NOT NULL DEFAULT 21,
	sort_order           INTEGER NOT NULL DEFAULT 0,
	FOREIGN KEY (company_id, recurring_invoice_id) REFERENCES recurring_invoices(company_id, id)
);
INSERT INTO recurring_invoice_items (
	id, company_id, recurring_invoice_id, description, quantity, unit, unit_price,
	vat_rate_percent, sort_order
)
SELECT
	id, 1, recurring_invoice_id, description, quantity, unit, unit_price,
	vat_rate_percent, sort_order
FROM recurring_invoice_items__old;
DROP TABLE recurring_invoice_items__old;
CREATE INDEX idx_recurring_invoice_items_company ON recurring_invoice_items(company_id);
CREATE INDEX idx_recurring_invoice_items_parent  ON recurring_invoice_items(recurring_invoice_id);

-- Partition: expenses (parent).
-- Adding the column is enough on the parent side; the UNIQUE(company_id, id)
-- index makes the parent a valid composite-FK target for the rebuilt child.
ALTER TABLE expenses ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX        idx_expenses_company    ON expenses(company_id);
CREATE UNIQUE INDEX idx_expenses_company_id ON expenses(company_id, id);

-- Rebuild: expense_items gains company_id and a composite FK
-- (company_id, expense_id) -> expenses(company_id, id) so an item can never
-- reference an expense owned by a different company. ON DELETE CASCADE keeps
-- the original deletion semantics from migration 023.
-- Uses the CREATE __new + DROP original + RENAME pattern (mirrored from the
-- invoice_sequences rebuild above) to avoid silently rewriting FK references
-- in any future dependent table during the table rename.
CREATE TABLE expense_items__new (
	id               INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id       INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id),
	expense_id       INTEGER NOT NULL,
	description      TEXT    NOT NULL DEFAULT '',
	quantity         INTEGER NOT NULL DEFAULT 100,
	unit             TEXT    NOT NULL DEFAULT 'ks',
	unit_price       INTEGER NOT NULL DEFAULT 0,
	vat_rate_percent INTEGER NOT NULL DEFAULT 0,
	vat_amount       INTEGER NOT NULL DEFAULT 0,
	total_amount     INTEGER NOT NULL DEFAULT 0,
	sort_order       INTEGER NOT NULL DEFAULT 0,
	created_at       TEXT    NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY (company_id, expense_id) REFERENCES expenses(company_id, id) ON DELETE CASCADE
);
INSERT INTO expense_items__new (
	id, company_id, expense_id, description, quantity, unit, unit_price,
	vat_rate_percent, vat_amount, total_amount, sort_order, created_at
)
SELECT
	id, 1, expense_id, description, quantity, unit, unit_price,
	vat_rate_percent, vat_amount, total_amount, sort_order, created_at
FROM expense_items;
DROP TABLE expense_items;
ALTER TABLE expense_items__new RENAME TO expense_items;
CREATE INDEX idx_expense_items_expense_id ON expense_items(expense_id);
CREATE INDEX idx_expense_items_company    ON expense_items(company_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse: expense graph composite FK on child, then parent column drop.
-- The child must be rebuilt first so the composite FK referencing the parent's
-- (company_id, id) is gone before we drop company_id from expenses.

-- Rebuild expense_items back to single-column FK ON DELETE CASCADE.
-- Mirror of Up: CREATE __new + DROP original + RENAME so we never rename the
-- live table out of the way (would silently rewrite FK references in any
-- future dependent table).
DROP INDEX IF EXISTS idx_expense_items_company;
DROP INDEX IF EXISTS idx_expense_items_expense_id;
CREATE TABLE expense_items__new (
	id               INTEGER PRIMARY KEY AUTOINCREMENT,
	expense_id       INTEGER NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
	description      TEXT    NOT NULL DEFAULT '',
	quantity         INTEGER NOT NULL DEFAULT 100,
	unit             TEXT    NOT NULL DEFAULT 'ks',
	unit_price       INTEGER NOT NULL DEFAULT 0,
	vat_rate_percent INTEGER NOT NULL DEFAULT 0,
	vat_amount       INTEGER NOT NULL DEFAULT 0,
	total_amount     INTEGER NOT NULL DEFAULT 0,
	sort_order       INTEGER NOT NULL DEFAULT 0,
	created_at       TEXT    NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO expense_items__new (
	id, expense_id, description, quantity, unit, unit_price,
	vat_rate_percent, vat_amount, total_amount, sort_order, created_at
)
SELECT
	id, expense_id, description, quantity, unit, unit_price,
	vat_rate_percent, vat_amount, total_amount, sort_order, created_at
FROM expense_items
WHERE company_id = 1;
DROP TABLE expense_items;
ALTER TABLE expense_items__new RENAME TO expense_items;
CREATE INDEX idx_expense_items_expense_id ON expense_items(expense_id);

-- Reverse: expenses parent column add.
DROP INDEX IF EXISTS idx_expenses_company_id;
DROP INDEX IF EXISTS idx_expenses_company;
ALTER TABLE expenses DROP COLUMN company_id;

-- Reverse: invoice graph composite FK on children, then parent column drops.
-- Children must be rebuilt first so the composite FK referencing the parent's
-- (company_id, id) is gone before we drop company_id from the parents.

-- Rebuild recurring_invoice_items back to single-column FK.
DROP INDEX IF EXISTS idx_recurring_invoice_items_parent;
DROP INDEX IF EXISTS idx_recurring_invoice_items_company;
ALTER TABLE recurring_invoice_items RENAME TO recurring_invoice_items__new;
CREATE TABLE recurring_invoice_items (
	id                   INTEGER PRIMARY KEY AUTOINCREMENT,
	recurring_invoice_id INTEGER NOT NULL REFERENCES recurring_invoices(id),
	description          TEXT    NOT NULL,
	quantity             INTEGER NOT NULL,
	unit                 TEXT    NOT NULL DEFAULT 'ks',
	unit_price           INTEGER NOT NULL,
	vat_rate_percent     INTEGER NOT NULL DEFAULT 21,
	sort_order           INTEGER NOT NULL DEFAULT 0
);
INSERT INTO recurring_invoice_items (
	id, recurring_invoice_id, description, quantity, unit, unit_price,
	vat_rate_percent, sort_order
)
SELECT
	id, recurring_invoice_id, description, quantity, unit, unit_price,
	vat_rate_percent, sort_order
FROM recurring_invoice_items__new
WHERE company_id = 1;
DROP TABLE recurring_invoice_items__new;

-- Rebuild invoice_items back to single-column FK ON DELETE CASCADE.
DROP INDEX IF EXISTS idx_invoice_items_company;
DROP INDEX IF EXISTS idx_invoice_items_invoice_id;
ALTER TABLE invoice_items RENAME TO invoice_items__new;
CREATE TABLE invoice_items (
	id               INTEGER PRIMARY KEY AUTOINCREMENT,
	invoice_id       INTEGER NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
	description      TEXT    NOT NULL,
	quantity         INTEGER NOT NULL DEFAULT 100,
	unit             TEXT    NOT NULL DEFAULT 'ks',
	unit_price       INTEGER NOT NULL DEFAULT 0,
	vat_rate_percent INTEGER NOT NULL DEFAULT 0,
	vat_amount       INTEGER NOT NULL DEFAULT 0,
	total_amount     INTEGER NOT NULL DEFAULT 0,
	sort_order       INTEGER NOT NULL DEFAULT 0,
	created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
INSERT INTO invoice_items (
	id, invoice_id, description, quantity, unit, unit_price,
	vat_rate_percent, vat_amount, total_amount, sort_order, created_at
)
SELECT
	id, invoice_id, description, quantity, unit, unit_price,
	vat_rate_percent, vat_amount, total_amount, sort_order, created_at
FROM invoice_items__new
WHERE company_id = 1;
DROP TABLE invoice_items__new;
CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);

-- Reverse parent column adds.
DROP INDEX IF EXISTS idx_recurring_invoices_company_id;
DROP INDEX IF EXISTS idx_invoices_company_id;
DROP INDEX IF EXISTS idx_recurring_invoices_company;
DROP INDEX IF EXISTS idx_invoices_company;
ALTER TABLE recurring_invoices DROP COLUMN company_id;
ALTER TABLE invoices           DROP COLUMN company_id;

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

-- Reverse: invoice_sequences rebuild back to UNIQUE(prefix, year).
-- Best-effort: only company 1's sequences survive the downgrade.
-- Mirror of Up: build under a temp name first, then drop+rename, to avoid
-- silently rewriting the FK reference in invoices.sequence_id when the
-- live table is renamed.
DROP INDEX IF EXISTS idx_invoice_sequences_company;
DROP INDEX IF EXISTS idx_invoice_sequences_deleted_at;
DROP INDEX IF EXISTS idx_invoice_sequences_year;
CREATE TABLE invoice_sequences__new (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	prefix         TEXT    NOT NULL,
	next_number    INTEGER NOT NULL DEFAULT 1,
	year           INTEGER NOT NULL,
	format_pattern TEXT    NOT NULL DEFAULT '{prefix}{year}{number:04d}',
	created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	updated_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	deleted_at     TEXT,
	UNIQUE(prefix, year)
);
INSERT INTO invoice_sequences__new (id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at)
SELECT id, prefix, next_number, year, format_pattern, created_at, updated_at, deleted_at FROM invoice_sequences
WHERE company_id = 1;
DROP TABLE invoice_sequences;
ALTER TABLE invoice_sequences__new RENAME TO invoice_sequences;
CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year);
CREATE INDEX idx_invoice_sequences_deleted_at ON invoice_sequences(deleted_at);

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
