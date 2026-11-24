package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

// dateFilterCases are issue dates around the month of March 2025; only the
// first and the last day of the month lie inside the filter.
var dateFilterCases = []struct {
	name    string
	date    time.Time
	inRange bool
}{
	{"day before the period", time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC), false},
	{"first day of the period", time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), true},
	{"last day of the period", time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC), true},
	{"day after the period", time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), false},
}

func TestExpenseRepository_List_DateFilterIncludesBoundaryDays(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	idByCase := map[string]int64{}
	for _, tc := range dateFilterCases {
		e := testutil.SeedExpense(t, db, &domain.Expense{Description: tc.name, IssueDate: tc.date})
		idByCase[tc.name] = e.ID
	}

	from := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)
	got, total, err := repo.List(ctx, domain.ExpenseFilter{DateFrom: &from, DateTo: &to})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	listed := map[int64]bool{}
	for _, e := range got {
		listed[e.ID] = true
	}
	for _, tc := range dateFilterCases {
		if listed[idByCase[tc.name]] != tc.inRange {
			t.Errorf("%s: listed = %v, want %v", tc.name, listed[idByCase[tc.name]], tc.inRange)
		}
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
}

func TestInvoiceRepository_List_DateFilterIncludesBoundaryDays(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	idByCase := map[string]int64{}
	for i, tc := range dateFilterCases {
		inv := makeRepoInvoice(customerID, seqID)
		inv.InvoiceNumber = fmt.Sprintf("FVDATE%04d", i)
		inv.IssueDate = tc.date
		inv.DueDate = tc.date.AddDate(0, 0, 14)
		inv.DeliveryDate = tc.date
		if err := repo.Create(ctx, inv); err != nil {
			t.Fatalf("Create(%s) error: %v", tc.name, err)
		}
		idByCase[tc.name] = inv.ID
	}

	from := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)
	got, total, err := repo.List(ctx, domain.InvoiceFilter{DateFrom: &from, DateTo: &to})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	listed := map[int64]bool{}
	for _, inv := range got {
		listed[inv.ID] = true
	}
	for _, tc := range dateFilterCases {
		if listed[idByCase[tc.name]] != tc.inRange {
			t.Errorf("%s: listed = %v, want %v", tc.name, listed[idByCase[tc.name]], tc.inRange)
		}
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
}
