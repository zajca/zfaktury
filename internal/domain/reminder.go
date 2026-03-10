package domain

import "time"

// PaymentReminder represents a payment reminder sent for an overdue invoice.
type PaymentReminder struct {
	ID             int64
	InvoiceID      int64
	ReminderNumber int
	SentAt         time.Time
	SentTo         string
	Subject        string
	BodyPreview    string
	CreatedAt      time.Time
}
