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

func TestExpenseService_GetByID_Valid(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.ID != e.ID {
		t.Errorf("ID = %d, want %d", got.ID, e.ID)
	}
	if got.Description != "Test expense" {
		t.Errorf("Description = %q, want %q", got.Description, "Test expense")
	}
}

func TestExpenseService_GetByID_NotFound(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent expense")
	}
}

func TestExpenseService_Delete_Valid(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, e.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify it's gone.
	_, err := svc.GetByID(ctx, e.ID)
	if err == nil {
		t.Error("expected error after deleting expense")
	}
}

func TestExpenseService_Delete_NotFound(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent expense")
	}
}

func TestExpenseService_MarkTaxReviewed_Valid(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e1 := makeExpense()
	if err := svc.Create(ctx, e1); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	e2 := makeExpense()
	if err := svc.Create(ctx, e2); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.MarkTaxReviewed(ctx, []int64{e1.ID, e2.ID}); err != nil {
		t.Fatalf("MarkTaxReviewed() error: %v", err)
	}

	// Verify both are marked.
	got1, err := svc.GetByID(ctx, e1.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got1.TaxReviewedAt == nil {
		t.Error("expected TaxReviewedAt to be set for e1")
	}

	got2, err := svc.GetByID(ctx, e2.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got2.TaxReviewedAt == nil {
		t.Error("expected TaxReviewedAt to be set for e2")
	}
}

func TestExpenseService_MarkTaxReviewed_EmptyIDs(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	err := svc.MarkTaxReviewed(ctx, []int64{})
	if err == nil {
		t.Error("expected error for empty IDs")
	}
}

func TestExpenseService_MarkTaxReviewed_TooManyIDs(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	ids := make([]int64, 501)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	err := svc.MarkTaxReviewed(ctx, ids)
	if err == nil {
		t.Error("expected error for too many IDs")
	}
}

func TestExpenseService_UnmarkTaxReviewed_Valid(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Mark first.
	if err := svc.MarkTaxReviewed(ctx, []int64{e.ID}); err != nil {
		t.Fatalf("MarkTaxReviewed() error: %v", err)
	}

	// Verify marked.
	got, err := svc.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.TaxReviewedAt == nil {
		t.Fatal("expected TaxReviewedAt to be set")
	}

	// Unmark.
	if err := svc.UnmarkTaxReviewed(ctx, []int64{e.ID}); err != nil {
		t.Fatalf("UnmarkTaxReviewed() error: %v", err)
	}

	// Verify unmarked.
	got, err = svc.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.TaxReviewedAt != nil {
		t.Error("expected TaxReviewedAt to be nil after unmark")
	}
}

func TestExpenseService_UnmarkTaxReviewed_EmptyIDs(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	err := svc.UnmarkTaxReviewed(ctx, []int64{})
	if err == nil {
		t.Error("expected error for empty IDs")
	}
}

func TestExpenseService_UnmarkTaxReviewed_TooManyIDs(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	ids := make([]int64, 501)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	err := svc.UnmarkTaxReviewed(ctx, ids)
	if err == nil {
		t.Error("expected error for too many IDs")
	}
}

func TestExpenseService_MarkTaxReviewed_DedupIDs(t *testing.T) {
	svc := newExpenseTestStack(t)
	ctx := context.Background()

	e := makeExpense()
	if err := svc.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Pass duplicate IDs -- should not error.
	if err := svc.MarkTaxReviewed(ctx, []int64{e.ID, e.ID, e.ID}); err != nil {
		t.Fatalf("MarkTaxReviewed() with duplicate IDs error: %v", err)
	}

	got, err := svc.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.TaxReviewedAt == nil {
		t.Error("expected TaxReviewedAt to be set")
	}
}

func Test_dedupIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    []int64
		expected []int64
	}{
		{"empty", []int64{}, []int64{}},
		{"no duplicates", []int64{1, 2, 3}, []int64{1, 2, 3}},
		{"all duplicates", []int64{5, 5, 5}, []int64{5}},
		{"mixed", []int64{1, 2, 1, 3, 2, 4}, []int64{1, 2, 3, 4}},
		{"single", []int64{42}, []int64{42}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupIDs(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("got[%d] = %d, want %d", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
