package service

import (
	"context"
	"fmt"
	"time"

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

	result := &AnnualTaxBase{}

	for _, inv := range invoices {
		// Use DeliveryDate if set, otherwise IssueDate.
		effectiveDate := inv.DeliveryDate
		if effectiveDate.IsZero() {
			effectiveDate = inv.IssueDate
		}
		if effectiveDate.Before(dateFrom) || effectiveDate.After(dateTo) {
			continue
		}

		// Only include sent, paid, overdue invoices.
		if inv.Status != domain.InvoiceStatusSent &&
			inv.Status != domain.InvoiceStatusPaid &&
			inv.Status != domain.InvoiceStatusOverdue {
			continue
		}

		// Skip proformas.
		if inv.Type == domain.InvoiceTypeProforma {
			continue
		}

		result.InvoiceIDs = append(result.InvoiceIDs, inv.ID)

		// Credit notes subtract from revenue.
		if inv.Type == domain.InvoiceTypeCreditNote {
			result.Revenue -= inv.SubtotalAmount
		} else {
			result.Revenue += inv.SubtotalAmount
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

	for _, exp := range expenses {
		if exp.IssueDate.Before(dateFrom) || exp.IssueDate.After(dateTo) {
			continue
		}
		// Only include tax-reviewed expenses.
		if exp.TaxReviewedAt == nil {
			continue
		}

		result.ExpenseIDs = append(result.ExpenseIDs, exp.ID)

		// Expense base = (Amount - VATAmount) * BusinessPercent / 100
		baseAmount := exp.Amount - exp.VATAmount
		businessPct := exp.BusinessPercent
		if businessPct == 0 {
			businessPct = 100
		}
		result.Expenses += baseAmount.Multiply(float64(businessPct) / 100.0)
	}

	return result, nil
}
