use std::fmt;
use std::ops::{Add, AddAssign, Neg, Sub, SubAssign};

/// Monetary value in the smallest currency unit (halere/cents).
/// For example, 100 CZK is stored as 10000 (100 * 100).
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Default)]
pub struct Amount(i64);

impl Amount {
    /// Zero amount.
    pub const ZERO: Amount = Amount(0);

    /// Create an Amount from whole units and fractional units.
    /// For CZK: crowns and halere. For EUR/USD: dollars/euros and cents.
    pub const fn new(whole: i64, fraction: i64) -> Self {
        Amount(whole * 100 + fraction)
    }

    /// Create an Amount from raw halere/cents value.
    pub const fn from_halere(halere: i64) -> Self {
        Amount(halere)
    }

    /// Convert a float64 value to Amount by rounding to nearest cent/haler.
    pub fn from_float(f: f64) -> Self {
        Amount((f * 100.0).round() as i64)
    }

    /// Return the raw halere/cents value.
    pub fn halere(self) -> i64 {
        self.0
    }

    /// Return the amount as f64 in whole currency units (e.g. CZK).
    pub fn to_czk(self) -> f64 {
        self.0 as f64 / 100.0
    }

    /// Multiply by a float factor, rounded to nearest cent/haler.
    pub fn multiply(self, factor: f64) -> Self {
        Amount((self.0 as f64 * factor).round() as i64)
    }

    /// Returns true if the amount is zero.
    pub fn is_zero(self) -> bool {
        self.0 == 0
    }

    /// Returns true if the amount is negative.
    pub fn is_negative(self) -> bool {
        self.0 < 0
    }
}

impl Add for Amount {
    type Output = Self;
    fn add(self, rhs: Self) -> Self {
        Amount(self.0 + rhs.0)
    }
}

impl AddAssign for Amount {
    fn add_assign(&mut self, rhs: Self) {
        self.0 += rhs.0;
    }
}

impl Sub for Amount {
    type Output = Self;
    fn sub(self, rhs: Self) -> Self {
        Amount(self.0 - rhs.0)
    }
}

impl SubAssign for Amount {
    fn sub_assign(&mut self, rhs: Self) {
        self.0 -= rhs.0;
    }
}

impl Neg for Amount {
    type Output = Self;
    fn neg(self) -> Self {
        Amount(-self.0)
    }
}

impl fmt::Display for Amount {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let whole = self.0 / 100;
        let fraction = (self.0 % 100).abs();
        if self.0 < 0 && whole == 0 {
            write!(f, "-{whole}.{fraction:02}")
        } else {
            write!(f, "{whole}.{fraction:02}")
        }
    }
}

impl From<i64> for Amount {
    fn from(halere: i64) -> Self {
        Amount(halere)
    }
}

impl From<Amount> for i64 {
    fn from(a: Amount) -> i64 {
        a.0
    }
}

/// Currency code constants.
pub const CURRENCY_CZK: &str = "CZK";
pub const CURRENCY_EUR: &str = "EUR";
pub const CURRENCY_USD: &str = "USD";

#[cfg(test)]
mod tests {
    use super::*;

    // -- new() tests --

    #[test]
    fn test_new() {
        assert_eq!(Amount::new(100, 50), Amount::from_halere(10050));
        assert_eq!(Amount::new(0, 0), Amount::from_halere(0));
        assert_eq!(Amount::new(1, 0), Amount::from_halere(100));
        assert_eq!(Amount::new(0, 99), Amount::from_halere(99));
        assert_eq!(Amount::new(-1, -50), Amount::from_halere(-150));
    }

    // -- from_float() tests --

    #[test]
    fn test_from_float() {
        assert_eq!(Amount::from_float(100.50), Amount::from_halere(10050));
        assert_eq!(Amount::from_float(0.0), Amount::from_halere(0));
        assert_eq!(Amount::from_float(-25.50), Amount::from_halere(-2550));
        assert_eq!(Amount::from_float(0.005), Amount::from_halere(1)); // rounds up
        assert_eq!(Amount::from_float(0.004), Amount::from_halere(0)); // rounds down
        assert_eq!(Amount::from_float(-0.99), Amount::from_halere(-99));
    }

    // -- from_halere() tests --

    #[test]
    fn test_from_halere() {
        assert_eq!(Amount::from_halere(10050).halere(), 10050);
        assert_eq!(Amount::from_halere(0), Amount::ZERO);
        assert_eq!(Amount::from_halere(-99).halere(), -99);
    }

    // -- to_czk() tests --

    #[test]
    fn test_to_czk() {
        assert_eq!(Amount::from_halere(10050).to_czk(), 100.50);
        assert_eq!(Amount::from_halere(0).to_czk(), 0.0);
        assert_eq!(Amount::from_halere(-2550).to_czk(), -25.50);
    }

    // -- Display formatting tests --

    #[test]
    fn test_display_positive() {
        assert_eq!(Amount::from_halere(10050).to_string(), "100.50");
        assert_eq!(Amount::from_halere(100).to_string(), "1.00");
        assert_eq!(Amount::from_halere(1).to_string(), "0.01");
        assert_eq!(Amount::from_halere(10).to_string(), "0.10");
    }

    #[test]
    fn test_display_zero() {
        assert_eq!(Amount::from_halere(0).to_string(), "0.00");
    }

    #[test]
    fn test_display_negative() {
        assert_eq!(Amount::from_halere(-2550).to_string(), "-25.50");
        assert_eq!(Amount::from_halere(-100).to_string(), "-1.00");
    }

    #[test]
    fn test_display_negative_fraction_only() {
        // The special -0.XX case where whole part is 0 but amount is negative
        assert_eq!(Amount::from_halere(-99).to_string(), "-0.99");
        assert_eq!(Amount::from_halere(-1).to_string(), "-0.01");
        assert_eq!(Amount::from_halere(-50).to_string(), "-0.50");
    }

    // -- Add tests --

    #[test]
    fn test_add() {
        assert_eq!(Amount::from_halere(100) + Amount::from_halere(200), Amount::from_halere(300));
        assert_eq!(Amount::from_halere(100) + Amount::from_halere(-50), Amount::from_halere(50));
        assert_eq!(
            Amount::from_halere(-100) + Amount::from_halere(-200),
            Amount::from_halere(-300)
        );
    }

    #[test]
    fn test_add_assign() {
        let mut a = Amount::from_halere(100);
        a += Amount::from_halere(50);
        assert_eq!(a, Amount::from_halere(150));
    }

    // -- Sub tests --

    #[test]
    fn test_sub() {
        assert_eq!(Amount::from_halere(300) - Amount::from_halere(100), Amount::from_halere(200));
        assert_eq!(Amount::from_halere(100) - Amount::from_halere(200), Amount::from_halere(-100));
    }

    #[test]
    fn test_sub_assign() {
        let mut a = Amount::from_halere(300);
        a -= Amount::from_halere(100);
        assert_eq!(a, Amount::from_halere(200));
    }

    // -- Neg tests --

    #[test]
    fn test_neg() {
        assert_eq!(-Amount::from_halere(100), Amount::from_halere(-100));
        assert_eq!(-Amount::from_halere(-100), Amount::from_halere(100));
        assert_eq!(-Amount::from_halere(0), Amount::from_halere(0));
    }

    // -- multiply() tests --

    #[test]
    fn test_multiply() {
        assert_eq!(Amount::from_halere(10000).multiply(0.21), Amount::from_halere(2100));
        assert_eq!(Amount::from_halere(10000).multiply(0.0), Amount::from_halere(0));
        assert_eq!(Amount::from_halere(10000).multiply(1.0), Amount::from_halere(10000));
        assert_eq!(Amount::from_halere(10000).multiply(2.5), Amount::from_halere(25000));
        assert_eq!(Amount::from_halere(333).multiply(0.21), Amount::from_halere(70)); // 69.93 rounds to 70
        assert_eq!(Amount::from_halere(-10000).multiply(0.21), Amount::from_halere(-2100));
    }

    // -- is_zero() / is_negative() tests --

    #[test]
    fn test_is_zero() {
        assert!(Amount::from_halere(0).is_zero());
        assert!(Amount::ZERO.is_zero());
        assert!(!Amount::from_halere(1).is_zero());
        assert!(!Amount::from_halere(-1).is_zero());
    }

    #[test]
    fn test_is_negative() {
        assert!(Amount::from_halere(-1).is_negative());
        assert!(Amount::from_halere(-10000).is_negative());
        assert!(!Amount::from_halere(0).is_negative());
        assert!(!Amount::from_halere(1).is_negative());
    }

    // -- Ord comparison tests --

    #[test]
    fn test_ord() {
        assert!(Amount::from_halere(100) > Amount::from_halere(50));
        assert!(Amount::from_halere(-100) < Amount::from_halere(50));
        assert!(Amount::from_halere(100) >= Amount::from_halere(100));
        assert!(Amount::from_halere(0) <= Amount::from_halere(0));

        let mut amounts = vec![
            Amount::from_halere(300),
            Amount::from_halere(100),
            Amount::from_halere(200),
            Amount::from_halere(-50),
        ];
        amounts.sort();
        assert_eq!(
            amounts,
            vec![
                Amount::from_halere(-50),
                Amount::from_halere(100),
                Amount::from_halere(200),
                Amount::from_halere(300)
            ]
        );
    }

    // -- Default tests --

    #[test]
    fn test_default() {
        assert_eq!(Amount::default(), Amount::ZERO);
    }

    // -- Edge cases --

    #[test]
    fn test_edge_cases_large_values() {
        let large = Amount::from_halere(i64::MAX / 2);
        assert_eq!(large.halere(), i64::MAX / 2);

        let neg_large = Amount::from_halere(i64::MIN / 2);
        assert_eq!(neg_large.halere(), i64::MIN / 2);

        // Reasonable large additions
        let a = Amount::from_halere(1_000_000_000_000); // 10 billion CZK
        let b = Amount::from_halere(2_000_000_000_000);
        assert_eq!((a + b).halere(), 3_000_000_000_000);
    }

    // -- Copy semantics --

    #[test]
    fn test_copy_semantics() {
        let a = Amount::from_halere(100);
        let b = a; // Copy
        assert_eq!(a, b);
        assert_eq!(a.halere(), 100);
    }

    // -- Hash consistency --

    #[test]
    fn test_hash_consistency() {
        use std::collections::HashSet;
        let mut set = HashSet::new();
        set.insert(Amount::from_halere(100));
        set.insert(Amount::from_halere(100));
        assert_eq!(set.len(), 1);
        set.insert(Amount::from_halere(200));
        assert_eq!(set.len(), 2);
    }

    // -- From trait conversions --

    #[test]
    fn test_from_i64() {
        let a: Amount = 12345_i64.into();
        assert_eq!(a.halere(), 12345);
    }

    #[test]
    fn test_into_i64() {
        let a = Amount::from_halere(12345);
        let v: i64 = a.into();
        assert_eq!(v, 12345);
    }
}
