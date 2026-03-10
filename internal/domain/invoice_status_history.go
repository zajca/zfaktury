package domain

import "time"

// InvoiceStatusChange represents a recorded change of an invoice's status.
type InvoiceStatusChange struct {
	ID        int64
	InvoiceID int64
	OldStatus string
	NewStatus string
	ChangedAt time.Time
	Note      string
}
