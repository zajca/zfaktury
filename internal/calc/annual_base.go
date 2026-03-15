package calc

import (
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// InvoiceForBase holds the minimal invoice data needed for annual tax base calculation.
type InvoiceForBase struct {
	ID             int64
	Type           string
	Status         string
	DeliveryDate   time.Time
	IssueDate      time.Time
	SubtotalAmount domain.Amount
}

// ExpenseForBase holds the minimal expense data needed for annual tax base calculation.
type ExpenseForBase struct {
	ID              int64
	IssueDate       time.Time
	Amount          domain.Amount
	VATAmount       domain.Amount
	BusinessPercent int
	TaxReviewed     bool
}

// AnnualBaseResult contains the calculated annual revenue, expenses, and the IDs
// of invoices and expenses that contributed to the totals.
type AnnualBaseResult struct {
	Revenue    domain.Amount
	Expenses   domain.Amount
	InvoiceIDs []int64
	ExpenseIDs []int64
}

// CalculateAnnualTotals computes aggregate revenue and expenses for the given year.
// It filters invoices and expenses by date range and applicable status/type rules.
func CalculateAnnualTotals(invoices []InvoiceForBase, expenses []ExpenseForBase, year int) AnnualBaseResult {
	dateFrom := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	var result AnnualBaseResult

	for _, inv := range invoices {
		effectiveDate := inv.DeliveryDate
		if effectiveDate.IsZero() {
			effectiveDate = inv.IssueDate
		}

		if effectiveDate.Before(dateFrom) || effectiveDate.After(dateTo) {
			continue
		}

		if inv.Status != domain.InvoiceStatusSent &&
			inv.Status != domain.InvoiceStatusPaid &&
			inv.Status != domain.InvoiceStatusOverdue {
			continue
		}

		if inv.Type == domain.InvoiceTypeProforma {
			continue
		}

		if inv.Type == domain.InvoiceTypeCreditNote {
			result.Revenue = result.Revenue.Sub(inv.SubtotalAmount)
		} else {
			result.Revenue = result.Revenue.Add(inv.SubtotalAmount)
		}
		result.InvoiceIDs = append(result.InvoiceIDs, inv.ID)
	}

	for _, exp := range expenses {
		if exp.IssueDate.Before(dateFrom) || exp.IssueDate.After(dateTo) {
			continue
		}

		if !exp.TaxReviewed {
			continue
		}

		baseAmount := exp.Amount.Sub(exp.VATAmount)

		businessPct := exp.BusinessPercent
		if businessPct == 0 {
			businessPct = 100
		}

		result.Expenses = result.Expenses.Add(baseAmount.Multiply(float64(businessPct) / 100.0))
		result.ExpenseIDs = append(result.ExpenseIDs, exp.ID)
	}

	return result
}
