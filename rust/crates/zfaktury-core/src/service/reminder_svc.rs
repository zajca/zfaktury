use std::sync::Arc;

use zfaktury_domain::{DomainError, PaymentReminder};

use crate::repository::traits::{InvoiceRepo, ReminderRepo, SettingsRepo};

/// Trait for sending emails, abstracted for testability.
pub trait EmailSender: Send + Sync {
    fn send(&self, to: &str, subject: &str, body: &str) -> Result<(), DomainError>;
}

/// Service for payment reminder management.
#[allow(dead_code)]
pub struct ReminderService {
    reminder_repo: Arc<dyn ReminderRepo + Send + Sync>,
    invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
    email_sender: Option<Arc<dyn EmailSender>>,
    settings_repo: Arc<dyn SettingsRepo + Send + Sync>,
}

impl ReminderService {
    pub fn new(
        reminder_repo: Arc<dyn ReminderRepo + Send + Sync>,
        invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
        email_sender: Option<Arc<dyn EmailSender>>,
        settings_repo: Arc<dyn SettingsRepo + Send + Sync>,
    ) -> Self {
        Self {
            reminder_repo,
            invoice_repo,
            email_sender,
            settings_repo,
        }
    }

    /// Sends a payment reminder for the given invoice.
    pub fn send_reminder(&self, invoice_id: i64) -> Result<PaymentReminder, DomainError> {
        let inv = self.invoice_repo.get_by_id(invoice_id)?;

        // Validate invoice is overdue.
        use zfaktury_domain::InvoiceStatus;
        if inv.status != InvoiceStatus::Overdue {
            let today = chrono::Local::now().date_naive();
            if !(inv.status == InvoiceStatus::Sent && today > inv.due_date) {
                return Err(DomainError::InvoiceNotOverdue);
            }
        }

        // Check customer email.
        let customer_email = inv
            .customer
            .as_ref()
            .map(|c| c.email.clone())
            .filter(|e| !e.is_empty())
            .ok_or(DomainError::NoCustomerEmail)?;

        // Determine reminder level.
        let count = self.reminder_repo.count_by_invoice_id(invoice_id)?;
        let level = std::cmp::min(count + 1, 3);

        let now = chrono::Local::now().naive_local();
        let subject = format!(
            "Payment reminder #{} for invoice {}",
            level, inv.invoice_number
        );

        // Send email if configured.
        if let Some(ref sender) = self.email_sender {
            sender.send(
                &customer_email,
                &subject,
                &format!("Please pay invoice {}", inv.invoice_number),
            )?;
        }

        let mut reminder = PaymentReminder {
            id: 0,
            invoice_id,
            reminder_number: level as i32,
            sent_at: now,
            sent_to: customer_email,
            subject,
            body_preview: String::new(),
            created_at: now,
        };
        self.reminder_repo.create(&mut reminder)?;
        Ok(reminder)
    }

    /// Returns all reminders for the given invoice.
    pub fn get_reminders(&self, invoice_id: i64) -> Result<Vec<PaymentReminder>, DomainError> {
        self.reminder_repo.list_by_invoice_id(invoice_id)
    }
}
