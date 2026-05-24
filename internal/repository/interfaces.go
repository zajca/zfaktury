package repository

import (
	"context"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// ContactRepo defines the persistence interface for contacts.
//
// All methods are scoped to a single company via the companyID parameter;
// rows belonging to other companies are invisible.
type ContactRepo interface {
	Create(ctx context.Context, companyID int64, contact *domain.Contact) error
	Update(ctx context.Context, companyID int64, contact *domain.Contact) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.Contact, error)
	List(ctx context.Context, companyID int64, filter domain.ContactFilter) ([]domain.Contact, int, error)
	FindByICO(ctx context.Context, companyID int64, ico string) (*domain.Contact, error)
}

// InvoiceRepo defines the persistence interface for invoices.
//
// All methods are scoped to a single company via the companyID parameter;
// rows belonging to other companies are invisible.
type InvoiceRepo interface {
	Create(ctx context.Context, companyID int64, invoice *domain.Invoice) error
	Update(ctx context.Context, companyID int64, invoice *domain.Invoice) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.Invoice, error)
	List(ctx context.Context, companyID int64, filter domain.InvoiceFilter) ([]domain.Invoice, int, error)
	UpdateStatus(ctx context.Context, companyID, id int64, status string) error
	GetNextNumber(ctx context.Context, companyID, sequenceID int64) (string, error)
	GetRelatedInvoices(ctx context.Context, companyID, invoiceID int64) ([]domain.Invoice, error)
	FindByRelatedInvoice(ctx context.Context, companyID, relatedID int64, relationType string) (*domain.Invoice, error)
}

// ExpenseRepo defines the persistence interface for expenses.
//
// All methods are scoped to a single company via the companyID parameter;
// rows belonging to other companies are invisible.
type ExpenseRepo interface {
	Create(ctx context.Context, companyID int64, expense *domain.Expense) error
	Update(ctx context.Context, companyID int64, expense *domain.Expense) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.Expense, error)
	List(ctx context.Context, companyID int64, filter domain.ExpenseFilter) ([]domain.Expense, int, error)
	MarkTaxReviewed(ctx context.Context, companyID int64, ids []int64) error
	UnmarkTaxReviewed(ctx context.Context, companyID int64, ids []int64) error
}

// InvoiceSequenceRepo defines the persistence interface for invoice sequences.
//
// All methods are scoped to a single company via the companyID parameter;
// the UNIQUE(company_id, prefix, year) constraint added by migration 025
// means the same prefix+year tuple can be reused across companies.
type InvoiceSequenceRepo interface {
	Create(ctx context.Context, companyID int64, seq *domain.InvoiceSequence) error
	Update(ctx context.Context, companyID int64, seq *domain.InvoiceSequence) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.InvoiceSequence, error)
	List(ctx context.Context, companyID int64) ([]domain.InvoiceSequence, error)
	GetByPrefixAndYear(ctx context.Context, companyID int64, prefix string, year int) (*domain.InvoiceSequence, error)
	CountInvoicesBySequenceID(ctx context.Context, companyID, sequenceID int64) (int, error)
	MaxUsedNumber(ctx context.Context, companyID, sequenceID int64) (int, error)
}

// CategoryRepo defines the persistence interface for expense categories.
//
// All methods are scoped to a single company via the companyID parameter.
type CategoryRepo interface {
	Create(ctx context.Context, companyID int64, cat *domain.ExpenseCategory) error
	Update(ctx context.Context, companyID int64, cat *domain.ExpenseCategory) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.ExpenseCategory, error)
	GetByKey(ctx context.Context, companyID int64, key string) (*domain.ExpenseCategory, error)
	List(ctx context.Context, companyID int64) ([]domain.ExpenseCategory, error)
}

// DocumentRepo defines the persistence interface for expense documents.
//
// All methods are scoped to a single company via the companyID parameter.
type DocumentRepo interface {
	Create(ctx context.Context, companyID int64, doc *domain.ExpenseDocument) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.ExpenseDocument, error)
	ListByExpenseID(ctx context.Context, companyID, expenseID int64) ([]domain.ExpenseDocument, error)
	Delete(ctx context.Context, companyID, id int64) error
	CountByExpenseID(ctx context.Context, companyID, expenseID int64) (int, error)
}

// InvoiceDocumentRepo defines the persistence interface for invoice documents.
//
// All methods are scoped to a single company via the companyID parameter.
type InvoiceDocumentRepo interface {
	Create(ctx context.Context, companyID int64, doc *domain.InvoiceDocument) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.InvoiceDocument, error)
	ListByInvoiceID(ctx context.Context, companyID, invoiceID int64) ([]domain.InvoiceDocument, error)
	Delete(ctx context.Context, companyID, id int64) error
	CountByInvoiceID(ctx context.Context, companyID, invoiceID int64) (int, error)
}

// RecurringInvoiceRepo defines the persistence interface for recurring invoices.
//
// All methods are scoped to a single company via the companyID parameter.
type RecurringInvoiceRepo interface {
	Create(ctx context.Context, companyID int64, ri *domain.RecurringInvoice) error
	Update(ctx context.Context, companyID int64, ri *domain.RecurringInvoice) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.RecurringInvoice, error)
	List(ctx context.Context, companyID int64) ([]domain.RecurringInvoice, error)
	ListDue(ctx context.Context, companyID int64, date time.Time) ([]domain.RecurringInvoice, error)
	Deactivate(ctx context.Context, companyID, id int64) error
}

// RecurringExpenseRepo defines the persistence interface for recurring expenses.
//
// All methods are scoped to a single company via the companyID parameter.
type RecurringExpenseRepo interface {
	Create(ctx context.Context, companyID int64, re *domain.RecurringExpense) error
	Update(ctx context.Context, companyID int64, re *domain.RecurringExpense) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.RecurringExpense, error)
	List(ctx context.Context, companyID int64, limit, offset int) ([]domain.RecurringExpense, int, error)
	ListActive(ctx context.Context, companyID int64) ([]domain.RecurringExpense, error)
	ListDue(ctx context.Context, companyID int64, asOfDate time.Time) ([]domain.RecurringExpense, error)
	Deactivate(ctx context.Context, companyID, id int64) error
	Activate(ctx context.Context, companyID, id int64) error
}

// StatusHistoryRepo defines the persistence interface for invoice status change records.
//
// All methods are scoped to a single company via the companyID parameter.
type StatusHistoryRepo interface {
	Create(ctx context.Context, companyID int64, change *domain.InvoiceStatusChange) error
	ListByInvoiceID(ctx context.Context, companyID, invoiceID int64) ([]domain.InvoiceStatusChange, error)
}

// ReminderRepo defines the persistence interface for payment reminders.
//
// All methods are scoped to a single company via the companyID parameter.
type ReminderRepo interface {
	Create(ctx context.Context, companyID int64, reminder *domain.PaymentReminder) error
	ListByInvoiceID(ctx context.Context, companyID, invoiceID int64) ([]domain.PaymentReminder, error)
	CountByInvoiceID(ctx context.Context, companyID, invoiceID int64) (int, error)
}

// SettingsRepo defines the persistence interface for application settings.
//
// All methods are scoped to a single company via the companyID parameter;
// settings belonging to other companies are invisible.
type SettingsRepo interface {
	GetAll(ctx context.Context, companyID int64) (map[string]string, error)
	Get(ctx context.Context, companyID int64, key string) (string, error)
	Set(ctx context.Context, companyID int64, key, value string) error
	SetBulk(ctx context.Context, companyID int64, settings map[string]string) error
}

// VATReturnRepo defines the persistence interface for VAT returns.
//
// All methods are scoped to a single company via the companyID parameter.
type VATReturnRepo interface {
	Create(ctx context.Context, companyID int64, vr *domain.VATReturn) error
	Update(ctx context.Context, companyID int64, vr *domain.VATReturn) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.VATReturn, error)
	List(ctx context.Context, companyID int64, year int) ([]domain.VATReturn, error)
	GetByPeriod(ctx context.Context, companyID int64, year, month, quarter int, filingType string) (*domain.VATReturn, error)
	LinkInvoices(ctx context.Context, companyID, vatReturnID int64, invoiceIDs []int64) error
	LinkExpenses(ctx context.Context, companyID, vatReturnID int64, expenseIDs []int64) error
	GetLinkedInvoiceIDs(ctx context.Context, companyID, vatReturnID int64) ([]int64, error)
	GetLinkedExpenseIDs(ctx context.Context, companyID, vatReturnID int64) ([]int64, error)
}

// VATControlStatementRepo defines the persistence interface for VAT control statements.
//
// All methods are scoped to a single company via the companyID parameter.
type VATControlStatementRepo interface {
	Create(ctx context.Context, companyID int64, cs *domain.VATControlStatement) error
	Update(ctx context.Context, companyID int64, cs *domain.VATControlStatement) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.VATControlStatement, error)
	List(ctx context.Context, companyID int64, year int) ([]domain.VATControlStatement, error)
	GetByPeriod(ctx context.Context, companyID int64, year, month int, filingType string) (*domain.VATControlStatement, error)
	CreateLines(ctx context.Context, companyID int64, lines []domain.VATControlStatementLine) error
	DeleteLines(ctx context.Context, companyID, controlStatementID int64) error
	GetLines(ctx context.Context, companyID, controlStatementID int64) ([]domain.VATControlStatementLine, error)
}

// IncomeTaxReturnRepo defines the persistence interface for income tax returns.
//
// All methods are scoped to a single company via the companyID parameter.
type IncomeTaxReturnRepo interface {
	Create(ctx context.Context, companyID int64, itr *domain.IncomeTaxReturn) error
	Update(ctx context.Context, companyID int64, itr *domain.IncomeTaxReturn) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.IncomeTaxReturn, error)
	List(ctx context.Context, companyID int64, year int) ([]domain.IncomeTaxReturn, error)
	GetByYear(ctx context.Context, companyID int64, year int, filingType string) (*domain.IncomeTaxReturn, error)
	LinkInvoices(ctx context.Context, companyID, id int64, invoiceIDs []int64) error
	LinkExpenses(ctx context.Context, companyID, id int64, expenseIDs []int64) error
	GetLinkedInvoiceIDs(ctx context.Context, companyID, id int64) ([]int64, error)
	GetLinkedExpenseIDs(ctx context.Context, companyID, id int64) ([]int64, error)
}

// SocialInsuranceOverviewRepo defines the persistence interface for social insurance overviews.
//
// All methods are scoped to a single company via the companyID parameter.
type SocialInsuranceOverviewRepo interface {
	Create(ctx context.Context, companyID int64, sio *domain.SocialInsuranceOverview) error
	Update(ctx context.Context, companyID int64, sio *domain.SocialInsuranceOverview) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.SocialInsuranceOverview, error)
	List(ctx context.Context, companyID int64, year int) ([]domain.SocialInsuranceOverview, error)
	GetByYear(ctx context.Context, companyID int64, year int, filingType string) (*domain.SocialInsuranceOverview, error)
}

// TaxYearSettingsRepo defines the persistence interface for per-year tax settings.
//
// All methods are scoped to a single company via the companyID parameter.
type TaxYearSettingsRepo interface {
	GetByYear(ctx context.Context, companyID int64, year int) (*domain.TaxYearSettings, error)
	Upsert(ctx context.Context, companyID int64, tys *domain.TaxYearSettings) error
}

// TaxPrepaymentRepo defines the persistence interface for monthly tax prepayments.
//
// All methods are scoped to a single company via the companyID parameter.
type TaxPrepaymentRepo interface {
	ListByYear(ctx context.Context, companyID int64, year int) ([]domain.TaxPrepayment, error)
	UpsertAll(ctx context.Context, companyID int64, year int, prepayments []domain.TaxPrepayment) error
	SumByYear(ctx context.Context, companyID int64, year int) (taxTotal, socialTotal, healthTotal domain.Amount, err error)
}

// HealthInsuranceOverviewRepo defines the persistence interface for health insurance overviews.
//
// All methods are scoped to a single company via the companyID parameter.
type HealthInsuranceOverviewRepo interface {
	Create(ctx context.Context, companyID int64, hio *domain.HealthInsuranceOverview) error
	Update(ctx context.Context, companyID int64, hio *domain.HealthInsuranceOverview) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.HealthInsuranceOverview, error)
	List(ctx context.Context, companyID int64, year int) ([]domain.HealthInsuranceOverview, error)
	GetByYear(ctx context.Context, companyID int64, year int, filingType string) (*domain.HealthInsuranceOverview, error)
}

// VIESSummaryRepo defines the persistence interface for VIES summaries.
//
// All methods are scoped to a single company via the companyID parameter.
type VIESSummaryRepo interface {
	Create(ctx context.Context, companyID int64, vs *domain.VIESSummary) error
	Update(ctx context.Context, companyID int64, vs *domain.VIESSummary) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.VIESSummary, error)
	List(ctx context.Context, companyID int64, year int) ([]domain.VIESSummary, error)
	GetByPeriod(ctx context.Context, companyID int64, year, quarter int, filingType string) (*domain.VIESSummary, error)
	CreateLines(ctx context.Context, companyID int64, lines []domain.VIESSummaryLine) error
	DeleteLines(ctx context.Context, companyID, viesSummaryID int64) error
	GetLines(ctx context.Context, companyID, viesSummaryID int64) ([]domain.VIESSummaryLine, error)
}

// TaxSpouseCreditRepo defines the persistence interface for spouse tax credits.
//
// All methods are scoped to a single company via the companyID parameter.
type TaxSpouseCreditRepo interface {
	Upsert(ctx context.Context, companyID int64, credit *domain.TaxSpouseCredit) error
	GetByYear(ctx context.Context, companyID int64, year int) (*domain.TaxSpouseCredit, error)
	DeleteByYear(ctx context.Context, companyID int64, year int) error
}

// TaxChildCreditRepo defines the persistence interface for child tax credits.
//
// All methods are scoped to a single company via the companyID parameter.
type TaxChildCreditRepo interface {
	Create(ctx context.Context, companyID int64, credit *domain.TaxChildCredit) error
	Update(ctx context.Context, companyID int64, credit *domain.TaxChildCredit) error
	Delete(ctx context.Context, companyID, id int64) error
	ListByYear(ctx context.Context, companyID int64, year int) ([]domain.TaxChildCredit, error)
}

// TaxPersonalCreditsRepo defines the persistence interface for personal tax credits.
//
// All methods are scoped to a single company via the companyID parameter.
type TaxPersonalCreditsRepo interface {
	Upsert(ctx context.Context, companyID int64, credits *domain.TaxPersonalCredits) error
	GetByYear(ctx context.Context, companyID int64, year int) (*domain.TaxPersonalCredits, error)
}

// TaxDeductionRepo defines the persistence interface for tax deductions.
//
// All methods are scoped to a single company via the companyID parameter.
type TaxDeductionRepo interface {
	Create(ctx context.Context, companyID int64, ded *domain.TaxDeduction) error
	Update(ctx context.Context, companyID int64, ded *domain.TaxDeduction) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.TaxDeduction, error)
	ListByYear(ctx context.Context, companyID int64, year int) ([]domain.TaxDeduction, error)
}

// InvestmentDocumentRepo defines the persistence interface for investment documents.
//
// All methods are scoped to a single company via the companyID parameter.
type InvestmentDocumentRepo interface {
	Create(ctx context.Context, companyID int64, doc *domain.InvestmentDocument) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.InvestmentDocument, error)
	ListByYear(ctx context.Context, companyID int64, year int) ([]domain.InvestmentDocument, error)
	Delete(ctx context.Context, companyID, id int64) error
	UpdateExtraction(ctx context.Context, companyID, id int64, status string, extractionError string) error
}

// CapitalIncomeRepo defines the persistence interface for capital income entries.
//
// All methods are scoped to a single company via the companyID parameter.
type CapitalIncomeRepo interface {
	Create(ctx context.Context, companyID int64, entry *domain.CapitalIncomeEntry) error
	Update(ctx context.Context, companyID int64, entry *domain.CapitalIncomeEntry) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.CapitalIncomeEntry, error)
	ListByYear(ctx context.Context, companyID int64, year int) ([]domain.CapitalIncomeEntry, error)
	ListByDocumentID(ctx context.Context, companyID, documentID int64) ([]domain.CapitalIncomeEntry, error)
	SumByYear(ctx context.Context, companyID int64, year int) (grossTotal, taxTotal, netTotal domain.Amount, err error)
	DeleteByDocumentID(ctx context.Context, companyID, documentID int64) error
}

// SecurityTransactionRepo defines the persistence interface for security transactions.
//
// All methods are scoped to a single company via the companyID parameter.
type SecurityTransactionRepo interface {
	Create(ctx context.Context, companyID int64, tx *domain.SecurityTransaction) error
	Update(ctx context.Context, companyID int64, tx *domain.SecurityTransaction) error
	Delete(ctx context.Context, companyID, id int64) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.SecurityTransaction, error)
	ListByYear(ctx context.Context, companyID int64, year int) ([]domain.SecurityTransaction, error)
	ListByDocumentID(ctx context.Context, companyID, documentID int64) ([]domain.SecurityTransaction, error)
	ListBuysForFIFO(ctx context.Context, companyID int64, assetName, assetType string) ([]domain.SecurityTransaction, error)
	ListSellsByYear(ctx context.Context, companyID int64, year int) ([]domain.SecurityTransaction, error)
	UpdateFIFOResults(ctx context.Context, companyID, id int64, costBasis, computedGain, exemptAmount domain.Amount, timeTestExempt bool) error
	DeleteByDocumentID(ctx context.Context, companyID, documentID int64) error
}

// AuditLogRepo defines the persistence interface for audit log entries.
type AuditLogRepo interface {
	Create(ctx context.Context, entry *domain.AuditLogEntry) error
	ListByEntity(ctx context.Context, entityType string, entityID int64) ([]domain.AuditLogEntry, error)
	List(ctx context.Context, filter domain.AuditLogFilter) ([]domain.AuditLogEntry, int, error)
}

// FakturoidImportLogRepo defines the persistence interface for Fakturoid import logs.
//
// All methods are scoped to a single company via the companyID parameter.
type FakturoidImportLogRepo interface {
	Create(ctx context.Context, companyID int64, entry *domain.FakturoidImportLog) error
	FindByFakturoidID(ctx context.Context, companyID int64, entityType string, fakturoidID int64) (*domain.FakturoidImportLog, error)
	ListByEntityType(ctx context.Context, companyID int64, entityType string) ([]domain.FakturoidImportLog, error)
}

// BackupHistoryRepo defines the persistence interface for backup history records.
type BackupHistoryRepo interface {
	Create(ctx context.Context, record *domain.BackupRecord) error
	Update(ctx context.Context, record *domain.BackupRecord) error
	GetByID(ctx context.Context, id int64) (*domain.BackupRecord, error)
	List(ctx context.Context) ([]domain.BackupRecord, error)
	Delete(ctx context.Context, id int64) error
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
//
// All methods are scoped to a single company via the companyID parameter.
type TaxDeductionDocumentRepo interface {
	Create(ctx context.Context, companyID int64, doc *domain.TaxDeductionDocument) error
	GetByID(ctx context.Context, companyID, id int64) (*domain.TaxDeductionDocument, error)
	ListByDeductionID(ctx context.Context, companyID, deductionID int64) ([]domain.TaxDeductionDocument, error)
	Delete(ctx context.Context, companyID, id int64) error
	UpdateExtraction(ctx context.Context, companyID, id int64, amount domain.Amount, confidence float64) error
}

// CompanyRepo persists Company aggregates.
//
// All other per-company repositories receive companyID as an explicit
// parameter and filter by it; CompanyRepo itself is global —
// it knows about all companies regardless of which is currently active.
type CompanyRepo interface {
	Create(ctx context.Context, c domain.Company) (int64, error)
	GetByID(ctx context.Context, id int64) (domain.Company, error)
	List(ctx context.Context) ([]domain.Company, error)
	Update(ctx context.Context, c domain.Company) error
	SoftDelete(ctx context.Context, id int64) error
	CountActive(ctx context.Context) (int, error)
}
