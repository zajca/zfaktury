use std::sync::Arc;

use zfaktury_domain::{
    DomainError, TaxChildCredit, TaxDeduction, TaxPersonalCredits, TaxSpouseCredit,
};

use super::audit_svc::AuditService;
use crate::repository::traits::{
    TaxChildCreditRepo, TaxDeductionRepo, TaxPersonalCreditsRepo, TaxSpouseCreditRepo,
};

/// Service for tax credits and deductions management.
pub struct TaxCreditsService {
    spouse_repo: Arc<dyn TaxSpouseCreditRepo + Send + Sync>,
    child_repo: Arc<dyn TaxChildCreditRepo + Send + Sync>,
    personal_repo: Arc<dyn TaxPersonalCreditsRepo + Send + Sync>,
    deduction_repo: Arc<dyn TaxDeductionRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl TaxCreditsService {
    pub fn new(
        spouse_repo: Arc<dyn TaxSpouseCreditRepo + Send + Sync>,
        child_repo: Arc<dyn TaxChildCreditRepo + Send + Sync>,
        personal_repo: Arc<dyn TaxPersonalCreditsRepo + Send + Sync>,
        deduction_repo: Arc<dyn TaxDeductionRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            spouse_repo,
            child_repo,
            personal_repo,
            deduction_repo,
            audit,
        }
    }

    // --- Spouse credit ---

    pub fn upsert_spouse(&self, credit: &mut TaxSpouseCredit) -> Result<(), DomainError> {
        if credit.year < 2000 || credit.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        if credit.months_claimed < 1 || credit.months_claimed > 12 {
            return Err(DomainError::InvalidInput);
        }
        self.spouse_repo.upsert(credit)?;
        if let Some(ref audit) = self.audit {
            audit.log(
                "tax_spouse_credit",
                credit.year as i64,
                "upsert",
                None,
                None,
            );
        }
        Ok(())
    }

    pub fn get_spouse(&self, year: i32) -> Result<TaxSpouseCredit, DomainError> {
        self.spouse_repo.get_by_year(year)
    }

    pub fn delete_spouse(&self, year: i32) -> Result<(), DomainError> {
        self.spouse_repo.delete_by_year(year)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_spouse_credit", year as i64, "delete", None, None);
        }
        Ok(())
    }

    // --- Child credit ---

    pub fn create_child(&self, credit: &mut TaxChildCredit) -> Result<(), DomainError> {
        if credit.year < 2000 || credit.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        if credit.child_order < 1 || credit.child_order > 3 {
            return Err(DomainError::InvalidInput);
        }
        if credit.months_claimed < 1 || credit.months_claimed > 12 {
            return Err(DomainError::InvalidInput);
        }
        self.child_repo.create(credit)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_child_credit", credit.id, "create", None, None);
        }
        Ok(())
    }

    pub fn update_child(&self, credit: &mut TaxChildCredit) -> Result<(), DomainError> {
        if credit.year < 2000 || credit.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        if credit.child_order < 1 || credit.child_order > 3 {
            return Err(DomainError::InvalidInput);
        }
        if credit.months_claimed < 1 || credit.months_claimed > 12 {
            return Err(DomainError::InvalidInput);
        }
        self.child_repo.update(credit)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_child_credit", credit.id, "update", None, None);
        }
        Ok(())
    }

    pub fn delete_child(&self, id: i64) -> Result<(), DomainError> {
        self.child_repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_child_credit", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn list_children(&self, year: i32) -> Result<Vec<TaxChildCredit>, DomainError> {
        self.child_repo.list_by_year(year)
    }

    // --- Personal credits ---

    pub fn upsert_personal(&self, credits: &mut TaxPersonalCredits) -> Result<(), DomainError> {
        if credits.year < 2000 || credits.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        self.personal_repo.upsert(credits)?;
        if let Some(ref audit) = self.audit {
            audit.log(
                "tax_personal_credits",
                credits.year as i64,
                "upsert",
                None,
                None,
            );
        }
        Ok(())
    }

    pub fn get_personal(&self, year: i32) -> Result<TaxPersonalCredits, DomainError> {
        self.personal_repo.get_by_year(year)
    }

    // --- Deduction ---

    pub fn create_deduction(&self, ded: &mut TaxDeduction) -> Result<(), DomainError> {
        if ded.year < 2000 || ded.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        self.deduction_repo.create(ded)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_deduction", ded.id, "create", None, None);
        }
        Ok(())
    }

    pub fn update_deduction(&self, ded: &mut TaxDeduction) -> Result<(), DomainError> {
        if ded.year < 2000 || ded.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        self.deduction_repo.update(ded)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_deduction", ded.id, "update", None, None);
        }
        Ok(())
    }

    pub fn delete_deduction(&self, id: i64) -> Result<(), DomainError> {
        self.deduction_repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_deduction", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_deduction(&self, id: i64) -> Result<TaxDeduction, DomainError> {
        self.deduction_repo.get_by_id(id)
    }

    pub fn list_deductions(&self, year: i32) -> Result<Vec<TaxDeduction>, DomainError> {
        self.deduction_repo.list_by_year(year)
    }
}
