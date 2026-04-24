package domain

import "time"

// Deduction category constants (nezdanitelne casti zakladu dane).
const (
	DeductionMortgage      = "mortgage"       // uroky z hypoteky/stavebniho sporeni
	DeductionLifeInsurance = "life_insurance" // zivotni pojisteni
	DeductionPension       = "pension"        // penzijni sporeni
	DeductionDonation      = "donation"       // dary
	DeductionUnionDues     = "union_dues"     // odborove prispevky
)

// TaxDeduction represents a single tax deduction entry for a year.
// Multiple entries per category per year are supported.
type TaxDeduction struct {
	ID            int64
	Year          int
	Category      string // one of Deduction* constants
	Description   string
	ClaimedAmount Amount // user-entered or AI-extracted
	MaxAmount     Amount // statutory cap (computed by service)
	AllowedAmount Amount // min(claimed, remaining_cap) - computed
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TaxDeductionDocument represents a proof document linked to a deduction.
type TaxDeductionDocument struct {
	ID              int64
	TaxDeductionID  int64
	Filename        string
	ContentType     string
	StoragePath     string
	Size            int64
	ExtractedAmount Amount  // AI-extracted value
	Confidence      float64 // 0.0-1.0
	CreatedAt       time.Time
	DeletedAt       *time.Time
}

// TaxExtractionResult holds the AI-extracted data from a tax deduction proof document.
// Contains all fields required to pre-fill a TaxDeduction entry for the tax return
// (DPFO — Priznani k dani z prijmu fyzickych osob).
type TaxExtractionResult struct {
	// Suggested deduction category (one of Deduction* constants or empty/unknown).
	Category string
	// Name of the institution that issued the proof (bank, insurance company,
	// pension provider, donee, union).
	ProviderName string
	// ICO of the provider/donee (if present on the document).
	ProviderICO string
	// Contract number / variable symbol / proof number.
	ContractNumber string
	// Document issue date in YYYY-MM-DD format.
	DocumentDate string
	// Tax year this deduction applies to.
	PeriodYear int
	// Amount in CZK (whole crowns) — eligible deduction amount.
	AmountCZK int
	// Full amount in halere for precision.
	AmountHalere Amount
	// Purpose of the donation (for donations) or similar free-text description.
	Purpose string
	// Short suggested description for the deduction entry.
	DescriptionSuggestion string
	// Confidence score from the model (0.0–1.0).
	Confidence float64
	// Legacy field: the year the parent deduction record belongs to.
	Year int
}
