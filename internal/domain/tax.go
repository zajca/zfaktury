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
	Year    int `json:"year"`
	Month   int `json:"month"`   // 1-12, 0 if quarterly
	Quarter int `json:"quarter"` // 1-4, 0 if monthly
}

// VATReturn represents a VAT return (Přiznání k DPH).
type VATReturn struct {
	ID         int64     `json:"id"`
	Period     TaxPeriod `json:"period"`
	FilingType string    `json:"filing_type"`

	// Output VAT (daň na výstupu)
	OutputVATBase21   Amount `json:"output_vat_base_21"`
	OutputVATAmount21 Amount `json:"output_vat_amount_21"`
	OutputVATBase12   Amount `json:"output_vat_base_12"`
	OutputVATAmount12 Amount `json:"output_vat_amount_12"`

	// Input VAT (daň na vstupu)
	InputVATBase21   Amount `json:"input_vat_base_21"`
	InputVATAmount21 Amount `json:"input_vat_amount_21"`
	InputVATBase12   Amount `json:"input_vat_base_12"`
	InputVATAmount12 Amount `json:"input_vat_amount_12"`

	// Result
	TotalVATDue    Amount `json:"total_vat_due"`
	TotalVATCredit Amount `json:"total_vat_credit"`
	NetVAT         Amount `json:"net_vat"` // positive = pay, negative = refund

	Status    string     `json:"status"`
	FiledAt   *time.Time `json:"filed_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// VATControlStatement represents a VAT control statement (Kontrolní hlášení).
type VATControlStatement struct {
	ID         int64     `json:"id"`
	Period     TaxPeriod `json:"period"`
	FilingType string    `json:"filing_type"`

	Status    string     `json:"status"`
	FiledAt   *time.Time `json:"filed_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// VIESSummary represents a VIES recapitulative statement (Souhrnné hlášení).
type VIESSummary struct {
	ID         int64     `json:"id"`
	Period     TaxPeriod `json:"period"`
	FilingType string    `json:"filing_type"`

	Status    string     `json:"status"`
	FiledAt   *time.Time `json:"filed_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
