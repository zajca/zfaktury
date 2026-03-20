use std::sync::Arc;

use zfaktury_domain::{DomainError, InvoiceStatusChange};

use crate::repository::traits::{InvoiceRepo, StatusHistoryRepo};

/// Service for overdue detection and invoice status history.
pub struct OverdueService {
    invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
    history_repo: Arc<dyn StatusHistoryRepo + Send + Sync>,
}

impl OverdueService {
    pub fn new(
        invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
        history_repo: Arc<dyn StatusHistoryRepo + Send + Sync>,
    ) -> Self {
        Self {
            invoice_repo,
            history_repo,
        }
    }

    /// Finds all sent invoices past due date and marks them as overdue.
    pub fn check_overdue(&self) -> Result<i32, DomainError> {
        use zfaktury_domain::{InvoiceFilter, InvoiceStatus};
        let now = chrono::Local::now().naive_local();

        // List all sent invoices and check due dates.
        let (invoices, _) = self.invoice_repo.list(&InvoiceFilter {
            status: Some(InvoiceStatus::Sent),
            limit: 10000,
            ..Default::default()
        })?;

        let today = chrono::Local::now().date_naive();
        let mut count = 0;

        for inv in &invoices {
            if inv.due_date < today {
                if let Err(e) = self.invoice_repo.update_status(inv.id, "overdue") {
                    log::error!("failed to mark invoice {} as overdue: {}", inv.id, e);
                    continue;
                }
                count += 1;

                let mut change = InvoiceStatusChange {
                    id: 0,
                    invoice_id: inv.id,
                    old_status: InvoiceStatus::Sent,
                    new_status: InvoiceStatus::Overdue,
                    changed_at: now,
                    note: "automatically marked as overdue".to_string(),
                };
                if let Err(e) = self.history_repo.create(&mut change) {
                    log::error!(
                        "failed to record status change for invoice {}: {}",
                        inv.id,
                        e
                    );
                }
            }
        }

        Ok(count)
    }

    /// Returns the status change history for a given invoice.
    pub fn get_history(&self, invoice_id: i64) -> Result<Vec<InvoiceStatusChange>, DomainError> {
        self.history_repo.list_by_invoice_id(invoice_id)
    }
}
