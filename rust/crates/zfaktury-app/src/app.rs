use std::path::Path;
use std::sync::Arc;

use anyhow::{Context, Result};
use zfaktury_core::service::{
    AuditService, BackupService, CategoryService, ContactService, DashboardService,
    DocumentService, ExpenseService, HealthInsuranceService, ImportService, IncomeTaxReturnService,
    InvestmentDocumentService, InvestmentIncomeService, InvoiceDocumentService, InvoiceService,
    OverdueService, RecurringExpenseService, RecurringInvoiceService, ReminderService,
    ReportService, SequenceService, SettingsService, SocialInsuranceService, TaxCalendarService,
    TaxCreditsService, TaxDeductionDocumentService, TaxYearSettingsService,
    VATControlStatementService, VATReturnService, VIESSummaryService,
};
use zfaktury_db::connection::open_connection;
use zfaktury_db::migrate::run_migrations;
use zfaktury_db::repos::audit_log_repo::SqliteAuditLogRepo;
use zfaktury_db::repos::backup_repo::SqliteBackupRepo;
use zfaktury_db::repos::capital_income_repo::SqliteCapitalIncomeRepo;
use zfaktury_db::repos::category_repo::SqliteCategoryRepo;
use zfaktury_db::repos::contact_repo::SqliteContactRepo;
use zfaktury_db::repos::dashboard_repo::SqliteDashboardRepo;
use zfaktury_db::repos::document_repo::SqliteDocumentRepo;
use zfaktury_db::repos::expense_repo::SqliteExpenseRepo;
use zfaktury_db::repos::health_insurance_repo::SqliteHealthInsuranceRepo;
use zfaktury_db::repos::income_tax_return_repo::SqliteIncomeTaxReturnRepo;
use zfaktury_db::repos::investment_document_repo::SqliteInvestmentDocumentRepo;
use zfaktury_db::repos::invoice_document_repo::SqliteInvoiceDocumentRepo;
use zfaktury_db::repos::invoice_repo::SqliteInvoiceRepo;
use zfaktury_db::repos::recurring_expense_repo::SqliteRecurringExpenseRepo;
use zfaktury_db::repos::recurring_invoice_repo::SqliteRecurringInvoiceRepo;
use zfaktury_db::repos::reminder_repo::SqliteReminderRepo;
use zfaktury_db::repos::report_repo::SqliteReportRepo;
use zfaktury_db::repos::security_transaction_repo::SqliteSecurityTransactionRepo;
use zfaktury_db::repos::sequence_repo::SqliteSequenceRepo;
use zfaktury_db::repos::settings_repo::SqliteSettingsRepo;
use zfaktury_db::repos::social_insurance_repo::SqliteSocialInsuranceRepo;
use zfaktury_db::repos::status_history_repo::SqliteStatusHistoryRepo;
use zfaktury_db::repos::tax_child_credit_repo::SqliteTaxChildCreditRepo;
use zfaktury_db::repos::tax_deduction_document_repo::SqliteTaxDeductionDocumentRepo;
use zfaktury_db::repos::tax_deduction_repo::SqliteTaxDeductionRepo;
use zfaktury_db::repos::tax_personal_credits_repo::SqliteTaxPersonalCreditsRepo;
use zfaktury_db::repos::tax_prepayment_repo::SqliteTaxPrepaymentRepo;
use zfaktury_db::repos::tax_spouse_credit_repo::SqliteTaxSpouseCreditRepo;
use zfaktury_db::repos::tax_year_settings_repo::SqliteTaxYearSettingsRepo;
use zfaktury_db::repos::vat_control_repo::SqliteVATControlRepo;
use zfaktury_db::repos::vat_return_repo::SqliteVATReturnRepo;
use zfaktury_db::repos::vies_repo::SqliteVIESRepo;

/// Shared application state holding all service instances.
#[allow(dead_code)]
pub struct AppServices {
    // --- Core entity services (12, existing) ---
    pub dashboard: Arc<DashboardService>,
    pub invoices: Arc<InvoiceService>,
    pub expenses: Arc<ExpenseService>,
    pub contacts: Arc<ContactService>,
    pub settings: Arc<SettingsService>,
    pub categories: Arc<CategoryService>,
    pub sequences: Arc<SequenceService>,
    pub audit: Arc<AuditService>,
    pub recurring_invoices: Arc<RecurringInvoiceService>,
    pub recurring_expenses: Arc<RecurringExpenseService>,
    pub vat_returns: Arc<VATReturnService>,
    pub reports: Arc<ReportService>,

    // --- Newly wired services (18) ---
    pub backup: Arc<BackupService>,
    pub documents: Arc<DocumentService>,
    pub health_insurance: Arc<HealthInsuranceService>,
    pub income_tax: Arc<IncomeTaxReturnService>,
    pub investment_documents: Arc<InvestmentDocumentService>,
    pub investment_income: Arc<InvestmentIncomeService>,
    pub invoice_documents: Arc<InvoiceDocumentService>,
    pub overdue: Arc<OverdueService>,
    pub reminders: Arc<ReminderService>,
    pub social_insurance: Arc<SocialInsuranceService>,
    pub tax_calendar: Arc<TaxCalendarService>,
    pub tax_credits: Arc<TaxCreditsService>,
    pub tax_deduction_documents: Arc<TaxDeductionDocumentService>,
    pub tax_year_settings: Arc<TaxYearSettingsService>,
    pub vat_control: Arc<VATControlStatementService>,
    pub vies: Arc<VIESSummaryService>,
    pub import: Arc<ImportService>,
}

impl AppServices {
    /// Create all services wired to the given database and data directory.
    /// Opens a separate connection for each repository (SQLite WAL mode allows concurrent readers).
    pub fn new(db_path: &Path, data_dir: &Path) -> Result<Self> {
        // Run migrations on a dedicated connection.
        let migrate_conn =
            open_connection(db_path).context("opening db connection for migrations")?;
        run_migrations(&migrate_conn).context("running database migrations")?;
        drop(migrate_conn);

        let data_dir_str = data_dir.to_string_lossy().to_string();

        // Helper macro to reduce boilerplate for opening connections.
        macro_rules! conn {
            ($label:expr) => {
                open_connection(db_path).context(concat!("opening db for ", $label))?
            };
        }

        // --- Audit (dependency for many services) ---
        let audit_repo = Arc::new(SqliteAuditLogRepo::new(conn!("audit")));
        let audit = Arc::new(AuditService::new(audit_repo));

        // --- Core entity repos & services ---
        let contact_repo = Arc::new(SqliteContactRepo::new(conn!("contacts")));
        let contacts = Arc::new(ContactService::new(contact_repo, None, Some(audit.clone())));

        let sequence_repo = Arc::new(SqliteSequenceRepo::new(conn!("sequences")));
        let sequences = Arc::new(SequenceService::new(sequence_repo, Some(audit.clone())));

        let invoice_repo = Arc::new(SqliteInvoiceRepo::new(conn!("invoices")));
        let invoices = Arc::new(InvoiceService::new(
            invoice_repo,
            contacts.clone(),
            Some(sequences.clone()),
            Some(audit.clone()),
        ));

        let expense_repo = Arc::new(SqliteExpenseRepo::new(conn!("expenses")));
        let expenses = Arc::new(ExpenseService::new(expense_repo, Some(audit.clone())));

        let settings_repo = Arc::new(SqliteSettingsRepo::new(conn!("settings")));
        let settings = Arc::new(SettingsService::new(settings_repo, Some(audit.clone())));

        let category_repo = Arc::new(SqliteCategoryRepo::new(conn!("categories")));
        let categories = Arc::new(CategoryService::new(category_repo, Some(audit.clone())));

        let dashboard_repo = Arc::new(SqliteDashboardRepo::new(conn!("dashboard")));
        let dashboard = Arc::new(DashboardService::new(dashboard_repo));

        let recurring_invoice_repo =
            Arc::new(SqliteRecurringInvoiceRepo::new(conn!("recurring_invoices")));
        let recurring_invoices = Arc::new(RecurringInvoiceService::new(
            recurring_invoice_repo,
            invoices.clone(),
            Some(audit.clone()),
        ));

        let recurring_expense_repo =
            Arc::new(SqliteRecurringExpenseRepo::new(conn!("recurring_expenses")));
        let recurring_expenses = Arc::new(RecurringExpenseService::new(
            recurring_expense_repo,
            expenses.clone(),
            Some(audit.clone()),
        ));

        // VATReturnService needs its own repo instances.
        let vat_return_repo = Arc::new(SqliteVATReturnRepo::new(conn!("vat_returns")));
        let vat_invoice_repo = Arc::new(SqliteInvoiceRepo::new(conn!("vat_invoices")));
        let vat_expense_repo = Arc::new(SqliteExpenseRepo::new(conn!("vat_expenses")));
        let vat_settings_repo = Arc::new(SqliteSettingsRepo::new(conn!("vat_settings")));
        let vat_returns = Arc::new(VATReturnService::new(
            vat_return_repo,
            vat_invoice_repo,
            vat_expense_repo,
            vat_settings_repo,
            Some(audit.clone()),
        ));

        let report_repo = Arc::new(SqliteReportRepo::new(conn!("reports")));
        let reports = Arc::new(ReportService::new(report_repo));

        // --- Newly wired services ---

        // BackupService
        let backup_repo = Arc::new(SqliteBackupRepo::new(conn!("backup")));
        let backup = Arc::new(BackupService::new(backup_repo, data_dir_str.clone()));

        // DocumentService
        let document_repo = Arc::new(SqliteDocumentRepo::new(conn!("documents")));
        let documents = Arc::new(DocumentService::new(
            document_repo,
            data_dir_str.clone(),
            Some(audit.clone()),
        ));

        // InvoiceDocumentService
        let invoice_document_repo =
            Arc::new(SqliteInvoiceDocumentRepo::new(conn!("invoice_documents")));
        let invoice_documents = Arc::new(InvoiceDocumentService::new(
            invoice_document_repo,
            data_dir_str.clone(),
            Some(audit.clone()),
        ));

        // OverdueService
        let overdue_invoice_repo = Arc::new(SqliteInvoiceRepo::new(conn!("overdue_invoices")));
        let status_history_repo = Arc::new(SqliteStatusHistoryRepo::new(conn!("status_history")));
        let overdue = Arc::new(OverdueService::new(
            overdue_invoice_repo,
            status_history_repo,
        ));

        // ReminderService (no email sender in desktop app for now)
        let reminder_repo = Arc::new(SqliteReminderRepo::new(conn!("reminders")));
        let reminder_invoice_repo = Arc::new(SqliteInvoiceRepo::new(conn!("reminder_invoices")));
        let reminder_settings_repo = Arc::new(SqliteSettingsRepo::new(conn!("reminder_settings")));
        let reminders = Arc::new(ReminderService::new(
            reminder_repo,
            reminder_invoice_repo,
            None, // email sender wired when SMTP is configured
            reminder_settings_repo,
        ));

        // TaxCalendarService (stateless)
        let tax_calendar = Arc::new(TaxCalendarService::new());

        // TaxYearSettingsService
        let tax_year_settings_repo =
            Arc::new(SqliteTaxYearSettingsRepo::new(conn!("tax_year_settings")));
        let tax_prepayment_repo = Arc::new(SqliteTaxPrepaymentRepo::new(conn!("tax_prepayments")));
        let tax_year_settings = Arc::new(TaxYearSettingsService::new(
            tax_year_settings_repo,
            tax_prepayment_repo,
            Some(audit.clone()),
        ));

        // TaxCreditsService
        let tax_spouse_repo = Arc::new(SqliteTaxSpouseCreditRepo::new(conn!("tax_spouse_credits")));
        let tax_child_repo = Arc::new(SqliteTaxChildCreditRepo::new(conn!("tax_child_credits")));
        let tax_personal_repo = Arc::new(SqliteTaxPersonalCreditsRepo::new(conn!(
            "tax_personal_credits"
        )));
        let tax_deduction_repo = Arc::new(SqliteTaxDeductionRepo::new(conn!("tax_deductions")));
        let tax_credits = Arc::new(TaxCreditsService::new(
            tax_spouse_repo,
            tax_child_repo,
            tax_personal_repo,
            tax_deduction_repo.clone(),
            Some(audit.clone()),
        ));

        // TaxDeductionDocumentService
        let tax_deduction_doc_repo = Arc::new(SqliteTaxDeductionDocumentRepo::new(conn!(
            "tax_deduction_documents"
        )));
        let tax_deduction_documents = Arc::new(TaxDeductionDocumentService::new(
            tax_deduction_doc_repo,
            tax_deduction_repo,
            data_dir_str.clone(),
            Some(audit.clone()),
        ));

        // InvestmentIncomeService
        let capital_repo = Arc::new(SqliteCapitalIncomeRepo::new(conn!("capital_income")));
        let security_repo = Arc::new(SqliteSecurityTransactionRepo::new(conn!(
            "security_transactions"
        )));
        let investment_income = Arc::new(InvestmentIncomeService::new(
            capital_repo.clone(),
            security_repo.clone(),
            Some(audit.clone()),
        ));

        // InvestmentDocumentService
        let investment_doc_repo = Arc::new(SqliteInvestmentDocumentRepo::new(conn!(
            "investment_documents"
        )));
        let inv_doc_capital_repo = Arc::new(SqliteCapitalIncomeRepo::new(conn!("inv_doc_capital")));
        let inv_doc_security_repo = Arc::new(SqliteSecurityTransactionRepo::new(conn!(
            "inv_doc_security"
        )));
        let investment_documents = Arc::new(InvestmentDocumentService::new(
            investment_doc_repo,
            inv_doc_capital_repo,
            inv_doc_security_repo,
            data_dir_str.clone(),
            Some(audit.clone()),
        ));

        // IncomeTaxReturnService
        let income_tax_repo = Arc::new(SqliteIncomeTaxReturnRepo::new(conn!("income_tax_returns")));
        let it_invoice_repo = Arc::new(SqliteInvoiceRepo::new(conn!("it_invoices")));
        let it_expense_repo = Arc::new(SqliteExpenseRepo::new(conn!("it_expenses")));
        let it_settings_repo = Arc::new(SqliteSettingsRepo::new(conn!("it_settings")));
        let it_tys_repo = Arc::new(SqliteTaxYearSettingsRepo::new(conn!(
            "it_tax_year_settings"
        )));
        let it_prepayment_repo = Arc::new(SqliteTaxPrepaymentRepo::new(conn!("it_prepayments")));
        let income_tax = Arc::new(IncomeTaxReturnService::new(
            income_tax_repo,
            it_invoice_repo,
            it_expense_repo,
            it_settings_repo,
            it_tys_repo,
            it_prepayment_repo,
            Some(audit.clone()),
        ));

        // SocialInsuranceService
        let social_repo = Arc::new(SqliteSocialInsuranceRepo::new(conn!("social_insurance")));
        let si_invoice_repo = Arc::new(SqliteInvoiceRepo::new(conn!("si_invoices")));
        let si_expense_repo = Arc::new(SqliteExpenseRepo::new(conn!("si_expenses")));
        let si_settings_repo = Arc::new(SqliteSettingsRepo::new(conn!("si_settings")));
        let si_tys_repo = Arc::new(SqliteTaxYearSettingsRepo::new(conn!(
            "si_tax_year_settings"
        )));
        let si_prepayment_repo = Arc::new(SqliteTaxPrepaymentRepo::new(conn!("si_prepayments")));
        let social_insurance = Arc::new(SocialInsuranceService::new(
            social_repo,
            si_invoice_repo,
            si_expense_repo,
            si_settings_repo,
            si_tys_repo,
            si_prepayment_repo,
            Some(audit.clone()),
        ));

        // HealthInsuranceService
        let health_repo = Arc::new(SqliteHealthInsuranceRepo::new(conn!("health_insurance")));
        let hi_invoice_repo = Arc::new(SqliteInvoiceRepo::new(conn!("hi_invoices")));
        let hi_expense_repo = Arc::new(SqliteExpenseRepo::new(conn!("hi_expenses")));
        let hi_settings_repo = Arc::new(SqliteSettingsRepo::new(conn!("hi_settings")));
        let hi_tys_repo = Arc::new(SqliteTaxYearSettingsRepo::new(conn!(
            "hi_tax_year_settings"
        )));
        let hi_prepayment_repo = Arc::new(SqliteTaxPrepaymentRepo::new(conn!("hi_prepayments")));
        let health_insurance = Arc::new(HealthInsuranceService::new(
            health_repo,
            hi_invoice_repo,
            hi_expense_repo,
            hi_settings_repo,
            hi_tys_repo,
            hi_prepayment_repo,
            Some(audit.clone()),
        ));

        // VATControlStatementService
        let vat_control_repo = Arc::new(SqliteVATControlRepo::new(conn!("vat_control")));
        let vc_invoice_repo = Arc::new(SqliteInvoiceRepo::new(conn!("vc_invoices")));
        let vc_expense_repo = Arc::new(SqliteExpenseRepo::new(conn!("vc_expenses")));
        let vc_contact_repo = Arc::new(SqliteContactRepo::new(conn!("vc_contacts")));
        let vat_control = Arc::new(VATControlStatementService::new(
            vat_control_repo,
            vc_invoice_repo,
            vc_expense_repo,
            vc_contact_repo,
            Some(audit.clone()),
        ));

        // VIESSummaryService
        let vies_repo = Arc::new(SqliteVIESRepo::new(conn!("vies")));
        let vies_invoice_repo = Arc::new(SqliteInvoiceRepo::new(conn!("vies_invoices")));
        let vies_contact_repo = Arc::new(SqliteContactRepo::new(conn!("vies_contacts")));
        let vies = Arc::new(VIESSummaryService::new(
            vies_repo,
            vies_invoice_repo,
            vies_contact_repo,
            Some(audit.clone()),
        ));

        // ImportService (depends on ExpenseService, DocumentService; OCR not configured)
        let import = Arc::new(ImportService::new(
            expenses.clone(),
            documents.clone(),
            None, // OCR provider not configured for desktop app
        ));

        Ok(Self {
            // Core (existing)
            dashboard,
            invoices,
            expenses,
            contacts,
            settings,
            categories,
            sequences,
            audit,
            recurring_invoices,
            recurring_expenses,
            vat_returns,
            reports,
            // Newly wired
            backup,
            documents,
            health_insurance,
            income_tax,
            investment_documents,
            investment_income,
            invoice_documents,
            overdue,
            reminders,
            social_insurance,
            tax_calendar,
            tax_credits,
            tax_deduction_documents,
            tax_year_settings,
            vat_control,
            vies,
            import,
        })
    }
}
