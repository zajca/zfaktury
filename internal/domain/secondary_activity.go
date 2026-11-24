package domain

import "fmt"

// SecondaryActivityReason is why the self-employment is a secondary activity
// (vedlejší SVČ) in some months (§ 9 odst. 6 zákona č. 155/1995 Sb.).
type SecondaryActivityReason string

const (
	SecondaryReasonNone       SecondaryActivityReason = ""
	SecondaryReasonEmployment SecondaryActivityReason = "employment" // ČSSZ row 14, zam
	SecondaryReasonPension    SecondaryActivityReason = "pension"    // row 15, duchod
	SecondaryReasonChildCare  SecondaryActivityReason = "child_care" // row 16, pdite
	SecondaryReasonMaternity  SecondaryActivityReason = "maternity"  // row 17, ppm
	SecondaryReasonCare       SecondaryActivityReason = "care"       // row 18, pece
	SecondaryReasonStudent    SecondaryActivityReason = "student"    // row 19, ndite
)

// ValidateSecondaryActivity checks that the months of secondary activity are
// months of activity and that a reason is given exactly when some months are set.
func ValidateSecondaryActivity(activity, secondary ActivityMonths, reason SecondaryActivityReason) error {
	if secondary > FullYearActivityMonths {
		return &UserMessageError{Message: "měsíce vedlejší činnosti obsahují měsíc mimo leden až prosinec"}
	}
	if secondary&^activity != 0 {
		return &UserMessageError{Message: "měsíce vedlejší činnosti musí být měsíce činnosti"}
	}
	switch reason {
	case SecondaryReasonNone, SecondaryReasonEmployment, SecondaryReasonPension, SecondaryReasonChildCare,
		SecondaryReasonMaternity, SecondaryReasonCare, SecondaryReasonStudent:
	default:
		return &UserMessageError{Message: fmt.Sprintf("neznámý důvod vedlejší činnosti: %q", reason)}
	}
	if secondary != 0 && reason == SecondaryReasonNone {
		return &UserMessageError{Message: "vyberte důvod vedlejší činnosti"}
	}
	if secondary == 0 && reason != SecondaryReasonNone {
		return &UserMessageError{Message: "důvod vedlejší činnosti vyžaduje alespoň jeden měsíc"}
	}
	return nil
}

// AdvanceAsSecondary reports whether the advances of the payment year are paid
// as for a secondary activity: the last month of activity is a secondary month
// (ČSSZ zal@ved).
func (s *SocialInsuranceOverview) AdvanceAsSecondary() bool {
	months := s.ActivityMonths.Months()
	if len(months) == 0 {
		return false
	}
	return s.SecondaryMonths.Has(months[len(months)-1])
}
