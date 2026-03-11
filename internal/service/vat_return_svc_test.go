package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestVATReturnService_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType: domain.FilingTypeRegular,
	}

	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if vr.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if vr.Status != "draft" {
		t.Errorf("Status = %q, want %q", vr.Status, "draft")
	}
}

func TestVATReturnService_Create_DuplicateRegular(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Second regular filing for same period should fail.
	vr2 := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr2); err == nil {
		t.Error("expected error for duplicate regular filing")
	}
}

func TestVATReturnService_Create_InvalidInput(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	ctx := context.Background()

	// Missing year.
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err == nil {
		t.Error("expected error for missing year")
	}

	// Missing month and quarter.
	vr2 := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr2); err == nil {
		t.Error("expected error for missing month and quarter")
	}
}

func TestVATReturnService_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.GetByID(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Period.Month != 1 {
		t.Errorf("Month = %d, want 1", got.Period.Month)
	}
}

func TestVATReturnService_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, vr.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := svc.GetByID(ctx, vr.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestVATReturnService_Delete_Filed(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Mark as filed.
	_, err := svc.MarkFiled(ctx, vr.ID)
	if err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	// Try to delete.
	if err := svc.Delete(ctx, vr.ID); err == nil {
		t.Error("expected error when deleting filed vat return")
	}
}

func TestVATReturnService_MarkFiled(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 2,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	filed, err := svc.MarkFiled(ctx, vr.ID)
	if err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}
	if filed.Status != "filed" {
		t.Errorf("Status = %q, want %q", filed.Status, "filed")
	}
	if filed.FiledAt == nil {
		t.Error("expected FiledAt to be set")
	}

	// Cannot mark as filed again.
	_, err = svc.MarkFiled(ctx, vr.ID)
	if err == nil {
		t.Error("expected error for double filing")
	}
}

func TestVATReturnService_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	ctx := context.Background()

	for _, m := range []int{1, 2, 3} {
		vr := &domain.VATReturn{
			Period: domain.TaxPeriod{
				Year:  2025,
				Month: m,
			},
			FilingType: domain.FilingTypeRegular,
		}
		if err := svc.Create(ctx, vr); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	returns, err := svc.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns) != 3 {
		t.Errorf("List() returned %d items, want 3", len(returns))
	}
}

func TestVATReturnService_Recalculate(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	ctx := context.Background()

	// Create a contact and invoice in March 2025.
	contact := testutil.SeedContact(t, db, nil)

	// Seed invoice with delivery date in March 2025, status=sent.
	march15 := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	inv := &domain.Invoice{
		InvoiceNumber: "FV20250001",
		Type:          domain.InvoiceTypeRegular,
		Status:        domain.InvoiceStatusSent,
		IssueDate:     march15,
		DueDate:       march15.AddDate(0, 0, 14),
		DeliveryDate:  march15,
		CustomerID:    contact.ID,
		CurrencyCode:  domain.CurrencyCZK,
		PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{
				Description:    "Consulting",
				Quantity:       100, // 1.00
				Unit:           "hod",
				UnitPrice:      100000, // 1000.00 CZK
				VATRatePercent: 21,
			},
		},
	}
	inv.CalculateTotals()
	now := time.Now()
	inv.CreatedAt = now
	inv.UpdatedAt = now

	// Insert invoice directly.
	result, err := db.ExecContext(ctx, `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.InvoiceNumber, inv.Type, inv.Status,
		inv.IssueDate.Format("2006-01-02"), inv.DueDate.Format("2006-01-02"), inv.DeliveryDate.Format("2006-01-02"),
		"", "", inv.CustomerID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		inv.SubtotalAmount, inv.VATAmount, inv.TotalAmount, 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice: %v", err)
	}
	invID, _ := result.LastInsertId()
	inv.ID = invID

	// Insert invoice items.
	for i := range inv.Items {
		item := &inv.Items[i]
		item.InvoiceID = invID
		itemResult, err := db.ExecContext(ctx, `
			INSERT INTO invoice_items (
				invoice_id, description, quantity, unit, unit_price,
				vat_rate_percent, vat_amount, total_amount, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.InvoiceID, item.Description, item.Quantity, item.Unit, item.UnitPrice,
			item.VATRatePercent, item.VATAmount, item.TotalAmount, item.SortOrder,
		)
		if err != nil {
			t.Fatalf("inserting invoice item: %v", err)
		}
		itemID, _ := itemResult.LastInsertId()
		item.ID = itemID
	}

	// Create VAT return for March 2025.
	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Recalculate.
	result2, err := svc.Recalculate(ctx, vr.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	// The invoice has 1 item: 100 qty * 100000 unit_price / 100 = 100000 base (1000.00 CZK).
	// VAT at 21% = 21000 (210.00 CZK).
	if result2.OutputVATBase21 != 100000 {
		t.Errorf("OutputVATBase21 = %d, want 100000", result2.OutputVATBase21)
	}
	if result2.OutputVATAmount21 != 21000 {
		t.Errorf("OutputVATAmount21 = %d, want 21000", result2.OutputVATAmount21)
	}
	if result2.TotalOutputVAT != 21000 {
		t.Errorf("TotalOutputVAT = %d, want 21000", result2.TotalOutputVAT)
	}
	if result2.NetVAT != 21000 {
		t.Errorf("NetVAT = %d, want 21000", result2.NetVAT)
	}
}

func TestPeriodDateRange(t *testing.T) {
	tests := []struct {
		name     string
		period   domain.TaxPeriod
		wantFrom string
		wantTo   string
	}{
		{
			name:     "monthly January",
			period:   domain.TaxPeriod{Year: 2025, Month: 1},
			wantFrom: "2025-01-01",
			wantTo:   "2025-01-31",
		},
		{
			name:     "monthly February non-leap",
			period:   domain.TaxPeriod{Year: 2025, Month: 2},
			wantFrom: "2025-02-01",
			wantTo:   "2025-02-28",
		},
		{
			name:     "quarterly Q1",
			period:   domain.TaxPeriod{Year: 2025, Quarter: 1},
			wantFrom: "2025-01-01",
			wantTo:   "2025-03-31",
		},
		{
			name:     "quarterly Q4",
			period:   domain.TaxPeriod{Year: 2025, Quarter: 4},
			wantFrom: "2025-10-01",
			wantTo:   "2025-12-31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to := periodDateRange(tt.period)
			if from.Format("2006-01-02") != tt.wantFrom {
				t.Errorf("from = %s, want %s", from.Format("2006-01-02"), tt.wantFrom)
			}
			if to.Format("2006-01-02") != tt.wantTo {
				t.Errorf("to = %s, want %s", to.Format("2006-01-02"), tt.wantTo)
			}
		})
	}
}

// --- Additional tests ---

func setupVATReturnSvc(t *testing.T) (*VATReturnService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	svc := NewVATReturnService(vatRepo, invRepo, expRepo, setRepo)
	return svc, db
}

func TestVATReturnService_List_Empty(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	result, err := svc.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("List() returned %d items, want 0", len(result))
	}
}

func TestVATReturnService_List_DefaultYear(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	// List with year=0 should default to current year.
	result, err := svc.List(ctx, 0)
	if err != nil {
		t.Fatalf("List(0) error: %v", err)
	}
	// Just verify it doesn't error; may return 0 items.
	_ = result
}

func TestVATReturnService_List_FiltersByYear(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	// Create returns for different years and months.
	entries := []struct{ year, month int }{
		{2024, 1},
		{2025, 2},
		{2025, 3},
	}
	for _, e := range entries {
		vr := &domain.VATReturn{
			Period:     domain.TaxPeriod{Year: e.year, Month: e.month},
			FilingType: domain.FilingTypeRegular,
		}
		if err := svc.Create(ctx, vr); err != nil {
			t.Fatalf("Create(%d/%d) error: %v", e.year, e.month, err)
		}
	}

	result2025, err := svc.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List(2025) error: %v", err)
	}
	if len(result2025) != 2 {
		t.Errorf("List(2025) returned %d items, want 2", len(result2025))
	}

	result2024, err := svc.List(ctx, 2024)
	if err != nil {
		t.Fatalf("List(2024) error: %v", err)
	}
	if len(result2024) != 1 {
		t.Errorf("List(2024) returned %d items, want 1", len(result2024))
	}
}

func TestVATReturnService_Recalculate_Filed(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 4},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Mark as filed.
	if _, err := svc.MarkFiled(ctx, vr.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	// Recalculate filed return should fail.
	_, err := svc.Recalculate(ctx, vr.ID)
	if err == nil {
		t.Error("Recalculate() should return error for filed return")
	}
}

func TestVATReturnService_Recalculate_WithExpenses(t *testing.T) {
	svc, db := setupVATReturnSvc(t)
	ctx := context.Background()

	// Seed a vendor contact.
	vendor := testutil.SeedContact(t, db, &domain.Contact{
		Name: "Vendor s.r.o.",
		DIC:  "CZ55667788",
	})

	// Seed a tax-deductible expense in March 2025.
	march10 := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	testutil.SeedExpense(t, db, &domain.Expense{
		VendorID:        &vendor.ID,
		ExpenseNumber:   "VF2025001",
		Description:     "Office supplies",
		IssueDate:       march10,
		Amount:          domain.NewAmount(12100, 0), // 12100.00 CZK (10000 base + 2100 VAT)
		VATRatePercent:  21,
		VATAmount:       domain.NewAmount(2100, 0),
		IsTaxDeductible: true,
		BusinessPercent: 100,
	})

	// Create VAT return for March 2025.
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, vr.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	// Input VAT at 21%: base = 12100 - 2100 = 10000 (1000000 halere), VAT = 2100 (210000 halere).
	if result.InputVATBase21 != domain.NewAmount(10000, 0) {
		t.Errorf("InputVATBase21 = %d, want %d", result.InputVATBase21, domain.NewAmount(10000, 0))
	}
	if result.InputVATAmount21 != domain.NewAmount(2100, 0) {
		t.Errorf("InputVATAmount21 = %d, want %d", result.InputVATAmount21, domain.NewAmount(2100, 0))
	}
	if result.TotalInputVAT != domain.NewAmount(2100, 0) {
		t.Errorf("TotalInputVAT = %d, want %d", result.TotalInputVAT, domain.NewAmount(2100, 0))
	}
	// NetVAT = 0 output - 2100 input = -2100 (refund).
	if result.NetVAT != -domain.NewAmount(2100, 0) {
		t.Errorf("NetVAT = %d, want %d", result.NetVAT, -domain.NewAmount(2100, 0))
	}
}

func TestVATReturnService_Recalculate_WithInvoiceAndExpense(t *testing.T) {
	svc, db := setupVATReturnSvc(t)
	ctx := context.Background()

	// Seed customer contact.
	customer := testutil.SeedContact(t, db, &domain.Contact{
		Name: "Customer a.s.",
		DIC:  "CZ11223344",
	})

	// Seed vendor contact.
	vendor := testutil.SeedContact(t, db, &domain.Contact{
		Name: "Vendor s.r.o.",
		DIC:  "CZ55667788",
	})

	march15 := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

	// Insert invoice directly with correct dates (SeedInvoice uses "now" for dates).
	now := time.Now()
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
		"FV20250099", domain.InvoiceTypeRegular, domain.InvoiceStatusSent,
		march15.Format("2006-01-02"), march15.AddDate(0, 0, 14).Format("2006-01-02"), march15.Format("2006-01-02"),
		"", "", customer.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		100000, 21000, 121000, 0, // 1000 CZK base, 210 CZK VAT, 1210 CZK total
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice: %v", err)
	}
	var invID int64
	row := db.QueryRowContext(ctx, "SELECT last_insert_rowid()")
	if err := row.Scan(&invID); err != nil {
		t.Fatalf("getting invoice id: %v", err)
	}

	// Insert invoice item.
	_, err = db.ExecContext(ctx, `
		INSERT INTO invoice_items (
			invoice_id, description, quantity, unit, unit_price,
			vat_rate_percent, vat_amount, total_amount, sort_order
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invID, "Service", 100, "ks", 100000,
		21, 21000, 121000, 0,
	)
	if err != nil {
		t.Fatalf("inserting invoice item: %v", err)
	}

	// Seed expense: 6050 CZK (5000 base + 1050 VAT).
	testutil.SeedExpense(t, db, &domain.Expense{
		VendorID:        &vendor.ID,
		ExpenseNumber:   "VF2025010",
		Description:     "Materials",
		IssueDate:       march15,
		Amount:          domain.NewAmount(6050, 0),
		VATRatePercent:  21,
		VATAmount:       domain.NewAmount(1050, 0),
		IsTaxDeductible: true,
		BusinessPercent: 100,
	})

	// Create VAT return.
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, vr.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	// Output: 100000 base, 21000 VAT.
	if result.OutputVATBase21 != 100000 {
		t.Errorf("OutputVATBase21 = %d, want 100000", result.OutputVATBase21)
	}
	if result.OutputVATAmount21 != 21000 {
		t.Errorf("OutputVATAmount21 = %d, want 21000", result.OutputVATAmount21)
	}

	// Input: 5000 CZK = 500000 halere base, 1050 CZK = 105000 halere VAT.
	if result.InputVATBase21 != domain.NewAmount(5000, 0) {
		t.Errorf("InputVATBase21 = %d, want %d", result.InputVATBase21, domain.NewAmount(5000, 0))
	}
	if result.InputVATAmount21 != domain.NewAmount(1050, 0) {
		t.Errorf("InputVATAmount21 = %d, want %d", result.InputVATAmount21, domain.NewAmount(1050, 0))
	}

	// Net = 21000 - 105000 = -84000 (refund).
	expectedNet := domain.Amount(21000) - domain.NewAmount(1050, 0)
	if result.NetVAT != expectedNet {
		t.Errorf("NetVAT = %d, want %d", result.NetVAT, expectedNet)
	}
}

func TestVATReturnService_Recalculate_ZeroID(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	_, err := svc.Recalculate(ctx, 0)
	if err == nil {
		t.Error("Recalculate(0) should return error")
	}
}

func TestVATReturnService_Recalculate_NotFound(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	_, err := svc.Recalculate(ctx, 9999)
	if err == nil {
		t.Error("Recalculate(9999) should return error for non-existent ID")
	}
}

func TestVATReturnService_GenerateXML_Basic(t *testing.T) {
	svc, db := setupVATReturnSvc(t)
	ctx := context.Background()

	// Insert DIC setting.
	_, err := db.ExecContext(ctx, "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("inserting DIC setting: %v", err)
	}

	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.GenerateXML(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GenerateXML() error: %v", err)
	}
	if len(result.XMLData) == 0 {
		t.Error("GenerateXML() should produce non-empty XML")
	}

	// Verify XML is persisted.
	fetched, err := svc.GetByID(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if len(fetched.XMLData) == 0 {
		t.Error("XMLData should be persisted after GenerateXML")
	}
}

func TestVATReturnService_GenerateXML_NoDIC(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Without DIC setting, GenerateXML should fail.
	_, err := svc.GenerateXML(ctx, vr.ID)
	if err == nil {
		t.Error("GenerateXML() should return error when DIC setting is missing")
	}
}

func TestVATReturnService_GenerateXML_ZeroID(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	_, err := svc.GenerateXML(ctx, 0)
	if err == nil {
		t.Error("GenerateXML(0) should return error")
	}
}

func TestVATReturnService_GetXMLData_Basic(t *testing.T) {
	svc, db := setupVATReturnSvc(t)
	ctx := context.Background()

	// Insert DIC setting.
	_, err := db.ExecContext(ctx, "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("inserting DIC setting: %v", err)
	}

	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 5},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Before generating XML, GetXMLData should return nil/empty.
	data, err := svc.GetXMLData(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetXMLData() error: %v", err)
	}
	if len(data) != 0 {
		t.Error("GetXMLData() should return empty before GenerateXML")
	}

	// Generate XML.
	if _, err := svc.GenerateXML(ctx, vr.ID); err != nil {
		t.Fatalf("GenerateXML() error: %v", err)
	}

	// Now GetXMLData should return data.
	data, err = svc.GetXMLData(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetXMLData() after generate error: %v", err)
	}
	if len(data) == 0 {
		t.Error("GetXMLData() should return non-empty after GenerateXML")
	}
}

func TestVATReturnService_GetXMLData_ZeroID(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	_, err := svc.GetXMLData(ctx, 0)
	if err == nil {
		t.Error("GetXMLData(0) should return error")
	}
}

func TestVATReturnService_GetXMLData_NotFound(t *testing.T) {
	svc, _ := setupVATReturnSvc(t)
	ctx := context.Background()

	_, err := svc.GetXMLData(ctx, 9999)
	if err == nil {
		t.Error("GetXMLData(9999) should return error for non-existent ID")
	}
}

func TestVATReturnService_Recalculate_PartialBusinessPercent(t *testing.T) {
	svc, db := setupVATReturnSvc(t)
	ctx := context.Background()

	vendor := testutil.SeedContact(t, db, &domain.Contact{
		Name: "Vendor",
		DIC:  "CZ11112222",
	})

	march5 := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)
	// Expense with 60% business use.
	testutil.SeedExpense(t, db, &domain.Expense{
		VendorID:        &vendor.ID,
		ExpenseNumber:   "VF2025020",
		Description:     "Car fuel",
		IssueDate:       march5,
		Amount:          domain.NewAmount(12100, 0), // 12100 CZK total
		VATRatePercent:  21,
		VATAmount:       domain.NewAmount(2100, 0), // 2100 CZK VAT
		IsTaxDeductible: true,
		BusinessPercent: 60,
	})

	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, vr.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	// Input VAT at 60%: base = (12100-2100)*0.6 = 6000 CZK = 600000 halere.
	// VAT = 2100*0.6 = 1260 CZK = 126000 halere.
	expectedBase := domain.NewAmount(10000, 0).Multiply(0.6)
	expectedVAT := domain.NewAmount(2100, 0).Multiply(0.6)

	if result.InputVATBase21 != expectedBase {
		t.Errorf("InputVATBase21 = %d, want %d", result.InputVATBase21, expectedBase)
	}
	if result.InputVATAmount21 != expectedVAT {
		t.Errorf("InputVATAmount21 = %d, want %d", result.InputVATAmount21, expectedVAT)
	}
}
