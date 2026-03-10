package domain

import "time"

// ExpenseCategory represents a category for classifying expenses.
type ExpenseCategory struct {
	ID        int64
	Key       string
	LabelCS   string
	LabelEN   string
	Color     string
	SortOrder int
	IsDefault bool
	CreatedAt time.Time
	DeletedAt *time.Time
}
