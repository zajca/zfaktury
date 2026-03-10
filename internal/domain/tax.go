package domain

import "time"

// Filing type constants for tax submissions.
const (
	FilingTypeRegular       = "regular"
	FilingTypeCorrective    = "corrective"
	FilingTypeSupplementary = "supplementary"
)

// TaxPeriod identifies a tax reporting period.
type TaxPeriod struct {
	Year    int
	Month   int   // 1-12, 0 if quarterly
	Quarter int // 1-4, 0 if monthly
}

// VATReturn represents a VAT return (Přiznání k DPH).
type VATReturn struct {
	ID         int64
	Period     TaxPeriod
	FilingType string

	// Output VAT (daň na výstupu)
	OutputVATBase21   Amount
	OutputVATAmount21 Amount
	OutputVATBase12   Amount
	OutputVATAmount12 Amount

	// Input VAT (daň na vstupu)
	InputVATBase21   Amount
	InputVATAmount21 Amount
	InputVATBase12   Amount
	InputVATAmount12 Amount

	// Result
	TotalVATDue    Amount
	TotalVATCredit Amount
	NetVAT         Amount // positive = pay, negative = refund

	Status    string
	FiledAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// VATControlStatement represents a VAT control statement (Kontrolní hlášení).
type VATControlStatement struct {
	ID         int64
	Period     TaxPeriod
	FilingType string

	Status    string
	FiledAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// VIESSummary represents a VIES recapitulative statement (Souhrnné hlášení).
type VIESSummary struct {
	ID         int64
	Period     TaxPeriod
	FilingType string

	Status    string
	FiledAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
