package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newRecurringExpenseTestStack(t *testing.T) (*RecurringExpenseService, *ExpenseService) {
	t.Helper()
	db := testutil.NewTestDB(t)
	expRepo := repository.NewExpenseRepository(db)
	expSvc := NewExpenseService(expRepo, nil)
	recRepo := repository.NewRecurringExpenseRepository(db)
	recSvc := NewRecurringExpenseService(recRepo, expSvc, nil)
	return recSvc, expSvc
}

func makeRecurringExpense() *domain.RecurringExpense {
	return &domain.RecurringExpense{
		Name:            "Monthly hosting",
		Description:     "Cloud hosting fee",
		Amount:          domain.NewAmount(1000, 0),
		CurrencyCode:    domain.CurrencyCZK,
		Frequency:       "monthly",
		NextIssueDate:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		IsActive:        true,
		BusinessPercent: 100,
		PaymentMethod:   "bank_transfer",
	}
}

func TestRecurringExpenseService_Create_Valid(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if re.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestRecurringExpenseService_Create_EmptyName(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.Name = ""
	if err := svc.Create(ctx, re); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRecurringExpenseService_Create_EmptyDescription(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.Description = ""
	if err := svc.Create(ctx, re); err == nil {
		t.Error("expected error for empty description")
	}
}

func TestRecurringExpenseService_Create_ZeroAmount(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.Amount = 0
	if err := svc.Create(ctx, re); err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestRecurringExpenseService_Create_MissingNextIssueDate(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.NextIssueDate = time.Time{}
	if err := svc.Create(ctx, re); err == nil {
		t.Error("expected error for missing next issue date")
	}
}

func TestRecurringExpenseService_Create_InvalidFrequency(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.Frequency = "biweekly"
	if err := svc.Create(ctx, re); err == nil {
		t.Error("expected error for invalid frequency")
	}
}

func TestRecurringExpenseService_Create_InvalidBusinessPercent(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
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
			re := makeRecurringExpense()
			re.BusinessPercent = tt.percent
			if err := svc.Create(ctx, re); err == nil {
				t.Error("expected error for invalid business percent")
			}
		})
	}
}

func TestRecurringExpenseService_Create_DefaultCurrency(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.CurrencyCode = ""
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if re.CurrencyCode != domain.CurrencyCZK {
		t.Errorf("CurrencyCode = %q, want %q", re.CurrencyCode, domain.CurrencyCZK)
	}
}

func TestRecurringExpenseService_Create_VATCalculation(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.Amount = domain.NewAmount(12100, 0)
	re.VATRatePercent = 21
	re.VATAmount = 0

	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	expectedVAT := re.Amount.Multiply(21.0 / 121.0)
	if re.VATAmount != expectedVAT {
		t.Errorf("VATAmount = %d, want %d", re.VATAmount, expectedVAT)
	}
}

func TestRecurringExpenseService_Update_Valid(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	re.Name = "Updated name"
	if err := svc.Update(ctx, re); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestRecurringExpenseService_Update_ZeroID(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	if err := svc.Update(ctx, re); err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestRecurringExpenseService_Delete_ZeroID(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	if err := svc.Delete(ctx, 0); err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestRecurringExpenseService_Delete_Success(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(ctx, re.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should not be found after delete.
	if _, err := svc.GetByID(ctx, re.ID); err == nil {
		t.Error("expected error after delete")
	}
}

func TestRecurringExpenseService_GetByID_ZeroID(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	if _, err := svc.GetByID(ctx, 0); err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestRecurringExpenseService_GetByID_Success(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.GetByID(ctx, re.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != re.Name {
		t.Errorf("Name = %q, want %q", got.Name, re.Name)
	}
}

func TestRecurringExpenseService_Activate_ZeroID(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	if err := svc.Activate(ctx, 0); err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestRecurringExpenseService_Activate_Success(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Deactivate(ctx, re.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	if err := svc.Activate(ctx, re.ID); err != nil {
		t.Fatalf("Activate: %v", err)
	}
}

func TestRecurringExpenseService_Deactivate_ZeroID(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	if err := svc.Deactivate(ctx, 0); err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestRecurringExpenseService_Deactivate_Success(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Deactivate(ctx, re.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
}

func TestRecurringExpenseService_List_HighLimit(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	// Limit > 100 should be capped.
	items, count, err := svc.List(ctx, 200, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
	if len(items) != 0 {
		t.Errorf("len = %d, want 0", len(items))
	}

	// Negative offset should be treated as 0.
	items2, _, err := svc.List(ctx, 10, -5)
	if err != nil {
		t.Fatalf("List with negative offset: %v", err)
	}
	_ = items2
}

func TestRecurringExpenseService_List_DefaultLimit(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		re := makeRecurringExpense()
		if err := svc.Create(ctx, re); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	items, total, err := svc.List(ctx, 0, 0)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(items) != 3 {
		t.Errorf("len = %d, want 3", len(items))
	}
}

func TestRecurringExpenseService_GeneratePending(t *testing.T) {
	svc, expSvc := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.NextIssueDate = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	asOf := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	count, err := svc.GeneratePending(ctx, asOf)
	if err != nil {
		t.Fatalf("GeneratePending() error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Check that expense was created.
	expenses, total, err := expSvc.List(ctx, domain.ExpenseFilter{})
	if err != nil {
		t.Fatalf("List expenses error: %v", err)
	}
	if total != 1 {
		t.Errorf("total expenses = %d, want 1", total)
	}
	if expenses[0].Description != re.Description {
		t.Errorf("expense description = %q, want %q", expenses[0].Description, re.Description)
	}

	// Check that next_issue_date was advanced.
	updated, err := svc.GetByID(ctx, re.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	expectedNext := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !updated.NextIssueDate.Equal(expectedNext) {
		t.Errorf("NextIssueDate = %s, want %s", updated.NextIssueDate.Format("2006-01-02"), expectedNext.Format("2006-01-02"))
	}
}

func TestRecurringExpenseService_GeneratePending_Idempotent(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.NextIssueDate = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	asOf := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	// First run generates.
	count1, err := svc.GeneratePending(ctx, asOf)
	if err != nil {
		t.Fatalf("GeneratePending() 1st error: %v", err)
	}
	if count1 != 1 {
		t.Errorf("1st count = %d, want 1", count1)
	}

	// Second run should generate nothing (next_issue_date advanced past asOf).
	count2, err := svc.GeneratePending(ctx, asOf)
	if err != nil {
		t.Fatalf("GeneratePending() 2nd error: %v", err)
	}
	if count2 != 0 {
		t.Errorf("2nd count = %d, want 0", count2)
	}
}

func TestRecurringExpenseService_GeneratePending_DeactivatesAtEndDate(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	endDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	re := makeRecurringExpense()
	re.NextIssueDate = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	re.EndDate = &endDate
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	asOf := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	count, err := svc.GeneratePending(ctx, asOf)
	if err != nil {
		t.Fatalf("GeneratePending() error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Check that recurring expense was deactivated (next would be April 1, past end March 15).
	updated, err := svc.GetByID(ctx, re.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if updated.IsActive {
		t.Error("expected recurring expense to be deactivated after passing end date")
	}
}

func TestRecurringExpenseService_GeneratePending_SkipsInactive(t *testing.T) {
	svc, _ := newRecurringExpenseTestStack(t)
	ctx := context.Background()

	re := makeRecurringExpense()
	re.NextIssueDate = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if err := svc.Create(ctx, re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Deactivate(ctx, re.ID); err != nil {
		t.Fatalf("Deactivate() error: %v", err)
	}

	asOf := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	count, err := svc.GeneratePending(ctx, asOf)
	if err != nil {
		t.Fatalf("GeneratePending() error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 for inactive recurring expense", count)
	}
}
