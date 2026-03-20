use std::sync::Arc;

use zfaktury_domain::{
    CapitalIncomeEntry, DomainError, InvestmentYearSummary, SecurityTransaction,
};

use super::audit_svc::AuditService;
use crate::repository::traits::{CapitalIncomeRepo, SecurityTransactionRepo};

/// Service for investment income management (capital income + security transactions).
pub struct InvestmentIncomeService {
    capital_repo: Arc<dyn CapitalIncomeRepo + Send + Sync>,
    security_repo: Arc<dyn SecurityTransactionRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl InvestmentIncomeService {
    pub fn new(
        capital_repo: Arc<dyn CapitalIncomeRepo + Send + Sync>,
        security_repo: Arc<dyn SecurityTransactionRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            capital_repo,
            security_repo,
            audit,
        }
    }

    // --- Capital Income CRUD ---

    pub fn create_capital_entry(&self, entry: &mut CapitalIncomeEntry) -> Result<(), DomainError> {
        entry.net_amount = entry.gross_amount - entry.withheld_tax_cz - entry.withheld_tax_foreign;
        self.capital_repo.create(entry)?;
        if let Some(ref audit) = self.audit {
            audit.log("capital_income", entry.id, "create", None, None);
        }
        Ok(())
    }

    pub fn update_capital_entry(&self, entry: &mut CapitalIncomeEntry) -> Result<(), DomainError> {
        entry.net_amount = entry.gross_amount - entry.withheld_tax_cz - entry.withheld_tax_foreign;
        self.capital_repo.update(entry)?;
        if let Some(ref audit) = self.audit {
            audit.log("capital_income", entry.id, "update", None, None);
        }
        Ok(())
    }

    pub fn delete_capital_entry(&self, id: i64) -> Result<(), DomainError> {
        self.capital_repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("capital_income", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_capital_entry(&self, id: i64) -> Result<CapitalIncomeEntry, DomainError> {
        self.capital_repo.get_by_id(id)
    }

    pub fn list_capital_entries(&self, year: i32) -> Result<Vec<CapitalIncomeEntry>, DomainError> {
        self.capital_repo.list_by_year(year)
    }

    // --- Security Transaction CRUD ---

    pub fn create_security_transaction(
        &self,
        tx: &mut SecurityTransaction,
    ) -> Result<(), DomainError> {
        self.security_repo.create(tx)?;
        if let Some(ref audit) = self.audit {
            audit.log("security_transaction", tx.id, "create", None, None);
        }
        Ok(())
    }

    pub fn update_security_transaction(
        &self,
        tx: &mut SecurityTransaction,
    ) -> Result<(), DomainError> {
        self.security_repo.update(tx)?;
        if let Some(ref audit) = self.audit {
            audit.log("security_transaction", tx.id, "update", None, None);
        }
        Ok(())
    }

    pub fn delete_security_transaction(&self, id: i64) -> Result<(), DomainError> {
        self.security_repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("security_transaction", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_security_transaction(&self, id: i64) -> Result<SecurityTransaction, DomainError> {
        self.security_repo.get_by_id(id)
    }

    pub fn list_security_transactions(
        &self,
        year: i32,
    ) -> Result<Vec<SecurityTransaction>, DomainError> {
        self.security_repo.list_by_year(year)
    }

    // --- Year Summary ---

    pub fn get_year_summary(&self, year: i32) -> Result<InvestmentYearSummary, DomainError> {
        let (capital_gross, capital_tax, capital_net) = self.capital_repo.sum_by_year(year)?;
        let sells = self.security_repo.list_sells_by_year(year)?;

        let mut other_gross = zfaktury_domain::Amount::ZERO;
        let mut other_expenses = zfaktury_domain::Amount::ZERO;
        let mut other_exempt = zfaktury_domain::Amount::ZERO;

        for sell in &sells {
            other_gross += sell.total_amount - sell.fees;
            other_expenses += sell.cost_basis;
            other_exempt += sell.exempt_amount;
        }
        let other_net = other_gross - other_expenses - other_exempt;

        Ok(InvestmentYearSummary {
            year,
            capital_income_gross: capital_gross,
            capital_income_tax: capital_tax,
            capital_income_net: capital_net,
            other_income_gross: other_gross,
            other_income_expenses: other_expenses,
            other_income_exempt: other_exempt,
            other_income_net: other_net,
        })
    }
}
