use std::path::Path;
use std::sync::Arc;

use anyhow::{Context, Result};
use zfaktury_core::service::{
    AuditService, CategoryService, ContactService, DashboardService, ExpenseService,
    InvoiceService, RecurringExpenseService, RecurringInvoiceService, ReportService,
    SequenceService, SettingsService, VATReturnService,
};
use zfaktury_db::connection::open_connection;
use zfaktury_db::migrate::run_migrations;
use zfaktury_db::repos::audit_log_repo::SqliteAuditLogRepo;
use zfaktury_db::repos::category_repo::SqliteCategoryRepo;
use zfaktury_db::repos::contact_repo::SqliteContactRepo;
use zfaktury_db::repos::dashboard_repo::SqliteDashboardRepo;
use zfaktury_db::repos::expense_repo::SqliteExpenseRepo;
use zfaktury_db::repos::invoice_repo::SqliteInvoiceRepo;
use zfaktury_db::repos::recurring_expense_repo::SqliteRecurringExpenseRepo;
use zfaktury_db::repos::recurring_invoice_repo::SqliteRecurringInvoiceRepo;
use zfaktury_db::repos::report_repo::SqliteReportRepo;
use zfaktury_db::repos::sequence_repo::SqliteSequenceRepo;
use zfaktury_db::repos::settings_repo::SqliteSettingsRepo;
use zfaktury_db::repos::vat_return_repo::SqliteVATReturnRepo;

/// Shared application state holding all service instances.
#[allow(dead_code)]
pub struct AppServices {
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
}

impl AppServices {
    /// Create all services wired to the given database path.
    /// Opens a separate connection for each repository (SQLite WAL mode allows concurrent readers).
    pub fn new(db_path: &Path) -> Result<Self> {
        // Run migrations on a dedicated connection.
        let migrate_conn =
            open_connection(db_path).context("opening db connection for migrations")?;
        run_migrations(&migrate_conn).context("running database migrations")?;
        drop(migrate_conn);

        // Create a connection per repository.
        let audit_repo = Arc::new(SqliteAuditLogRepo::new(
            open_connection(db_path).context("opening db for audit")?,
        ));
        let audit = Arc::new(AuditService::new(audit_repo));

        let contact_repo = Arc::new(SqliteContactRepo::new(
            open_connection(db_path).context("opening db for contacts")?,
        ));
        let contacts = Arc::new(ContactService::new(contact_repo, None, Some(audit.clone())));

        let sequence_repo = Arc::new(SqliteSequenceRepo::new(
            open_connection(db_path).context("opening db for sequences")?,
        ));
        let sequences = Arc::new(SequenceService::new(sequence_repo, Some(audit.clone())));

        let invoice_repo = Arc::new(SqliteInvoiceRepo::new(
            open_connection(db_path).context("opening db for invoices")?,
        ));
        let invoices = Arc::new(InvoiceService::new(
            invoice_repo,
            contacts.clone(),
            Some(sequences.clone()),
            Some(audit.clone()),
        ));

        let expense_repo = Arc::new(SqliteExpenseRepo::new(
            open_connection(db_path).context("opening db for expenses")?,
        ));
        let expenses = Arc::new(ExpenseService::new(expense_repo, Some(audit.clone())));

        let settings_repo = Arc::new(SqliteSettingsRepo::new(
            open_connection(db_path).context("opening db for settings")?,
        ));
        let settings = Arc::new(SettingsService::new(settings_repo, Some(audit.clone())));

        let category_repo = Arc::new(SqliteCategoryRepo::new(
            open_connection(db_path).context("opening db for categories")?,
        ));
        let categories = Arc::new(CategoryService::new(category_repo, Some(audit.clone())));

        let dashboard_repo = Arc::new(SqliteDashboardRepo::new(
            open_connection(db_path).context("opening db for dashboard")?,
        ));
        let dashboard = Arc::new(DashboardService::new(dashboard_repo));

        let recurring_invoice_repo = Arc::new(SqliteRecurringInvoiceRepo::new(
            open_connection(db_path).context("opening db for recurring invoices")?,
        ));
        let recurring_invoices = Arc::new(RecurringInvoiceService::new(
            recurring_invoice_repo,
            invoices.clone(),
            Some(audit.clone()),
        ));

        let recurring_expense_repo = Arc::new(SqliteRecurringExpenseRepo::new(
            open_connection(db_path).context("opening db for recurring expenses")?,
        ));
        let recurring_expenses = Arc::new(RecurringExpenseService::new(
            recurring_expense_repo,
            expenses.clone(),
            Some(audit.clone()),
        ));

        let vat_return_repo = Arc::new(SqliteVATReturnRepo::new(
            open_connection(db_path).context("opening db for vat returns")?,
        ));
        // VATReturnService needs its own invoice/expense/settings repo instances.
        let vat_invoice_repo = Arc::new(SqliteInvoiceRepo::new(
            open_connection(db_path).context("opening db for vat invoices")?,
        ));
        let vat_expense_repo = Arc::new(SqliteExpenseRepo::new(
            open_connection(db_path).context("opening db for vat expenses")?,
        ));
        let vat_settings_repo = Arc::new(SqliteSettingsRepo::new(
            open_connection(db_path).context("opening db for vat settings")?,
        ));
        let vat_returns = Arc::new(VATReturnService::new(
            vat_return_repo,
            vat_invoice_repo,
            vat_expense_repo,
            vat_settings_repo,
            Some(audit.clone()),
        ));

        let report_repo = Arc::new(SqliteReportRepo::new(
            open_connection(db_path).context("opening db for reports")?,
        ));
        let reports = Arc::new(ReportService::new(report_repo));

        Ok(Self {
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
        })
    }
}
