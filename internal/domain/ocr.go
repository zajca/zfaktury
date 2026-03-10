package domain

// OCRResult contains structured data extracted from a document scan.
type OCRResult struct {
	VendorName     string
	VendorICO      string
	VendorDIC      string
	InvoiceNumber  string
	IssueDate      string // YYYY-MM-DD
	DueDate        string // YYYY-MM-DD
	TotalAmount    Amount // in halere
	VATAmount      Amount // in halere
	VATRatePercent int
	CurrencyCode   string
	Description    string
	Items          []OCRItem
	RawText        string
	Confidence     float64 // 0.0-1.0
}

// OCRItem represents a single line item extracted from a document.
type OCRItem struct {
	Description    string
	Quantity       Amount // in cents (e.g. 100 = 1.00)
	UnitPrice      Amount // in halere
	VATRatePercent int
	TotalAmount    Amount // in halere
}
