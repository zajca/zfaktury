use std::collections::HashMap;

use chrono::NaiveDate;
use zfaktury_domain::{
    Amount, AuditLogEntry, AuditLogFilter, BackupRecord, CapitalIncomeEntry, Contact,
    ContactFilter, DomainError, Expense, ExpenseCategory, ExpenseDocument, ExpenseFilter,
    FakturoidImportLog, HealthInsuranceOverview, IncomeTaxReturn, InvestmentDocument, Invoice,
    InvoiceDocument, InvoiceFilter, InvoiceSequence, InvoiceStatusChange, PaymentReminder,
    RecurringExpense, RecurringInvoice, SecurityTransaction, SocialInsuranceOverview,
    TaxChildCredit, TaxDeduction, TaxDeductionDocument, TaxPersonalCredits, TaxPrepayment,
    TaxSpouseCredit, TaxYearSettings, VATControlStatement, VATControlStatementLine, VATReturn,
    VIESSummary, VIESSummaryLine,
};

use super::types::{
    CategoryAmount, CustomerRevenue, MonthlyAmount, QuarterlyAmount, RecentExpense, RecentInvoice,
};

type Result<T> = std::result::Result<T, DomainError>;

/// Persistence interface for contacts.
pub trait ContactRepo {
    fn create(&self, contact: &mut Contact) -> Result<()>;
    fn update(&self, contact: &mut Contact) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<Contact>;
    fn list(&self, filter: &ContactFilter) -> Result<(Vec<Contact>, i64)>;
    fn find_by_ico(&self, ico: &str) -> Result<Contact>;
}

/// Persistence interface for invoices.
pub trait InvoiceRepo {
    fn create(&self, invoice: &mut Invoice) -> Result<()>;
    fn update(&self, invoice: &mut Invoice) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<Invoice>;
    fn list(&self, filter: &InvoiceFilter) -> Result<(Vec<Invoice>, i64)>;
    fn update_status(&self, id: i64, status: &str) -> Result<()>;
    fn get_next_number(&self, sequence_id: i64) -> Result<String>;
    fn get_related_invoices(&self, invoice_id: i64) -> Result<Vec<Invoice>>;
    fn find_by_related_invoice(
        &self,
        related_id: i64,
        relation_type: &str,
    ) -> Result<Option<Invoice>>;
}

/// Persistence interface for expenses.
pub trait ExpenseRepo {
    fn create(&self, expense: &mut Expense) -> Result<()>;
    fn update(&self, expense: &mut Expense) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<Expense>;
    fn list(&self, filter: &ExpenseFilter) -> Result<(Vec<Expense>, i64)>;
    fn mark_tax_reviewed(&self, ids: &[i64]) -> Result<()>;
    fn unmark_tax_reviewed(&self, ids: &[i64]) -> Result<()>;
}

/// Persistence interface for invoice numbering sequences.
pub trait InvoiceSequenceRepo {
    fn create(&self, seq: &mut InvoiceSequence) -> Result<()>;
    fn update(&self, seq: &mut InvoiceSequence) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<InvoiceSequence>;
    fn list(&self) -> Result<Vec<InvoiceSequence>>;
    fn get_by_prefix_and_year(&self, prefix: &str, year: i32) -> Result<InvoiceSequence>;
    fn count_invoices_by_sequence_id(&self, sequence_id: i64) -> Result<i64>;
    fn max_used_number(&self, sequence_id: i64) -> Result<i64>;
}

/// Persistence interface for expense categories.
pub trait CategoryRepo {
    fn create(&self, cat: &mut ExpenseCategory) -> Result<()>;
    fn update(&self, cat: &mut ExpenseCategory) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<ExpenseCategory>;
    fn get_by_key(&self, key: &str) -> Result<ExpenseCategory>;
    fn list(&self) -> Result<Vec<ExpenseCategory>>;
}

/// Persistence interface for expense documents.
pub trait DocumentRepo {
    fn create(&self, doc: &mut ExpenseDocument) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<ExpenseDocument>;
    fn list_by_expense_id(&self, expense_id: i64) -> Result<Vec<ExpenseDocument>>;
    fn delete(&self, id: i64) -> Result<()>;
    fn count_by_expense_id(&self, expense_id: i64) -> Result<i64>;
}

/// Persistence interface for invoice documents.
pub trait InvoiceDocumentRepo {
    fn create(&self, doc: &mut InvoiceDocument) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<InvoiceDocument>;
    fn list_by_invoice_id(&self, invoice_id: i64) -> Result<Vec<InvoiceDocument>>;
    fn delete(&self, id: i64) -> Result<()>;
    fn count_by_invoice_id(&self, invoice_id: i64) -> Result<i64>;
}

/// Persistence interface for recurring invoices.
pub trait RecurringInvoiceRepo {
    fn create(&self, ri: &mut RecurringInvoice) -> Result<()>;
    fn update(&self, ri: &mut RecurringInvoice) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<RecurringInvoice>;
    fn list(&self) -> Result<Vec<RecurringInvoice>>;
    fn list_due(&self, date: NaiveDate) -> Result<Vec<RecurringInvoice>>;
    fn deactivate(&self, id: i64) -> Result<()>;
}

/// Persistence interface for recurring expenses.
pub trait RecurringExpenseRepo {
    fn create(&self, re: &mut RecurringExpense) -> Result<()>;
    fn update(&self, re: &mut RecurringExpense) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<RecurringExpense>;
    fn list(&self, limit: i32, offset: i32) -> Result<(Vec<RecurringExpense>, i64)>;
    fn list_active(&self) -> Result<Vec<RecurringExpense>>;
    fn list_due(&self, as_of_date: NaiveDate) -> Result<Vec<RecurringExpense>>;
    fn deactivate(&self, id: i64) -> Result<()>;
    fn activate(&self, id: i64) -> Result<()>;
}

/// Persistence interface for invoice status change records.
pub trait StatusHistoryRepo {
    fn create(&self, change: &mut InvoiceStatusChange) -> Result<()>;
    fn list_by_invoice_id(&self, invoice_id: i64) -> Result<Vec<InvoiceStatusChange>>;
}

/// Persistence interface for payment reminders.
pub trait ReminderRepo {
    fn create(&self, reminder: &mut PaymentReminder) -> Result<()>;
    fn list_by_invoice_id(&self, invoice_id: i64) -> Result<Vec<PaymentReminder>>;
    fn count_by_invoice_id(&self, invoice_id: i64) -> Result<i64>;
}

/// Persistence interface for application settings (key-value store).
pub trait SettingsRepo {
    fn get_all(&self) -> Result<HashMap<String, String>>;
    fn get(&self, key: &str) -> Result<String>;
    fn set(&self, key: &str, value: &str) -> Result<()>;
    fn set_bulk(&self, settings: &HashMap<String, String>) -> Result<()>;
}

/// Persistence interface for VAT returns.
pub trait VATReturnRepo {
    fn create(&self, vr: &mut VATReturn) -> Result<()>;
    fn update(&self, vr: &mut VATReturn) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<VATReturn>;
    fn list(&self, year: i32) -> Result<Vec<VATReturn>>;
    fn get_by_period(
        &self,
        year: i32,
        month: i32,
        quarter: i32,
        filing_type: &str,
    ) -> Result<VATReturn>;
    fn link_invoices(&self, vat_return_id: i64, invoice_ids: &[i64]) -> Result<()>;
    fn link_expenses(&self, vat_return_id: i64, expense_ids: &[i64]) -> Result<()>;
    fn get_linked_invoice_ids(&self, vat_return_id: i64) -> Result<Vec<i64>>;
    fn get_linked_expense_ids(&self, vat_return_id: i64) -> Result<Vec<i64>>;
}

/// Persistence interface for VAT control statements.
pub trait VATControlStatementRepo {
    fn create(&self, cs: &mut VATControlStatement) -> Result<()>;
    fn update(&self, cs: &mut VATControlStatement) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<VATControlStatement>;
    fn list(&self, year: i32) -> Result<Vec<VATControlStatement>>;
    fn get_by_period(
        &self,
        year: i32,
        month: i32,
        filing_type: &str,
    ) -> Result<VATControlStatement>;
    fn create_lines(&self, lines: &[VATControlStatementLine]) -> Result<()>;
    fn delete_lines(&self, control_statement_id: i64) -> Result<()>;
    fn get_lines(&self, control_statement_id: i64) -> Result<Vec<VATControlStatementLine>>;
}

/// Persistence interface for VIES summaries.
pub trait VIESSummaryRepo {
    fn create(&self, vs: &mut VIESSummary) -> Result<()>;
    fn update(&self, vs: &mut VIESSummary) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<VIESSummary>;
    fn list(&self, year: i32) -> Result<Vec<VIESSummary>>;
    fn get_by_period(&self, year: i32, quarter: i32, filing_type: &str) -> Result<VIESSummary>;
    fn create_lines(&self, lines: &[VIESSummaryLine]) -> Result<()>;
    fn delete_lines(&self, vies_summary_id: i64) -> Result<()>;
    fn get_lines(&self, vies_summary_id: i64) -> Result<Vec<VIESSummaryLine>>;
}

/// Persistence interface for income tax returns.
pub trait IncomeTaxReturnRepo {
    fn create(&self, itr: &mut IncomeTaxReturn) -> Result<()>;
    fn update(&self, itr: &mut IncomeTaxReturn) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<IncomeTaxReturn>;
    fn list(&self, year: i32) -> Result<Vec<IncomeTaxReturn>>;
    fn get_by_year(&self, year: i32, filing_type: &str) -> Result<IncomeTaxReturn>;
    fn link_invoices(&self, id: i64, invoice_ids: &[i64]) -> Result<()>;
    fn link_expenses(&self, id: i64, expense_ids: &[i64]) -> Result<()>;
    fn get_linked_invoice_ids(&self, id: i64) -> Result<Vec<i64>>;
    fn get_linked_expense_ids(&self, id: i64) -> Result<Vec<i64>>;
}

/// Persistence interface for social insurance overviews.
pub trait SocialInsuranceOverviewRepo {
    fn create(&self, sio: &mut SocialInsuranceOverview) -> Result<()>;
    fn update(&self, sio: &mut SocialInsuranceOverview) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<SocialInsuranceOverview>;
    fn list(&self, year: i32) -> Result<Vec<SocialInsuranceOverview>>;
    fn get_by_year(&self, year: i32, filing_type: &str) -> Result<SocialInsuranceOverview>;
}

/// Persistence interface for health insurance overviews.
pub trait HealthInsuranceOverviewRepo {
    fn create(&self, hio: &mut HealthInsuranceOverview) -> Result<()>;
    fn update(&self, hio: &mut HealthInsuranceOverview) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<HealthInsuranceOverview>;
    fn list(&self, year: i32) -> Result<Vec<HealthInsuranceOverview>>;
    fn get_by_year(&self, year: i32, filing_type: &str) -> Result<HealthInsuranceOverview>;
}

/// Persistence interface for per-year tax settings.
pub trait TaxYearSettingsRepo {
    fn get_by_year(&self, year: i32) -> Result<TaxYearSettings>;
    fn upsert(&self, tys: &mut TaxYearSettings) -> Result<()>;
}

/// Persistence interface for monthly tax prepayments.
pub trait TaxPrepaymentRepo {
    fn list_by_year(&self, year: i32) -> Result<Vec<TaxPrepayment>>;
    fn upsert_all(&self, year: i32, prepayments: &[TaxPrepayment]) -> Result<()>;
    fn sum_by_year(&self, year: i32) -> Result<(Amount, Amount, Amount)>;
}

/// Persistence interface for spouse tax credits.
pub trait TaxSpouseCreditRepo {
    fn upsert(&self, credit: &mut TaxSpouseCredit) -> Result<()>;
    fn get_by_year(&self, year: i32) -> Result<TaxSpouseCredit>;
    fn delete_by_year(&self, year: i32) -> Result<()>;
}

/// Persistence interface for child tax credits.
pub trait TaxChildCreditRepo {
    fn create(&self, credit: &mut TaxChildCredit) -> Result<()>;
    fn update(&self, credit: &mut TaxChildCredit) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn list_by_year(&self, year: i32) -> Result<Vec<TaxChildCredit>>;
}

/// Persistence interface for personal tax credits.
pub trait TaxPersonalCreditsRepo {
    fn upsert(&self, credits: &mut TaxPersonalCredits) -> Result<()>;
    fn get_by_year(&self, year: i32) -> Result<TaxPersonalCredits>;
}

/// Persistence interface for tax deductions.
pub trait TaxDeductionRepo {
    fn create(&self, ded: &mut TaxDeduction) -> Result<()>;
    fn update(&self, ded: &mut TaxDeduction) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<TaxDeduction>;
    fn list_by_year(&self, year: i32) -> Result<Vec<TaxDeduction>>;
}

/// Persistence interface for tax deduction documents.
pub trait TaxDeductionDocumentRepo {
    fn create(&self, doc: &mut TaxDeductionDocument) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<TaxDeductionDocument>;
    fn list_by_deduction_id(&self, deduction_id: i64) -> Result<Vec<TaxDeductionDocument>>;
    fn delete(&self, id: i64) -> Result<()>;
    fn update_extraction(&self, id: i64, amount: Amount, confidence: f64) -> Result<()>;
}

/// Persistence interface for investment documents.
pub trait InvestmentDocumentRepo {
    fn create(&self, doc: &mut InvestmentDocument) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<InvestmentDocument>;
    fn list_by_year(&self, year: i32) -> Result<Vec<InvestmentDocument>>;
    fn delete(&self, id: i64) -> Result<()>;
    fn update_extraction(&self, id: i64, status: &str, extraction_error: &str) -> Result<()>;
}

/// Persistence interface for capital income entries.
pub trait CapitalIncomeRepo {
    fn create(&self, entry: &mut CapitalIncomeEntry) -> Result<()>;
    fn update(&self, entry: &mut CapitalIncomeEntry) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<CapitalIncomeEntry>;
    fn list_by_year(&self, year: i32) -> Result<Vec<CapitalIncomeEntry>>;
    fn list_by_document_id(&self, document_id: i64) -> Result<Vec<CapitalIncomeEntry>>;
    fn sum_by_year(&self, year: i32) -> Result<(Amount, Amount, Amount)>;
    fn delete_by_document_id(&self, document_id: i64) -> Result<()>;
}

/// Persistence interface for security transactions.
pub trait SecurityTransactionRepo {
    fn create(&self, tx: &mut SecurityTransaction) -> Result<()>;
    fn update(&self, tx: &mut SecurityTransaction) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<SecurityTransaction>;
    fn list_by_year(&self, year: i32) -> Result<Vec<SecurityTransaction>>;
    fn list_by_document_id(&self, document_id: i64) -> Result<Vec<SecurityTransaction>>;
    fn list_buys_for_fifo(
        &self,
        asset_name: &str,
        asset_type: &str,
    ) -> Result<Vec<SecurityTransaction>>;
    fn list_sells_by_year(&self, year: i32) -> Result<Vec<SecurityTransaction>>;
    fn update_fifo_results(
        &self,
        id: i64,
        cost_basis: Amount,
        computed_gain: Amount,
        exempt_amount: Amount,
        time_test_exempt: bool,
    ) -> Result<()>;
    fn delete_by_document_id(&self, document_id: i64) -> Result<()>;
}

/// Persistence interface for audit log entries.
pub trait AuditLogRepo {
    fn create(&self, entry: &mut AuditLogEntry) -> Result<()>;
    fn list_by_entity(&self, entity_type: &str, entity_id: i64) -> Result<Vec<AuditLogEntry>>;
    fn list(&self, filter: &AuditLogFilter) -> Result<(Vec<AuditLogEntry>, i64)>;
}

/// Persistence interface for Fakturoid import logs.
pub trait FakturoidImportLogRepo {
    fn create(&self, entry: &mut FakturoidImportLog) -> Result<()>;
    fn find_by_fakturoid_id(
        &self,
        entity_type: &str,
        fakturoid_id: i64,
    ) -> Result<Option<FakturoidImportLog>>;
    fn list_by_entity_type(&self, entity_type: &str) -> Result<Vec<FakturoidImportLog>>;
}

/// Persistence interface for backup history records.
pub trait BackupHistoryRepo {
    fn create(&self, record: &mut BackupRecord) -> Result<()>;
    fn update(&self, record: &mut BackupRecord) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<BackupRecord>;
    fn list(&self) -> Result<Vec<BackupRecord>>;
    fn delete(&self, id: i64) -> Result<()>;
}

/// Persistence interface for dashboard aggregations.
pub trait DashboardRepo {
    fn revenue_current_month(&self, year: i32, month: i32) -> Result<Amount>;
    fn expenses_current_month(&self, year: i32, month: i32) -> Result<Amount>;
    fn unpaid_invoices(&self) -> Result<(i64, Amount)>;
    fn overdue_invoices(&self) -> Result<(i64, Amount)>;
    fn monthly_revenue(&self, year: i32) -> Result<Vec<MonthlyAmount>>;
    fn monthly_expenses(&self, year: i32) -> Result<Vec<MonthlyAmount>>;
    fn recent_invoices(&self, limit: i32) -> Result<Vec<RecentInvoice>>;
    fn recent_expenses(&self, limit: i32) -> Result<Vec<RecentExpense>>;
}

/// Persistence interface for report aggregations.
pub trait ReportRepo {
    fn monthly_revenue(&self, year: i32) -> Result<Vec<MonthlyAmount>>;
    fn quarterly_revenue(&self, year: i32) -> Result<Vec<QuarterlyAmount>>;
    fn yearly_revenue(&self, year: i32) -> Result<Amount>;
    fn monthly_expenses(&self, year: i32) -> Result<Vec<MonthlyAmount>>;
    fn quarterly_expenses(&self, year: i32) -> Result<Vec<QuarterlyAmount>>;
    fn category_expenses(&self, year: i32) -> Result<Vec<CategoryAmount>>;
    fn top_customers(&self, year: i32, limit: i32) -> Result<Vec<CustomerRevenue>>;
    fn profit_loss_monthly(&self, year: i32) -> Result<(Vec<MonthlyAmount>, Vec<MonthlyAmount>)>;
}
