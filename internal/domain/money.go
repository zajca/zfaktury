package domain

import (
	"fmt"
	"math"
)

// Amount represents a monetary value in the smallest currency unit (halere/cents).
// For example, 100 CZK is stored as 10000 (100 * 100).
type Amount int64

// NewAmount creates an Amount from whole units and fractional units.
// For CZK: crowns and halere. For EUR/USD: dollars/euros and cents.
func NewAmount(whole int64, fraction int64) Amount {
	return Amount(whole*100 + fraction)
}

// FromFloat converts a float64 value to Amount by rounding to nearest cent/haler.
func FromFloat(f float64) Amount {
	return Amount(math.Round(f * 100))
}

// ToCZK returns the amount as a float64 in whole currency units.
func (a Amount) ToCZK() float64 {
	return float64(a) / 100.0
}

// String returns a human-readable representation of the amount (e.g. "123.45").
func (a Amount) String() string {
	whole := int64(a) / 100
	fraction := int64(a) % 100
	if fraction < 0 {
		fraction = -fraction
	}
	if a < 0 && whole == 0 {
		return fmt.Sprintf("-%d.%02d", whole, fraction)
	}
	return fmt.Sprintf("%d.%02d", whole, fraction)
}

// Add returns the sum of two amounts.
func (a Amount) Add(other Amount) Amount {
	return a + other
}

// Sub returns the difference of two amounts.
func (a Amount) Sub(other Amount) Amount {
	return a - other
}

// Multiply returns the amount multiplied by a factor, rounded to nearest cent.
func (a Amount) Multiply(factor float64) Amount {
	return Amount(math.Round(float64(a) * factor))
}

// IsZero returns true if the amount is zero.
func (a Amount) IsZero() bool {
	return a == 0
}

// IsNegative returns true if the amount is negative.
func (a Amount) IsNegative() bool {
	return a < 0
}

// Currency code constants.
const (
	CurrencyCZK = "CZK"
	CurrencyEUR = "EUR"
	CurrencyUSD = "USD"
)
