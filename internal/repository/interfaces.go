package repository

import (
	"context"

	"github.com/zajca/zfaktury/internal/domain"
)

// ContactRepo defines the persistence interface for contacts.
type ContactRepo interface {
	Create(ctx context.Context, contact *domain.Contact) error
	Update(ctx context.Context, contact *domain.Contact) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.Contact, error)
	List(ctx context.Context, filter domain.ContactFilter) ([]domain.Contact, int, error)
	FindByICO(ctx context.Context, ico string) (*domain.Contact, error)
}

// InvoiceRepo defines the persistence interface for invoices.
type InvoiceRepo interface {
	Create(ctx context.Context, invoice *domain.Invoice) error
	Update(ctx context.Context, invoice *domain.Invoice) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.Invoice, error)
	List(ctx context.Context, filter domain.InvoiceFilter) ([]domain.Invoice, int, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	GetNextNumber(ctx context.Context, sequenceID int64) (string, error)
}

// ExpenseRepo defines the persistence interface for expenses.
type ExpenseRepo interface {
	Create(ctx context.Context, expense *domain.Expense) error
	Update(ctx context.Context, expense *domain.Expense) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.Expense, error)
	List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, int, error)
}

// InvoiceSequenceRepo defines the persistence interface for invoice sequences.
type InvoiceSequenceRepo interface {
	Create(ctx context.Context, seq *domain.InvoiceSequence) error
	Update(ctx context.Context, seq *domain.InvoiceSequence) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.InvoiceSequence, error)
	List(ctx context.Context) ([]domain.InvoiceSequence, error)
	GetByPrefixAndYear(ctx context.Context, prefix string, year int) (*domain.InvoiceSequence, error)
	CountInvoicesBySequenceID(ctx context.Context, sequenceID int64) (int, error)
	MaxUsedNumber(ctx context.Context, sequenceID int64) (int, error)
}

// CategoryRepo defines the persistence interface for expense categories.
type CategoryRepo interface {
	Create(ctx context.Context, cat *domain.ExpenseCategory) error
	Update(ctx context.Context, cat *domain.ExpenseCategory) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.ExpenseCategory, error)
	GetByKey(ctx context.Context, key string) (*domain.ExpenseCategory, error)
	List(ctx context.Context) ([]domain.ExpenseCategory, error)
}

// SettingsRepo defines the persistence interface for application settings.
type SettingsRepo interface {
	GetAll(ctx context.Context) (map[string]string, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	SetBulk(ctx context.Context, settings map[string]string) error
}
