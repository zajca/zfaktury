package service

import (
	"context"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/calc"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// AnnualTaxBase holds the computed annual revenue and expenses for tax purposes.
type AnnualTaxBase struct {
	Revenue    domain.Amount
	Expenses   domain.Amount
	InvoiceIDs []int64
	ExpenseIDs []int64
}

// CalculateAnnualBase computes the annual revenue and expenses from invoices and expenses.
// Revenue = sum of SubtotalAmount from invoices where status IN (sent, paid, overdue),
// DeliveryDate (or IssueDate) in the given year, type is standard or credit_note.
// Expenses = sum of (Amount - VATAmount) * BusinessPercent/100 from tax-reviewed expenses.
func CalculateAnnualBase(
	ctx context.Context,
	invoiceRepo repository.InvoiceRepo,
	expenseRepo repository.ExpenseRepo,
	year int,
) (*AnnualTaxBase, error) {
	dateFrom := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	invoices, _, err := invoiceRepo.List(ctx, domain.InvoiceFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		Limit:    100000,
		Offset:   0,
	})
	if err != nil {
		return nil, fmt.Errorf("listing invoices for annual base: %w", err)
	}

	// Map invoices to calc inputs.
	calcInvoices := make([]calc.InvoiceForBase, len(invoices))
	for i, inv := range invoices {
		calcInvoices[i] = calc.InvoiceForBase{
			ID:             inv.ID,
			Type:           inv.Type,
			Status:         inv.Status,
			DeliveryDate:   inv.DeliveryDate,
			IssueDate:      inv.IssueDate,
			SubtotalAmount: inv.SubtotalAmount,
		}
	}

	expenses, _, err := expenseRepo.List(ctx, domain.ExpenseFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		Limit:    100000,
		Offset:   0,
	})
	if err != nil {
		return nil, fmt.Errorf("listing expenses for annual base: %w", err)
	}

	// Map expenses to calc inputs.
	calcExpenses := make([]calc.ExpenseForBase, len(expenses))
	for i, exp := range expenses {
		calcExpenses[i] = calc.ExpenseForBase{
			ID:              exp.ID,
			IssueDate:       exp.IssueDate,
			Amount:          exp.Amount,
			VATAmount:       exp.VATAmount,
			BusinessPercent: exp.BusinessPercent,
			TaxReviewed:     exp.TaxReviewedAt != nil,
		}
	}

	// Pure calculation.
	calcResult := calc.CalculateAnnualTotals(calcInvoices, calcExpenses, year)

	return &AnnualTaxBase{
		Revenue:    calcResult.Revenue,
		Expenses:   calcResult.Expenses,
		InvoiceIDs: calcResult.InvoiceIDs,
		ExpenseIDs: calcResult.ExpenseIDs,
	}, nil
}
