package domain

import "time"

// Invoice type constants.
const (
	InvoiceTypeRegular    = "regular"
	InvoiceTypeProforma   = "proforma"
	InvoiceTypeCreditNote = "credit_note"
)

// Invoice relation type constants.
const (
	RelationTypeSettlement = "settlement"
	RelationTypeCreditNote = "credit_note"
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
	ID            int64
	SequenceID    int64
	InvoiceNumber string
	Type          string // regular, proforma, credit_note
	Status        string // draft, sent, paid, overdue, cancelled

	IssueDate      time.Time
	DueDate        time.Time
	DeliveryDate   time.Time
	VariableSymbol string
	ConstantSymbol string

	// Customer
	CustomerID int64
	Customer   *Contact

	// Currency
	CurrencyCode string
	ExchangeRate Amount // stored as cents, e.g. 2534 = 25.34 CZK per unit

	// Payment
	PaymentMethod string
	BankAccount   string
	BankCode      string
	IBAN          string
	SWIFT         string

	// Amounts
	SubtotalAmount Amount
	VATAmount      Amount
	TotalAmount    Amount
	PaidAmount     Amount

	// Notes
	Notes         string
	InternalNotes string

	// Related invoice (for credit notes, settlements)
	RelatedInvoiceID *int64
	RelationType     string // "", "settlement", "credit_note"

	// Event timestamps
	SentAt *time.Time
	PaidAt *time.Time

	// Line items
	Items []InvoiceItem

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// InvoiceItem represents a single line item on an invoice.
type InvoiceItem struct {
	ID             int64
	InvoiceID      int64
	Description    string
	Quantity       Amount // stored as cents for decimal precision (e.g. 250 = 2.50)
	Unit           string // ks, hod, m2, etc.
	UnitPrice      Amount // price per unit in halere
	VATRatePercent int    // 0, 12, 21
	VATAmount      Amount
	TotalAmount    Amount // including VAT
	SortOrder      int
}

// InvoiceSequence defines a numbering sequence for invoices.
type InvoiceSequence struct {
	ID            int64
	Prefix        string // e.g. "FV"
	NextNumber    int    // next number to assign
	Year          int    // sequence year
	FormatPattern string // e.g. "{prefix}{year}{number:04d}"
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
