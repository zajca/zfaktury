package domain

import (
	"errors"
	"testing"
)

func TestValidateHealthNoMinimum(t *testing.T) {
	jan, _ := NewActivityMonths([]int{1})
	secondHalf, _ := NewActivityMonths([]int{7, 8, 9, 10, 11, 12})

	tests := []struct {
		name      string
		activity  ActivityMonths
		noMinimum ActivityMonths
		reason    HealthNoMinimumReason
		wantErr   bool
	}{
		{"nothing set", FullYearActivityMonths, 0, HealthNoMinimumNone, false},
		{"some months with a reason", FullYearActivityMonths, jan, HealthNoMinimumEmployee, false},
		{"all months", FullYearActivityMonths, FullYearActivityMonths, HealthNoMinimumRetirementAge, false},
		{"mask above the year", FullYearActivityMonths, ActivityMonths(8192), HealthNoMinimumZTP, true},
		{"month outside activity", secondHalf, jan, HealthNoMinimumZTP, true},
		{"unknown reason", FullYearActivityMonths, jan, HealthNoMinimumReason("x"), true},
		{"months without a reason", FullYearActivityMonths, jan, HealthNoMinimumNone, true},
		{"reason without months", FullYearActivityMonths, 0, HealthNoMinimumStateInsured, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHealthNoMinimum(tt.activity, tt.noMinimum, tt.reason)
			if tt.wantErr && !errors.Is(err, ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("error = %v, want nil", err)
			}
		})
	}
}
