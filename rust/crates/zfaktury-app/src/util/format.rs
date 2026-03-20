use chrono::NaiveDate;
use zfaktury_domain::Amount;

/// Format an Amount as Czech currency: "1 234,56 Kc".
/// Uses non-breaking spaces for thousands separators.
pub fn format_amount(amount: Amount) -> String {
    let halere = amount.halere();
    let negative = halere < 0;
    let abs = halere.unsigned_abs();
    let whole = abs / 100;
    let frac = abs % 100;

    let whole_str = format_thousands(whole);

    if negative {
        format!("-{whole_str},{frac:02} Kc")
    } else {
        format!("{whole_str},{frac:02} Kc")
    }
}

/// Format an Amount as a plain number with Czech comma: "1 234,56".
pub fn format_number(amount: Amount) -> String {
    let halere = amount.halere();
    let negative = halere < 0;
    let abs = halere.unsigned_abs();
    let whole = abs / 100;
    let frac = abs % 100;

    let whole_str = format_thousands(whole);

    if negative {
        format!("-{whole_str},{frac:02}")
    } else {
        format!("{whole_str},{frac:02}")
    }
}

/// Format a NaiveDate in Czech style: "1. 3. 2024".
pub fn format_date(date: NaiveDate) -> String {
    format!(
        "{}. {}. {}",
        date.format("%-d"),
        date.format("%-m"),
        date.format("%Y")
    )
}

/// Add thousands separators (space) to an unsigned integer.
fn format_thousands(n: u64) -> String {
    let s = n.to_string();
    let len = s.len();
    if len <= 3 {
        return s;
    }

    let mut result = String::with_capacity(len + len / 3);
    for (i, ch) in s.chars().enumerate() {
        if i > 0 && (len - i) % 3 == 0 {
            result.push('\u{00A0}'); // non-breaking space
        }
        result.push(ch);
    }
    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_format_amount_basic() {
        assert_eq!(format_amount(Amount::from_halere(10050)), "100,50 Kc");
        assert_eq!(format_amount(Amount::ZERO), "0,00 Kc");
        assert_eq!(format_amount(Amount::from_halere(-2550)), "-25,50 Kc");
    }

    #[test]
    fn test_format_amount_thousands() {
        let amt = Amount::from_halere(123456789);
        let s = format_amount(amt);
        // 1234567.89 -> "1 234 567,89 Kc" (with non-breaking spaces)
        assert!(s.contains("567,89 Kc"));
    }

    #[test]
    fn test_format_date() {
        let d = NaiveDate::from_ymd_opt(2024, 3, 1).unwrap();
        assert_eq!(format_date(d), "1. 3. 2024");

        let d2 = NaiveDate::from_ymd_opt(2024, 12, 25).unwrap();
        assert_eq!(format_date(d2), "25. 12. 2024");
    }
}
