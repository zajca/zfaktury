package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestCalculateAnnualBase_NoData(t *testing.T) {
	db := testutil.NewTestDB(t)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	ctx := context.Background()

	base, err := CalculateAnnualBase(ctx, invRepo, expRepo, 2025)
	if err != nil {
		t.Fatalf("CalculateAnnualBase() error: %v", err)
	}

	if base.Revenue != 0 {
		t.Errorf("Revenue = %d, want 0", base.Revenue)
	}
	if base.Expenses != 0 {
		t.Errorf("Expenses = %d, want 0", base.Expenses)
	}
	if len(base.InvoiceIDs) != 0 {
		t.Errorf("InvoiceIDs = %v, want empty", base.InvoiceIDs)
	}
	if len(base.ExpenseIDs) != 0 {
		t.Errorf("ExpenseIDs = %v, want empty", base.ExpenseIDs)
	}
}

func TestCalculateAnnualBase_WithInvoices(t *testing.T) {
	db := testutil.NewTestDB(t)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	ctx := context.Background()

	contact := testutil.SeedContact(t, db, nil)
	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// Sent invoice: 100,000 CZK subtotal.
	_, err := db.ExecContext(ctx, `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"FV20250001", domain.InvoiceTypeRegular, domain.InvoiceStatusSent,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(100000, 0), domain.NewAmount(21000, 0), domain.NewAmount(121000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice: %v", err)
	}

	// Paid invoice: 50,000 CZK subtotal.
	_, err = db.ExecContext(ctx, `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"FV20250002", domain.InvoiceTypeRegular, domain.InvoiceStatusPaid,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(50000, 0), 0, domain.NewAmount(50000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice 2: %v", err)
	}

	base, err := CalculateAnnualBase(ctx, invRepo, expRepo, 2025)
	if err != nil {
		t.Fatalf("CalculateAnnualBase() error: %v", err)
	}

	expectedRevenue := domain.NewAmount(100000, 0) + domain.NewAmount(50000, 0)
	if base.Revenue != expectedRevenue {
		t.Errorf("Revenue = %d, want %d", base.Revenue, expectedRevenue)
	}
	if len(base.InvoiceIDs) != 2 {
		t.Errorf("InvoiceIDs count = %d, want 2", len(base.InvoiceIDs))
	}
}

func TestCalculateAnnualBase_ExcludesDraftInvoices(t *testing.T) {
	db := testutil.NewTestDB(t)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	ctx := context.Background()

	contact := testutil.SeedContact(t, db, nil)
	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// Draft invoice -- should be excluded.
	_, err := db.ExecContext(ctx, `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"FV20250003", domain.InvoiceTypeRegular, domain.InvoiceStatusDraft,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(100000, 0), 0, domain.NewAmount(100000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice: %v", err)
	}

	base, err := CalculateAnnualBase(ctx, invRepo, expRepo, 2025)
	if err != nil {
		t.Fatalf("CalculateAnnualBase() error: %v", err)
	}

	if base.Revenue != 0 {
		t.Errorf("Revenue = %d, want 0 (draft excluded)", base.Revenue)
	}
	if len(base.InvoiceIDs) != 0 {
		t.Errorf("InvoiceIDs count = %d, want 0", len(base.InvoiceIDs))
	}
}

func TestCalculateAnnualBase_ExcludesProforma(t *testing.T) {
	db := testutil.NewTestDB(t)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	ctx := context.Background()

	contact := testutil.SeedContact(t, db, nil)
	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// Proforma invoice -- should be excluded.
	_, err := db.ExecContext(ctx, `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"ZF20250001", domain.InvoiceTypeProforma, domain.InvoiceStatusSent,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(100000, 0), 0, domain.NewAmount(100000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting proforma: %v", err)
	}

	base, err := CalculateAnnualBase(ctx, invRepo, expRepo, 2025)
	if err != nil {
		t.Fatalf("CalculateAnnualBase() error: %v", err)
	}

	if base.Revenue != 0 {
		t.Errorf("Revenue = %d, want 0 (proforma excluded)", base.Revenue)
	}
}

func TestCalculateAnnualBase_CreditNoteSubtractsRevenue(t *testing.T) {
	db := testutil.NewTestDB(t)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	ctx := context.Background()

	contact := testutil.SeedContact(t, db, nil)
	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// Regular sent invoice: 100,000 CZK.
	_, err := db.ExecContext(ctx, `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"FV20250010", domain.InvoiceTypeRegular, domain.InvoiceStatusSent,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(100000, 0), 0, domain.NewAmount(100000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting regular invoice: %v", err)
	}

	// Credit note: 20,000 CZK (subtracted from revenue).
	_, err = db.ExecContext(ctx, `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"DD20250001", domain.InvoiceTypeCreditNote, domain.InvoiceStatusSent,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(20000, 0), 0, domain.NewAmount(20000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting credit note: %v", err)
	}

	base, err := CalculateAnnualBase(ctx, invRepo, expRepo, 2025)
	if err != nil {
		t.Fatalf("CalculateAnnualBase() error: %v", err)
	}

	expectedRevenue := domain.NewAmount(100000, 0) - domain.NewAmount(20000, 0)
	if base.Revenue != expectedRevenue {
		t.Errorf("Revenue = %d, want %d (after credit note)", base.Revenue, expectedRevenue)
	}
	if len(base.InvoiceIDs) != 2 {
		t.Errorf("InvoiceIDs count = %d, want 2", len(base.InvoiceIDs))
	}
}

func TestCalculateAnnualBase_WithExpenses(t *testing.T) {
	db := testutil.NewTestDB(t)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	ctx := context.Background()

	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	// Tax-reviewed expense: 12,100 CZK total, 2,100 CZK VAT, 100% business.
	exp := testutil.SeedExpense(t, db, &domain.Expense{
		Description:     "Office supplies",
		IssueDate:       jan15,
		Amount:          domain.NewAmount(12100, 0),
		VATAmount:       domain.NewAmount(2100, 0),
		BusinessPercent: 100,
	})
	// Mark as tax-reviewed.
	expRepo.MarkTaxReviewed(ctx, []int64{exp.ID})

	// Non-reviewed expense -- should be excluded.
	testutil.SeedExpense(t, db, &domain.Expense{
		Description:     "Personal item",
		IssueDate:       jan15,
		Amount:          domain.NewAmount(5000, 0),
		BusinessPercent: 100,
	})

	base, err := CalculateAnnualBase(ctx, invRepo, expRepo, 2025)
	if err != nil {
		t.Fatalf("CalculateAnnualBase() error: %v", err)
	}

	// Expense base = (12100 - 2100) * 100/100 = 10000 CZK.
	expectedExpenses := domain.NewAmount(10000, 0)
	if base.Expenses != expectedExpenses {
		t.Errorf("Expenses = %d, want %d", base.Expenses, expectedExpenses)
	}
	if len(base.ExpenseIDs) != 1 {
		t.Errorf("ExpenseIDs count = %d, want 1", len(base.ExpenseIDs))
	}
}

func TestCalculateAnnualBase_ExpensePartialBusinessPercent(t *testing.T) {
	db := testutil.NewTestDB(t)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	ctx := context.Background()

	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	// Expense with 60% business use.
	exp := testutil.SeedExpense(t, db, &domain.Expense{
		Description:     "Car fuel",
		IssueDate:       jan15,
		Amount:          domain.NewAmount(12100, 0),
		VATAmount:       domain.NewAmount(2100, 0),
		BusinessPercent: 60,
	})
	expRepo.MarkTaxReviewed(ctx, []int64{exp.ID})

	base, err := CalculateAnnualBase(ctx, invRepo, expRepo, 2025)
	if err != nil {
		t.Fatalf("CalculateAnnualBase() error: %v", err)
	}

	// Expense base = (12100 - 2100) * 60/100 = 6000 CZK.
	expectedExpenses := domain.NewAmount(10000, 0).Multiply(0.6)
	if base.Expenses != expectedExpenses {
		t.Errorf("Expenses = %d, want %d", base.Expenses, expectedExpenses)
	}
}

func TestGetTaxConstants_KnownYear(t *testing.T) {
	constants, err := GetTaxConstants(2025)
	if err != nil {
		t.Fatalf("GetTaxConstants(2025) error: %v", err)
	}

	if constants.BasicCredit != domain.NewAmount(30840, 0) {
		t.Errorf("BasicCredit = %d, want %d", constants.BasicCredit, domain.NewAmount(30840, 0))
	}
	if constants.SocialRate != 292 {
		t.Errorf("SocialRate = %d, want 292", constants.SocialRate)
	}
	if constants.HealthRate != 135 {
		t.Errorf("HealthRate = %d, want 135", constants.HealthRate)
	}
}

func TestGetTaxConstants_UnknownYear(t *testing.T) {
	_, err := GetTaxConstants(1900)
	if err == nil {
		t.Error("expected error for unknown year")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestGetTaxConstants_AllConfiguredYears(t *testing.T) {
	for _, year := range []int{2024, 2025, 2026} {
		constants, err := GetTaxConstants(year)
		if err != nil {
			t.Fatalf("GetTaxConstants(%d) error: %v", year, err)
		}
		if constants.ProgressiveThreshold <= 0 {
			t.Errorf("year %d: ProgressiveThreshold = %d, expected > 0", year, constants.ProgressiveThreshold)
		}
		if len(constants.FlatRateCaps) == 0 {
			t.Errorf("year %d: FlatRateCaps is empty", year)
		}
	}
}
