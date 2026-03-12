package domain

import "time"

// FakturoidImportItem represents a single entity to be imported from Fakturoid.
type FakturoidImportItem struct {
	FakturoidID int64
	Status      string // "new", "duplicate", "conflict"
	Entity      any    // mapped domain entity (*Contact, *Invoice, *Expense)
	ExistingID  *int64 // set when Status is "duplicate"
	Reason      string // human-readable explanation for status
}

// FakturoidImportPreview contains the preview of what will be imported.
type FakturoidImportPreview struct {
	Contacts []FakturoidImportItem
	Invoices []FakturoidImportItem
	Expenses []FakturoidImportItem
}

// FakturoidImportResult contains the results of an import operation.
type FakturoidImportResult struct {
	ContactsCreated int
	ContactsSkipped int
	InvoicesCreated int
	InvoicesSkipped int
	ExpensesCreated int
	ExpensesSkipped int
	Errors          []string
}

// FakturoidImportLog represents a record of an imported entity.
type FakturoidImportLog struct {
	ID                  int64
	FakturoidEntityType string
	FakturoidID         int64
	LocalEntityType     string
	LocalID             int64
	ImportedAt          time.Time
}
