package domain

import "time"

// Expense represents a business expense / received invoice.
type Expense struct {
	ID            int64
	VendorID      *int64
	Vendor        *Contact
	ExpenseNumber string
	Category      string
	Description   string

	IssueDate    time.Time
	Amount       Amount
	CurrencyCode string
	ExchangeRate Amount

	VATRatePercent int
	VATAmount      Amount

	IsTaxDeductible bool
	BusinessPercent int // 0-100, percentage used for business
	PaymentMethod   string

	DocumentPath string
	Notes        string

	TaxReviewedAt *time.Time

	// Line items
	Items []ExpenseItem

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// ExpenseItem represents a single line item on an expense.
type ExpenseItem struct {
	ID             int64
	ExpenseID      int64
	Description    string
	Quantity       Amount // stored as cents for decimal precision (e.g. 250 = 2.50)
	Unit           string // ks, hod, m2, etc.
	UnitPrice      Amount // price per unit in halere
	VATRatePercent int    // 0, 12, 21
	VATAmount      Amount
	TotalAmount    Amount // including VAT
	SortOrder      int
}

// CalculateTotals recalculates Amount and VATAmount from expense items.
// When Items is empty, leaves fields untouched (backward compat).
func (e *Expense) CalculateTotals() {
	if len(e.Items) == 0 {
		return
	}

	var subtotal, vat, total Amount
	// Track which VAT rate has the highest subtotal share for dominant rate.
	rateSubtotals := make(map[int]Amount)

	for i := range e.Items {
		item := &e.Items[i]
		// item subtotal before VAT = quantity * unit_price / 100
		// (quantity is in cents, so we divide by 100)
		itemSubtotal := Amount(int64(item.Quantity) * int64(item.UnitPrice) / 100)
		itemVAT := itemSubtotal.Multiply(float64(item.VATRatePercent) / 100.0)
		item.VATAmount = itemVAT
		item.TotalAmount = itemSubtotal.Add(itemVAT)

		subtotal = subtotal.Add(itemSubtotal)
		vat = vat.Add(itemVAT)
		total = total.Add(item.TotalAmount)

		rateSubtotals[item.VATRatePercent] = rateSubtotals[item.VATRatePercent].Add(itemSubtotal)
	}

	e.Amount = total
	e.VATAmount = vat

	// Set dominant VAT rate (rate with highest subtotal share).
	var maxSubtotal Amount
	for rate, sub := range rateSubtotals {
		if sub > maxSubtotal {
			maxSubtotal = sub
			e.VATRatePercent = rate
		}
	}
}
