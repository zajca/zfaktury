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
		return fmt.Errorf("měsíce bez minima obsahují měsíc mimo leden až prosinec: %w", ErrInvalidInput)
	}
	if noMinimum&^activity != 0 {
		return fmt.Errorf("měsíce bez minima musí být měsíce činnosti: %w", ErrInvalidInput)
	}
	switch reason {
	case HealthNoMinimumNone, HealthNoMinimumEmployee, HealthNoMinimumStateInsured, HealthNoMinimumZTP, HealthNoMinimumRetirementAge:
	default:
		return fmt.Errorf("neznámý důvod, proč minimum neplatí: %q: %w", reason, ErrInvalidInput)
	}
	if noMinimum != 0 && reason == HealthNoMinimumNone {
		return fmt.Errorf("vyberte důvod, proč minimum neplatí: %w", ErrInvalidInput)
	}
	if noMinimum == 0 && reason != HealthNoMinimumNone {
		return fmt.Errorf("důvod, proč minimum neplatí, vyžaduje alespoň jeden měsíc: %w", ErrInvalidInput)
	}
	return nil
}
