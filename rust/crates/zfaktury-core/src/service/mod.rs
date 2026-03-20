//! Service layer: business logic on top of repository traits.
//!
//! Each service owns `Arc<dyn RepoTrait>` references and may depend on other
//! services (also via `Arc`). All services are `Send + Sync` for sharing in
//! an `Arc<AppServices>` container.

pub mod audit_svc;
pub mod backup_svc;
pub mod category_svc;
pub mod contact_svc;
pub mod dashboard_svc;
pub mod document_svc;
pub mod expense_svc;
pub mod health_insurance_svc;
pub mod import_svc;
pub mod income_tax_return_svc;
pub mod investment_document_svc;
pub mod investment_income_svc;
pub mod invoice_document_svc;
pub mod invoice_svc;
pub mod ocr_svc;
pub mod overdue_svc;
pub mod recurring_expense_svc;
pub mod recurring_invoice_svc;
pub mod reminder_svc;
pub mod report_svc;
pub mod sequence_svc;
pub mod settings_svc;
pub mod social_insurance_svc;
pub mod tax_calendar_svc;
pub mod tax_credits_svc;
pub mod tax_deduction_document_svc;
pub mod tax_year_settings_svc;
pub mod vat_control_svc;
pub mod vat_return_svc;
pub mod vies_svc;

// Re-export service structs for convenience.
pub use audit_svc::AuditService;
pub use backup_svc::BackupService;
pub use category_svc::CategoryService;
pub use contact_svc::ContactService;
pub use dashboard_svc::DashboardService;
pub use document_svc::DocumentService;
pub use expense_svc::ExpenseService;
pub use health_insurance_svc::HealthInsuranceService;
pub use income_tax_return_svc::IncomeTaxReturnService;
pub use investment_document_svc::InvestmentDocumentService;
pub use investment_income_svc::InvestmentIncomeService;
pub use invoice_document_svc::InvoiceDocumentService;
pub use invoice_svc::InvoiceService;
pub use ocr_svc::OCRService;
pub use overdue_svc::OverdueService;
pub use recurring_expense_svc::RecurringExpenseService;
pub use recurring_invoice_svc::RecurringInvoiceService;
pub use reminder_svc::ReminderService;
pub use report_svc::ReportService;
pub use sequence_svc::SequenceService;
pub use settings_svc::SettingsService;
pub use social_insurance_svc::SocialInsuranceService;
pub use tax_calendar_svc::TaxCalendarService;
pub use tax_credits_svc::TaxCreditsService;
pub use tax_deduction_document_svc::TaxDeductionDocumentService;
pub use tax_year_settings_svc::TaxYearSettingsService;
pub use vat_control_svc::VATControlStatementService;
pub use vat_return_svc::VATReturnService;
pub use vies_svc::VIESSummaryService;
