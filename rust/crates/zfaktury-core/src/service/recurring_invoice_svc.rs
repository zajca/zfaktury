use std::sync::Arc;

use chrono::NaiveDate;
use zfaktury_domain::{
    Amount, CURRENCY_CZK, DomainError, Invoice, InvoiceItem, InvoiceStatus, InvoiceType,
    RecurringInvoice, RelationType,
};

use super::audit_svc::AuditService;
use super::invoice_svc::InvoiceService;
use crate::repository::traits::RecurringInvoiceRepo;

/// Service for recurring invoice management.
pub struct RecurringInvoiceService {
    repo: Arc<dyn RecurringInvoiceRepo + Send + Sync>,
    invoices: Arc<InvoiceService>,
    audit: Option<Arc<AuditService>>,
}

impl RecurringInvoiceService {
    pub fn new(
        repo: Arc<dyn RecurringInvoiceRepo + Send + Sync>,
        invoices: Arc<InvoiceService>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            invoices,
            audit,
        }
    }

    pub fn create(&self, ri: &mut RecurringInvoice) -> Result<(), DomainError> {
        if ri.name.is_empty() || ri.customer_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        if ri.items.is_empty() {
            return Err(DomainError::NoItems);
        }
        if ri.next_issue_date == NaiveDate::default() {
            return Err(DomainError::InvalidInput);
        }
        if ri.currency_code.is_empty() {
            ri.currency_code = CURRENCY_CZK.to_string();
        }
        self.repo.create(ri)?;
        if let Some(ref audit) = self.audit {
            audit.log("recurring_invoice", ri.id, "create", None, None);
        }
        Ok(())
    }

    pub fn update(&self, ri: &mut RecurringInvoice) -> Result<(), DomainError> {
        if ri.id == 0 || ri.name.is_empty() || ri.customer_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        if ri.items.is_empty() {
            return Err(DomainError::NoItems);
        }
        self.repo.get_by_id(ri.id)?;
        self.repo.update(ri)?;
        if let Some(ref audit) = self.audit {
            audit.log("recurring_invoice", ri.id, "update", None, None);
        }
        Ok(())
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("recurring_invoice", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<RecurringInvoice, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list(&self) -> Result<Vec<RecurringInvoice>, DomainError> {
        self.repo.list()
    }

    /// Generates a single invoice from a recurring template.
    pub fn generate_invoice(&self, id: i64) -> Result<Invoice, DomainError> {
        let ri = self.repo.get_by_id(id)?;
        self.create_invoice_from_template(&ri, ri.next_issue_date)
    }

    /// Processes all due recurring invoices.
    pub fn process_due(&self) -> Result<i32, DomainError> {
        let today = chrono::Local::now().date_naive();
        let due_list = self.repo.list_due(today)?;
        let mut count = 0;

        for mut ri in due_list {
            if let Some(ref end_date) = ri.end_date {
                if today > *end_date {
                    self.repo.deactivate(ri.id)?;
                    continue;
                }
            }

            self.create_invoice_from_template(&ri, ri.next_issue_date)?;
            count += 1;

            ri.next_issue_date = ri.next_date();
            if let Some(ref end_date) = ri.end_date {
                if ri.next_issue_date > *end_date {
                    ri.is_active = false;
                }
            }
            self.repo.update(&mut ri)?;
        }

        Ok(count)
    }

    fn create_invoice_from_template(
        &self,
        ri: &RecurringInvoice,
        issue_date: NaiveDate,
    ) -> Result<Invoice, DomainError> {
        let now = chrono::Local::now().naive_local();
        let mut invoice = Invoice {
            id: 0,
            sequence_id: 0,
            invoice_number: String::new(),
            invoice_type: InvoiceType::Regular,
            status: InvoiceStatus::Draft,
            issue_date,
            due_date: issue_date + chrono::Duration::days(14),
            delivery_date: issue_date,
            variable_symbol: String::new(),
            constant_symbol: ri.constant_symbol.clone(),
            customer_id: ri.customer_id,
            customer: None,
            currency_code: ri.currency_code.clone(),
            exchange_rate: ri.exchange_rate,
            payment_method: ri.payment_method.clone(),
            bank_account: ri.bank_account.clone(),
            bank_code: ri.bank_code.clone(),
            iban: ri.iban.clone(),
            swift: ri.swift.clone(),
            subtotal_amount: Amount::ZERO,
            vat_amount: Amount::ZERO,
            total_amount: Amount::ZERO,
            paid_amount: Amount::ZERO,
            notes: ri.notes.clone(),
            internal_notes: String::new(),
            related_invoice_id: None,
            relation_type: RelationType::None,
            sent_at: None,
            paid_at: None,
            items: ri
                .items
                .iter()
                .map(|item| InvoiceItem {
                    id: 0,
                    invoice_id: 0,
                    description: item.description.clone(),
                    quantity: item.quantity,
                    unit: item.unit.clone(),
                    unit_price: item.unit_price,
                    vat_rate_percent: item.vat_rate_percent,
                    vat_amount: Amount::ZERO,
                    total_amount: Amount::ZERO,
                    sort_order: item.sort_order,
                })
                .collect(),
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.invoices.create(&mut invoice)?;
        Ok(invoice)
    }
}
