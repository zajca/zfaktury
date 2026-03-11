package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newExpenseTestStack(t *testing.T) *ExpenseService {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewExpenseRepository(db)
	return NewExpenseService(repo)
}

func makeExpense() *domain.Expense {
	return &domain.Expense{
		Description:     "Test expense",
		Amount:          domain.NewAmount(1000, 0),
		IssueDate:       time.Now(),
		CurrencyCode:    domain.CurrencyCZK,
		BusinessPercent: 100,
		PaymentMethod:   "bank_transfer",
	}
}

func TestExpenseService_Create_Valid(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if e.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestExpenseService_Create_EmptyDescription(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	e.Description = ""
	err := svc.Create(ctx, e)
	if err == nil {
		t.Error("expected error for empty description")
	}
}

func TestExpenseService_Create_ZeroAmount(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	e.Amount = 0
	err := svc.Create(ctx, e)
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestExpenseService_Create_MissingIssueDate(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	e.IssueDate = time.Time{}
	err := svc.Create(ctx, e)
	if err == nil {
		t.Error("expected error for missing issue date")
	}
}

func TestExpenseService_Create_DefaultCurrency(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	e.CurrencyCode = ""
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if e.CurrencyCode != domain.CurrencyCZK {
		t.Errorf("CurrencyCode = %q, want %q", e.CurrencyCode, domain.CurrencyCZK)
	}
}

func TestExpenseService_Create_DefaultBusinessPercent(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	e.BusinessPercent = 0
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if e.BusinessPercent != 100 {
		t.Errorf("BusinessPercent = %d, want 100", e.BusinessPercent)
	}
}

func TestExpenseService_Create_InvalidBusinessPercent(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		percent int
	}{
		{"negative", -10},
		{"over 100", 150},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := makeExpense()
			e.BusinessPercent = tt.percent
			err := svc.Create(ctx, e)
			if err == nil {
				t.Error("expected error for invalid business percent")
			}
		})
	}
}

func TestExpenseService_Create_VATCalculation(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	e.Amount = domain.NewAmount(12100, 0) // 12100.00 CZK including 21% VAT
	e.VATRatePercent = 21
	e.VATAmount = 0 // Should be auto-calculated.

	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// VAT from gross: 1210000 * 21 / 121 = 210000 (2100.00 CZK)
	// The formula is: amount * rate / (100 + rate)
	expectedVAT := e.Amount.Multiply(21.0 / 121.0)
	if e.VATAmount != expectedVAT {
		t.Errorf("VATAmount = %d, want %d", e.VATAmount, expectedVAT)
	}
}

func TestExpenseService_Create_VATNotOverridden(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	e.VATRatePercent = 21
	e.VATAmount = 5000 // Manually set -- should NOT be overridden.

	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if e.VATAmount != 5000 {
		t.Errorf("VATAmount = %d, want 5000 (should not be overridden when explicitly set)", e.VATAmount)
	}
}

func TestExpenseService_Update_Valid(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	e.Description = "Updated"
	if err := svc.Update(ctx, e); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestExpenseService_Update_ZeroID(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	err := svc.Update(ctx, e)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestExpenseService_Update_EmptyDescription(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	e.Description = ""
	err := svc.Update(ctx, e)
	if err == nil {
		t.Error("expected error for empty description")
	}
}

func TestExpenseService_Update_ZeroAmount(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	e.Amount = 0
	err := svc.Update(ctx, e)
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestExpenseService_Delete_ZeroID(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestExpenseService_GetByID_ZeroID(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestExpenseService_List_DefaultLimit(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		e := makeExpense()
		if err := svc.Create(ctx, e); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	expenses, total, err := svc.List(ctx, domain.ExpenseFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(expenses) != 3 {
		t.Errorf("len = %d, want 3", len(expenses))
	}
}
