package domain

import (
	"fmt"
	"math/bits"
)

// ActivityMonths is the set of calendar months in which a self-employed
// activity was performed for at least part of the month; bit 0 is January.
type ActivityMonths uint16

// FullYearActivityMonths has all twelve months set.
const FullYearActivityMonths ActivityMonths = 1<<12 - 1

// NewActivityMonths builds a mask from month numbers (1-12).
func NewActivityMonths(months []int) (ActivityMonths, error) {
	if len(months) == 0 {
		return 0, fmt.Errorf("at least one activity month is required: %w", ErrInvalidInput)
	}

	var mask ActivityMonths
	for _, month := range months {
		if month < 1 || month > 12 {
			return 0, fmt.Errorf("activity month %d out of range 1-12: %w", month, ErrInvalidInput)
		}
		bit := ActivityMonths(1 << (month - 1))
		if mask&bit != 0 {
			return 0, fmt.Errorf("duplicate activity month %d: %w", month, ErrInvalidInput)
		}
		mask |= bit
	}
	return mask, nil
}

// Validate checks that the mask has at least one month and no bits outside
// January-December.
func (m ActivityMonths) Validate() error {
	if m == 0 {
		return fmt.Errorf("at least one activity month is required: %w", ErrInvalidInput)
	}
	if m > FullYearActivityMonths {
		return fmt.Errorf("activity months mask %d has bits outside January-December: %w", m, ErrInvalidInput)
	}
	return nil
}

// Count returns the number of months with activity.
func (m ActivityMonths) Count() int {
	return bits.OnesCount16(uint16(m))
}

// Has reports whether the given month (1-12) has activity.
func (m ActivityMonths) Has(month int) bool {
	if month < 1 || month > 12 {
		return false
	}
	return m&ActivityMonths(1<<(month-1)) != 0
}

// IsFullYear reports whether all twelve months have activity.
func (m ActivityMonths) IsFullYear() bool {
	return m == FullYearActivityMonths
}

// Months returns the month numbers with activity in ascending order.
func (m ActivityMonths) Months() []int {
	months := make([]int, 0, m.Count())
	for month := 1; month <= 12; month++ {
		if m.Has(month) {
			months = append(months, month)
		}
	}
	return months
}
