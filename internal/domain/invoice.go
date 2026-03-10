package domain

import "time"

// Invoice type constants.
const (
	InvoiceTypeRegular    = "regular"
	InvoiceTypeProforma   = "proforma"
	InvoiceTypeCreditNote = "credit_note"
)

// Invoice status constants.
const (
	InvoiceStatusDraft     = "draft"
	InvoiceStatusSent      = "sent"
	InvoiceStatusPaid      = "paid"
	InvoiceStatusOverdue   = "overdue"
	InvoiceStatusCancelled = "cancelled"
)

// Invoice represents an issued invoice.
type Invoice struct {
	ID             int64  `json:"id"`
	SequenceID     int64  `json:"sequence_id"`
	InvoiceNumber  string `json:"invoice_number"`
	Type           string `json:"type"`   // regular, proforma, credit_note
	Status         string `json:"status"` // draft, sent, paid, overdue, cancelled

	IssueDate      time.Time `json:"issue_date"`
	DueDate        time.Time `json:"due_date"`
	DeliveryDate   time.Time `json:"delivery_date"`
	VariableSymbol string    `json:"variable_symbol"`
	ConstantSymbol string    `json:"constant_symbol"`

	// Customer
	CustomerID int64    `json:"customer_id"`
	Customer   *Contact `json:"customer,omitempty"`

	// Currency
	CurrencyCode string `json:"currency_code"`
	ExchangeRate Amount `json:"exchange_rate"` // stored as cents, e.g. 2534 = 25.34 CZK per unit

	// Payment
	PaymentMethod string `json:"payment_method"`
	BankAccount   string `json:"bank_account"`
	BankCode      string `json:"bank_code"`
	IBAN          string `json:"iban"`
	SWIFT         string `json:"swift"`

	// Amounts
	SubtotalAmount Amount `json:"subtotal_amount"`
	VATAmount      Amount `json:"vat_amount"`
	TotalAmount    Amount `json:"total_amount"`
	PaidAmount     Amount `json:"paid_amount"`

	// Notes
	Notes         string `json:"notes"`
	InternalNotes string `json:"internal_notes"`

	// Event timestamps
	SentAt *time.Time `json:"sent_at,omitempty"`
	PaidAt *time.Time `json:"paid_at,omitempty"`

	// Line items
	Items []InvoiceItem `json:"items"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// InvoiceItem represents a single line item on an invoice.
type InvoiceItem struct {
	ID             int64  `json:"id"`
	InvoiceID      int64  `json:"invoice_id"`
	Description    string `json:"description"`
	Quantity       Amount `json:"quantity"`        // stored as cents for decimal precision (e.g. 250 = 2.50)
	Unit           string `json:"unit"`            // ks, hod, m2, etc.
	UnitPrice      Amount `json:"unit_price"`      // price per unit in halere
	VATRatePercent int    `json:"vat_rate_percent"` // 0, 12, 21
	VATAmount      Amount `json:"vat_amount"`
	TotalAmount    Amount `json:"total_amount"` // including VAT
	SortOrder      int    `json:"sort_order"`
}

// InvoiceSequence defines a numbering sequence for invoices.
type InvoiceSequence struct {
	ID            int64  `json:"id"`
	Prefix        string `json:"prefix"`         // e.g. "FV"
	NextNumber    int    `json:"next_number"`     // next number to assign
	Year          int    `json:"year"`            // sequence year
	FormatPattern string `json:"format_pattern"`  // e.g. "{prefix}{year}{number:04d}"
}

// CalculateTotals recalculates subtotal, VAT, and total from invoice items.
func (inv *Invoice) CalculateTotals() {
	var subtotal, vat, total Amount
	for i := range inv.Items {
		item := &inv.Items[i]
		// item total before VAT = quantity * unit_price / 100
		// (quantity is in cents, so we divide by 100)
		itemSubtotal := Amount(int64(item.Quantity) * int64(item.UnitPrice) / 100)
		itemVAT := itemSubtotal.Multiply(float64(item.VATRatePercent) / 100.0)
		item.VATAmount = itemVAT
		item.TotalAmount = itemSubtotal.Add(itemVAT)

		subtotal = subtotal.Add(itemSubtotal)
		vat = vat.Add(itemVAT)
		total = total.Add(item.TotalAmount)
	}
	inv.SubtotalAmount = subtotal
	inv.VATAmount = vat
	inv.TotalAmount = total
}

// IsOverdue returns true if the invoice is past due and not yet fully paid.
func (inv *Invoice) IsOverdue() bool {
	if inv.Status == InvoiceStatusPaid || inv.Status == InvoiceStatusCancelled {
		return false
	}
	return time.Now().After(inv.DueDate)
}

// IsPaid returns true if the invoice has been fully paid.
func (inv *Invoice) IsPaid() bool {
	return inv.PaidAmount >= inv.TotalAmount && inv.TotalAmount > 0
}
