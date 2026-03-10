package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"

	"github.com/zajca/zfaktury/internal/repository"
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
