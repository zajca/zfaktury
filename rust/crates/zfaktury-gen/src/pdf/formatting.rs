//! Czech number and currency formatting utilities for PDF generation.

use zfaktury_domain::Amount;

/// Non-breaking space used as thousands separator in Czech locale.
const NBSP: char = '\u{a0}';

/// Format an Amount in Czech locale: "1 234,56" (with non-breaking spaces).
///
/// Uses comma as decimal separator and non-breaking space as thousands separator.
/// Always shows 2 decimal places.
pub fn format_czech_amount(amount: Amount) -> String {
    let halere = amount.halere();
    let is_negative = halere < 0;
    let abs_halere = halere.unsigned_abs();
    let whole = abs_halere / 100;
    let fraction = abs_halere % 100;

    let whole_str = format_with_thousands(whole);

    if is_negative {
        format!("-{},{:02}", whole_str, fraction)
    } else {
        format!("{},{:02}", whole_str, fraction)
    }
}

/// Format an Amount in Czech locale with CZK suffix: "1 234,56 CZK".
pub fn format_czech_amount_czk(amount: Amount) -> String {
    format!("{} CZK", format_czech_amount(amount))
}

/// Format a quantity Amount for display (e.g., "1,00" or "12,50").
pub fn format_quantity(amount: Amount) -> String {
    let halere = amount.halere();
    let whole = halere / 100;
    let fraction = (halere % 100).unsigned_abs();

    if fraction == 0 {
        format!("{}", whole)
    } else {
        format!("{},{:02}", whole, fraction)
    }
}

/// Insert non-breaking spaces as thousands separators.
fn format_with_thousands(n: u64) -> String {
    let s = n.to_string();
    let len = s.len();
    if len <= 3 {
        return s;
    }

    let mut result = String::with_capacity(len + len / 3);
    for (i, ch) in s.chars().enumerate() {
        if i > 0 && (len - i).is_multiple_of(3) {
            result.push(NBSP);
        }
        result.push(ch);
    }
    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_format_czech_amount_basic() {
        assert_eq!(format_czech_amount(Amount::new(100, 0)), "100,00");
        assert_eq!(format_czech_amount(Amount::new(0, 50)), "0,50");
        assert_eq!(format_czech_amount(Amount::new(0, 0)), "0,00");
    }

    #[test]
    fn test_format_czech_amount_thousands() {
        assert_eq!(format_czech_amount(Amount::new(1234, 56)), "1\u{a0}234,56");
        assert_eq!(format_czech_amount(Amount::new(12345, 0)), "12\u{a0}345,00");
        assert_eq!(
            format_czech_amount(Amount::new(123456, 78)),
            "123\u{a0}456,78"
        );
        assert_eq!(
            format_czech_amount(Amount::new(1_000_000, 0)),
            "1\u{a0}000\u{a0}000,00"
        );
    }

    #[test]
    fn test_format_czech_amount_negative() {
        assert_eq!(format_czech_amount(Amount::new(-100, 0)), "-100,00");
        // Amount::new(-1234, -56) = from_halere(-123456-56) -- but let's test with from_halere
        assert_eq!(
            format_czech_amount(Amount::from_halere(-123456)),
            "-1\u{a0}234,56"
        );
    }

    #[test]
    fn test_format_czech_amount_czk() {
        assert_eq!(
            format_czech_amount_czk(Amount::new(15000, 0)),
            "15\u{a0}000,00 CZK"
        );
    }

    #[test]
    fn test_format_quantity_whole() {
        assert_eq!(format_quantity(Amount::new(1, 0)), "1");
        assert_eq!(format_quantity(Amount::new(12, 0)), "12");
        assert_eq!(format_quantity(Amount::new(100, 0)), "100");
    }

    #[test]
    fn test_format_quantity_fractional() {
        assert_eq!(format_quantity(Amount::new(1, 50)), "1,50");
        assert_eq!(format_quantity(Amount::new(0, 25)), "0,25");
    }

    #[test]
    fn test_format_with_thousands() {
        assert_eq!(format_with_thousands(0), "0");
        assert_eq!(format_with_thousands(1), "1");
        assert_eq!(format_with_thousands(999), "999");
        assert_eq!(format_with_thousands(1000), "1\u{a0}000");
        assert_eq!(format_with_thousands(1234567), "1\u{a0}234\u{a0}567");
    }
}
