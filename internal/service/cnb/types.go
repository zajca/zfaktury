package cnb

// ExchangeRate represents a single exchange rate entry from the CNB daily rate sheet.
type ExchangeRate struct {
	Country  string
	Currency string
	Amount   int     // how many units the rate applies to
	Code     string  // ISO 4217 currency code (EUR, USD, etc.)
	Rate     float64 // CZK per Amount units
}
