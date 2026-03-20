use chrono::NaiveDateTime;
use std::fmt;

use crate::amount::Amount;

/// Spouse tax credit for a given year.
/// At most one record per year. Credit applies if spouse income < 68,000 CZK.
#[derive(Debug, Clone)]
pub struct TaxSpouseCredit {
    pub id: i64,
    pub year: i32,
    pub spouse_name: String,
    pub spouse_birth_number: String,
    pub spouse_income: Amount,
    /// ZTP/P holder -> doubled credit.
    pub spouse_ztp: bool,
    pub months_claimed: i32,
    pub credit_amount: Amount,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// Child tax benefit entry for a given year.
/// Multiple children per year are supported.
#[derive(Debug, Clone)]
pub struct TaxChildCredit {
    pub id: i64,
    pub year: i32,
    pub child_name: String,
    pub birth_number: String,
    /// 1, 2, 3 (3 = 3rd and subsequent).
    pub child_order: i32,
    pub months_claimed: i32,
    /// ZTP/P holder -> doubled benefit.
    pub ztp: bool,
    pub credit_amount: Amount,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// Personal tax credits (student, disability) for a year.
/// At most one record per year.
#[derive(Debug, Clone)]
pub struct TaxPersonalCredits {
    pub year: i32,
    pub is_student: bool,
    pub student_months: i32,
    /// 0=none, 1=1st/2nd degree, 2=3rd degree, 3=ZTP/P holder.
    pub disability_level: i32,
    pub credit_student: Amount,
    pub credit_disability: Amount,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// Deduction category (nezdanitelne casti zakladu dane).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum DeductionCategory {
    Mortgage,
    LifeInsurance,
    Pension,
    Donation,
    UnionDues,
}

impl fmt::Display for DeductionCategory {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            DeductionCategory::Mortgage => write!(f, "mortgage"),
            DeductionCategory::LifeInsurance => write!(f, "life_insurance"),
            DeductionCategory::Pension => write!(f, "pension"),
            DeductionCategory::Donation => write!(f, "donation"),
            DeductionCategory::UnionDues => write!(f, "union_dues"),
        }
    }
}

/// A single tax deduction entry for a year.
/// Multiple entries per category per year are supported.
#[derive(Debug, Clone)]
pub struct TaxDeduction {
    pub id: i64,
    pub year: i32,
    pub category: DeductionCategory,
    pub description: String,
    pub claimed_amount: Amount,
    pub max_amount: Amount,
    pub allowed_amount: Amount,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// A proof document linked to a deduction.
#[derive(Debug, Clone)]
pub struct TaxDeductionDocument {
    pub id: i64,
    pub tax_deduction_id: i64,
    pub filename: String,
    pub content_type: String,
    pub storage_path: String,
    pub size: i64,
    pub extracted_amount: Amount,
    pub confidence: f64,
    pub created_at: NaiveDateTime,
    pub deleted_at: Option<NaiveDateTime>,
}
