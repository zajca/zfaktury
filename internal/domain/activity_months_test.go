package domain

import (
	"errors"
	"testing"
)

func TestNewActivityMonths(t *testing.T) {
	tests := []struct {
		name       string
		months     []int
		wantMask   ActivityMonths
		wantCount  int
		wantMonths []int
	}{
		{
			name:       "second half of the year",
			months:     []int{7, 8, 9, 10, 11, 12},
			wantMask:   ActivityMonths(4032),
			wantCount:  6,
			wantMonths: []int{7, 8, 9, 10, 11, 12},
		},
		{
			name:       "all twelve months",
			months:     []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			wantMask:   FullYearActivityMonths,
			wantCount:  12,
			wantMonths: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
		{
			name:       "unsorted input with January and February missing",
			months:     []int{12, 3, 4, 5, 6, 7, 8, 9, 10, 11},
			wantMask:   ActivityMonths(4092),
			wantCount:  10,
			wantMonths: []int{3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewActivityMonths(tt.months)
			if err != nil {
				t.Fatalf("NewActivityMonths() error: %v", err)
			}
			if got != tt.wantMask {
				t.Errorf("mask = %d, want %d", got, tt.wantMask)
			}
			if got.Count() != tt.wantCount {
				t.Errorf("Count() = %d, want %d", got.Count(), tt.wantCount)
			}
			if got.IsFullYear() != (tt.wantMask == FullYearActivityMonths) {
				t.Errorf("IsFullYear() = %v, want %v", got.IsFullYear(), tt.wantMask == FullYearActivityMonths)
			}
			months := got.Months()
			if len(months) != len(tt.wantMonths) {
				t.Fatalf("Months() = %v, want %v", months, tt.wantMonths)
			}
			for i := range months {
				if months[i] != tt.wantMonths[i] {
					t.Errorf("Months() = %v, want %v", months, tt.wantMonths)
					break
				}
			}
		})
	}
}

func TestNewActivityMonthsMembership(t *testing.T) {
	mask, err := NewActivityMonths([]int{7, 8, 9, 10, 11, 12})
	if err != nil {
		t.Fatalf("NewActivityMonths() error: %v", err)
	}

	if mask.Has(6) {
		t.Error("Has(6) = true, want false")
	}
	if !mask.Has(7) {
		t.Error("Has(7) = false, want true")
	}
	if mask.Has(0) {
		t.Error("Has(0) = true, want false")
	}
	if mask.Has(13) {
		t.Error("Has(13) = true, want false")
	}
	if mask.IsFullYear() {
		t.Error("IsFullYear() = true, want false")
	}
}

func TestNewActivityMonthsInvalid(t *testing.T) {
	tests := []struct {
		name   string
		months []int
	}{
		{name: "empty slice", months: []int{}},
		{name: "nil slice", months: nil},
		{name: "month zero", months: []int{0}},
		{name: "month thirteen", months: []int{13}},
		{name: "duplicate month", months: []int{5, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewActivityMonths(tt.months)
			if err == nil {
				t.Fatal("NewActivityMonths() error = nil, want ErrInvalidInput")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("NewActivityMonths() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestActivityMonthsValidate(t *testing.T) {
	tests := []struct {
		name    string
		mask    ActivityMonths
		wantErr bool
	}{
		{name: "zero mask", mask: 0, wantErr: true},
		{name: "bit outside the year", mask: 4096, wantErr: true},
		{name: "single month", mask: 1, wantErr: false},
		{name: "full year", mask: FullYearActivityMonths, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mask.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate() error = nil, want ErrInvalidInput")
				}
				if !errors.Is(err, ErrInvalidInput) {
					t.Errorf("Validate() error = %v, want ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}
		})
	}
}
