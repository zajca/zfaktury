package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/calc"
	"github.com/zajca/zfaktury/internal/domain"
)

// taxConstantsResponse is the JSON response for tax year constants.
// All monetary amounts are in CZK (halere / 100).
type taxConstantsResponse struct {
	Year                      int              `json:"year"`
	BasicCredit               int64            `json:"basic_credit"`
	SpouseCredit              int64            `json:"spouse_credit"`
	SpouseIncomeLimit         int64            `json:"spouse_income_limit"`
	StudentCredit             int64            `json:"student_credit"`
	DisabilityCredit1         int64            `json:"disability_credit_1"`
	DisabilityCredit3         int64            `json:"disability_credit_3"`
	DisabilityZTPP            int64            `json:"disability_ztpp"`
	ChildBenefit1             int64            `json:"child_benefit_1"`
	ChildBenefit2             int64            `json:"child_benefit_2"`
	ChildBenefit3Plus         int64            `json:"child_benefit_3_plus"`
	MaxChildBonus             int64            `json:"max_child_bonus"`
	ProgressiveThreshold      int64            `json:"progressive_threshold"`
	FlatRateCaps              map[string]int64 `json:"flat_rate_caps"`
	DeductionCapMortgage      int64            `json:"deduction_cap_mortgage"`
	DeductionCapPension       int64            `json:"deduction_cap_pension"`
	DeductionCapLifeInsurance int64            `json:"deduction_cap_life_insurance"`
	DeductionCapUnion         int64            `json:"deduction_cap_union"`
	TimeTestYears             int              `json:"time_test_years"`
	SecurityExemptionLimit    int64            `json:"security_exemption_limit"`
}

// toCZK converts halere to CZK.
func toCZK(a domain.Amount) int64 {
	return int64(a) / 100
}

func taxConstantsFromService(year int, c calc.TaxYearConstants) taxConstantsResponse {
	flatRateCaps := make(map[string]int64, len(c.FlatRateCaps))
	for pct, cap := range c.FlatRateCaps {
		flatRateCaps[strconv.Itoa(pct)] = toCZK(cap)
	}

	return taxConstantsResponse{
		Year:                      year,
		BasicCredit:               toCZK(c.BasicCredit),
		SpouseCredit:              toCZK(c.SpouseCredit),
		SpouseIncomeLimit:         toCZK(c.SpouseIncomeLimit),
		StudentCredit:             toCZK(c.StudentCredit),
		DisabilityCredit1:         toCZK(c.DisabilityCredit1),
		DisabilityCredit3:         toCZK(c.DisabilityCredit3),
		DisabilityZTPP:            toCZK(c.DisabilityZTPP),
		ChildBenefit1:             toCZK(c.ChildBenefit1),
		ChildBenefit2:             toCZK(c.ChildBenefit2),
		ChildBenefit3Plus:         toCZK(c.ChildBenefit3Plus),
		MaxChildBonus:             toCZK(c.MaxChildBonus),
		ProgressiveThreshold:      toCZK(c.ProgressiveThreshold),
		FlatRateCaps:              flatRateCaps,
		DeductionCapMortgage:      toCZK(c.DeductionCapMortgage),
		DeductionCapPension:       toCZK(c.DeductionCapPension),
		DeductionCapLifeInsurance: toCZK(c.DeductionCapLifeInsurance),
		DeductionCapUnion:         toCZK(c.DeductionCapUnionDues),
		TimeTestYears:             c.TimeTestYears,
		SecurityExemptionLimit:    toCZK(c.SecurityExemptionLimit),
	}
}

// handleGetTaxConstants returns tax constants for a given year.
func handleGetTaxConstants(w http.ResponseWriter, r *http.Request) {
	yearStr := chi.URLParam(r, "year")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	constants, err := calc.GetTaxConstants(year)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, taxConstantsFromService(year, constants))
}
