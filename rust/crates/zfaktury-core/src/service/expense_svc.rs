use std::collections::HashSet;
use std::sync::Arc;

use zfaktury_domain::{Amount, CURRENCY_CZK, DomainError, Expense, ExpenseFilter};

use super::audit_svc::AuditService;
use crate::repository::traits::ExpenseRepo;

const MAX_BULK_IDS: usize = 500;

/// Service for expense management.
pub struct ExpenseService {
    repo: Arc<dyn ExpenseRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl ExpenseService {
    pub fn new(repo: Arc<dyn ExpenseRepo + Send + Sync>, audit: Option<Arc<AuditService>>) -> Self {
        Self { repo, audit }
    }

    pub fn create(&self, expense: &mut Expense) -> Result<(), DomainError> {
        if expense.description.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        if expense.amount == Amount::ZERO && expense.items.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        if expense.issue_date == chrono::NaiveDate::default() {
            return Err(DomainError::InvalidInput);
        }
        if expense.currency_code.is_empty() {
            expense.currency_code = CURRENCY_CZK.to_string();
        }
        if expense.business_percent == 0 {
            expense.business_percent = 100;
        }
        if expense.business_percent < 0 || expense.business_percent > 100 {
            return Err(DomainError::InvalidInput);
        }

        if !expense.items.is_empty() {
            expense.calculate_totals();
        } else if expense.vat_amount == Amount::ZERO && expense.vat_rate_percent > 0 {
            expense.vat_amount = expense.amount.multiply(
                expense.vat_rate_percent as f64 / (100.0 + expense.vat_rate_percent as f64),
            );
        }

        self.repo.create(expense)?;
        if let Some(ref audit) = self.audit {
            audit.log("expense", expense.id, "create", None, None);
        }
        Ok(())
    }

    pub fn update(&self, expense: &mut Expense) -> Result<(), DomainError> {
        if expense.id == 0 || expense.description.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        if expense.amount == Amount::ZERO && expense.items.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        if expense.business_percent < 0 || expense.business_percent > 100 {
            return Err(DomainError::InvalidInput);
        }

        if !expense.items.is_empty() {
            expense.calculate_totals();
        } else if expense.vat_amount == Amount::ZERO && expense.vat_rate_percent > 0 {
            expense.vat_amount = expense.amount.multiply(
                expense.vat_rate_percent as f64 / (100.0 + expense.vat_rate_percent as f64),
            );
        }

        self.repo.get_by_id(expense.id)?; // verify exists
        self.repo.update(expense)?;
        if let Some(ref audit) = self.audit {
            audit.log("expense", expense.id, "update", None, None);
        }
        Ok(())
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("expense", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<Expense, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list(&self, mut filter: ExpenseFilter) -> Result<(Vec<Expense>, i64), DomainError> {
        if filter.limit <= 0 {
            filter.limit = 20;
        }
        if filter.limit > 100 {
            filter.limit = 100;
        }
        if filter.offset < 0 {
            filter.offset = 0;
        }
        self.repo.list(&filter)
    }

    pub fn mark_tax_reviewed(&self, ids: &[i64]) -> Result<(), DomainError> {
        if ids.is_empty() || ids.len() > MAX_BULK_IDS {
            return Err(DomainError::InvalidInput);
        }
        let deduped = dedup_ids(ids);
        self.repo.mark_tax_reviewed(&deduped)
    }

    pub fn unmark_tax_reviewed(&self, ids: &[i64]) -> Result<(), DomainError> {
        if ids.is_empty() || ids.len() > MAX_BULK_IDS {
            return Err(DomainError::InvalidInput);
        }
        let deduped = dedup_ids(ids);
        self.repo.unmark_tax_reviewed(&deduped)
    }
}

fn dedup_ids(ids: &[i64]) -> Vec<i64> {
    let mut seen = HashSet::new();
    ids.iter().copied().filter(|id| seen.insert(*id)).collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_dedup_ids() {
        assert_eq!(dedup_ids(&[1, 2, 2, 3, 1]), vec![1, 2, 3]);
        assert_eq!(dedup_ids(&[]), Vec::<i64>::new());
    }
}
