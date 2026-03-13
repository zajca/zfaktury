package domain

import "time"

// InvoiceDocument represents a file attachment linked to an invoice.
type InvoiceDocument struct {
	ID          int64
	InvoiceID   int64
	Filename    string
	ContentType string
	StoragePath string
	Size        int64
	CreatedAt   time.Time
	DeletedAt   *time.Time
}
