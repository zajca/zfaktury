package domain

import "time"

// IncomeTaxReturn warning tokens raised during Recalculate. These are
// advisory only — they do not block XML generation or filing, but the UI
// surfaces them so the user can verify against the legal ground truth.
const (
	// WarningProgressiveRateReview signals the consolidated tax base
	// (§6 + §7 + §8 + §10) crossed 36× průměrná mzda for the year, putting
	// the upper portion into the §16 odst. 1 ZDP 23% bracket. The user must
	// verify the split is correct (existing CalculateIncomeTax handles it).
	WarningProgressiveRateReview = "progressive_rate_review"

	// WarningWithholdingPartialInclude is reserved for §38g odst. 6 ZDP
	// enforcement (out of scope for current MVP — left here so future
	// implementations can adopt the same identifier).
	WarningWithholdingPartialInclude = "withholding_partial_include"
)

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

	// §6 employment income aggregates (DPC/DPP/HPP)
	Section6GrossIncome          Amount // ř.31
	Section6IncomeWithoutAdvance Amount // ř.35 (informativní; §38h)
	Section6ForeignTax           Amount // ř.33
	Section6TaxBase              Amount // ř.34/36 = ř.31 - ř.33
	Section6AdvanceWithheld      Amount // ř.84 (po vrácení přeplatku z RZ)
	Section6WithholdingCredited  Amount // ř.87 (jen pokud uživatel zahrnul §36/6 do DAP)
	Section6MonthlyBonusPaid     Amount // ř.89 kc_vyplbonus (vyplacené zaměstnavatelem; NESLEVÍ z ChildBenefit)
	Section6CertsAdvance         int    // count -> potv_zam
	Section6CertsWithholding     int    // count -> potv_36
	Section6CertsBonus           int    // count -> potv_dazvyh

	XMLData []byte

	// Warnings carries non-blocking advisory tokens raised during Recalculate
	// (e.g. progressive 23% rate review for §16 odst. 1 ZDP). Persisted as a
	// comma-separated string in the warnings column. Empty slice = no warnings.
	Warnings []string

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
