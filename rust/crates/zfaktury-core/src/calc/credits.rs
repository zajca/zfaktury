use zfaktury_domain::Amount;

use super::constants::TaxYearConstants;

/// Computes the spouse tax credit.
///
/// Returns zero if spouse income is at or above the income limit.
/// The credit is proportional to months claimed (out of 12).
/// If `spouse_ztp` is true, the credit is doubled.
pub fn compute_spouse_credit(
    spouse_income: Amount,
    months_claimed: i32,
    spouse_ztp: bool,
    constants: &TaxYearConstants,
) -> Amount {
    if spouse_income >= constants.spouse_income_limit {
        return Amount::ZERO;
    }
    let mut credit = constants
        .spouse_credit
        .multiply(months_claimed as f64 / 12.0);
    if spouse_ztp {
        credit = credit.multiply(2.0);
    }
    credit
}

/// Computes personal disability and student credits.
///
/// `disability_level`: 0=none, 1=first/second degree, 2=third degree, 3=ZTP/P holder.
///
/// Returns `(disability_credit, student_credit)`.
pub fn compute_personal_credits(
    disability_level: i32,
    is_student: bool,
    student_months: i32,
    constants: &TaxYearConstants,
) -> (Amount, Amount) {
    let disability = match disability_level {
        1 => constants.disability_credit_1,
        2 => constants.disability_credit_3,
        3 => constants.disability_ztpp,
        _ => Amount::ZERO,
    };

    let student = if is_student && student_months > 0 {
        constants
            .student_credit
            .multiply(student_months as f64 / 12.0)
    } else {
        Amount::ZERO
    };

    (disability, student)
}

/// Input describing a single child for benefit computation.
pub struct ChildCreditInput {
    /// Child order: 1, 2, or 3+ (any value >= 3 is treated as 3+).
    pub child_order: i32,
    /// Number of months claimed.
    pub months_claimed: i32,
    /// Whether the child holds a ZTP card (doubles the benefit).
    pub ztp: bool,
}

/// Computes the total child benefit for all children.
pub fn compute_child_benefit(
    children: &[ChildCreditInput],
    constants: &TaxYearConstants,
) -> Amount {
    let mut total = Amount::ZERO;
    for child in children {
        let base = match child.child_order {
            1 => constants.child_benefit_1,
            2 => constants.child_benefit_2,
            _ => constants.child_benefit_3_plus,
        };
        let mut amount = base.multiply(child.months_claimed as f64 / 12.0);
        if child.ztp {
            amount = amount.multiply(2.0);
        }
        total += amount;
    }
    total
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::calc::constants::get_tax_constants;
    use rstest::rstest;

    fn constants() -> TaxYearConstants {
        get_tax_constants(2024).unwrap()
    }

    // -- Spouse credit tests --

    #[test]
    fn spouse_income_below_limit() {
        let c = constants();
        let result = compute_spouse_credit(Amount::new(50_000, 0), 12, false, &c);
        assert_eq!(result, c.spouse_credit);
    }

    #[test]
    fn spouse_income_at_limit() {
        let c = constants();
        // 68,000 CZK = the limit, should return 0
        let result = compute_spouse_credit(Amount::new(68_000, 0), 12, false, &c);
        assert_eq!(result, Amount::ZERO);
    }

    #[test]
    fn spouse_income_above_limit() {
        let c = constants();
        let result = compute_spouse_credit(Amount::new(100_000, 0), 12, false, &c);
        assert_eq!(result, Amount::ZERO);
    }

    #[test]
    fn spouse_ztp_doubles_credit() {
        let c = constants();
        let without_ztp = compute_spouse_credit(Amount::new(0, 0), 12, false, &c);
        let with_ztp = compute_spouse_credit(Amount::new(0, 0), 12, true, &c);
        assert_eq!(with_ztp, without_ztp.multiply(2.0));
    }

    #[test]
    fn spouse_proportional_months() {
        let c = constants();
        let full_year = compute_spouse_credit(Amount::new(0, 0), 12, false, &c);
        let half_year = compute_spouse_credit(Amount::new(0, 0), 6, false, &c);
        assert_eq!(half_year, full_year.multiply(0.5));
    }

    #[test]
    fn spouse_zero_months() {
        let c = constants();
        let result = compute_spouse_credit(Amount::new(0, 0), 0, false, &c);
        assert_eq!(result, Amount::ZERO);
    }

    // -- Personal credits tests --

    #[rstest]
    #[case(0, Amount::ZERO)]
    #[case(1, Amount::new(2_520, 0))]
    #[case(2, Amount::new(5_040, 0))]
    #[case(3, Amount::new(16_140, 0))]
    #[case(99, Amount::ZERO)] // invalid level -> no credit
    fn disability_levels(#[case] level: i32, #[case] expected: Amount) {
        let c = constants();
        let (disability, _) = compute_personal_credits(level, false, 0, &c);
        assert_eq!(disability, expected);
    }

    #[test]
    fn student_full_year() {
        let c = constants();
        let (_, student) = compute_personal_credits(0, true, 12, &c);
        assert_eq!(student, c.student_credit);
    }

    #[test]
    fn student_proportional_months() {
        let c = constants();
        let (_, student) = compute_personal_credits(0, true, 6, &c);
        assert_eq!(student, c.student_credit.multiply(6.0 / 12.0));
    }

    #[test]
    fn student_not_enrolled() {
        let c = constants();
        let (_, student) = compute_personal_credits(0, false, 12, &c);
        assert_eq!(student, Amount::ZERO);
    }

    #[test]
    fn student_zero_months() {
        let c = constants();
        let (_, student) = compute_personal_credits(0, true, 0, &c);
        assert_eq!(student, Amount::ZERO);
    }

    // -- Child benefit tests --

    #[test]
    fn child_first_full_year() {
        let c = constants();
        let result = compute_child_benefit(
            &[ChildCreditInput {
                child_order: 1,
                months_claimed: 12,
                ztp: false,
            }],
            &c,
        );
        assert_eq!(result, c.child_benefit_1);
    }

    #[test]
    fn child_second_full_year() {
        let c = constants();
        let result = compute_child_benefit(
            &[ChildCreditInput {
                child_order: 2,
                months_claimed: 12,
                ztp: false,
            }],
            &c,
        );
        assert_eq!(result, c.child_benefit_2);
    }

    #[test]
    fn child_third_plus_full_year() {
        let c = constants();
        let result = compute_child_benefit(
            &[ChildCreditInput {
                child_order: 3,
                months_claimed: 12,
                ztp: false,
            }],
            &c,
        );
        assert_eq!(result, c.child_benefit_3_plus);
    }

    #[test]
    fn child_higher_order_uses_3_plus() {
        let c = constants();
        let result = compute_child_benefit(
            &[ChildCreditInput {
                child_order: 5,
                months_claimed: 12,
                ztp: false,
            }],
            &c,
        );
        assert_eq!(result, c.child_benefit_3_plus);
    }

    #[test]
    fn child_ztp_doubles() {
        let c = constants();
        let without = compute_child_benefit(
            &[ChildCreditInput {
                child_order: 1,
                months_claimed: 12,
                ztp: false,
            }],
            &c,
        );
        let with = compute_child_benefit(
            &[ChildCreditInput {
                child_order: 1,
                months_claimed: 12,
                ztp: true,
            }],
            &c,
        );
        assert_eq!(with, without.multiply(2.0));
    }

    #[test]
    fn child_mixed_children() {
        let c = constants();
        let children = [
            ChildCreditInput {
                child_order: 1,
                months_claimed: 12,
                ztp: false,
            },
            ChildCreditInput {
                child_order: 2,
                months_claimed: 6,
                ztp: false,
            },
            ChildCreditInput {
                child_order: 3,
                months_claimed: 12,
                ztp: true,
            },
        ];
        let result = compute_child_benefit(&children, &c);
        let expected = c.child_benefit_1
            + c.child_benefit_2.multiply(6.0 / 12.0)
            + c.child_benefit_3_plus.multiply(2.0);
        assert_eq!(result, expected);
    }

    #[test]
    fn child_empty_list() {
        let c = constants();
        let result = compute_child_benefit(&[], &c);
        assert_eq!(result, Amount::ZERO);
    }
}
