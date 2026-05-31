package taxrules

import (
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// PensionContribution is one taxpayer-paid pension contribution for a month.
type PensionContribution struct {
	Year       int
	Month      time.Month
	PaidAmount domain.Amount
}

// PensionDeductionInput supports either an institution-confirmed deductible
// amount or monthly contributions. A single annual paid amount is intentionally
// not supported because mid-year threshold changes make it ambiguous.
type PensionDeductionInput struct {
	CertificateDeductibleAmount *domain.Amount
	MonthlyContributions        []PensionContribution
}

// MonthlyDeductionLine explains one contribution's deductible portion.
type MonthlyDeductionLine struct {
	Year             int
	Month            time.Month
	PaidAmount       domain.Amount
	Threshold        domain.Amount
	DeductibleAmount domain.Amount
}

// PensionDeductionResult is the calculated deductible pension amount plus an
// audit-friendly monthly breakdown.
type PensionDeductionResult struct {
	TotalPaid           domain.Amount
	DeductibleBeforeCap domain.Amount
	AllowedAmount       domain.Amount
	Breakdown           []MonthlyDeductionLine
}

// ComputePensionDeduction calculates the deductible amount for pension savings.
func ComputePensionDeduction(input PensionDeductionInput, rules PensionDeductionRules) (PensionDeductionResult, error) {
	if input.CertificateDeductibleAmount != nil && len(input.MonthlyContributions) > 0 {
		return PensionDeductionResult{}, fmt.Errorf("provide either certificate deductible amount or monthly contributions, not both: %w", domain.ErrInvalidInput)
	}
	if input.CertificateDeductibleAmount != nil {
		if *input.CertificateDeductibleAmount < 0 {
			return PensionDeductionResult{}, fmt.Errorf("certificate deductible amount must not be negative: %w", domain.ErrInvalidInput)
		}
		allowed := capAmount(*input.CertificateDeductibleAmount, rules.SharedSavingsCap)
		return PensionDeductionResult{
			DeductibleBeforeCap: *input.CertificateDeductibleAmount,
			AllowedAmount:       allowed,
		}, nil
	}

	result := PensionDeductionResult{
		Breakdown: make([]MonthlyDeductionLine, 0, len(input.MonthlyContributions)),
	}
	for _, c := range input.MonthlyContributions {
		if c.Month < time.January || c.Month > time.December {
			return PensionDeductionResult{}, fmt.Errorf("invalid pension contribution month %d: %w", c.Month, domain.ErrInvalidInput)
		}
		if c.PaidAmount < 0 {
			return PensionDeductionResult{}, fmt.Errorf("pension contribution %04d-%02d must not be negative: %w", c.Year, c.Month, domain.ErrInvalidInput)
		}

		threshold, err := rules.MonthlyThreshold.ValueAt(NewDate(c.Year, c.Month, 1))
		if err != nil {
			return PensionDeductionResult{}, fmt.Errorf("resolving pension threshold for %04d-%02d: %w", c.Year, c.Month, err)
		}
		deductible := c.PaidAmount - threshold
		if deductible < 0 {
			deductible = 0
		}

		result.TotalPaid += c.PaidAmount
		result.DeductibleBeforeCap += deductible
		result.Breakdown = append(result.Breakdown, MonthlyDeductionLine{
			Year:             c.Year,
			Month:            c.Month,
			PaidAmount:       c.PaidAmount,
			Threshold:        threshold,
			DeductibleAmount: deductible,
		})
	}
	result.AllowedAmount = capAmount(result.DeductibleBeforeCap, rules.SharedSavingsCap)
	return result, nil
}

func capAmount(amount, cap domain.Amount) domain.Amount {
	if cap > 0 && amount > cap {
		return cap
	}
	return amount
}
