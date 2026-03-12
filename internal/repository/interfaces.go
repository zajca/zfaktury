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
	FindByRelatedInvoice(ctx context.Context, relatedID int64, relationType string) (*domain.Invoice, error)
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

// IncomeTaxReturnRepo defines the persistence interface for income tax returns.
type IncomeTaxReturnRepo interface {
	Create(ctx context.Context, itr *domain.IncomeTaxReturn) error
	Update(ctx context.Context, itr *domain.IncomeTaxReturn) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.IncomeTaxReturn, error)
	List(ctx context.Context, year int) ([]domain.IncomeTaxReturn, error)
	GetByYear(ctx context.Context, year int, filingType string) (*domain.IncomeTaxReturn, error)
	LinkInvoices(ctx context.Context, id int64, invoiceIDs []int64) error
	LinkExpenses(ctx context.Context, id int64, expenseIDs []int64) error
	GetLinkedInvoiceIDs(ctx context.Context, id int64) ([]int64, error)
	GetLinkedExpenseIDs(ctx context.Context, id int64) ([]int64, error)
}

// SocialInsuranceOverviewRepo defines the persistence interface for social insurance overviews.
type SocialInsuranceOverviewRepo interface {
	Create(ctx context.Context, sio *domain.SocialInsuranceOverview) error
	Update(ctx context.Context, sio *domain.SocialInsuranceOverview) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.SocialInsuranceOverview, error)
	List(ctx context.Context, year int) ([]domain.SocialInsuranceOverview, error)
	GetByYear(ctx context.Context, year int, filingType string) (*domain.SocialInsuranceOverview, error)
}

// TaxYearSettingsRepo defines the persistence interface for per-year tax settings.
type TaxYearSettingsRepo interface {
	GetByYear(ctx context.Context, year int) (*domain.TaxYearSettings, error)
	Upsert(ctx context.Context, tys *domain.TaxYearSettings) error
}

// TaxPrepaymentRepo defines the persistence interface for monthly tax prepayments.
type TaxPrepaymentRepo interface {
	ListByYear(ctx context.Context, year int) ([]domain.TaxPrepayment, error)
	UpsertAll(ctx context.Context, year int, prepayments []domain.TaxPrepayment) error
	SumByYear(ctx context.Context, year int) (taxTotal, socialTotal, healthTotal domain.Amount, err error)
}

// HealthInsuranceOverviewRepo defines the persistence interface for health insurance overviews.
type HealthInsuranceOverviewRepo interface {
	Create(ctx context.Context, hio *domain.HealthInsuranceOverview) error
	Update(ctx context.Context, hio *domain.HealthInsuranceOverview) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.HealthInsuranceOverview, error)
	List(ctx context.Context, year int) ([]domain.HealthInsuranceOverview, error)
	GetByYear(ctx context.Context, year int, filingType string) (*domain.HealthInsuranceOverview, error)
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

// TaxSpouseCreditRepo defines the persistence interface for spouse tax credits.
type TaxSpouseCreditRepo interface {
	Upsert(ctx context.Context, credit *domain.TaxSpouseCredit) error
	GetByYear(ctx context.Context, year int) (*domain.TaxSpouseCredit, error)
	DeleteByYear(ctx context.Context, year int) error
}

// TaxChildCreditRepo defines the persistence interface for child tax credits.
type TaxChildCreditRepo interface {
	Create(ctx context.Context, credit *domain.TaxChildCredit) error
	Update(ctx context.Context, credit *domain.TaxChildCredit) error
	Delete(ctx context.Context, id int64) error
	ListByYear(ctx context.Context, year int) ([]domain.TaxChildCredit, error)
}

// TaxPersonalCreditsRepo defines the persistence interface for personal tax credits.
type TaxPersonalCreditsRepo interface {
	Upsert(ctx context.Context, credits *domain.TaxPersonalCredits) error
	GetByYear(ctx context.Context, year int) (*domain.TaxPersonalCredits, error)
}

// TaxDeductionRepo defines the persistence interface for tax deductions.
type TaxDeductionRepo interface {
	Create(ctx context.Context, ded *domain.TaxDeduction) error
	Update(ctx context.Context, ded *domain.TaxDeduction) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.TaxDeduction, error)
	ListByYear(ctx context.Context, year int) ([]domain.TaxDeduction, error)
}

// InvestmentDocumentRepo defines the persistence interface for investment documents.
type InvestmentDocumentRepo interface {
	Create(ctx context.Context, doc *domain.InvestmentDocument) error
	GetByID(ctx context.Context, id int64) (*domain.InvestmentDocument, error)
	ListByYear(ctx context.Context, year int) ([]domain.InvestmentDocument, error)
	Delete(ctx context.Context, id int64) error
	UpdateExtraction(ctx context.Context, id int64, status string, extractionError string) error
}

// CapitalIncomeRepo defines the persistence interface for capital income entries.
type CapitalIncomeRepo interface {
	Create(ctx context.Context, entry *domain.CapitalIncomeEntry) error
	Update(ctx context.Context, entry *domain.CapitalIncomeEntry) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.CapitalIncomeEntry, error)
	ListByYear(ctx context.Context, year int) ([]domain.CapitalIncomeEntry, error)
	ListByDocumentID(ctx context.Context, documentID int64) ([]domain.CapitalIncomeEntry, error)
	SumByYear(ctx context.Context, year int) (grossTotal, taxTotal, netTotal domain.Amount, err error)
	DeleteByDocumentID(ctx context.Context, documentID int64) error
}

// SecurityTransactionRepo defines the persistence interface for security transactions.
type SecurityTransactionRepo interface {
	Create(ctx context.Context, tx *domain.SecurityTransaction) error
	Update(ctx context.Context, tx *domain.SecurityTransaction) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.SecurityTransaction, error)
	ListByYear(ctx context.Context, year int) ([]domain.SecurityTransaction, error)
	ListByDocumentID(ctx context.Context, documentID int64) ([]domain.SecurityTransaction, error)
	ListBuysForFIFO(ctx context.Context, assetName, assetType string) ([]domain.SecurityTransaction, error)
	ListSellsByYear(ctx context.Context, year int) ([]domain.SecurityTransaction, error)
	UpdateFIFOResults(ctx context.Context, id int64, costBasis, computedGain, exemptAmount domain.Amount, timeTestExempt bool) error
	DeleteByDocumentID(ctx context.Context, documentID int64) error
}

// AuditLogRepo defines the persistence interface for audit log entries.
type AuditLogRepo interface {
	Create(ctx context.Context, entry *domain.AuditLogEntry) error
	ListByEntity(ctx context.Context, entityType string, entityID int64) ([]domain.AuditLogEntry, error)
}

// FakturoidImportLogRepo defines the persistence interface for Fakturoid import logs.
type FakturoidImportLogRepo interface {
	Create(ctx context.Context, entry *domain.FakturoidImportLog) error
	FindByFakturoidID(ctx context.Context, entityType string, fakturoidID int64) (*domain.FakturoidImportLog, error)
	ListByEntityType(ctx context.Context, entityType string) ([]domain.FakturoidImportLog, error)
}

// DashboardRepo defines the persistence interface for dashboard aggregations.
type DashboardRepo interface {
	RevenueCurrentMonth(ctx context.Context, year int, month int) (domain.Amount, error)
	ExpensesCurrentMonth(ctx context.Context, year int, month int) (domain.Amount, error)
	UnpaidInvoices(ctx context.Context) (count int, total domain.Amount, err error)
	OverdueInvoices(ctx context.Context) (count int, total domain.Amount, err error)
	MonthlyRevenue(ctx context.Context, year int) ([]MonthlyAmount, error)
	MonthlyExpenses(ctx context.Context, year int) ([]MonthlyAmount, error)
	RecentInvoices(ctx context.Context, limit int) ([]RecentInvoice, error)
	RecentExpenses(ctx context.Context, limit int) ([]RecentExpense, error)
}

// ReportRepo defines the persistence interface for report aggregations.
type ReportRepo interface {
	MonthlyRevenue(ctx context.Context, year int) ([]MonthlyAmount, error)
	QuarterlyRevenue(ctx context.Context, year int) ([]QuarterlyAmount, error)
	YearlyRevenue(ctx context.Context, year int) (domain.Amount, error)
	MonthlyExpenses(ctx context.Context, year int) ([]MonthlyAmount, error)
	QuarterlyExpenses(ctx context.Context, year int) ([]QuarterlyAmount, error)
	CategoryExpenses(ctx context.Context, year int) ([]CategoryAmount, error)
	TopCustomers(ctx context.Context, year int, limit int) ([]CustomerRevenue, error)
	ProfitLossMonthly(ctx context.Context, year int) (revenue []MonthlyAmount, expenses []MonthlyAmount, err error)
}

// TaxDeductionDocumentRepo defines the persistence interface for tax deduction documents.
type TaxDeductionDocumentRepo interface {
	Create(ctx context.Context, doc *domain.TaxDeductionDocument) error
	GetByID(ctx context.Context, id int64) (*domain.TaxDeductionDocument, error)
	ListByDeductionID(ctx context.Context, deductionID int64) ([]domain.TaxDeductionDocument, error)
	Delete(ctx context.Context, id int64) error
	UpdateExtraction(ctx context.Context, id int64, amount domain.Amount, confidence float64) error
}
