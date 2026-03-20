use std::collections::HashMap;
use zfaktury_domain::Amount;

use super::constants::TaxYearConstants;

/// Inputs needed for income tax calculation.
#[derive(Debug, Clone)]
pub struct IncomeTaxInput {
    pub total_revenue: Amount,
    pub actual_expenses: Amount,
    pub flat_rate_percent: i32,
    pub constants: TaxYearConstants,
    pub spouse_credit: Amount,
    pub disability_credit: Amount,
    pub student_credit: Amount,
    pub child_benefit: Amount,
    pub total_deductions: Amount,
    pub prepayments: Amount,
    pub capital_income_net: Amount,
    pub other_income_net: Amount,
}

/// Computed values from the income tax calculation.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IncomeTaxResult {
    pub flat_rate_amount: Amount,
    pub used_expenses: Amount,
    pub tax_base: Amount,
    pub tax_base_rounded: Amount,
    pub tax_at_15: Amount,
    pub tax_at_23: Amount,
    pub total_tax: Amount,
    pub credit_basic: Amount,
    pub total_credits: Amount,
    pub tax_after_credits: Amount,
    pub tax_after_benefit: Amount,
    pub tax_due: Amount,
}

/// Performs the full income tax calculation (steps 4-10 from Czech tax form).
pub fn calculate_income_tax(input: &IncomeTaxInput) -> IncomeTaxResult {
    // Step 4: Determine used expenses (flat rate vs actual).
    let (flat_rate_amount, used_expenses) = if input.flat_rate_percent > 0 {
        let flat_amount = input
            .total_revenue
            .multiply(input.flat_rate_percent as f64 / 100.0);
        let capped = cap_flat_rate(
            flat_amount,
            input.flat_rate_percent,
            &input.constants.flat_rate_caps,
        );
        (capped, capped)
    } else {
        (Amount::ZERO, input.actual_expenses)
    };

    // Step 5: Tax base (revenue - expenses + capital income + other income).
    let mut tax_base =
        input.total_revenue - used_expenses + input.capital_income_net + input.other_income_net;
    if tax_base.is_negative() {
        tax_base = Amount::ZERO;
    }
    let tax_base_before_deductions = tax_base;

    // Step 5b: Apply deductions (reduce tax base before rounding).
    tax_base -= input.total_deductions;
    if tax_base.is_negative() {
        tax_base = Amount::ZERO;
    }

    // Step 6: Round down to 100 CZK (10000 halere).
    let tax_base_rounded = Amount::from_halere((tax_base.halere() / 10000) * 10000);

    // Step 7: Progressive tax calculation.
    let threshold = input.constants.progressive_threshold;

    let (tax_at_15, tax_at_23) = if tax_base_rounded <= threshold {
        (tax_base_rounded.multiply(0.15), Amount::ZERO)
    } else {
        (
            threshold.multiply(0.15),
            (tax_base_rounded - threshold).multiply(0.23),
        )
    };
    let total_tax = tax_at_15 + tax_at_23;

    // Step 8: Tax credits.
    let credit_basic = input.constants.basic_credit;
    let total_credits =
        credit_basic + input.spouse_credit + input.disability_credit + input.student_credit;

    let mut tax_after_credits = total_tax - total_credits;
    if tax_after_credits.is_negative() {
        tax_after_credits = Amount::ZERO;
    }

    // Step 9: Child benefit (can go negative - it's a bonus).
    let tax_after_benefit = tax_after_credits - input.child_benefit;

    // Step 10: Prepayments (can be negative = refund).
    let tax_due = tax_after_benefit - input.prepayments;

    IncomeTaxResult {
        flat_rate_amount,
        used_expenses,
        tax_base: tax_base_before_deductions,
        tax_base_rounded,
        tax_at_15,
        tax_at_23,
        total_tax,
        credit_basic,
        total_credits,
        tax_after_credits,
        tax_after_benefit,
        tax_due,
    }
}

/// Apply flat rate cap if one exists for the given percent.
fn cap_flat_rate(amount: Amount, percent: i32, caps: &HashMap<i32, Amount>) -> Amount {
    if let Some(&cap) = caps.get(&percent)
        && amount > cap
    {
        return cap;
    }
    amount
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::calc::constants::get_tax_constants;
    use rstest::rstest;

    fn amt(whole: i64, fraction: i64) -> Amount {
        Amount::new(whole, fraction)
    }

    fn default_input(year: i32) -> IncomeTaxInput {
        IncomeTaxInput {
            total_revenue: Amount::ZERO,
            actual_expenses: Amount::ZERO,
            flat_rate_percent: 0,
            constants: get_tax_constants(year).unwrap(),
            spouse_credit: Amount::ZERO,
            disability_credit: Amount::ZERO,
            student_credit: Amount::ZERO,
            child_benefit: Amount::ZERO,
            total_deductions: Amount::ZERO,
            prepayments: Amount::ZERO,
            capital_income_net: Amount::ZERO,
            other_income_net: Amount::ZERO,
        }
    }

    #[test]
    fn test_zero_revenue_zero_tax() {
        let input = default_input(2024);
        let result = calculate_income_tax(&input);
        assert_eq!(result.tax_base, Amount::ZERO);
        assert_eq!(result.total_tax, Amount::ZERO);
        assert_eq!(result.tax_after_credits, Amount::ZERO);
        assert_eq!(result.tax_due, Amount::ZERO);
    }

    #[test]
    fn test_revenue_below_threshold_only_15_percent() {
        let mut input = default_input(2024);
        // Revenue 1,000,000 CZK, expenses 400,000 CZK -> tax base 600,000 CZK
        input.total_revenue = amt(1_000_000, 0);
        input.actual_expenses = amt(400_000, 0);

        let result = calculate_income_tax(&input);
        assert_eq!(result.tax_base, amt(600_000, 0));
        // Rounded to 100 CZK: 600,000 CZK
        assert_eq!(result.tax_base_rounded, amt(600_000, 0));
        // 600,000 * 0.15 = 90,000 CZK
        assert_eq!(result.tax_at_15, amt(90_000, 0));
        assert_eq!(result.tax_at_23, Amount::ZERO);
        assert_eq!(result.total_tax, amt(90_000, 0));
    }

    #[test]
    fn test_revenue_above_threshold_progressive() {
        let mut input = default_input(2024);
        // Revenue 3,000,000 CZK, expenses 500,000 CZK -> tax base 2,500,000 CZK
        input.total_revenue = amt(3_000_000, 0);
        input.actual_expenses = amt(500_000, 0);

        let result = calculate_income_tax(&input);
        assert_eq!(result.tax_base, amt(2_500_000, 0));
        assert_eq!(result.tax_base_rounded, amt(2_500_000, 0));

        // threshold = 1,582,812 CZK
        let threshold = amt(1_582_812, 0);
        let expected_15 = threshold.multiply(0.15);
        let expected_23 = (amt(2_500_000, 0) - threshold).multiply(0.23);
        assert_eq!(result.tax_at_15, expected_15);
        assert_eq!(result.tax_at_23, expected_23);
        assert_eq!(result.total_tax, expected_15 + expected_23);
    }

    #[test]
    fn test_flat_rate_60_percent_with_cap() {
        let mut input = default_input(2024);
        // Revenue 3,000,000 CZK, flat rate 60%
        // 60% of 3M = 1,800,000 but cap is 1,200,000
        input.total_revenue = amt(3_000_000, 0);
        input.flat_rate_percent = 60;

        let result = calculate_income_tax(&input);
        assert_eq!(result.flat_rate_amount, amt(1_200_000, 0)); // capped
        assert_eq!(result.used_expenses, amt(1_200_000, 0));
        assert_eq!(result.tax_base, amt(1_800_000, 0)); // 3M - 1.2M
    }

    #[test]
    fn test_flat_rate_60_percent_below_cap() {
        let mut input = default_input(2024);
        // Revenue 1,000,000 CZK, flat rate 60%
        // 60% of 1M = 600,000 (below 1,200,000 cap)
        input.total_revenue = amt(1_000_000, 0);
        input.flat_rate_percent = 60;

        let result = calculate_income_tax(&input);
        assert_eq!(result.flat_rate_amount, amt(600_000, 0));
        assert_eq!(result.used_expenses, amt(600_000, 0));
        assert_eq!(result.tax_base, amt(400_000, 0));
    }

    #[test]
    fn test_deductions_reduce_tax_base() {
        let mut input = default_input(2024);
        input.total_revenue = amt(1_000_000, 0);
        input.actual_expenses = amt(400_000, 0);
        // tax base = 600,000 CZK
        input.total_deductions = amt(100_000, 0);

        let result = calculate_income_tax(&input);
        // tax_base field stores pre-deduction value
        assert_eq!(result.tax_base, amt(600_000, 0));
        // Rounded base = (600,000 - 100,000) = 500,000 CZK
        assert_eq!(result.tax_base_rounded, amt(500_000, 0));
    }

    #[test]
    fn test_deductions_cannot_make_base_negative() {
        let mut input = default_input(2024);
        input.total_revenue = amt(100_000, 0);
        input.actual_expenses = amt(50_000, 0);
        // tax base = 50,000 CZK
        input.total_deductions = amt(200_000, 0); // larger than base

        let result = calculate_income_tax(&input);
        assert_eq!(result.tax_base_rounded, Amount::ZERO);
        assert_eq!(result.total_tax, Amount::ZERO);
    }

    #[test]
    fn test_credits_reduce_tax_capped_at_zero() {
        let mut input = default_input(2024);
        // Small revenue so tax is small
        input.total_revenue = amt(100_000, 0);
        input.actual_expenses = amt(50_000, 0);
        // tax base = 50,000 CZK, tax = 50,000 * 0.15 = 7,500 CZK
        // basic credit = 30,840 -> larger than tax -> capped at 0

        let result = calculate_income_tax(&input);
        assert_eq!(result.total_tax, amt(7_500, 0));
        assert_eq!(result.credit_basic, amt(30_840, 0));
        assert_eq!(result.tax_after_credits, Amount::ZERO); // capped at 0
    }

    #[test]
    fn test_child_benefit_makes_tax_negative() {
        let mut input = default_input(2024);
        input.total_revenue = amt(500_000, 0);
        input.actual_expenses = amt(200_000, 0);
        // tax base = 300,000 CZK, tax = 45,000, minus basic credit 30,840 = 14,160
        input.child_benefit = amt(30_000, 0);

        let result = calculate_income_tax(&input);
        let expected_after_credits = amt(45_000, 0) - amt(30_840, 0); // 14,160
        assert_eq!(result.tax_after_credits, expected_after_credits);
        // 14,160 - 30,000 = -15,840 (bonus)
        assert_eq!(
            result.tax_after_benefit,
            expected_after_credits - amt(30_000, 0)
        );
        assert!(result.tax_after_benefit.is_negative());
    }

    #[test]
    fn test_prepayments_positive_and_negative_tax_due() {
        let mut input = default_input(2024);
        input.total_revenue = amt(1_000_000, 0);
        input.actual_expenses = amt(400_000, 0);
        // tax base = 600,000 CZK, tax = 90,000 - 30,840 credit = 59,160

        // Prepayments less than tax -> positive tax_due
        input.prepayments = amt(50_000, 0);
        let result = calculate_income_tax(&input);
        assert_eq!(result.tax_after_credits, amt(59_160, 0));
        assert_eq!(result.tax_due, amt(59_160, 0) - amt(50_000, 0));

        // Prepayments more than tax -> negative tax_due (refund)
        input.prepayments = amt(70_000, 0);
        let result = calculate_income_tax(&input);
        assert_eq!(result.tax_due, amt(59_160, 0) - amt(70_000, 0));
        assert!(result.tax_due.is_negative());
    }

    #[test]
    fn test_rounding_to_100_czk() {
        let mut input = default_input(2024);
        // Revenue so that tax base is not round
        input.total_revenue = amt(555_555, 0);
        input.actual_expenses = amt(100_000, 0);
        // tax base = 455,555 CZK = 45,555,500 halere
        // rounded: (45555500 / 10000) * 10000 = 4555 * 10000 = 45,550,000 = 455,500 CZK

        let result = calculate_income_tax(&input);
        assert_eq!(result.tax_base, amt(455_555, 0));
        assert_eq!(result.tax_base_rounded, amt(455_500, 0));
    }

    #[rstest]
    #[case(2024)]
    #[case(2025)]
    #[case(2026)]
    fn test_basic_credit_matches_constants(#[case] year: i32) {
        let mut input = default_input(year);
        input.total_revenue = amt(1_000_000, 0);
        input.actual_expenses = amt(400_000, 0);

        let result = calculate_income_tax(&input);
        assert_eq!(
            result.credit_basic,
            get_tax_constants(year).unwrap().basic_credit
        );
    }

    #[test]
    fn test_capital_and_other_income_add_to_base() {
        let mut input = default_input(2024);
        input.total_revenue = amt(500_000, 0);
        input.actual_expenses = amt(200_000, 0);
        input.capital_income_net = amt(50_000, 0);
        input.other_income_net = amt(30_000, 0);

        let result = calculate_income_tax(&input);
        // 500,000 - 200,000 + 50,000 + 30,000 = 380,000
        assert_eq!(result.tax_base, amt(380_000, 0));
    }

    #[test]
    fn test_negative_revenue_minus_expenses_clamped_to_zero() {
        let mut input = default_input(2024);
        input.total_revenue = amt(100_000, 0);
        input.actual_expenses = amt(200_000, 0);

        let result = calculate_income_tax(&input);
        assert_eq!(result.tax_base, Amount::ZERO);
        assert_eq!(result.total_tax, Amount::ZERO);
    }

    #[test]
    fn test_all_credits_combined() {
        let mut input = default_input(2024);
        input.total_revenue = amt(2_000_000, 0);
        input.actual_expenses = amt(500_000, 0);
        input.spouse_credit = amt(24_840, 0);
        input.disability_credit = amt(2_520, 0);
        input.student_credit = amt(4_020, 0);

        let result = calculate_income_tax(&input);
        let expected_credits = amt(30_840, 0) + amt(24_840, 0) + amt(2_520, 0) + amt(4_020, 0);
        assert_eq!(result.total_credits, expected_credits);
    }

    mod proptests {
        use super::*;
        use proptest::prelude::*;

        proptest! {
            #[test]
            fn tax_base_is_never_negative(
                revenue in 0i64..500_000_000i64,
                expenses in 0i64..500_000_000i64,
            ) {
                let mut input = default_input(2024);
                input.total_revenue = Amount::from_halere(revenue);
                input.actual_expenses = Amount::from_halere(expenses);

                let result = calculate_income_tax(&input);
                prop_assert!(result.tax_base.halere() >= 0);
                prop_assert!(result.tax_base_rounded.halere() >= 0);
            }

            #[test]
            fn total_tax_equals_sum_of_brackets(
                revenue in 0i64..1_000_000_000i64,
                expenses in 0i64..500_000_000i64,
            ) {
                let mut input = default_input(2024);
                input.total_revenue = Amount::from_halere(revenue);
                input.actual_expenses = Amount::from_halere(expenses);

                let result = calculate_income_tax(&input);
                prop_assert_eq!(result.total_tax, result.tax_at_15 + result.tax_at_23);
            }

            #[test]
            fn tax_after_credits_is_non_negative(
                revenue in 0i64..1_000_000_000i64,
                expenses in 0i64..500_000_000i64,
            ) {
                let mut input = default_input(2024);
                input.total_revenue = Amount::from_halere(revenue);
                input.actual_expenses = Amount::from_halere(expenses);

                let result = calculate_income_tax(&input);
                prop_assert!(result.tax_after_credits.halere() >= 0);
            }

            #[test]
            fn tax_base_rounded_is_multiple_of_100_czk(
                revenue in 0i64..1_000_000_000i64,
                expenses in 0i64..500_000_000i64,
            ) {
                let mut input = default_input(2024);
                input.total_revenue = Amount::from_halere(revenue);
                input.actual_expenses = Amount::from_halere(expenses);

                let result = calculate_income_tax(&input);
                // 100 CZK = 10000 halere
                prop_assert_eq!(result.tax_base_rounded.halere() % 10000, 0);
            }
        }
    }
}
