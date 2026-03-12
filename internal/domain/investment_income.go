package domain

import "time"

// Investment document platform constants.
const (
	PlatformPortu      = "portu"
	PlatformZonky      = "zonky"
	PlatformTrading212 = "trading212"
	PlatformRevolut    = "revolut"
	PlatformOther      = "other"
)

// Extraction status constants.
const (
	ExtractionPending   = "pending"
	ExtractionExtracted = "extracted"
	ExtractionFailed    = "failed"
)

// Capital income category constants.
const (
	CapitalCategoryDividendCZ      = "dividend_cz"
	CapitalCategoryDividendForeign = "dividend_foreign"
	CapitalCategoryInterest        = "interest"
	CapitalCategoryCoupon          = "coupon"
	CapitalCategoryFundDist        = "fund_distribution"
	CapitalCategoryOther           = "other"
)

// Asset type constants.
const (
	AssetTypeStock  = "stock"
	AssetTypeETF    = "etf"
	AssetTypeBond   = "bond"
	AssetTypeFund   = "fund"
	AssetTypeCrypto = "crypto"
	AssetTypeOther  = "other"
)

// Transaction type constants.
const (
	TransactionTypeBuy  = "buy"
	TransactionTypeSell = "sell"
)

// InvestmentDocument represents an uploaded broker statement.
type InvestmentDocument struct {
	ID               int64
	Year             int
	Platform         string
	Filename         string
	ContentType      string
	StoragePath      string
	Size             int64
	ExtractionStatus string
	ExtractionError  string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CapitalIncomeEntry represents a single §8 income entry.
type CapitalIncomeEntry struct {
	ID                 int64
	Year               int
	DocumentID         *int64
	Category           string
	Description        string
	IncomeDate         time.Time
	GrossAmount        Amount
	WithheldTaxCZ      Amount
	WithheldTaxForeign Amount
	CountryCode        string
	NeedsDeclaring     bool
	NetAmount          Amount
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SecurityTransaction represents a single buy or sell (§10).
type SecurityTransaction struct {
	ID              int64
	Year            int
	DocumentID      *int64
	AssetType       string
	AssetName       string
	ISIN            string
	TransactionType string
	TransactionDate time.Time
	Quantity        int64 // 1/10000 units (1 share = 10000)
	UnitPrice       Amount
	TotalAmount     Amount
	Fees            Amount
	CurrencyCode    string
	ExchangeRate    int64 // rate * 10000 (for precision)
	CostBasis       Amount
	ComputedGain    Amount
	TimeTestExempt  bool
	ExemptAmount    Amount
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// InvestmentExtractionResult is what AI returns from a broker statement.
type InvestmentExtractionResult struct {
	Platform       string
	CapitalEntries []CapitalIncomeEntry
	Transactions   []SecurityTransaction
	Confidence     float64
	RawResponse    string
}

// InvestmentYearSummary aggregates investment income for a year.
type InvestmentYearSummary struct {
	Year int
	// §8 capital income
	CapitalIncomeGross Amount
	CapitalIncomeTax   Amount
	CapitalIncomeNet   Amount
	// §10 other income
	OtherIncomeGross    Amount
	OtherIncomeExpenses Amount
	OtherIncomeExempt   Amount
	OtherIncomeNet      Amount
}
