package domain

import "fmt"

// HealthNoMinimumReason is why the health minimum assessment base does not
// apply in some months (§ 3a odst. 3 zákona č. 592/1992 Sb.).
type HealthNoMinimumReason string

const (
	HealthNoMinimumNone          HealthNoMinimumReason = ""
	HealthNoMinimumEmployee      HealthNoMinimumReason = "employee"       // písm. b)
	HealthNoMinimumStateInsured  HealthNoMinimumReason = "state_insured"  // písm. c), § 3c
	HealthNoMinimumZTP           HealthNoMinimumReason = "ztp"            // § 3 odst. 8 písm. a)
	HealthNoMinimumRetirementAge HealthNoMinimumReason = "retirement_age" // § 3 odst. 8 písm. b)
)

// ValidateHealthNoMinimum checks that the months without minimum are months of
// activity and that a reason is given exactly when some months are set.
func ValidateHealthNoMinimum(activity, noMinimum ActivityMonths, reason HealthNoMinimumReason) error {
	if noMinimum > FullYearActivityMonths {
		return &UserMessageError{Message: "měsíce bez minima obsahují měsíc mimo leden až prosinec"}
	}
	if noMinimum&^activity != 0 {
		return &UserMessageError{Message: "měsíce bez minima musí být měsíce činnosti"}
	}
	switch reason {
	case HealthNoMinimumNone, HealthNoMinimumEmployee, HealthNoMinimumStateInsured, HealthNoMinimumZTP, HealthNoMinimumRetirementAge:
	default:
		return &UserMessageError{Message: fmt.Sprintf("neznámý důvod, proč minimum neplatí: %q", reason)}
	}
	if noMinimum != 0 && reason == HealthNoMinimumNone {
		return &UserMessageError{Message: "vyberte důvod, proč minimum neplatí"}
	}
	if noMinimum == 0 && reason != HealthNoMinimumNone {
		return &UserMessageError{Message: "důvod, proč minimum neplatí, vyžaduje alespoň jeden měsíc"}
	}
	return nil
}
