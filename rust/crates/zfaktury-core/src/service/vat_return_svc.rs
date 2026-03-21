use std::sync::Arc;

use chrono::NaiveDate;
use zfaktury_domain::{DomainError, FilingStatus, FilingType, TaxPeriod, VATReturn};

use super::audit_svc::AuditService;
use crate::repository::traits::{ExpenseRepo, InvoiceRepo, SettingsRepo, VATReturnRepo};

/// Service for VAT return management.
#[allow(dead_code)]
pub struct VATReturnService {
    repo: Arc<dyn VATReturnRepo + Send + Sync>,
    invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
    expense_repo: Arc<dyn ExpenseRepo + Send + Sync>,
    settings_repo: Arc<dyn SettingsRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl VATReturnService {
    pub fn new(
        repo: Arc<dyn VATReturnRepo + Send + Sync>,
        invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
        expense_repo: Arc<dyn ExpenseRepo + Send + Sync>,
        settings_repo: Arc<dyn SettingsRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            invoice_repo,
            expense_repo,
            settings_repo,
            audit,
        }
    }

    pub fn create(&self, vr: &mut VATReturn) -> Result<(), DomainError> {
        if vr.period.year < 2000 || vr.period.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        if vr.period.month == 0 && vr.period.quarter == 0 {
            return Err(DomainError::InvalidInput);
        }
        if vr.filing_type == FilingType::Regular
            && let Ok(_existing) = self.repo.get_by_period(
                vr.period.year,
                vr.period.month,
                vr.period.quarter,
                &vr.filing_type.to_string(),
            )
        {
            return Err(DomainError::FilingAlreadyExists);
        }
        if vr.status == FilingStatus::Draft {
            // default
        }
        self.repo.create(vr)?;
        if let Some(ref audit) = self.audit {
            audit.log("vat_return", vr.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<VATReturn, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list(&self, year: i32) -> Result<Vec<VATReturn>, DomainError> {
        let y = if year == 0 {
            {
                use chrono::Datelike;
                chrono::Local::now().date_naive().year()
            }
        } else {
            year
        };
        self.repo.list(y)
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let vr = self.repo.get_by_id(id)?;
        if vr.status == FilingStatus::Filed {
            return Err(DomainError::FilingAlreadyFiled);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("vat_return", id, "delete", None, None);
        }
        Ok(())
    }

    /// Recalculates VAT return from invoices and expenses for the period.
    pub fn recalculate(&self, id: i64) -> Result<VATReturn, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let mut vr = self.repo.get_by_id(id)?;
        if vr.status == FilingStatus::Filed {
            return Err(DomainError::FilingAlreadyFiled);
        }

        let (date_from, date_to) = period_date_range(&vr.period);

        let invoices = self.invoice_repo.list(&zfaktury_domain::InvoiceFilter {
            date_from: Some(date_from),
            date_to: Some(date_to),
            limit: 10000,
            ..Default::default()
        })?;

        let expenses = self.expense_repo.list(&zfaktury_domain::ExpenseFilter {
            date_from: Some(date_from),
            date_to: Some(date_to),
            limit: 10000,
            ..Default::default()
        })?;

        // Build calc inputs and calculate using crate::calc::vat.
        use crate::calc::vat::{
            VATExpenseInput, VATInvoiceInput, VATItemInput, calculate_vat_return,
        };
        use zfaktury_domain::{Amount, InvoiceStatus, InvoiceType};

        let mut calc_invoices = Vec::new();
        let mut invoice_ids = Vec::new();

        for inv in &invoices.0 {
            let delivery = inv.delivery_date;
            if delivery < date_from || delivery > date_to {
                continue;
            }
            if !matches!(
                inv.status,
                InvoiceStatus::Sent | InvoiceStatus::Paid | InvoiceStatus::Overdue
            ) {
                continue;
            }
            if inv.invoice_type == InvoiceType::Proforma {
                continue;
            }

            invoice_ids.push(inv.id);
            let full_inv = self.invoice_repo.get_by_id(inv.id)?;
            let mut items = Vec::new();
            for item in &full_inv.items {
                let base =
                    Amount::from_halere(item.quantity.halere() * item.unit_price.halere() / 100);
                items.push(VATItemInput {
                    base,
                    vat_amount: item.vat_amount,
                    vat_rate_percent: item.vat_rate_percent,
                });
            }
            calc_invoices.push(VATInvoiceInput {
                is_credit_note: full_inv.invoice_type == InvoiceType::CreditNote,
                items,
            });
        }

        let mut calc_expenses = Vec::new();
        let mut expense_ids = Vec::new();

        for exp in &expenses.0 {
            if !exp.is_tax_deductible {
                continue;
            }
            if exp.issue_date < date_from || exp.issue_date > date_to {
                continue;
            }
            expense_ids.push(exp.id);
            calc_expenses.push(VATExpenseInput {
                amount: exp.amount,
                vat_amount: exp.vat_amount,
                vat_rate_percent: exp.vat_rate_percent,
                business_percent: exp.business_percent,
            });
        }

        let result = calculate_vat_return(&calc_invoices, &calc_expenses);

        vr.output_vat_base_21 = result.output_vat_base_21;
        vr.output_vat_amount_21 = result.output_vat_amount_21;
        vr.output_vat_base_12 = result.output_vat_base_12;
        vr.output_vat_amount_12 = result.output_vat_amount_12;
        vr.input_vat_base_21 = result.input_vat_base_21;
        vr.input_vat_amount_21 = result.input_vat_amount_21;
        vr.input_vat_base_12 = result.input_vat_base_12;
        vr.input_vat_amount_12 = result.input_vat_amount_12;
        vr.total_output_vat = result.total_output_vat;
        vr.total_input_vat = result.total_input_vat;
        vr.net_vat = result.net_vat;

        self.repo.update(&mut vr)?;
        self.repo.link_invoices(vr.id, &invoice_ids)?;
        self.repo.link_expenses(vr.id, &expense_ids)?;

        if let Some(ref audit) = self.audit {
            audit.log("vat_return", id, "recalculate", None, None);
        }
        Ok(vr)
    }

    /// Marks a VAT return as filed.
    pub fn mark_filed(&self, id: i64) -> Result<VATReturn, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let mut vr = self.repo.get_by_id(id)?;
        if vr.status == FilingStatus::Filed {
            return Err(DomainError::FilingAlreadyFiled);
        }
        vr.status = FilingStatus::Filed;
        vr.filed_at = Some(chrono::Local::now().naive_local());
        self.repo.update(&mut vr)?;
        if let Some(ref audit) = self.audit {
            audit.log("vat_return", id, "mark_filed", None, None);
        }
        Ok(vr)
    }
}

/// Returns the start and end dates for a tax period.
pub fn period_date_range(p: &TaxPeriod) -> (NaiveDate, NaiveDate) {
    if p.month > 0 {
        let from = NaiveDate::from_ymd_opt(p.year, p.month as u32, 1).unwrap();
        let to = if p.month == 12 {
            NaiveDate::from_ymd_opt(p.year + 1, 1, 1).unwrap() - chrono::Duration::days(1)
        } else {
            NaiveDate::from_ymd_opt(p.year, p.month as u32 + 1, 1).unwrap()
                - chrono::Duration::days(1)
        };
        (from, to)
    } else {
        let start_month = ((p.quarter - 1) * 3 + 1) as u32;
        let from = NaiveDate::from_ymd_opt(p.year, start_month, 1).unwrap();
        let to = if start_month + 3 > 12 {
            NaiveDate::from_ymd_opt(p.year + 1, 1, 1).unwrap() - chrono::Duration::days(1)
        } else {
            NaiveDate::from_ymd_opt(p.year, start_month + 3, 1).unwrap() - chrono::Duration::days(1)
        };
        (from, to)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_period_date_range_month() {
        let p = TaxPeriod {
            year: 2024,
            month: 3,
            quarter: 0,
        };
        let (from, to) = period_date_range(&p);
        assert_eq!(from, NaiveDate::from_ymd_opt(2024, 3, 1).unwrap());
        assert_eq!(to, NaiveDate::from_ymd_opt(2024, 3, 31).unwrap());
    }

    #[test]
    fn test_period_date_range_quarter() {
        let p = TaxPeriod {
            year: 2024,
            month: 0,
            quarter: 2,
        };
        let (from, to) = period_date_range(&p);
        assert_eq!(from, NaiveDate::from_ymd_opt(2024, 4, 1).unwrap());
        assert_eq!(to, NaiveDate::from_ymd_opt(2024, 6, 30).unwrap());
    }

    #[test]
    fn test_period_date_range_december() {
        let p = TaxPeriod {
            year: 2024,
            month: 12,
            quarter: 0,
        };
        let (from, to) = period_date_range(&p);
        assert_eq!(from, NaiveDate::from_ymd_opt(2024, 12, 1).unwrap());
        assert_eq!(to, NaiveDate::from_ymd_opt(2024, 12, 31).unwrap());
    }
}
