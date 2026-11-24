package domain

import (
	"errors"
	"testing"
)

func TestValidateSecondaryActivity(t *testing.T) {
	jan, _ := NewActivityMonths([]int{1})
	secondHalf, _ := NewActivityMonths([]int{7, 8, 9, 10, 11, 12})

	tests := []struct {
		name      string
		activity  ActivityMonths
		secondary ActivityMonths
		reason    SecondaryActivityReason
		wantErr   bool
	}{
		{"nothing set", FullYearActivityMonths, 0, SecondaryReasonNone, false},
		{"some months with a reason", FullYearActivityMonths, secondHalf, SecondaryReasonEmployment, false},
		{"all months", FullYearActivityMonths, FullYearActivityMonths, SecondaryReasonStudent, false},
		{"mask above the year", FullYearActivityMonths, ActivityMonths(8192), SecondaryReasonPension, true},
		{"month outside activity", secondHalf, jan, SecondaryReasonCare, true},
		{"unknown reason", FullYearActivityMonths, jan, SecondaryActivityReason("x"), true},
		{"months without a reason", FullYearActivityMonths, jan, SecondaryReasonNone, true},
		{"reason without months", FullYearActivityMonths, 0, SecondaryReasonMaternity, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecondaryActivity(tt.activity, tt.secondary, tt.reason)
			if tt.wantErr && !errors.Is(err, ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("error = %v, want nil", err)
			}
		})
	}
}

func TestAdvanceAsSecondary(t *testing.T) {
	months := func(m ...int) ActivityMonths {
		got, err := NewActivityMonths(m)
		if err != nil {
			t.Fatalf("NewActivityMonths(%v) error: %v", m, err)
		}
		return got
	}
	tests := []struct {
		name      string
		activity  ActivityMonths
		secondary ActivityMonths
		want      bool
	}{
		{"secondary July to December", FullYearActivityMonths, months(7, 8, 9, 10, 11, 12), true},
		{"secondary January to June", FullYearActivityMonths, months(1, 2, 3, 4, 5, 6), false},
		{"no secondary months", FullYearActivityMonths, 0, false},
		{"activity January to June, June secondary", months(1, 2, 3, 4, 5, 6), months(6), true},
		{"no activity months", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sio := &SocialInsuranceOverview{ActivityMonths: tt.activity, SecondaryMonths: tt.secondary}
			if got := sio.AdvanceAsSecondary(); got != tt.want {
				t.Errorf("AdvanceAsSecondary() = %v, want %v", got, tt.want)
			}
		})
	}
}
