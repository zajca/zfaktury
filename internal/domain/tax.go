package domain

import "time"

// Filing type constants for tax submissions.
const (
	FilingTypeRegular       = "regular"
	FilingTypeCorrective    = "corrective"
	FilingTypeSupplementary = "supplementary"
)

// Filing status constants.
const (
	FilingStatusDraft = "draft"
	FilingStatusReady = "ready"
	FilingStatusFiled = "filed"
)

// Control statement section constants.
const (
	ControlSectionA4 = "A4"
	ControlSectionA5 = "A5"
	ControlSectionB2 = "B2"
	ControlSectionB3 = "B3"
)

// ControlStatementThreshold is the amount threshold (in halere) for individual vs aggregated
// transactions in control statements. 10,000 CZK = 1,000,000 haleru.
const ControlStatementThreshold Amount = 1_000_000

// VIES service code constants.
const (
	VIESServiceCode3 = "3" // services (Article 196 directive)
)

// TaxPeriod identifies a tax reporting period.
type TaxPeriod struct {
	Year    int
	Month   int // 1-12, 0 if quarterly
	Quarter int // 1-4, 0 if monthly
}

// VATReturn represents a VAT return (Priznanf k DPH).
type VATReturn struct {
	ID         int64
	Period     TaxPeriod
	FilingType string

	// Output VAT (dan na vystupu) - standard rates
	OutputVATBase21   Amount
	OutputVATAmount21 Amount
	OutputVATBase12   Amount
	OutputVATAmount12 Amount
	OutputVATBase0    Amount

	// Reverse charge output (preneseni danove povinnosti)
	ReverseChargeBase21   Amount
	ReverseChargeAmount21 Amount
	ReverseChargeBase12   Amount
	ReverseChargeAmount12 Amount

	// Input VAT (dan na vstupu)
	InputVATBase21   Amount
	InputVATAmount21 Amount
	InputVATBase12   Amount
	InputVATAmount12 Amount

	// Result
	TotalOutputVAT Amount // total output VAT due
	TotalInputVAT  Amount // total input VAT credit
	NetVAT         Amount // positive = pay, negative = refund

	// XML blob for re-download
	XMLData []byte

	Status    string
	FiledAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// VATReturnInvoice is a junction record linking a VAT return to an invoice.
type VATReturnInvoice struct {
	VATReturnID int64
	InvoiceID   int64
}

// VATReturnExpense is a junction record linking a VAT return to an expense.
type VATReturnExpense struct {
	VATReturnID int64
	ExpenseID   int64
}

// VATControlStatement represents a VAT control statement (Kontrolni hlaseni).
type VATControlStatement struct {
	ID         int64
	Period     TaxPeriod
	FilingType string

	// XML blob for re-download
	XMLData []byte

	Status    string
	FiledAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// VATControlStatementLine represents a single line in a control statement.
type VATControlStatementLine struct {
	ID                 int64
	ControlStatementID int64
	Section            string // A4, A5, B2, B3
	PartnerDIC         string
	DocumentNumber     string // evidence number (cislo dokladu)
	DPPD               string // datum povinnosti priznat dan (YYYY-MM-DD)
	Base               Amount
	VAT                Amount
	VATRatePercent     int
	InvoiceID          *int64 // source invoice (nullable for expenses)
	ExpenseID          *int64 // source expense (nullable for invoices)
}

// VIESSummary represents a VIES recapitulative statement (Souhrnne hlaseni).
type VIESSummary struct {
	ID         int64
	Period     TaxPeriod
	FilingType string

	// XML blob for re-download
	XMLData []byte

	Status    string
	FiledAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// VIESSummaryLine represents a single line in a VIES summary (one per EU partner).
type VIESSummaryLine struct {
	ID            int64
	VIESSummaryID int64
	PartnerDIC    string
	CountryCode   string // 2-letter ISO code (e.g. "DE", "SK")
	TotalAmount   Amount // base amount in CZK (no VAT for intra-EU)
	ServiceCode   string // "3" for services
}
