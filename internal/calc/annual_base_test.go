package calc

import (
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestCalculateAnnualTotals(t *testing.T) {
	year := 2025
	inYear := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	outOfYear := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		invoices       []InvoiceForBase
		expenses       []ExpenseForBase
		wantRevenue    domain.Amount
		wantExpenses   domain.Amount
		wantInvoiceIDs []int64
		wantExpenseIDs []int64
	}{
		{
			name:           "no invoices no expenses",
			invoices:       nil,
			expenses:       nil,
			wantRevenue:    0,
			wantExpenses:   0,
			wantInvoiceIDs: nil,
			wantExpenseIDs: nil,
		},
		{
			name: "regular invoice in year added to revenue",
			invoices: []InvoiceForBase{
				{ID: 1, Type: domain.InvoiceTypeRegular, Status: domain.InvoiceStatusPaid, IssueDate: inYear, SubtotalAmount: domain.NewAmount(10000, 0)},
			},
			wantRevenue:    domain.NewAmount(10000, 0),
			wantInvoiceIDs: []int64{1},
			wantExpenseIDs: nil,
		},
		{
			name: "proforma invoice excluded",
			invoices: []InvoiceForBase{
				{ID: 2, Type: domain.InvoiceTypeProforma, Status: domain.InvoiceStatusPaid, IssueDate: inYear, SubtotalAmount: domain.NewAmount(5000, 0)},
			},
			wantRevenue:    0,
			wantInvoiceIDs: nil,
			wantExpenseIDs: nil,
		},
		{
			name: "draft invoice excluded",
			invoices: []InvoiceForBase{
				{ID: 3, Type: domain.InvoiceTypeRegular, Status: domain.InvoiceStatusDraft, IssueDate: inYear, SubtotalAmount: domain.NewAmount(5000, 0)},
			},
			wantRevenue:    0,
			wantInvoiceIDs: nil,
			wantExpenseIDs: nil,
		},
		{
			name: "cancelled invoice excluded",
			invoices: []InvoiceForBase{
				{ID: 4, Type: domain.InvoiceTypeRegular, Status: domain.InvoiceStatusCancelled, IssueDate: inYear, SubtotalAmount: domain.NewAmount(5000, 0)},
			},
			wantRevenue:    0,
			wantInvoiceIDs: nil,
			wantExpenseIDs: nil,
		},
		{
			name: "credit note subtracted from revenue",
			invoices: []InvoiceForBase{
				{ID: 5, Type: domain.InvoiceTypeCreditNote, Status: domain.InvoiceStatusSent, IssueDate: inYear, SubtotalAmount: domain.NewAmount(3000, 0)},
			},
			wantRevenue:    -domain.NewAmount(3000, 0),
			wantInvoiceIDs: []int64{5},
			wantExpenseIDs: nil,
		},
		{
			name: "invoice with DeliveryDate in year included",
			invoices: []InvoiceForBase{
				{ID: 6, Type: domain.InvoiceTypeRegular, Status: domain.InvoiceStatusPaid, IssueDate: outOfYear, DeliveryDate: inYear, SubtotalAmount: domain.NewAmount(7000, 0)},
			},
			wantRevenue:    domain.NewAmount(7000, 0),
			wantInvoiceIDs: []int64{6},
			wantExpenseIDs: nil,
		},
		{
			name: "invoice with DeliveryDate outside year excluded",
			invoices: []InvoiceForBase{
				{ID: 7, Type: domain.InvoiceTypeRegular, Status: domain.InvoiceStatusPaid, IssueDate: inYear, DeliveryDate: outOfYear, SubtotalAmount: domain.NewAmount(7000, 0)},
			},
			wantRevenue:    0,
			wantInvoiceIDs: nil,
			wantExpenseIDs: nil,
		},
		{
			name: "invoice without DeliveryDate uses IssueDate",
			invoices: []InvoiceForBase{
				{ID: 8, Type: domain.InvoiceTypeRegular, Status: domain.InvoiceStatusOverdue, IssueDate: inYear, SubtotalAmount: domain.NewAmount(2000, 0)},
			},
			wantRevenue:    domain.NewAmount(2000, 0),
			wantInvoiceIDs: []int64{8},
			wantExpenseIDs: nil,
		},
		{
			name: "expense in year tax-reviewed included",
			expenses: []ExpenseForBase{
				{ID: 10, IssueDate: inYear, Amount: domain.NewAmount(12100, 0), VATAmount: domain.NewAmount(2100, 0), BusinessPercent: 100, TaxReviewed: true},
			},
			wantRevenue:    0,
			wantExpenses:   domain.NewAmount(10000, 0),
			wantInvoiceIDs: nil,
			wantExpenseIDs: []int64{10},
		},
		{
			name: "expense not tax-reviewed excluded",
			expenses: []ExpenseForBase{
				{ID: 11, IssueDate: inYear, Amount: domain.NewAmount(12100, 0), VATAmount: domain.NewAmount(2100, 0), BusinessPercent: 100, TaxReviewed: false},
			},
			wantRevenue:    0,
			wantExpenses:   0,
			wantInvoiceIDs: nil,
			wantExpenseIDs: nil,
		},
		{
			name: "expense outside year excluded",
			expenses: []ExpenseForBase{
				{ID: 12, IssueDate: outOfYear, Amount: domain.NewAmount(12100, 0), VATAmount: domain.NewAmount(2100, 0), BusinessPercent: 100, TaxReviewed: true},
			},
			wantRevenue:    0,
			wantExpenses:   0,
			wantInvoiceIDs: nil,
			wantExpenseIDs: nil,
		},
		{
			name: "expense business percent 50 takes half of base",
			expenses: []ExpenseForBase{
				{ID: 13, IssueDate: inYear, Amount: domain.NewAmount(12100, 0), VATAmount: domain.NewAmount(2100, 0), BusinessPercent: 50, TaxReviewed: true},
			},
			wantRevenue:    0,
			wantExpenses:   domain.NewAmount(5000, 0),
			wantInvoiceIDs: nil,
			wantExpenseIDs: []int64{13},
		},
		{
			name: "expense business percent 0 treated as 100",
			expenses: []ExpenseForBase{
				{ID: 14, IssueDate: inYear, Amount: domain.NewAmount(12100, 0), VATAmount: domain.NewAmount(2100, 0), BusinessPercent: 0, TaxReviewed: true},
			},
			wantRevenue:    0,
			wantExpenses:   domain.NewAmount(10000, 0),
			wantInvoiceIDs: nil,
			wantExpenseIDs: []int64{14},
		},
		{
			name: "mixed scenario with IDs verified",
			invoices: []InvoiceForBase{
				{ID: 100, Type: domain.InvoiceTypeRegular, Status: domain.InvoiceStatusPaid, IssueDate: inYear, SubtotalAmount: domain.NewAmount(20000, 0)},
				{ID: 101, Type: domain.InvoiceTypeProforma, Status: domain.InvoiceStatusPaid, IssueDate: inYear, SubtotalAmount: domain.NewAmount(5000, 0)},
				{ID: 102, Type: domain.InvoiceTypeCreditNote, Status: domain.InvoiceStatusSent, IssueDate: inYear, SubtotalAmount: domain.NewAmount(2000, 0)},
				{ID: 103, Type: domain.InvoiceTypeRegular, Status: domain.InvoiceStatusDraft, IssueDate: inYear, SubtotalAmount: domain.NewAmount(3000, 0)},
				{ID: 104, Type: domain.InvoiceTypeRegular, Status: domain.InvoiceStatusSent, DeliveryDate: outOfYear, IssueDate: inYear, SubtotalAmount: domain.NewAmount(4000, 0)},
			},
			expenses: []ExpenseForBase{
				{ID: 200, IssueDate: inYear, Amount: domain.NewAmount(24200, 0), VATAmount: domain.NewAmount(4200, 0), BusinessPercent: 100, TaxReviewed: true},
				{ID: 201, IssueDate: inYear, Amount: domain.NewAmount(6050, 0), VATAmount: domain.NewAmount(1050, 0), BusinessPercent: 50, TaxReviewed: true},
				{ID: 202, IssueDate: inYear, Amount: domain.NewAmount(10000, 0), VATAmount: domain.NewAmount(0, 0), BusinessPercent: 100, TaxReviewed: false},
			},
			// Revenue: 20000 (ID=100) - 2000 (ID=102 credit note) = 18000
			// ID=101 proforma excluded, ID=103 draft excluded, ID=104 delivery date out of year excluded
			wantRevenue: domain.NewAmount(18000, 0),
			// Expenses: (24200-4200)*1.0=20000 (ID=200) + (6050-1050)*0.5=2500 (ID=201) = 22500
			// ID=202 not tax-reviewed excluded
			wantExpenses:   domain.NewAmount(22500, 0),
			wantInvoiceIDs: []int64{100, 102},
			wantExpenseIDs: []int64{200, 201},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateAnnualTotals(tt.invoices, tt.expenses, year)

			if result.Revenue != tt.wantRevenue {
				t.Errorf("Revenue = %v, want %v", result.Revenue, tt.wantRevenue)
			}
			if result.Expenses != tt.wantExpenses {
				t.Errorf("Expenses = %v, want %v", result.Expenses, tt.wantExpenses)
			}
			if !int64SliceEqual(result.InvoiceIDs, tt.wantInvoiceIDs) {
				t.Errorf("InvoiceIDs = %v, want %v", result.InvoiceIDs, tt.wantInvoiceIDs)
			}
			if !int64SliceEqual(result.ExpenseIDs, tt.wantExpenseIDs) {
				t.Errorf("ExpenseIDs = %v, want %v", result.ExpenseIDs, tt.wantExpenseIDs)
			}
		})
	}
}

func int64SliceEqual(a, b []int64) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
