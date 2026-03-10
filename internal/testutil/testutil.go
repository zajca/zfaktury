package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/zajca/zfaktury/internal/database"
	"github.com/zajca/zfaktury/internal/domain"

	_ "modernc.org/sqlite"
)

var invoiceCounter atomic.Int64

// NewTestDB creates an in-memory SQLite database with all migrations applied.
// After migrations, it patches TEXT timestamp columns to DATETIME so the
// modernc.org/sqlite driver can scan them back into time.Time values.
// The database is automatically closed when the test completes.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?_time_format=sqlite")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		t.Fatalf("setting foreign_keys pragma: %v", err)
	}

	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(database.MigrationsFS())

	if err := goose.SetDialect("sqlite3"); err != nil {
		db.Close()
		t.Fatalf("setting goose dialect: %v", err)
	}

	// Disable FK checks during migrations (table recreation may temporarily break references).
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		db.Close()
		t.Fatalf("disabling FK for migrations: %v", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		db.Close()
		t.Fatalf("running migrations: %v", err)
	}

	// Patch TEXT timestamp columns to DATETIME so the sqlite driver
	// can auto-parse them into time.Time on scan.
	patchTimestampColumns(t, db)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		t.Fatalf("re-enabling FK after migrations: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// patchTimestampColumns recreates tables so that columns storing timestamps
// use the DATETIME type instead of TEXT. The modernc.org/sqlite driver only
// auto-parses time values for DATE/DATETIME/TIMESTAMP typed columns.
func patchTimestampColumns(t *testing.T, db *sql.DB) {
	t.Helper()

	stmts := []string{
		// -- contacts --
		`CREATE TABLE contacts_tmp (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL DEFAULT 'company' CHECK (type IN ('company', 'individual')),
			name TEXT NOT NULL,
			ico TEXT, dic TEXT, street TEXT, city TEXT, zip TEXT,
			country TEXT NOT NULL DEFAULT 'CZ',
			email TEXT, phone TEXT, web TEXT,
			bank_account TEXT, bank_code TEXT, iban TEXT, swift TEXT,
			payment_terms_days INTEGER NOT NULL DEFAULT 14,
			tags TEXT, notes TEXT,
			is_favorite INTEGER NOT NULL DEFAULT 0,
			vat_unreliable INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			deleted_at DATETIME
		)`,
		`INSERT INTO contacts_tmp SELECT * FROM contacts`,
		`DROP TABLE contacts`,
		`ALTER TABLE contacts_tmp RENAME TO contacts`,
		`CREATE INDEX idx_contacts_type ON contacts(type)`,
		`CREATE INDEX idx_contacts_ico ON contacts(ico)`,
		`CREATE INDEX idx_contacts_name ON contacts(name)`,
		`CREATE INDEX idx_contacts_deleted_at ON contacts(deleted_at)`,

		// -- invoices --
		`CREATE TABLE invoices_tmp (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sequence_id INTEGER REFERENCES invoice_sequences(id),
			invoice_number TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL DEFAULT 'regular' CHECK (type IN ('regular', 'proforma', 'credit_note')),
			status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'sent', 'paid', 'overdue', 'cancelled')),
			issue_date DATETIME NOT NULL,
			due_date DATETIME NOT NULL,
			delivery_date DATETIME,
			variable_symbol TEXT, constant_symbol TEXT,
			customer_id INTEGER NOT NULL REFERENCES contacts(id),
			currency_code TEXT NOT NULL DEFAULT 'CZK',
			exchange_rate INTEGER NOT NULL DEFAULT 100,
			payment_method TEXT NOT NULL DEFAULT 'bank_transfer' CHECK (payment_method IN ('bank_transfer', 'cash', 'card', 'other')),
			bank_account TEXT, bank_code TEXT, iban TEXT, swift TEXT,
			subtotal_amount INTEGER NOT NULL DEFAULT 0,
			vat_amount INTEGER NOT NULL DEFAULT 0,
			total_amount INTEGER NOT NULL DEFAULT 0,
			paid_amount INTEGER NOT NULL DEFAULT 0,
			notes TEXT, internal_notes TEXT,
			sent_at DATETIME, paid_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			deleted_at DATETIME
		)`,
		`INSERT INTO invoices_tmp SELECT * FROM invoices`,
		`DROP TABLE invoices`,
		`ALTER TABLE invoices_tmp RENAME TO invoices`,
		`CREATE INDEX idx_invoices_customer_id ON invoices(customer_id)`,
		`CREATE INDEX idx_invoices_sequence_id ON invoices(sequence_id)`,
		`CREATE INDEX idx_invoices_type ON invoices(type)`,
		`CREATE INDEX idx_invoices_status ON invoices(status)`,
		`CREATE INDEX idx_invoices_issue_date ON invoices(issue_date)`,
		`CREATE INDEX idx_invoices_due_date ON invoices(due_date)`,
		`CREATE INDEX idx_invoices_invoice_number ON invoices(invoice_number)`,
		`CREATE INDEX idx_invoices_variable_symbol ON invoices(variable_symbol)`,
		`CREATE INDEX idx_invoices_deleted_at ON invoices(deleted_at)`,

		// -- expenses --
		`CREATE TABLE expenses_tmp (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vendor_id INTEGER REFERENCES contacts(id),
			expense_number TEXT, category TEXT,
			description TEXT NOT NULL,
			issue_date DATETIME NOT NULL,
			amount INTEGER NOT NULL DEFAULT 0,
			currency_code TEXT NOT NULL DEFAULT 'CZK',
			exchange_rate INTEGER NOT NULL DEFAULT 100,
			vat_rate_percent INTEGER NOT NULL DEFAULT 0,
			vat_amount INTEGER NOT NULL DEFAULT 0,
			is_tax_deductible INTEGER NOT NULL DEFAULT 1,
			business_percent INTEGER NOT NULL DEFAULT 100,
			payment_method TEXT NOT NULL DEFAULT 'bank_transfer' CHECK (payment_method IN ('bank_transfer', 'cash', 'card', 'other')),
			document_path TEXT, notes TEXT,
			created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			deleted_at DATETIME
		)`,
		`INSERT INTO expenses_tmp SELECT * FROM expenses`,
		`DROP TABLE expenses`,
		`ALTER TABLE expenses_tmp RENAME TO expenses`,
		`CREATE INDEX idx_expenses_vendor_id ON expenses(vendor_id)`,
		`CREATE INDEX idx_expenses_category ON expenses(category)`,
		`CREATE INDEX idx_expenses_issue_date ON expenses(issue_date)`,
		`CREATE INDEX idx_expenses_deleted_at ON expenses(deleted_at)`,

		// -- invoice_sequences --
		`CREATE TABLE invoice_sequences_tmp (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			prefix TEXT NOT NULL,
			next_number INTEGER NOT NULL DEFAULT 1,
			year INTEGER NOT NULL,
			format_pattern TEXT NOT NULL DEFAULT '{prefix}{year}{number:04d}',
			created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			deleted_at DATETIME,
			UNIQUE(prefix, year)
		)`,
		`INSERT INTO invoice_sequences_tmp SELECT * FROM invoice_sequences`,
		`DROP TABLE invoice_sequences`,
		`ALTER TABLE invoice_sequences_tmp RENAME TO invoice_sequences`,
		`CREATE INDEX idx_invoice_sequences_year ON invoice_sequences(year)`,
		`CREATE INDEX idx_invoice_sequences_deleted_at ON invoice_sequences(deleted_at)`,

		// -- expense_categories --
		`CREATE TABLE expense_categories_tmp (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			label_cs TEXT NOT NULL,
			label_en TEXT NOT NULL,
			color TEXT NOT NULL DEFAULT '#6B7280',
			sort_order INTEGER NOT NULL DEFAULT 0,
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			deleted_at DATETIME
		)`,
		`INSERT INTO expense_categories_tmp SELECT * FROM expense_categories`,
		`DROP TABLE expense_categories`,
		`ALTER TABLE expense_categories_tmp RENAME TO expense_categories`,
		`CREATE UNIQUE INDEX idx_expense_categories_key ON expense_categories(key)`,
		`CREATE INDEX idx_expense_categories_deleted_at ON expense_categories(deleted_at)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("patching timestamp columns: %v\nstatement: %s", err, stmt)
		}
	}
}

// SeedContact inserts a contact into the database with sensible defaults.
// Fields from the provided contact override defaults. Returns the contact with its assigned ID.
func SeedContact(t *testing.T, db *sql.DB, c *domain.Contact) *domain.Contact {
	t.Helper()

	if c == nil {
		c = &domain.Contact{}
	}
	if c.Type == "" {
		c.Type = domain.ContactTypeCompany
	}
	if c.Name == "" {
		c.Name = "Test Company s.r.o."
	}
	if c.Country == "" {
		c.Country = "CZ"
	}

	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	result, err := db.ExecContext(context.Background(), `
		INSERT INTO contacts (
			type, name, ico, dic, street, city, zip, country,
			email, phone, web, bank_account, bank_code, iban, swift,
			payment_terms_days, tags, notes, is_favorite, vat_unreliable,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Type, c.Name, c.ICO, c.DIC, c.Street, c.City, c.ZIP, c.Country,
		c.Email, c.Phone, c.Web, c.BankAccount, c.BankCode, c.IBAN, c.SWIFT,
		c.PaymentTermsDays, c.Tags, c.Notes, c.IsFavorite, c.VATUnreliable,
		c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("seeding contact: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("getting contact id: %v", err)
	}
	c.ID = id
	return c
}

// SeedInvoice inserts an invoice with items into the database.
// A customer contact must already exist. Returns the invoice with assigned IDs.
func SeedInvoice(t *testing.T, db *sql.DB, customerID int64, items []domain.InvoiceItem) *domain.Invoice {
	t.Helper()

	now := time.Now()
	inv := &domain.Invoice{
		InvoiceNumber: fmt.Sprintf("FV%s-%d", now.Format("20060102150405"), invoiceCounter.Add(1)),
		Type:          domain.InvoiceTypeRegular,
		Status:        domain.InvoiceStatusDraft,
		IssueDate:     now,
		DueDate:       now.AddDate(0, 0, 14),
		DeliveryDate:  now,
		CustomerID:    customerID,
		CurrencyCode:  domain.CurrencyCZK,
		ExchangeRate:  100,
		PaymentMethod: "bank_transfer",
		CreatedAt:     now,
		UpdatedAt:     now,
		Items:         items,
	}

	// Calculate totals.
	inv.CalculateTotals()

	// Use NULL for sequence_id when it's 0 to avoid FK constraint violation.
	var seqID any
	if inv.SequenceID > 0 {
		seqID = inv.SequenceID
	}

	result, err := db.ExecContext(context.Background(), `
		INSERT INTO invoices (
			sequence_id, invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes, sent_at, paid_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		seqID, inv.InvoiceNumber, inv.Type, inv.Status,
		inv.IssueDate, inv.DueDate, inv.DeliveryDate, inv.VariableSymbol, inv.ConstantSymbol,
		inv.CustomerID, inv.CurrencyCode, inv.ExchangeRate,
		inv.PaymentMethod, inv.BankAccount, inv.BankCode, inv.IBAN, inv.SWIFT,
		inv.SubtotalAmount, inv.VATAmount, inv.TotalAmount, inv.PaidAmount,
		inv.Notes, inv.InternalNotes, inv.SentAt, inv.PaidAt,
		inv.CreatedAt, inv.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("seeding invoice: %v", err)
	}

	invoiceID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("getting invoice id: %v", err)
	}
	inv.ID = invoiceID

	for i := range inv.Items {
		item := &inv.Items[i]
		item.InvoiceID = invoiceID

		itemResult, err := db.ExecContext(context.Background(), `
			INSERT INTO invoice_items (
				invoice_id, description, quantity, unit, unit_price,
				vat_rate_percent, vat_amount, total_amount, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.InvoiceID, item.Description, item.Quantity, item.Unit, item.UnitPrice,
			item.VATRatePercent, item.VATAmount, item.TotalAmount, item.SortOrder,
		)
		if err != nil {
			t.Fatalf("seeding invoice item %d: %v", i, err)
		}
		itemID, err := itemResult.LastInsertId()
		if err != nil {
			t.Fatalf("getting invoice item id: %v", err)
		}
		item.ID = itemID
	}

	return inv
}

// SeedExpense inserts an expense into the database with sensible defaults.
// Fields from the provided expense override defaults.
func SeedExpense(t *testing.T, db *sql.DB, e *domain.Expense) *domain.Expense {
	t.Helper()

	if e == nil {
		e = &domain.Expense{}
	}
	if e.Description == "" {
		e.Description = "Test Expense"
	}
	if e.Amount == 0 {
		e.Amount = domain.NewAmount(1000, 0)
	}
	if e.IssueDate.IsZero() {
		e.IssueDate = time.Now()
	}
	if e.CurrencyCode == "" {
		e.CurrencyCode = domain.CurrencyCZK
	}
	if e.BusinessPercent == 0 {
		e.BusinessPercent = 100
	}
	if e.PaymentMethod == "" {
		e.PaymentMethod = "bank_transfer"
	}

	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now

	result, err := db.ExecContext(context.Background(), `
		INSERT INTO expenses (
			vendor_id, expense_number, category, description,
			issue_date, amount, currency_code, exchange_rate,
			vat_rate_percent, vat_amount,
			is_tax_deductible, business_percent, payment_method,
			document_path, notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.VendorID, e.ExpenseNumber, e.Category, e.Description,
		e.IssueDate, e.Amount, e.CurrencyCode, e.ExchangeRate,
		e.VATRatePercent, e.VATAmount,
		e.IsTaxDeductible, e.BusinessPercent, e.PaymentMethod,
		e.DocumentPath, e.Notes,
		e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("seeding expense: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("getting expense id: %v", err)
	}
	e.ID = id
	return e
}

// SeedInvoiceSequence inserts an invoice sequence into the database.
func SeedInvoiceSequence(t *testing.T, db *sql.DB, prefix string, year int) int64 {
	t.Helper()

	result, err := db.ExecContext(context.Background(), `
		INSERT INTO invoice_sequences (prefix, next_number, year, format_pattern)
		VALUES (?, 1, ?, '{prefix}{year}{number:04d}')`, prefix, year)
	if err != nil {
		t.Fatalf("seeding invoice sequence: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("getting sequence id: %v", err)
	}
	return id
}
