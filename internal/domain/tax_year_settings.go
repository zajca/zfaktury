package domain

import "time"

// TaxYearSettings stores per-year tax configuration such as flat rate percent.
type TaxYearSettings struct {
	Year            int
	FlatRatePercent int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TaxPrepayment stores monthly prepayment amounts for a given year.
type TaxPrepayment struct {
	Year         int
	Month        int
	TaxAmount    Amount
	SocialAmount Amount
	HealthAmount Amount
}
