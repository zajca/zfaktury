package domain

import "time"

// Deduction category constants (nezdanitelne casti zakladu dane).
const (
	DeductionMortgage      = "mortgage"       // uroky z hypoteky/stavebniho sporeni
	DeductionLifeInsurance = "life_insurance"  // zivotni pojisteni
	DeductionPension       = "pension"         // penzijni sporeni
	DeductionDonation      = "donation"        // dary
	DeductionUnionDues     = "union_dues"      // odborove prispevky
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

// TaxExtractionResult holds the AI-extracted data from a tax proof document.
type TaxExtractionResult struct {
	AmountCZK  int
	Year       int
	Confidence float64
}
