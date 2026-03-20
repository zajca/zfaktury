# Phase 2: Persistence Layer

**Status:** Draft
**Timeline:** 2-3 weeks
**Dependencies:** Phase 1 (Domain types in `zfaktury-core`)

## Summary

This phase ports the entire Go persistence layer -- 36 repository interfaces, their SQLite implementations, and 21 migration files -- into Rust. Repository traits are defined in `zfaktury-core` (no runtime dependency on SQLite), and concrete implementations live in a new `zfaktury-db` crate backed by `rusqlite`. A migration bridge handles existing Go databases that used goose, so users upgrading from Go to Rust see zero data loss.

## Objectives

1. Define all 36 repository traits in `zfaktury-core/src/repository/`
2. Implement all repositories with `rusqlite` in `zfaktury-db`
3. Port 21 migration files from goose format to refinery format
4. Build a goose-to-refinery bridge for existing databases
5. Achieve 90%+ test coverage on all implementations via in-memory SQLite

## Crates Created/Modified

| Crate | Purpose |
|-------|---------|
| `zfaktury-core` | Repository trait definitions in `src/repository/` |
| `zfaktury-db` (new) | `rusqlite` implementations, migrations, bridge |
| `rust/migrations/` | SQL migration files (refinery format) |

### zfaktury-db Cargo.toml

```toml
[package]
name = "zfaktury-db"
version = "0.1.0"
edition = "2021"

[dependencies]
zfaktury-core = { path = "../zfaktury-core" }
rusqlite = { version = "0.32", features = ["bundled", "backup"] }
refinery = { version = "0.8", features = ["rusqlite"] }
chrono = { version = "0.4", features = ["serde"] }
thiserror = "2"

[dev-dependencies]
tempfile = "3"
```

The `bundled` feature compiles SQLite from source, eliminating any system dependency -- matching the Go project's CGO-free `modernc.org/sqlite` approach.

## Architecture

```
zfaktury-core/src/repository/
  mod.rs              -- pub mod declarations, re-exports
  traits.rs           -- all 36 trait definitions
  types.rs            -- reporting structs (MonthlyAmount, etc.)

zfaktury-db/src/
  lib.rs              -- pub mod, Database struct, connection pool
  connection.rs       -- connection setup, pragma init
  migrate.rs          -- refinery runner + goose bridge
  error.rs            -- DbError -> DomainError mapping
  repos/
    mod.rs
    contact.rs
    invoice.rs
    expense.rs
    invoice_sequence.rs
    category.rs
    document.rs        -- ExpenseDocument (DocumentRepo)
    invoice_document.rs
    recurring_invoice.rs
    recurring_expense.rs
    status_history.rs
    reminder.rs
    settings.rs
    vat_return.rs
    vat_control.rs
    vies.rs
    income_tax_return.rs
    social_insurance.rs
    health_insurance.rs
    tax_year_settings.rs
    tax_prepayment.rs
    tax_spouse_credit.rs
    tax_child_credit.rs
    tax_personal_credits.rs
    tax_deduction.rs
    tax_deduction_document.rs
    investment_document.rs
    capital_income.rs
    security_transaction.rs
    audit_log.rs
    fakturoid_import.rs
    backup.rs
    dashboard.rs
    report.rs
  scan.rs             -- row scanning helper functions

rust/migrations/
  V1__initial_schema.sql
  V4__expense_categories.sql
  V5__vat_unreliable_timestamp.sql
  V6__expense_documents.sql
  V7__invoice_relations.sql
  V8__tax_review.sql
  V9__recurring_invoices.sql
  V10__recurring_expenses.sql
  V12__invoice_status_history.sql
  V13__payment_reminders.sql
  V14__vat_filings.sql
  V15__annual_tax.sql
  V16__tax_prepayments.sql
  V17__tax_credits_deductions.sql
  V18__spouse_months.sql
  V19__investment_income.sql
  V20__fakturoid_import_log.sql
  V21__audit_log_expand.sql
  V22__invoice_documents.sql
  V23__expense_items.sql
  V24__backup_history.sql
```

Version numbers intentionally match the Go originals (V1, V4, V5, ...) -- gaps are permitted by refinery and preserve the audit trail.

## Detailed Design

### 1. Error Types

```rust
// zfaktury-db/src/error.rs

use zfaktury_core::domain::DomainError;

#[derive(Debug, thiserror::Error)]
pub enum DbError {
    #[error("entity not found")]
    NotFound,

    #[error("database error: {0}")]
    Rusqlite(#[from] rusqlite::Error),

    #[error("migration error: {0}")]
    Migration(String),

    #[error("date parse error: {0}")]
    DateParse(String),

    #[error("bridge error: {0}")]
    Bridge(String),
}

impl From<DbError> for DomainError {
    fn from(err: DbError) -> Self {
        match err {
            DbError::NotFound => DomainError::NotFound,
            other => DomainError::Internal(other.to_string()),
        }
    }
}
```

All repository implementations return `Result<T, DomainError>` to match the trait signatures. The `DbError` is an internal type converted at the boundary.

### 2. Connection Setup

```rust
// zfaktury-db/src/connection.rs

use rusqlite::Connection;

pub fn open_connection(path: &str) -> Result<Connection, DbError> {
    let conn = Connection::open(path)?;
    conn.execute_batch(
        "PRAGMA journal_mode=WAL;
         PRAGMA foreign_keys=ON;
         PRAGMA busy_timeout=5000;"
    )?;
    Ok(conn)
}

pub fn open_in_memory() -> Result<Connection, DbError> {
    let conn = Connection::open_in_memory()?;
    conn.execute_batch(
        "PRAGMA foreign_keys=ON;"
    )?;
    Ok(conn)
}
```

In-memory connections skip WAL mode (not applicable) and busy_timeout (single connection). Both are used in tests.

### 3. Repository Trait Definitions (zfaktury-core)

All 36 traits are defined in `zfaktury-core/src/repository/traits.rs`. They depend only on domain types and `std` -- no `rusqlite`, no `async`, no framework types.

The Go interfaces use `context.Context` as first parameter -- in Rust this is omitted since we are synchronous and cancellation is handled differently (the caller can drop the future/handle).

Return type conventions:
- Go `(*T, error)` where nil means not found -> Rust `Result<T, DomainError>` with `DomainError::NotFound`
- Go `([]T, int, error)` for paginated lists -> Rust `Result<(Vec<T>, i64), DomainError>` where i64 is total count
- Go `(*T, error)` for optional lookups (FindBy*) -> Rust `Result<Option<T>, DomainError>`
- Go `error` for void operations -> Rust `Result<(), DomainError>`

#### 3.1 Entity Repositories

```rust
// ContactRepo -- 6 methods
pub trait ContactRepo {
    fn create(&self, contact: &Contact) -> Result<Contact, DomainError>;
    fn update(&self, contact: &Contact) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<Contact, DomainError>;
    fn list(&self, filter: &ContactFilter) -> Result<(Vec<Contact>, i64), DomainError>;
    fn find_by_ico(&self, ico: &str) -> Result<Option<Contact>, DomainError>;
}
```

Note: `create` returns the entity with the auto-generated ID populated. The Go version mutates the pointer in place; the Rust version returns a new value.

```rust
// InvoiceRepo -- 9 methods
pub trait InvoiceRepo {
    fn create(&self, invoice: &Invoice) -> Result<Invoice, DomainError>;
    fn update(&self, invoice: &Invoice) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<Invoice, DomainError>;
    fn list(&self, filter: &InvoiceFilter) -> Result<(Vec<Invoice>, i64), DomainError>;
    fn update_status(&self, id: i64, status: &str) -> Result<(), DomainError>;
    fn get_next_number(&self, sequence_id: i64) -> Result<String, DomainError>;
    fn get_related_invoices(&self, invoice_id: i64) -> Result<Vec<Invoice>, DomainError>;
    fn find_by_related_invoice(
        &self,
        related_id: i64,
        relation_type: &str,
    ) -> Result<Option<Invoice>, DomainError>;
}
```

```rust
// ExpenseRepo -- 7 methods
pub trait ExpenseRepo {
    fn create(&self, expense: &Expense) -> Result<Expense, DomainError>;
    fn update(&self, expense: &Expense) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<Expense, DomainError>;
    fn list(&self, filter: &ExpenseFilter) -> Result<(Vec<Expense>, i64), DomainError>;
    fn mark_tax_reviewed(&self, ids: &[i64]) -> Result<(), DomainError>;
    fn unmark_tax_reviewed(&self, ids: &[i64]) -> Result<(), DomainError>;
}
```

```rust
// InvoiceSequenceRepo -- 8 methods
pub trait InvoiceSequenceRepo {
    fn create(&self, seq: &InvoiceSequence) -> Result<InvoiceSequence, DomainError>;
    fn update(&self, seq: &InvoiceSequence) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<InvoiceSequence, DomainError>;
    fn list(&self) -> Result<Vec<InvoiceSequence>, DomainError>;
    fn get_by_prefix_and_year(
        &self,
        prefix: &str,
        year: i32,
    ) -> Result<Option<InvoiceSequence>, DomainError>;
    fn count_invoices_by_sequence_id(&self, sequence_id: i64) -> Result<i32, DomainError>;
    fn max_used_number(&self, sequence_id: i64) -> Result<i32, DomainError>;
}
```

```rust
// CategoryRepo -- 6 methods
pub trait CategoryRepo {
    fn create(&self, cat: &ExpenseCategory) -> Result<ExpenseCategory, DomainError>;
    fn update(&self, cat: &ExpenseCategory) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<ExpenseCategory, DomainError>;
    fn get_by_key(&self, key: &str) -> Result<Option<ExpenseCategory>, DomainError>;
    fn list(&self) -> Result<Vec<ExpenseCategory>, DomainError>;
}
```

```rust
// DocumentRepo (expense documents) -- 5 methods
pub trait DocumentRepo {
    fn create(&self, doc: &ExpenseDocument) -> Result<ExpenseDocument, DomainError>;
    fn get_by_id(&self, id: i64) -> Result<ExpenseDocument, DomainError>;
    fn list_by_expense_id(&self, expense_id: i64) -> Result<Vec<ExpenseDocument>, DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn count_by_expense_id(&self, expense_id: i64) -> Result<i32, DomainError>;
}
```

```rust
// InvoiceDocumentRepo -- 5 methods
pub trait InvoiceDocumentRepo {
    fn create(&self, doc: &InvoiceDocument) -> Result<InvoiceDocument, DomainError>;
    fn get_by_id(&self, id: i64) -> Result<InvoiceDocument, DomainError>;
    fn list_by_invoice_id(&self, invoice_id: i64) -> Result<Vec<InvoiceDocument>, DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn count_by_invoice_id(&self, invoice_id: i64) -> Result<i32, DomainError>;
}
```

```rust
// RecurringInvoiceRepo -- 7 methods
pub trait RecurringInvoiceRepo {
    fn create(&self, ri: &RecurringInvoice) -> Result<RecurringInvoice, DomainError>;
    fn update(&self, ri: &RecurringInvoice) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<RecurringInvoice, DomainError>;
    fn list(&self) -> Result<Vec<RecurringInvoice>, DomainError>;
    fn list_due(&self, date: NaiveDate) -> Result<Vec<RecurringInvoice>, DomainError>;
    fn deactivate(&self, id: i64) -> Result<(), DomainError>;
}
```

```rust
// RecurringExpenseRepo -- 9 methods
pub trait RecurringExpenseRepo {
    fn create(&self, re: &RecurringExpense) -> Result<RecurringExpense, DomainError>;
    fn update(&self, re: &RecurringExpense) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<RecurringExpense, DomainError>;
    fn list(&self, limit: i64, offset: i64) -> Result<(Vec<RecurringExpense>, i64), DomainError>;
    fn list_active(&self) -> Result<Vec<RecurringExpense>, DomainError>;
    fn list_due(&self, as_of_date: NaiveDate) -> Result<Vec<RecurringExpense>, DomainError>;
    fn deactivate(&self, id: i64) -> Result<(), DomainError>;
    fn activate(&self, id: i64) -> Result<(), DomainError>;
}
```

```rust
// StatusHistoryRepo -- 2 methods
pub trait StatusHistoryRepo {
    fn create(&self, change: &InvoiceStatusChange) -> Result<(), DomainError>;
    fn list_by_invoice_id(
        &self,
        invoice_id: i64,
    ) -> Result<Vec<InvoiceStatusChange>, DomainError>;
}
```

```rust
// ReminderRepo -- 3 methods
pub trait ReminderRepo {
    fn create(&self, reminder: &PaymentReminder) -> Result<(), DomainError>;
    fn list_by_invoice_id(
        &self,
        invoice_id: i64,
    ) -> Result<Vec<PaymentReminder>, DomainError>;
    fn count_by_invoice_id(&self, invoice_id: i64) -> Result<i32, DomainError>;
}
```

```rust
// SettingsRepo -- 4 methods
pub trait SettingsRepo {
    fn get_all(&self) -> Result<HashMap<String, String>, DomainError>;
    fn get(&self, key: &str) -> Result<Option<String>, DomainError>;
    fn set(&self, key: &str, value: &str) -> Result<(), DomainError>;
    fn set_bulk(&self, settings: &HashMap<String, String>) -> Result<(), DomainError>;
}
```

#### 3.2 Tax/VAT Repositories

```rust
// VATReturnRepo -- 10 methods
pub trait VATReturnRepo {
    fn create(&self, vr: &VATReturn) -> Result<VATReturn, DomainError>;
    fn update(&self, vr: &VATReturn) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<VATReturn, DomainError>;
    fn list(&self, year: i32) -> Result<Vec<VATReturn>, DomainError>;
    fn get_by_period(
        &self,
        year: i32,
        month: i32,
        quarter: i32,
        filing_type: &str,
    ) -> Result<Option<VATReturn>, DomainError>;
    fn link_invoices(&self, vat_return_id: i64, invoice_ids: &[i64]) -> Result<(), DomainError>;
    fn link_expenses(&self, vat_return_id: i64, expense_ids: &[i64]) -> Result<(), DomainError>;
    fn get_linked_invoice_ids(&self, vat_return_id: i64) -> Result<Vec<i64>, DomainError>;
    fn get_linked_expense_ids(&self, vat_return_id: i64) -> Result<Vec<i64>, DomainError>;
}
```

```rust
// VATControlStatementRepo -- 9 methods
pub trait VATControlStatementRepo {
    fn create(&self, cs: &VATControlStatement) -> Result<VATControlStatement, DomainError>;
    fn update(&self, cs: &VATControlStatement) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<VATControlStatement, DomainError>;
    fn list(&self, year: i32) -> Result<Vec<VATControlStatement>, DomainError>;
    fn get_by_period(
        &self,
        year: i32,
        month: i32,
        filing_type: &str,
    ) -> Result<Option<VATControlStatement>, DomainError>;
    fn create_lines(&self, lines: &[VATControlStatementLine]) -> Result<(), DomainError>;
    fn delete_lines(&self, control_statement_id: i64) -> Result<(), DomainError>;
    fn get_lines(
        &self,
        control_statement_id: i64,
    ) -> Result<Vec<VATControlStatementLine>, DomainError>;
}
```

```rust
// VIESSummaryRepo -- 9 methods
pub trait VIESSummaryRepo {
    fn create(&self, vs: &VIESSummary) -> Result<VIESSummary, DomainError>;
    fn update(&self, vs: &VIESSummary) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<VIESSummary, DomainError>;
    fn list(&self, year: i32) -> Result<Vec<VIESSummary>, DomainError>;
    fn get_by_period(
        &self,
        year: i32,
        quarter: i32,
        filing_type: &str,
    ) -> Result<Option<VIESSummary>, DomainError>;
    fn create_lines(&self, lines: &[VIESSummaryLine]) -> Result<(), DomainError>;
    fn delete_lines(&self, vies_summary_id: i64) -> Result<(), DomainError>;
    fn get_lines(&self, vies_summary_id: i64) -> Result<Vec<VIESSummaryLine>, DomainError>;
}
```

```rust
// IncomeTaxReturnRepo -- 10 methods
pub trait IncomeTaxReturnRepo {
    fn create(&self, itr: &IncomeTaxReturn) -> Result<IncomeTaxReturn, DomainError>;
    fn update(&self, itr: &IncomeTaxReturn) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<IncomeTaxReturn, DomainError>;
    fn list(&self, year: i32) -> Result<Vec<IncomeTaxReturn>, DomainError>;
    fn get_by_year(
        &self,
        year: i32,
        filing_type: &str,
    ) -> Result<Option<IncomeTaxReturn>, DomainError>;
    fn link_invoices(&self, id: i64, invoice_ids: &[i64]) -> Result<(), DomainError>;
    fn link_expenses(&self, id: i64, expense_ids: &[i64]) -> Result<(), DomainError>;
    fn get_linked_invoice_ids(&self, id: i64) -> Result<Vec<i64>, DomainError>;
    fn get_linked_expense_ids(&self, id: i64) -> Result<Vec<i64>, DomainError>;
}
```

```rust
// SocialInsuranceOverviewRepo -- 6 methods
pub trait SocialInsuranceOverviewRepo {
    fn create(&self, sio: &SocialInsuranceOverview) -> Result<SocialInsuranceOverview, DomainError>;
    fn update(&self, sio: &SocialInsuranceOverview) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<SocialInsuranceOverview, DomainError>;
    fn list(&self, year: i32) -> Result<Vec<SocialInsuranceOverview>, DomainError>;
    fn get_by_year(
        &self,
        year: i32,
        filing_type: &str,
    ) -> Result<Option<SocialInsuranceOverview>, DomainError>;
}
```

```rust
// HealthInsuranceOverviewRepo -- 6 methods (same shape as SocialInsuranceOverviewRepo)
pub trait HealthInsuranceOverviewRepo {
    fn create(&self, hio: &HealthInsuranceOverview) -> Result<HealthInsuranceOverview, DomainError>;
    fn update(&self, hio: &HealthInsuranceOverview) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<HealthInsuranceOverview, DomainError>;
    fn list(&self, year: i32) -> Result<Vec<HealthInsuranceOverview>, DomainError>;
    fn get_by_year(
        &self,
        year: i32,
        filing_type: &str,
    ) -> Result<Option<HealthInsuranceOverview>, DomainError>;
}
```

```rust
// TaxYearSettingsRepo -- 2 methods
pub trait TaxYearSettingsRepo {
    fn get_by_year(&self, year: i32) -> Result<Option<TaxYearSettings>, DomainError>;
    fn upsert(&self, tys: &TaxYearSettings) -> Result<(), DomainError>;
}
```

```rust
// TaxPrepaymentRepo -- 3 methods
pub trait TaxPrepaymentRepo {
    fn list_by_year(&self, year: i32) -> Result<Vec<TaxPrepayment>, DomainError>;
    fn upsert_all(&self, year: i32, prepayments: &[TaxPrepayment]) -> Result<(), DomainError>;
    /// Returns (tax_total, social_total, health_total).
    fn sum_by_year(&self, year: i32) -> Result<(Amount, Amount, Amount), DomainError>;
}
```

```rust
// TaxSpouseCreditRepo -- 3 methods
pub trait TaxSpouseCreditRepo {
    fn upsert(&self, credit: &TaxSpouseCredit) -> Result<(), DomainError>;
    fn get_by_year(&self, year: i32) -> Result<Option<TaxSpouseCredit>, DomainError>;
    fn delete_by_year(&self, year: i32) -> Result<(), DomainError>;
}
```

```rust
// TaxChildCreditRepo -- 4 methods
pub trait TaxChildCreditRepo {
    fn create(&self, credit: &TaxChildCredit) -> Result<TaxChildCredit, DomainError>;
    fn update(&self, credit: &TaxChildCredit) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn list_by_year(&self, year: i32) -> Result<Vec<TaxChildCredit>, DomainError>;
}
```

```rust
// TaxPersonalCreditsRepo -- 2 methods
pub trait TaxPersonalCreditsRepo {
    fn upsert(&self, credits: &TaxPersonalCredits) -> Result<(), DomainError>;
    fn get_by_year(&self, year: i32) -> Result<Option<TaxPersonalCredits>, DomainError>;
}
```

```rust
// TaxDeductionRepo -- 5 methods
pub trait TaxDeductionRepo {
    fn create(&self, ded: &TaxDeduction) -> Result<TaxDeduction, DomainError>;
    fn update(&self, ded: &TaxDeduction) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<TaxDeduction, DomainError>;
    fn list_by_year(&self, year: i32) -> Result<Vec<TaxDeduction>, DomainError>;
}
```

```rust
// TaxDeductionDocumentRepo -- 5 methods
pub trait TaxDeductionDocumentRepo {
    fn create(&self, doc: &TaxDeductionDocument) -> Result<TaxDeductionDocument, DomainError>;
    fn get_by_id(&self, id: i64) -> Result<TaxDeductionDocument, DomainError>;
    fn list_by_deduction_id(
        &self,
        deduction_id: i64,
    ) -> Result<Vec<TaxDeductionDocument>, DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn update_extraction(
        &self,
        id: i64,
        amount: Amount,
        confidence: f64,
    ) -> Result<(), DomainError>;
}
```

#### 3.3 Investment Repositories

```rust
// InvestmentDocumentRepo -- 5 methods
pub trait InvestmentDocumentRepo {
    fn create(&self, doc: &InvestmentDocument) -> Result<InvestmentDocument, DomainError>;
    fn get_by_id(&self, id: i64) -> Result<InvestmentDocument, DomainError>;
    fn list_by_year(&self, year: i32) -> Result<Vec<InvestmentDocument>, DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn update_extraction(
        &self,
        id: i64,
        status: &str,
        extraction_error: &str,
    ) -> Result<(), DomainError>;
}
```

```rust
// CapitalIncomeRepo -- 8 methods
pub trait CapitalIncomeRepo {
    fn create(&self, entry: &CapitalIncomeEntry) -> Result<CapitalIncomeEntry, DomainError>;
    fn update(&self, entry: &CapitalIncomeEntry) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<CapitalIncomeEntry, DomainError>;
    fn list_by_year(&self, year: i32) -> Result<Vec<CapitalIncomeEntry>, DomainError>;
    fn list_by_document_id(
        &self,
        document_id: i64,
    ) -> Result<Vec<CapitalIncomeEntry>, DomainError>;
    /// Returns (gross_total, tax_total, net_total).
    fn sum_by_year(&self, year: i32) -> Result<(Amount, Amount, Amount), DomainError>;
    fn delete_by_document_id(&self, document_id: i64) -> Result<(), DomainError>;
}
```

```rust
// SecurityTransactionRepo -- 11 methods
pub trait SecurityTransactionRepo {
    fn create(&self, tx: &SecurityTransaction) -> Result<SecurityTransaction, DomainError>;
    fn update(&self, tx: &SecurityTransaction) -> Result<(), DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<SecurityTransaction, DomainError>;
    fn list_by_year(&self, year: i32) -> Result<Vec<SecurityTransaction>, DomainError>;
    fn list_by_document_id(
        &self,
        document_id: i64,
    ) -> Result<Vec<SecurityTransaction>, DomainError>;
    fn list_buys_for_fifo(
        &self,
        asset_name: &str,
        asset_type: &str,
    ) -> Result<Vec<SecurityTransaction>, DomainError>;
    fn list_sells_by_year(&self, year: i32) -> Result<Vec<SecurityTransaction>, DomainError>;
    fn update_fifo_results(
        &self,
        id: i64,
        cost_basis: Amount,
        computed_gain: Amount,
        exempt_amount: Amount,
        time_test_exempt: bool,
    ) -> Result<(), DomainError>;
    fn delete_by_document_id(&self, document_id: i64) -> Result<(), DomainError>;
}
```

#### 3.4 Reporting/Audit Repositories

```rust
// AuditLogRepo -- 3 methods
pub trait AuditLogRepo {
    fn create(&self, entry: &AuditLogEntry) -> Result<(), DomainError>;
    fn list_by_entity(
        &self,
        entity_type: &str,
        entity_id: i64,
    ) -> Result<Vec<AuditLogEntry>, DomainError>;
    fn list(
        &self,
        filter: &AuditLogFilter,
    ) -> Result<(Vec<AuditLogEntry>, i64), DomainError>;
}
```

```rust
// FakturoidImportLogRepo -- 3 methods
pub trait FakturoidImportLogRepo {
    fn create(&self, entry: &FakturoidImportLog) -> Result<(), DomainError>;
    fn find_by_fakturoid_id(
        &self,
        entity_type: &str,
        fakturoid_id: i64,
    ) -> Result<Option<FakturoidImportLog>, DomainError>;
    fn list_by_entity_type(
        &self,
        entity_type: &str,
    ) -> Result<Vec<FakturoidImportLog>, DomainError>;
}
```

```rust
// BackupHistoryRepo -- 5 methods
pub trait BackupHistoryRepo {
    fn create(&self, record: &BackupRecord) -> Result<BackupRecord, DomainError>;
    fn update(&self, record: &BackupRecord) -> Result<(), DomainError>;
    fn get_by_id(&self, id: i64) -> Result<BackupRecord, DomainError>;
    fn list(&self) -> Result<Vec<BackupRecord>, DomainError>;
    fn delete(&self, id: i64) -> Result<(), DomainError>;
}
```

```rust
// DashboardRepo -- 8 methods
pub trait DashboardRepo {
    fn revenue_current_month(&self, year: i32, month: i32) -> Result<Amount, DomainError>;
    fn expenses_current_month(&self, year: i32, month: i32) -> Result<Amount, DomainError>;
    /// Returns (count, total_amount).
    fn unpaid_invoices(&self) -> Result<(i32, Amount), DomainError>;
    /// Returns (count, total_amount).
    fn overdue_invoices(&self) -> Result<(i32, Amount), DomainError>;
    fn monthly_revenue(&self, year: i32) -> Result<Vec<MonthlyAmount>, DomainError>;
    fn monthly_expenses(&self, year: i32) -> Result<Vec<MonthlyAmount>, DomainError>;
    fn recent_invoices(&self, limit: i32) -> Result<Vec<RecentInvoice>, DomainError>;
    fn recent_expenses(&self, limit: i32) -> Result<Vec<RecentExpense>, DomainError>;
}
```

```rust
// ReportRepo -- 8 methods
pub trait ReportRepo {
    fn monthly_revenue(&self, year: i32) -> Result<Vec<MonthlyAmount>, DomainError>;
    fn quarterly_revenue(&self, year: i32) -> Result<Vec<QuarterlyAmount>, DomainError>;
    fn yearly_revenue(&self, year: i32) -> Result<Amount, DomainError>;
    fn monthly_expenses(&self, year: i32) -> Result<Vec<MonthlyAmount>, DomainError>;
    fn quarterly_expenses(&self, year: i32) -> Result<Vec<QuarterlyAmount>, DomainError>;
    fn category_expenses(&self, year: i32) -> Result<Vec<CategoryAmount>, DomainError>;
    fn top_customers(&self, year: i32, limit: i32) -> Result<Vec<CustomerRevenue>, DomainError>;
    /// Returns (revenue_months, expense_months).
    fn profit_loss_monthly(
        &self,
        year: i32,
    ) -> Result<(Vec<MonthlyAmount>, Vec<MonthlyAmount>), DomainError>;
}
```

#### 3.5 Reporting Types (zfaktury-core)

These live in `zfaktury-core/src/repository/types.rs` and are used by DashboardRepo and ReportRepo:

```rust
use crate::domain::Amount;
use chrono::NaiveDate;

pub struct MonthlyAmount {
    pub month: i32,
    pub amount: Amount,
}

pub struct QuarterlyAmount {
    pub quarter: i32,
    pub amount: Amount,
}

pub struct CategoryAmount {
    pub category: String,
    pub amount: Amount,
}

pub struct CustomerRevenue {
    pub customer_id: i64,
    pub customer_name: String,
    pub total: Amount,
    pub invoice_count: i32,
}

pub struct RecentInvoice {
    pub id: i64,
    pub invoice_number: String,
    pub customer_id: i64,
    pub total_amount: Amount,
    pub status: String,
    pub issue_date: NaiveDate,
}

pub struct RecentExpense {
    pub id: i64,
    pub description: String,
    pub category: String,
    pub amount: Amount,
    pub issue_date: NaiveDate,
}
```

### 4. Implementation Patterns

#### 4.1 Repository Struct

Each implementation is a simple struct holding a reference to the connection:

```rust
// zfaktury-db/src/repos/contact.rs

use rusqlite::Connection;
use zfaktury_core::repository::ContactRepo;

pub struct SqliteContactRepo<'a> {
    conn: &'a Connection,
}

impl<'a> SqliteContactRepo<'a> {
    pub fn new(conn: &'a Connection) -> Self {
        Self { conn }
    }
}

impl<'a> ContactRepo for SqliteContactRepo<'a> {
    fn create(&self, contact: &Contact) -> Result<Contact, DomainError> {
        self.conn.execute(
            "INSERT INTO contacts (type, name, ico, dic, street, city, zip, country,
             email, phone, web, bank_account, bank_code, iban, swift,
             payment_terms_days, tags, notes, is_favorite, vat_unreliable)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13,
                     ?14, ?15, ?16, ?17, ?18, ?19, ?20)",
            rusqlite::params![
                contact.contact_type,
                contact.name,
                contact.ico,
                // ... remaining fields
            ],
        ).map_err(|e| DomainError::Internal(e.to_string()))?;

        let id = self.conn.last_insert_rowid();
        self.get_by_id(id)
    }

    fn get_by_id(&self, id: i64) -> Result<Contact, DomainError> {
        self.conn
            .query_row(
                "SELECT id, type, name, ico, dic, ... FROM contacts
                 WHERE id = ?1 AND deleted_at IS NULL",
                [id],
                |row| scan_contact(row),
            )
            .map_err(|e| match e {
                rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
                other => DomainError::Internal(other.to_string()),
            })
    }

    // ... remaining methods
}
```

#### 4.2 Row Scanning Helpers

Extracted into `zfaktury-db/src/scan.rs` to avoid repetition (mirrors Go's `scanInvoiceRow`):

```rust
// zfaktury-db/src/scan.rs

use rusqlite::Row;
use chrono::NaiveDate;

pub fn scan_contact(row: &Row) -> rusqlite::Result<Contact> {
    Ok(Contact {
        id: row.get("id")?,
        contact_type: row.get("type")?,
        name: row.get("name")?,
        ico: row.get("ico")?,
        dic: row.get("dic")?,
        // ... all fields
        created_at: parse_datetime(row.get::<_, String>("created_at")?)?,
        updated_at: parse_datetime(row.get::<_, String>("updated_at")?)?,
    })
}

/// Parse ISO 8601 date string to NaiveDate.
/// Handles both "2006-01-02" and "2006-01-02T00:00:00Z" formats.
pub fn parse_date(value: &str) -> rusqlite::Result<NaiveDate> {
    // Try date-only first
    if let Ok(d) = NaiveDate::parse_from_str(value, "%Y-%m-%d") {
        return Ok(d);
    }
    // Try full timestamp, extract date portion
    if value.len() >= 10 {
        if let Ok(d) = NaiveDate::parse_from_str(&value[..10], "%Y-%m-%d") {
            return Ok(d);
        }
    }
    Err(rusqlite::Error::FromSqlConversionFailure(
        0,
        rusqlite::types::Type::Text,
        Box::new(std::fmt::Error),
    ))
}

/// Parse optional date from nullable TEXT column.
pub fn parse_date_opt(value: Option<String>) -> rusqlite::Result<Option<NaiveDate>> {
    match value {
        Some(v) if !v.is_empty() => Ok(Some(parse_date(&v)?)),
        _ => Ok(None),
    }
}

/// Parse ISO 8601 datetime string to NaiveDateTime.
/// Handles "2006-01-02T15:04:05Z" and "2006-01-02 15:04:05" formats.
pub fn parse_datetime(value: String) -> rusqlite::Result<NaiveDateTime> {
    // Try RFC3339 first
    if let Ok(dt) = NaiveDateTime::parse_from_str(&value, "%Y-%m-%dT%H:%M:%SZ") {
        return Ok(dt);
    }
    // Try space-separated
    if let Ok(dt) = NaiveDateTime::parse_from_str(&value, "%Y-%m-%d %H:%M:%S") {
        return Ok(dt);
    }
    Err(rusqlite::Error::FromSqlConversionFailure(
        0,
        rusqlite::types::Type::Text,
        Box::new(std::fmt::Error),
    ))
}
```

This mirrors the Go `helpers.go` pattern: `parseDate`, `parseDateOptional`, `parseDatePtr` have direct Rust equivalents. The fallback logic for mismatched date/datetime formats is preserved identically.

#### 4.3 Soft Delete Pattern

All entities with `deleted_at` follow the same pattern:

```rust
// SELECT queries always filter:
"... WHERE deleted_at IS NULL ..."

// DELETE sets the timestamp instead of removing the row:
fn delete(&self, id: i64) -> Result<(), DomainError> {
    let affected = self.conn.execute(
        "UPDATE contacts SET deleted_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
         WHERE id = ?1 AND deleted_at IS NULL",
        [id],
    ).map_err(|e| DomainError::Internal(e.to_string()))?;

    if affected == 0 {
        return Err(DomainError::NotFound);
    }
    Ok(())
}
```

#### 4.4 Pagination Pattern

Paginated list methods use `COUNT(*) OVER()` window function:

```rust
fn list(&self, filter: &ContactFilter) -> Result<(Vec<Contact>, i64), DomainError> {
    let mut sql = String::from(
        "SELECT *, COUNT(*) OVER() AS total_count
         FROM contacts WHERE deleted_at IS NULL"
    );
    let mut params: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();

    if let Some(ref search) = filter.search {
        sql.push_str(" AND (name LIKE ?1 OR ico LIKE ?1)");
        params.push(Box::new(format!("%{search}%")));
    }

    sql.push_str(" ORDER BY name ASC");
    sql.push_str(&format!(" LIMIT {} OFFSET {}", filter.limit, filter.offset));

    let mut stmt = self.conn.prepare(&sql)?;
    let rows = stmt.query_map(rusqlite::params_from_iter(&params), |row| {
        let total: i64 = row.get("total_count")?;
        let contact = scan_contact(row)?;
        Ok((contact, total))
    })?;

    let mut contacts = Vec::new();
    let mut total_count = 0i64;
    for row in rows {
        let (contact, total) = row?;
        total_count = total;
        contacts.push(contact);
    }

    Ok((contacts, total_count))
}
```

#### 4.5 Link Table Pattern (Many-to-Many)

VAT returns, income tax returns, etc. link to invoices/expenses via junction tables:

```rust
fn link_invoices(
    &self,
    vat_return_id: i64,
    invoice_ids: &[i64],
) -> Result<(), DomainError> {
    // Delete existing links
    self.conn.execute(
        "DELETE FROM vat_return_invoices WHERE vat_return_id = ?1",
        [vat_return_id],
    )?;

    // Insert new links
    let mut stmt = self.conn.prepare(
        "INSERT INTO vat_return_invoices (vat_return_id, invoice_id) VALUES (?1, ?2)"
    )?;
    for invoice_id in invoice_ids {
        stmt.execute(rusqlite::params![vat_return_id, invoice_id])?;
    }

    Ok(())
}
```

#### 4.6 Boolean Mapping

SQLite stores booleans as INTEGER 0/1. rusqlite handles this natively:

```rust
// Reading: rusqlite auto-converts 0/1 to bool
let is_favorite: bool = row.get("is_favorite")?;

// Writing: bool converts to 0/1 automatically
rusqlite::params![contact.is_favorite]
```

#### 4.7 Amount Mapping

All monetary values are stored as INTEGER (halere/cents). The `Amount` newtype wraps `i64`:

```rust
// Reading
let total: i64 = row.get("total_amount")?;
let amount = Amount(total);

// Writing
rusqlite::params![invoice.total_amount.0]
```

If `Amount` implements `rusqlite::types::FromSql` and `ToSql`, this simplifies to:

```rust
impl rusqlite::types::FromSql for Amount {
    fn column_result(value: rusqlite::types::ValueRef<'_>) -> rusqlite::types::FromSqlResult<Self> {
        i64::column_result(value).map(Amount)
    }
}

impl rusqlite::types::ToSql for Amount {
    fn to_sql(&self) -> rusqlite::Result<rusqlite::types::ToSqlOutput<'_>> {
        self.0.to_sql()
    }
}
```

These impls live in `zfaktury-db` (not `zfaktury-core`) to avoid the `rusqlite` dependency in core. Use a newtype wrapper or implement the conversion at the boundary.

### 5. Migrations

#### 5.1 Refinery Format

Migration files are placed in `rust/migrations/` and embedded at compile time:

```rust
// zfaktury-db/src/migrate.rs

use refinery::embed_migrations;

embed_migrations!("../migrations");

pub fn run_migrations(conn: &mut rusqlite::Connection) -> Result<(), DbError> {
    // Run the goose bridge first if needed
    bridge_goose_to_refinery(conn)?;

    // Then run refinery migrations
    migrations::runner()
        .run(conn)
        .map_err(|e| DbError::Migration(e.to_string()))?;

    Ok(())
}
```

#### 5.2 Migration File Mapping

The SQL content is ported identically from the Go goose files. Only the filename format changes:

| Go (goose) | Rust (refinery) |
|------------|-----------------|
| `001_initial_schema.sql` | `V1__initial_schema.sql` |
| `004_expense_categories.sql` | `V4__expense_categories.sql` |
| `005_vat_unreliable_to_timestamp.sql` | `V5__vat_unreliable_timestamp.sql` |
| `006_expense_documents.sql` | `V6__expense_documents.sql` |
| `007_invoice_relations.sql` | `V7__invoice_relations.sql` |
| `008_tax_review.sql` | `V8__tax_review.sql` |
| `009_recurring_invoices.sql` | `V9__recurring_invoices.sql` |
| `010_recurring_expenses.sql` | `V10__recurring_expenses.sql` |
| `012_invoice_status_history.sql` | `V12__invoice_status_history.sql` |
| `013_payment_reminders.sql` | `V13__payment_reminders.sql` |
| `014_vat_filings.sql` | `V14__vat_filings.sql` |
| `015_annual_tax.sql` | `V15__annual_tax.sql` |
| `016_tax_prepayments.sql` | `V16__tax_prepayments.sql` |
| `017_tax_credits_deductions.sql` | `V17__tax_credits_deductions.sql` |
| `018_spouse_months.sql` | `V18__spouse_months.sql` |
| `019_investment_income.sql` | `V19__investment_income.sql` |
| `020_fakturoid_import_log.sql` | `V20__fakturoid_import_log.sql` |
| `021_audit_log_expand_actions.sql` | `V21__audit_log_expand.sql` |
| `022_invoice_documents.sql` | `V22__invoice_documents.sql` |
| `023_expense_items.sql` | `V23__expense_items.sql` |
| `024_backup_history.sql` | `V24__backup_history.sql` |

Each file contains only the `-- +goose Up` portion (CREATE TABLE, ALTER TABLE, etc.). The `-- +goose Down` section is stripped -- refinery does not support down migrations, and the Go project never uses them in production.

#### 5.3 Goose-to-Refinery Bridge

When the Rust binary opens a database that was previously managed by Go/goose, it must populate refinery's `refinery_schema_history` table before running new migrations. This ensures refinery does not attempt to re-apply already-applied migrations.

```rust
// zfaktury-db/src/migrate.rs

/// Maps goose version numbers to refinery version numbers.
/// Goose versions that don't correspond to migration files (e.g., version 0
/// which is goose's internal baseline) are skipped.
const GOOSE_TO_REFINERY: &[(i64, i32)] = &[
    (1, 1),   // 001_initial_schema
    (4, 4),   // 004_expense_categories
    (5, 5),   // 005_vat_unreliable_timestamp
    (6, 6),   // 006_expense_documents
    (7, 7),   // 007_invoice_relations
    (8, 8),   // 008_tax_review
    (9, 9),   // 009_recurring_invoices
    (10, 10), // 010_recurring_expenses
    (12, 12), // 012_invoice_status_history
    (13, 13), // 013_payment_reminders
    (14, 14), // 014_vat_filings
    (15, 15), // 015_annual_tax
    (16, 16), // 016_tax_prepayments
    (17, 17), // 017_tax_credits_deductions
    (18, 18), // 018_spouse_months
    (19, 19), // 019_investment_income
    (20, 20), // 020_fakturoid_import_log
    (21, 21), // 021_audit_log_expand
    (22, 22), // 022_invoice_documents
    (23, 23), // 023_expense_items
    (24, 24), // 024_backup_history
];

fn bridge_goose_to_refinery(conn: &rusqlite::Connection) -> Result<(), DbError> {
    // Check if goose table exists
    let goose_exists: bool = conn
        .query_row(
            "SELECT COUNT(*) > 0 FROM sqlite_master
             WHERE type='table' AND name='goose_db_version'",
            [],
            |row| row.get(0),
        )
        .map_err(|e| DbError::Bridge(e.to_string()))?;

    if !goose_exists {
        return Ok(()); // Fresh database, no bridge needed
    }

    // Check if refinery table already exists (bridge already ran)
    let refinery_exists: bool = conn
        .query_row(
            "SELECT COUNT(*) > 0 FROM sqlite_master
             WHERE type='table' AND name='refinery_schema_history'",
            [],
            |row| row.get(0),
        )
        .map_err(|e| DbError::Bridge(e.to_string()))?;

    if refinery_exists {
        return Ok(()); // Bridge already ran
    }

    // Read current goose version
    let goose_version: i64 = conn
        .query_row(
            "SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version
             WHERE is_applied = 1",
            [],
            |row| row.get(0),
        )
        .map_err(|e| DbError::Bridge(e.to_string()))?;

    // Create refinery_schema_history table
    conn.execute_batch(
        "CREATE TABLE refinery_schema_history (
            version INTEGER PRIMARY KEY,
            name VARCHAR(255),
            applied_on VARCHAR(255),
            checksum VARCHAR(255)
        );"
    ).map_err(|e| DbError::Bridge(e.to_string()))?;

    // Insert entries for all goose versions up to the current one
    let mut stmt = conn.prepare(
        "INSERT INTO refinery_schema_history (version, name, applied_on, checksum)
         VALUES (?1, ?2, ?3, ?4)"
    ).map_err(|e| DbError::Bridge(e.to_string()))?;

    let now = chrono::Utc::now().format("%Y-%m-%d %H:%M:%S").to_string();

    for &(goose_ver, refinery_ver) in GOOSE_TO_REFINERY {
        if goose_ver > goose_version {
            break; // Only bridge versions that were actually applied
        }
        let name = migration_name_for_version(refinery_ver);
        stmt.execute(rusqlite::params![refinery_ver, name, now, "bridged"])
            .map_err(|e| DbError::Bridge(e.to_string()))?;
    }

    Ok(())
}

fn migration_name_for_version(version: i32) -> &'static str {
    match version {
        1 => "initial_schema",
        4 => "expense_categories",
        5 => "vat_unreliable_timestamp",
        6 => "expense_documents",
        7 => "invoice_relations",
        8 => "tax_review",
        9 => "recurring_invoices",
        10 => "recurring_expenses",
        12 => "invoice_status_history",
        13 => "payment_reminders",
        14 => "vat_filings",
        15 => "annual_tax",
        16 => "tax_prepayments",
        17 => "tax_credits_deductions",
        18 => "spouse_months",
        19 => "investment_income",
        20 => "fakturoid_import_log",
        21 => "audit_log_expand",
        22 => "invoice_documents",
        23 => "expense_items",
        24 => "backup_history",
        _ => "unknown",
    }
}
```

### 6. Database Struct

The top-level entry point for the persistence layer:

```rust
// zfaktury-db/src/lib.rs

pub struct Database {
    conn: rusqlite::Connection,
}

impl Database {
    /// Open an existing database or create a new one at the given path.
    /// Runs all pending migrations (including goose bridge if needed).
    pub fn open(path: &str) -> Result<Self, DbError> {
        let mut conn = connection::open_connection(path)?;
        migrate::run_migrations(&mut conn)?;
        Ok(Self { conn })
    }

    /// Open an in-memory database with all migrations applied.
    /// Used for testing.
    pub fn open_in_memory() -> Result<Self, DbError> {
        let mut conn = connection::open_in_memory()?;
        migrate::run_migrations(&mut conn)?;
        Ok(Self { conn })
    }

    /// Get a reference to the underlying connection.
    pub fn conn(&self) -> &rusqlite::Connection {
        &self.conn
    }

    // Factory methods for repositories
    pub fn contacts(&self) -> SqliteContactRepo<'_> {
        SqliteContactRepo::new(&self.conn)
    }

    pub fn invoices(&self) -> SqliteInvoiceRepo<'_> {
        SqliteInvoiceRepo::new(&self.conn)
    }

    pub fn expenses(&self) -> SqliteExpenseRepo<'_> {
        SqliteExpenseRepo::new(&self.conn)
    }

    // ... factory method for each of the 36 repos
}
```

### 7. Test Strategy

#### 7.1 Test Infrastructure

Each repo test file creates an in-memory database with all migrations applied:

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use crate::Database;

    fn test_db() -> Database {
        Database::open_in_memory().expect("failed to create test database")
    }

    #[test]
    fn test_create_and_get_contact() {
        let db = test_db();
        let repo = db.contacts();

        let contact = Contact {
            id: 0,
            contact_type: "company".to_string(),
            name: "Test s.r.o.".to_string(),
            ico: Some("12345678".to_string()),
            // ... remaining fields with defaults
        };

        let created = repo.create(&contact).unwrap();
        assert!(created.id > 0);
        assert_eq!(created.name, "Test s.r.o.");

        let fetched = repo.get_by_id(created.id).unwrap();
        assert_eq!(fetched.name, created.name);
        assert_eq!(fetched.ico, Some("12345678".to_string()));
    }
}
```

#### 7.2 Required Test Categories Per Repository

Each of the 36 repository implementations must have tests covering:

1. **CRUD operations:** Create, read (get_by_id), update, delete
2. **Soft delete correctness:** Deleted entities excluded from list/get queries
3. **Not found error:** get_by_id on non-existent or deleted ID returns `DomainError::NotFound`
4. **Filter/search:** Each filter field tested in isolation and combination
5. **Pagination:** Limit, offset, total count accuracy
6. **Unique constraints:** Duplicate detection where applicable (ICO, invoice number, etc.)
7. **Link tables:** Insert, replace, read back linked IDs (for VAT return, income tax repos)
8. **Edge cases:** Empty lists, zero amounts, optional fields as None

#### 7.3 Migration Tests

```rust
#[test]
fn test_all_migrations_apply_cleanly() {
    let db = Database::open_in_memory().unwrap();
    // Verify key tables exist
    let tables: Vec<String> = db.conn()
        .prepare("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
        .unwrap()
        .query_map([], |row| row.get(0))
        .unwrap()
        .collect::<Result<Vec<_>, _>>()
        .unwrap();

    assert!(tables.contains(&"contacts".to_string()));
    assert!(tables.contains(&"invoices".to_string()));
    assert!(tables.contains(&"expenses".to_string()));
    assert!(tables.contains(&"settings".to_string()));
    assert!(tables.contains(&"audit_log".to_string()));
    // ... all expected tables
}
```

#### 7.4 Bridge Tests

```rust
#[test]
fn test_goose_bridge_fresh_database() {
    // Fresh database should skip bridge entirely
    let db = Database::open_in_memory().unwrap();
    // Verify refinery_schema_history exists (created by refinery itself)
    // Verify goose_db_version does NOT exist
}

#[test]
fn test_goose_bridge_existing_goose_database() {
    let mut conn = connection::open_in_memory().unwrap();

    // Simulate a goose-managed database at version 24
    conn.execute_batch("
        CREATE TABLE goose_db_version (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            version_id INTEGER NOT NULL,
            is_applied INTEGER NOT NULL,
            tstamp TEXT NOT NULL DEFAULT (datetime('now'))
        );
        INSERT INTO goose_db_version (version_id, is_applied)
        VALUES (0, 1), (1, 1), (4, 1), (5, 1), (6, 1), (7, 1), (8, 1),
               (9, 1), (10, 1), (12, 1), (13, 1), (14, 1), (15, 1),
               (16, 1), (17, 1), (18, 1), (19, 1), (20, 1), (21, 1),
               (22, 1), (23, 1), (24, 1);
    ").unwrap();

    // Also create the actual tables (simulating a real goose-migrated DB)
    // ... run the raw SQL from each migration ...

    // Now run the Rust migration path
    migrate::run_migrations(&mut conn).unwrap();

    // Verify refinery_schema_history was populated
    let count: i64 = conn.query_row(
        "SELECT COUNT(*) FROM refinery_schema_history",
        [],
        |row| row.get(0),
    ).unwrap();
    assert_eq!(count, 21); // 21 migration files

    // Verify no migrations were re-applied (tables still intact, no errors)
}

#[test]
fn test_goose_bridge_partial_migration() {
    // Database at goose version 10 (only first 8 migrations applied)
    // Bridge should only populate refinery history for V1 through V10
    // Refinery should then apply V12 through V24 as new migrations
}

#[test]
fn test_goose_bridge_idempotent() {
    // Running bridge twice should not duplicate entries
}
```

### 8. Implementation Order

The repositories have dependencies (invoices need contacts, documents need invoices, etc.). Implementation order:

**Week 1: Foundation + Core Entities**

1. `zfaktury-db` crate scaffold: `Cargo.toml`, `lib.rs`, `connection.rs`, `error.rs`, `scan.rs`
2. Migration files: port all 21 SQL files to refinery format
3. `migrate.rs`: refinery runner (no bridge yet)
4. `ContactRepo`: full CRUD, filter, search, pagination
5. `CategoryRepo`: full CRUD, get_by_key
6. `SettingsRepo`: get/set/bulk
7. `InvoiceSequenceRepo`: full CRUD, prefix+year lookup, counting
8. `InvoiceRepo`: full CRUD, status update, numbering, relations
9. `ExpenseRepo`: full CRUD, tax review marking

**Week 2: Supporting Entities + Tax**

10. `DocumentRepo` (expense documents)
11. `InvoiceDocumentRepo`
12. `RecurringInvoiceRepo`
13. `RecurringExpenseRepo`
14. `StatusHistoryRepo`
15. `ReminderRepo`
16. `VATReturnRepo` (including link tables)
17. `VATControlStatementRepo` (including lines)
18. `VIESSummaryRepo` (including lines)
19. `IncomeTaxReturnRepo` (including link tables)
20. `SocialInsuranceOverviewRepo`
21. `HealthInsuranceOverviewRepo`
22. `TaxYearSettingsRepo`
23. `TaxPrepaymentRepo`
24. `TaxSpouseCreditRepo`
25. `TaxChildCreditRepo`
26. `TaxPersonalCreditsRepo`
27. `TaxDeductionRepo`
28. `TaxDeductionDocumentRepo`

**Week 3: Investments, Reporting, Bridge**

29. `InvestmentDocumentRepo`
30. `CapitalIncomeRepo`
31. `SecurityTransactionRepo`
32. `AuditLogRepo`
33. `FakturoidImportLogRepo`
34. `BackupHistoryRepo`
35. `DashboardRepo`
36. `ReportRepo`
37. Goose-to-refinery bridge implementation + tests
38. Full integration test suite review

### 9. DB Conventions (enforced in all implementations)

| Convention | Implementation |
|-----------|----------------|
| WAL mode | `PRAGMA journal_mode=WAL` on connection open |
| Foreign keys | `PRAGMA foreign_keys=ON` on connection open |
| Busy timeout | `PRAGMA busy_timeout=5000` on connection open |
| Dates as TEXT | ISO 8601 format, parsed via `scan.rs` helpers |
| Money as INTEGER | Stored as halere (i64), mapped to `Amount` newtype |
| Soft deletes | `deleted_at IS NULL` in all WHERE clauses for soft-deletable entities |
| Pagination | `LIMIT ? OFFSET ?` with `COUNT(*) OVER()` for total count |
| Parameterized queries | All values via `rusqlite::params![]`, never string interpolation |
| Nullable fields | `Option<T>` in Rust, NULL in SQLite |
| Booleans | `bool` in Rust, INTEGER 0/1 in SQLite |

### 10. Trait Counting Verification

Total: **36 traits**, **195 methods**

| # | Trait | Methods |
|---|-------|---------|
| 1 | ContactRepo | 6 |
| 2 | InvoiceRepo | 9 |
| 3 | ExpenseRepo | 7 |
| 4 | InvoiceSequenceRepo | 8 |
| 5 | CategoryRepo | 6 |
| 6 | DocumentRepo | 5 |
| 7 | InvoiceDocumentRepo | 5 |
| 8 | RecurringInvoiceRepo | 7 |
| 9 | RecurringExpenseRepo | 9 |
| 10 | StatusHistoryRepo | 2 |
| 11 | ReminderRepo | 3 |
| 12 | SettingsRepo | 4 |
| 13 | VATReturnRepo | 10 |
| 14 | VATControlStatementRepo | 9 |
| 15 | VIESSummaryRepo | 9 |
| 16 | IncomeTaxReturnRepo | 10 |
| 17 | SocialInsuranceOverviewRepo | 6 |
| 18 | HealthInsuranceOverviewRepo | 6 |
| 19 | TaxYearSettingsRepo | 2 |
| 20 | TaxPrepaymentRepo | 3 |
| 21 | TaxSpouseCreditRepo | 3 |
| 22 | TaxChildCreditRepo | 4 |
| 23 | TaxPersonalCreditsRepo | 2 |
| 24 | TaxDeductionRepo | 5 |
| 25 | TaxDeductionDocumentRepo | 5 |
| 26 | InvestmentDocumentRepo | 5 |
| 27 | CapitalIncomeRepo | 8 |
| 28 | SecurityTransactionRepo | 11 |
| 29 | AuditLogRepo | 3 |
| 30 | FakturoidImportLogRepo | 3 |
| 31 | BackupHistoryRepo | 5 |
| 32 | DashboardRepo | 8 |
| 33 | ReportRepo | 8 |
| 34 | MonthlyAmount (type) | -- |
| 35 | QuarterlyAmount (type) | -- |
| 36 | CategoryAmount (type) | -- |

Note: The "36" count includes the reporting types (MonthlyAmount, QuarterlyAmount, CategoryAmount, CustomerRevenue, RecentInvoice, RecentExpense) which are not traits but are part of the repository layer's public API. The actual trait count is **33 traits** + **6 supporting types**.

## Acceptance Criteria

1. All 33 repository traits defined in `zfaktury-core/src/repository/`
2. All 6 reporting types defined in `zfaktury-core/src/repository/types.rs`
3. All 33 rusqlite implementations pass integration tests with in-memory SQLite
4. All 21 migration files ported to refinery format with identical SQL
5. Migration bridge handles existing Go databases (goose version 1 through 24)
6. CRUD + soft delete + filter tests for every entity repository
7. Pagination tests (limit, offset, total count) for all paginated repos
8. Link table tests (VAT return invoices/expenses, income tax invoices/expenses)
9. Bridge tests: fresh DB, full goose DB, partial goose DB, idempotency
10. `cargo test -p zfaktury-db` -- all tests pass
11. `cargo clippy -p zfaktury-db -- -D warnings` -- no warnings

## Test Coverage Requirements

| Component | Target |
|-----------|--------|
| Repository trait definitions | N/A (no logic) |
| rusqlite implementations | 90%+ line coverage |
| Migration runner | 100% |
| Goose bridge | 100% |
| Scan helpers | 100% |

## Review Checklist

- [ ] All 33 trait signatures match Go interface methods exactly (accounting for Rust idioms)
- [ ] `create` methods return the entity with generated ID
- [ ] Error types use `DomainError::NotFound` for missing entities
- [ ] `DomainError::NotFound` returned when `rusqlite::Error::QueryReturnedNoRows` encountered
- [ ] Soft deletes filter correctly in all list/get queries
- [ ] Date parsing uses `scan.rs` helpers -- no `.unwrap()` on parse
- [ ] Amount columns read as i64 and convert to `Amount` newtype
- [ ] No SQL injection possible -- all queries use `rusqlite::params![]`
- [ ] Migration SQL identical to Go originals (only goose directives stripped)
- [ ] Migration version numbers match (V1, V4, V5, ... V24 -- gaps preserved)
- [ ] Bridge correctly maps goose version to refinery schema history
- [ ] Bridge is idempotent (running twice causes no errors or duplicates)
- [ ] `zfaktury-core` has no dependency on `rusqlite`
- [ ] All `Option<T>` fields correspond to nullable columns in the schema
- [ ] Pagination uses `COUNT(*) OVER()` window function
- [ ] Foreign key constraints honored in test data setup

## Open Questions

1. **Async vs sync:** This RFC assumes synchronous `rusqlite`. If Phase 3 (HTTP layer) uses `tokio`, we may need `tokio::task::spawn_blocking` wrappers or switch to `rusqlite` with `tokio-rusqlite`. Decision deferred to Phase 3 RFC.

2. **Transaction support:** Some service methods need multi-repo transactions (e.g., creating an invoice with items). The trait signatures don't expose transactions. Options: (a) add a `with_transaction` method to `Database`, (b) pass `&Transaction` instead of `&Connection`. Recommend option (a) with a closure-based API:
   ```rust
   db.with_transaction(|tx| {
       let invoices = SqliteInvoiceRepo::new(tx);
       let items = SqliteInvoiceItemRepo::new(tx);
       invoices.create(&invoice)?;
       for item in &items_list {
           items.create(item)?;
       }
       Ok(())
   })?;
   ```

3. **Connection pooling:** Single `Connection` is sufficient for SQLite in WAL mode with a single writer. If concurrency demands grow, `r2d2-rusqlite` can be added later without changing the trait layer.
