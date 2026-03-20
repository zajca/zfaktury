use std::sync::Arc;

use chrono::NaiveDate;
use zfaktury_domain::{Amount, CURRENCY_CZK, DomainError, Expense, RecurringExpense};

use super::audit_svc::AuditService;
use super::expense_svc::ExpenseService;
use crate::repository::traits::RecurringExpenseRepo;

/// Service for recurring expense management.
pub struct RecurringExpenseService {
    repo: Arc<dyn RecurringExpenseRepo + Send + Sync>,
    expenses: Arc<ExpenseService>,
    audit: Option<Arc<AuditService>>,
}

impl RecurringExpenseService {
    pub fn new(
        repo: Arc<dyn RecurringExpenseRepo + Send + Sync>,
        expenses: Arc<ExpenseService>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            expenses,
            audit,
        }
    }

    pub fn create(&self, re: &mut RecurringExpense) -> Result<(), DomainError> {
        if re.name.is_empty() || re.description.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        if re.amount == Amount::ZERO {
            return Err(DomainError::InvalidInput);
        }
        if re.next_issue_date == NaiveDate::default() {
            return Err(DomainError::InvalidInput);
        }
        if re.currency_code.is_empty() {
            re.currency_code = CURRENCY_CZK.to_string();
        }
        if re.business_percent == 0 {
            re.business_percent = 100;
        }
        if re.business_percent < 0 || re.business_percent > 100 {
            return Err(DomainError::InvalidInput);
        }
        if re.payment_method.is_empty() {
            re.payment_method = "bank_transfer".to_string();
        }
        if re.vat_amount == Amount::ZERO && re.vat_rate_percent > 0 {
            re.vat_amount = re
                .amount
                .multiply(re.vat_rate_percent as f64 / (100.0 + re.vat_rate_percent as f64));
        }
        self.repo.create(re)?;
        if let Some(ref audit) = self.audit {
            audit.log("recurring_expense", re.id, "create", None, None);
        }
        Ok(())
    }

    pub fn update(&self, re: &mut RecurringExpense) -> Result<(), DomainError> {
        if re.id == 0 || re.name.is_empty() || re.description.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        if re.amount == Amount::ZERO || re.next_issue_date == NaiveDate::default() {
            return Err(DomainError::InvalidInput);
        }
        if re.business_percent < 0 || re.business_percent > 100 {
            return Err(DomainError::InvalidInput);
        }
        if re.vat_amount == Amount::ZERO && re.vat_rate_percent > 0 {
            re.vat_amount = re
                .amount
                .multiply(re.vat_rate_percent as f64 / (100.0 + re.vat_rate_percent as f64));
        }
        self.repo.get_by_id(re.id)?;
        self.repo.update(re)?;
        if let Some(ref audit) = self.audit {
            audit.log("recurring_expense", re.id, "update", None, None);
        }
        Ok(())
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("recurring_expense", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<RecurringExpense, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list(
        &self,
        mut limit: i32,
        mut offset: i32,
    ) -> Result<(Vec<RecurringExpense>, i64), DomainError> {
        if limit <= 0 {
            limit = 20;
        }
        if limit > 100 {
            limit = 100;
        }
        if offset < 0 {
            offset = 0;
        }
        self.repo.list(limit, offset)
    }

    pub fn activate(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.activate(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("recurring_expense", id, "activate", None, None);
        }
        Ok(())
    }

    pub fn deactivate(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.deactivate(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("recurring_expense", id, "deactivate", None, None);
        }
        Ok(())
    }

    /// Generates pending expenses from due recurring templates.
    pub fn generate_pending(&self, as_of_date: NaiveDate) -> Result<i32, DomainError> {
        let due = self.repo.list_due(as_of_date)?;
        let mut generated = 0;

        for mut re in due {
            let now = chrono::Local::now().naive_local();
            let mut expense = Expense {
                id: 0,
                vendor_id: re.vendor_id,
                vendor: None,
                expense_number: String::new(),
                category: re.category.clone(),
                description: re.description.clone(),
                issue_date: re.next_issue_date,
                amount: re.amount,
                currency_code: re.currency_code.clone(),
                exchange_rate: re.exchange_rate,
                vat_rate_percent: re.vat_rate_percent,
                vat_amount: re.vat_amount,
                is_tax_deductible: re.is_tax_deductible,
                business_percent: re.business_percent,
                payment_method: re.payment_method.clone(),
                document_path: String::new(),
                notes: re.notes.clone(),
                tax_reviewed_at: None,
                items: Vec::new(),
                created_at: now,
                updated_at: now,
                deleted_at: None,
            };

            self.expenses.create(&mut expense)?;
            generated += 1;

            re.next_issue_date = re.next_date();
            if let Some(ref end_date) = re.end_date
                && re.next_issue_date > *end_date
            {
                re.is_active = false;
            }
            self.repo.update(&mut re)?;
        }

        Ok(generated)
    }
}
