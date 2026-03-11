package domain

// ImportResult holds the result of importing an expense from a document upload.
type ImportResult struct {
	Expense  Expense
	Document ExpenseDocument
	OCR      *OCRResult // nil if OCR not configured or failed
}
