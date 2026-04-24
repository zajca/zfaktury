package domain

import "time"

// IncomeTaxReturn represents DPFO (danove priznani fyzickych osob).
type IncomeTaxReturn struct {
	ID         int64
	Year       int
	FilingType string // regular, corrective, supplementary

	// Section 7 - business income
	TotalRevenue    Amount // celkove prijmy z podnikani (from invoices)
	ActualExpenses  Amount // skutecne vydaje (from expenses)
	FlatRatePercent int    // 0=actual, 60/80/40/30=pausal
	FlatRateAmount  Amount // computed flat-rate cap applied
	UsedExpenses    Amount // expenses actually used (actual or flat-rate)

	// Tax base
	TaxBase         Amount // prijmy - vydaje
	TotalDeductions Amount // nezdanitelne casti zakladu dane
	TaxBaseRounded  Amount // rounded down to 100 CZK

	// Deduction breakdown per §15 category (sum of AllowedAmount per category for the year).
	// Used for EPO DPFDP5 XML attributes in VetaD.
	DeductionMortgage      Amount // uroky z hypoteky / stavebniho sporeni
	DeductionLifeInsurance Amount // zivotni pojisteni
	DeductionPension       Amount // penzijni sporeni
	DeductionDonation      Amount // dary
	DeductionUnionDues     Amount // odborove prispevky

	// Tax calculation (15% / 23% progressive)
	TaxAt15  Amount
	TaxAt23  Amount
	TotalTax Amount // before credits

	// Credits (slevy na dani)
	CreditBasic      Amount // sleva na poplatnika
	CreditSpouse     Amount // sleva na manzela/ku
	CreditDisability Amount // invalidita
	CreditStudent    Amount // student
	TotalCredits     Amount
	TaxAfterCredits  Amount

	// Child benefit (danove zvyhodneni)
	ChildBenefit    Amount
	TaxAfterBenefit Amount // final tax (can go negative = bonus)

	// Prepayments & result
	Prepayments Amount // zaplacene zalohy
	TaxDue      Amount // doplatek (+) or preplatek (-)

	// §8 capital income
	CapitalIncomeGross Amount
	CapitalIncomeTax   Amount
	CapitalIncomeNet   Amount
	// §10 other income (securities, crypto)
	OtherIncomeGross    Amount
	OtherIncomeExpenses Amount
	OtherIncomeExempt   Amount
	OtherIncomeNet      Amount

	XMLData   []byte
	Status    string // draft, ready, filed
	FiledAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SocialInsuranceOverview represents Prehled OSVC pro CSSZ.
type SocialInsuranceOverview struct {
	ID         int64
	Year       int
	FilingType string

	TotalRevenue        Amount
	TotalExpenses       Amount // used expenses (actual or flat-rate)
	TaxBase             Amount // prijmy - vydaje
	AssessmentBase      Amount // vymericaci zaklad = 50% of TaxBase
	MinAssessmentBase   Amount // minimum for the year
	FinalAssessmentBase Amount // max(Assessment, Min)

	InsuranceRate    int    // 292 = 29.2% (permille*10)
	TotalInsurance   Amount // pojistne
	Prepayments      Amount // zaplacene zalohy za rok
	Difference       Amount // doplatek/preplatek
	NewMonthlyPrepay Amount // nova mesicni zaloha

	XMLData   []byte
	Status    string
	FiledAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// HealthInsuranceOverview represents Prehled OSVC pro ZP.
type HealthInsuranceOverview struct {
	ID         int64
	Year       int
	FilingType string

	TotalRevenue        Amount
	TotalExpenses       Amount
	TaxBase             Amount
	AssessmentBase      Amount // 50% of TaxBase
	MinAssessmentBase   Amount
	FinalAssessmentBase Amount

	InsuranceRate    int // 135 = 13.5% (permille*10)
	TotalInsurance   Amount
	Prepayments      Amount
	Difference       Amount
	NewMonthlyPrepay Amount

	XMLData   []byte
	Status    string
	FiledAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
