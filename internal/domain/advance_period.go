package domain

// AdvancePeriod is the monthly advance due from FromMonth (1-12) of the payment
// year until the next period starts.
type AdvancePeriod struct {
	FromMonth     int
	MonthlyBase   Amount
	MonthlyPrepay Amount
}

// Warning tokens of an insurance overview calculation.
const (
	// WarningAdvanceMinimumMissing: the payment year has no rule set, so the
	// advance is not raised to its minimum.
	WarningAdvanceMinimumMissing = "advance_minimum_missing"
	// WarningFilingMethodMissing: the tax year has no filing method; the
	// deadline assumes the return was filed by 1 April.
	WarningFilingMethodMissing = "filing_method_missing"
	// WarningAdvanceDiffersFromXML: the overview was filed earlier than the
	// deadline used for the XML, and the advance due from the start month
	// differs from the one reported in the XML.
	WarningAdvanceDiffersFromXML = "advance_differs_from_xml"
)
