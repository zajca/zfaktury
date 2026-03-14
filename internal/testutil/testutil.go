package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/zajca/zfaktury/internal/database"
	"github.com/zajca/zfaktury/internal/domain"

	_ "modernc.org/sqlite"
)

var (
	invoiceCounter atomic.Int64
	dbCounter      atomic.Int64
)

// NewTestDB creates an in-memory SQLite database with all migrations applied.
// The database is automatically closed when the test completes.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use a unique named in-memory database with shared cache so all connections
	// from the pool see the same data. This avoids issues where goose migrations
	// run on one connection but queries use another (each getting a separate :memory: DB).
	dbName := fmt.Sprintf("testdb_%d_%d", time.Now().UnixNano(), dbCounter.Add(1))
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_time_format=sqlite", dbName)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}

	// Disable FK checks during migrations (table recreation may temporarily break references).
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		_ = db.Close()
		t.Fatalf("disabling foreign_keys pragma: %v", err)
	}

	// Use per-instance goose provider to avoid global state race conditions
	// when multiple test packages run concurrently.
	migrationsFS, err := fs.Sub(database.MigrationsFS(), "migrations")
	if err != nil {
		_ = db.Close()
		t.Fatalf("getting migrations sub-fs: %v", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrationsFS)
	if err != nil {
		_ = db.Close()
		t.Fatalf("creating goose provider: %v", err)
	}

	if _, err := provider.Up(context.Background()); err != nil {
		_ = db.Close()
		t.Fatalf("running migrations: %v", err)
	}

	// Re-enable FK checks after migrations.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		t.Fatalf("enabling foreign_keys pragma: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db
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
			payment_terms_days, tags, notes, is_favorite, vat_unreliable_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Type, c.Name, c.ICO, c.DIC, c.Street, c.City, c.ZIP, c.Country,
		c.Email, c.Phone, c.Web, c.BankAccount, c.BankCode, c.IBAN, c.SWIFT,
		c.PaymentTermsDays, c.Tags, c.Notes, c.IsFavorite, c.VATUnreliableAt,
		c.CreatedAt.Format(time.RFC3339), c.UpdatedAt.Format(time.RFC3339),
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
		inv.IssueDate.Format("2006-01-02"), inv.DueDate.Format("2006-01-02"), inv.DeliveryDate.Format("2006-01-02"), inv.VariableSymbol, inv.ConstantSymbol,
		inv.CustomerID, inv.CurrencyCode, inv.ExchangeRate,
		inv.PaymentMethod, inv.BankAccount, inv.BankCode, inv.IBAN, inv.SWIFT,
		inv.SubtotalAmount, inv.VATAmount, inv.TotalAmount, inv.PaidAmount,
		inv.Notes, inv.InternalNotes, inv.SentAt, inv.PaidAt,
		inv.CreatedAt.Format(time.RFC3339), inv.UpdatedAt.Format(time.RFC3339),
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
		e.IssueDate.Format("2006-01-02"), e.Amount, e.CurrencyCode, e.ExchangeRate,
		e.VATRatePercent, e.VATAmount,
		e.IsTaxDeductible, e.BusinessPercent, e.PaymentMethod,
		e.DocumentPath, e.Notes,
		e.CreatedAt.Format(time.RFC3339), e.UpdatedAt.Format(time.RFC3339),
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
