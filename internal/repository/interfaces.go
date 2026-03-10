package repository

import (
	"context"
	"time"

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
	GetRelatedInvoices(ctx context.Context, invoiceID int64) ([]domain.Invoice, error)
}

// ExpenseRepo defines the persistence interface for expenses.
type ExpenseRepo interface {
	Create(ctx context.Context, expense *domain.Expense) error
	Update(ctx context.Context, expense *domain.Expense) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.Expense, error)
	List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, int, error)
	MarkTaxReviewed(ctx context.Context, ids []int64) error
	UnmarkTaxReviewed(ctx context.Context, ids []int64) error
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

// DocumentRepo defines the persistence interface for expense documents.
type DocumentRepo interface {
	Create(ctx context.Context, doc *domain.ExpenseDocument) error
	GetByID(ctx context.Context, id int64) (*domain.ExpenseDocument, error)
	ListByExpenseID(ctx context.Context, expenseID int64) ([]domain.ExpenseDocument, error)
	Delete(ctx context.Context, id int64) error
	CountByExpenseID(ctx context.Context, expenseID int64) (int, error)
}

// RecurringInvoiceRepo defines the persistence interface for recurring invoices.
type RecurringInvoiceRepo interface {
	Create(ctx context.Context, ri *domain.RecurringInvoice) error
	Update(ctx context.Context, ri *domain.RecurringInvoice) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.RecurringInvoice, error)
	List(ctx context.Context) ([]domain.RecurringInvoice, error)
	ListDue(ctx context.Context, date time.Time) ([]domain.RecurringInvoice, error)
	Deactivate(ctx context.Context, id int64) error
}

// RecurringExpenseRepo defines the persistence interface for recurring expenses.
type RecurringExpenseRepo interface {
	Create(ctx context.Context, re *domain.RecurringExpense) error
	Update(ctx context.Context, re *domain.RecurringExpense) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.RecurringExpense, error)
	List(ctx context.Context, limit, offset int) ([]domain.RecurringExpense, int, error)
	ListActive(ctx context.Context) ([]domain.RecurringExpense, error)
	ListDue(ctx context.Context, asOfDate time.Time) ([]domain.RecurringExpense, error)
	Deactivate(ctx context.Context, id int64) error
	Activate(ctx context.Context, id int64) error
}

// StatusHistoryRepo defines the persistence interface for invoice status change records.
type StatusHistoryRepo interface {
	Create(ctx context.Context, change *domain.InvoiceStatusChange) error
	ListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.InvoiceStatusChange, error)
}

// ReminderRepo defines the persistence interface for payment reminders.
type ReminderRepo interface {
	Create(ctx context.Context, reminder *domain.PaymentReminder) error
	ListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.PaymentReminder, error)
	CountByInvoiceID(ctx context.Context, invoiceID int64) (int, error)
}

// SettingsRepo defines the persistence interface for application settings.
type SettingsRepo interface {
	GetAll(ctx context.Context) (map[string]string, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	SetBulk(ctx context.Context, settings map[string]string) error
}

// VATReturnRepo defines the persistence interface for VAT returns.
type VATReturnRepo interface {
	Create(ctx context.Context, vr *domain.VATReturn) error
	Update(ctx context.Context, vr *domain.VATReturn) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.VATReturn, error)
	List(ctx context.Context, year int) ([]domain.VATReturn, error)
	GetByPeriod(ctx context.Context, year, month, quarter int, filingType string) (*domain.VATReturn, error)
	LinkInvoices(ctx context.Context, vatReturnID int64, invoiceIDs []int64) error
	LinkExpenses(ctx context.Context, vatReturnID int64, expenseIDs []int64) error
	GetLinkedInvoiceIDs(ctx context.Context, vatReturnID int64) ([]int64, error)
	GetLinkedExpenseIDs(ctx context.Context, vatReturnID int64) ([]int64, error)
}

// VATControlStatementRepo defines the persistence interface for VAT control statements.
type VATControlStatementRepo interface {
	Create(ctx context.Context, cs *domain.VATControlStatement) error
	Update(ctx context.Context, cs *domain.VATControlStatement) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.VATControlStatement, error)
	List(ctx context.Context, year int) ([]domain.VATControlStatement, error)
	GetByPeriod(ctx context.Context, year, month int, filingType string) (*domain.VATControlStatement, error)
	CreateLines(ctx context.Context, lines []domain.VATControlStatementLine) error
	DeleteLines(ctx context.Context, controlStatementID int64) error
	GetLines(ctx context.Context, controlStatementID int64) ([]domain.VATControlStatementLine, error)
}

// VIESSummaryRepo defines the persistence interface for VIES summaries.
type VIESSummaryRepo interface {
	Create(ctx context.Context, vs *domain.VIESSummary) error
	Update(ctx context.Context, vs *domain.VIESSummary) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.VIESSummary, error)
	List(ctx context.Context, year int) ([]domain.VIESSummary, error)
	GetByPeriod(ctx context.Context, year, quarter int, filingType string) (*domain.VIESSummary, error)
	CreateLines(ctx context.Context, lines []domain.VIESSummaryLine) error
	DeleteLines(ctx context.Context, viesSummaryID int64) error
	GetLines(ctx context.Context, viesSummaryID int64) ([]domain.VIESSummaryLine, error)
}
