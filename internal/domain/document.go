package domain

import "time"

// ExpenseDocument represents a file attachment linked to an expense.
type ExpenseDocument struct {
	ID          int64
	ExpenseID   int64
	Filename    string
	ContentType string
	StoragePath string
	Size        int64
	CreatedAt   time.Time
	DeletedAt   *time.Time
}
