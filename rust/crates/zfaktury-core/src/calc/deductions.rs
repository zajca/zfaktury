use std::collections::HashMap;

use zfaktury_domain::{Amount, DeductionCategory};

use super::constants::TaxYearConstants;

/// Input describing a single tax deduction claim.
pub struct DeductionInput {
    pub category: DeductionCategory,
    pub claimed_amount: Amount,
}

/// Result of computing allowed deduction amounts.
pub struct DeductionResult {
    /// Allowed amounts, parallel to the input slice.
    pub allowed_amounts: Vec<Amount>,
    /// Total allowed across all deductions.
    pub total_allowed: Amount,
}

/// Computes allowed deduction amounts with statutory caps.
///
/// Each category has a maximum cap; multiple deductions of the same category share the cap.
/// The donation (charity) cap is 15% of the tax base.
pub fn compute_deductions(
    deductions: &[DeductionInput],
    tax_base: Amount,
    constants: &TaxYearConstants,
) -> DeductionResult {
    let mut category_caps: HashMap<DeductionCategory, Amount> = HashMap::new();
    category_caps.insert(
        DeductionCategory::Mortgage,
        constants.deduction_cap_mortgage,
    );
    category_caps.insert(
        DeductionCategory::LifeInsurance,
        constants.deduction_cap_life_insurance,
    );
    category_caps.insert(DeductionCategory::Pension, constants.deduction_cap_pension);
    category_caps.insert(
        DeductionCategory::UnionDues,
        constants.deduction_cap_union_dues,
    );
    category_caps.insert(DeductionCategory::Donation, tax_base.multiply(0.15));

    let mut remaining_cap: HashMap<DeductionCategory, Amount> = category_caps.clone();

    let mut allowed_amounts = Vec::with_capacity(deductions.len());
    let mut total_allowed = Amount::ZERO;

    for d in deductions {
        let remaining = match remaining_cap.get(&d.category) {
            Some(&r) => r,
            None => {
                // Unknown category (e.g. Other): allow nothing.
                allowed_amounts.push(Amount::ZERO);
                continue;
            }
        };

        let mut allowed = d.claimed_amount;
        if allowed > remaining {
            allowed = remaining;
        }
        if allowed < Amount::ZERO {
            allowed = Amount::ZERO;
        }

        remaining_cap.insert(d.category, remaining - allowed);
        allowed_amounts.push(allowed);
        total_allowed += allowed;
    }

    DeductionResult {
        allowed_amounts,
        total_allowed,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::calc::constants::get_tax_constants;

    fn constants() -> TaxYearConstants {
        get_tax_constants(2024).unwrap()
    }

    #[test]
    fn mortgage_under_cap() {
        let c = constants();
        let result = compute_deductions(
            &[DeductionInput {
                category: DeductionCategory::Mortgage,
                claimed_amount: Amount::new(100_000, 0),
            }],
            Amount::new(1_000_000, 0),
            &c,
        );
        assert_eq!(result.allowed_amounts[0], Amount::new(100_000, 0));
        assert_eq!(result.total_allowed, Amount::new(100_000, 0));
    }

    #[test]
    fn mortgage_over_cap() {
        let c = constants();
        let result = compute_deductions(
            &[DeductionInput {
                category: DeductionCategory::Mortgage,
                claimed_amount: Amount::new(200_000, 0),
            }],
            Amount::new(1_000_000, 0),
            &c,
        );
        // Cap is 150,000 CZK
        assert_eq!(result.allowed_amounts[0], Amount::new(150_000, 0));
        assert_eq!(result.total_allowed, Amount::new(150_000, 0));
    }

    #[test]
    fn life_insurance_at_cap() {
        let c = constants();
        let result = compute_deductions(
            &[DeductionInput {
                category: DeductionCategory::LifeInsurance,
                claimed_amount: Amount::new(24_000, 0),
            }],
            Amount::new(1_000_000, 0),
            &c,
        );
        assert_eq!(result.allowed_amounts[0], Amount::new(24_000, 0));
    }

    #[test]
    fn pension_over_cap() {
        let c = constants();
        let result = compute_deductions(
            &[DeductionInput {
                category: DeductionCategory::Pension,
                claimed_amount: Amount::new(50_000, 0),
            }],
            Amount::new(1_000_000, 0),
            &c,
        );
        // Cap is 24,000 CZK
        assert_eq!(result.allowed_amounts[0], Amount::new(24_000, 0));
    }

    #[test]
    fn union_dues_cap() {
        let c = constants();
        let result = compute_deductions(
            &[DeductionInput {
                category: DeductionCategory::UnionDues,
                claimed_amount: Amount::new(5_000, 0),
            }],
            Amount::new(1_000_000, 0),
            &c,
        );
        // Cap is 3,000 CZK
        assert_eq!(result.allowed_amounts[0], Amount::new(3_000, 0));
    }

    #[test]
    fn donation_15_percent_of_tax_base() {
        let c = constants();
        let tax_base = Amount::new(1_000_000, 0);
        let result = compute_deductions(
            &[DeductionInput {
                category: DeductionCategory::Donation,
                claimed_amount: Amount::new(200_000, 0),
            }],
            tax_base,
            &c,
        );
        // 15% of 1,000,000 = 150,000
        assert_eq!(result.allowed_amounts[0], Amount::new(150_000, 0));
    }

    #[test]
    fn donation_under_15_percent() {
        let c = constants();
        let tax_base = Amount::new(1_000_000, 0);
        let result = compute_deductions(
            &[DeductionInput {
                category: DeductionCategory::Donation,
                claimed_amount: Amount::new(100_000, 0),
            }],
            tax_base,
            &c,
        );
        assert_eq!(result.allowed_amounts[0], Amount::new(100_000, 0));
    }

    #[test]
    fn multiple_same_category_share_cap() {
        let c = constants();
        let result = compute_deductions(
            &[
                DeductionInput {
                    category: DeductionCategory::Mortgage,
                    claimed_amount: Amount::new(100_000, 0),
                },
                DeductionInput {
                    category: DeductionCategory::Mortgage,
                    claimed_amount: Amount::new(100_000, 0),
                },
            ],
            Amount::new(1_000_000, 0),
            &c,
        );
        // First gets 100k, second gets only 50k (remaining of 150k cap)
        assert_eq!(result.allowed_amounts[0], Amount::new(100_000, 0));
        assert_eq!(result.allowed_amounts[1], Amount::new(50_000, 0));
        assert_eq!(result.total_allowed, Amount::new(150_000, 0));
    }

    #[test]
    fn empty_deductions() {
        let c = constants();
        let result = compute_deductions(&[], Amount::new(1_000_000, 0), &c);
        assert!(result.allowed_amounts.is_empty());
        assert_eq!(result.total_allowed, Amount::ZERO);
    }

    #[test]
    fn mixed_categories() {
        let c = constants();
        let result = compute_deductions(
            &[
                DeductionInput {
                    category: DeductionCategory::Mortgage,
                    claimed_amount: Amount::new(100_000, 0),
                },
                DeductionInput {
                    category: DeductionCategory::Pension,
                    claimed_amount: Amount::new(20_000, 0),
                },
                DeductionInput {
                    category: DeductionCategory::LifeInsurance,
                    claimed_amount: Amount::new(30_000, 0),
                },
            ],
            Amount::new(1_000_000, 0),
            &c,
        );
        assert_eq!(result.allowed_amounts[0], Amount::new(100_000, 0));
        assert_eq!(result.allowed_amounts[1], Amount::new(20_000, 0));
        // Life insurance cap is 24k, claimed 30k -> capped at 24k
        assert_eq!(result.allowed_amounts[2], Amount::new(24_000, 0));
        assert_eq!(
            result.total_allowed,
            Amount::new(100_000, 0) + Amount::new(20_000, 0) + Amount::new(24_000, 0)
        );
    }
}
