//! Shared helpers for XML generation.

use zfaktury_domain::{Amount, FilingType};

/// Taxpayer identification info needed across multiple XML generators.
pub struct TaxpayerInfo {
    /// DIC without "CZ" prefix (e.g. "12345678").
    pub dic: String,
    pub first_name: String,
    pub last_name: String,
    pub street: String,
    pub house_num: String,
    pub zip: String,
    pub city: String,
    pub phone: String,
    pub email: String,
    /// Financial office code (c_ufo, 3-digit).
    pub ufo_code: String,
    /// Workplace code (c_pracufo, 4-digit).
    pub prac_ufo: String,
    /// NACE/OKEC code.
    pub okec: String,
    /// Submission date in DD.MM.YYYY format. Empty string means "today".
    pub submission_date: String,
}

/// Convert halere to whole CZK with standard math rounding.
/// Used by VAT return and control statement generators.
pub fn to_whole_czk_rounded(amount: Amount) -> i64 {
    ((amount.halere() as f64) / 100.0).round() as i64
}

/// Convert halere to whole CZK by truncation toward zero.
/// Used by income tax and social insurance generators.
pub fn to_whole_czk_truncated(amount: Amount) -> i64 {
    amount.halere() / 100
}

/// Convert domain FilingType to DPH filing code.
/// Regular -> "B", Corrective -> "O", Supplementary -> "D".
pub fn dph_filing_type_code(ft: &FilingType) -> &'static str {
    match ft {
        FilingType::Regular => "B",
        FilingType::Corrective => "O",
        FilingType::Supplementary => "D",
    }
}

/// Convert domain FilingType to control statement d_typ code.
/// Regular -> "R", Corrective -> "N", Supplementary -> "O".
pub fn control_statement_filing_type_code(ft: &FilingType) -> &'static str {
    match ft {
        FilingType::Regular => "R",
        FilingType::Corrective => "N",
        FilingType::Supplementary => "O",
    }
}

/// Convert domain FilingType to VIES filing code.
/// Regular -> "B", Corrective -> "O", Supplementary -> "N".
pub fn vies_filing_type_code(ft: &FilingType) -> &'static str {
    match ft {
        FilingType::Regular => "B",
        FilingType::Corrective => "O",
        FilingType::Supplementary => "N",
    }
}

/// Convert domain FilingType to DPFO (income tax) filing code.
/// Regular -> "B", Corrective -> "O", Supplementary -> "D".
pub fn dpfo_filing_type_code(ft: &FilingType) -> &'static str {
    match ft {
        FilingType::Regular => "B",
        FilingType::Corrective => "O",
        FilingType::Supplementary => "D",
    }
}

/// Convert domain FilingType to CSSZ (social insurance) filing code.
/// Regular -> "N" (nova), Corrective -> "O" (opravna), Supplementary -> "Z" (zmena).
pub fn cssz_filing_type_code(ft: &FilingType) -> &'static str {
    match ft {
        FilingType::Regular => "N",
        FilingType::Corrective => "O",
        FilingType::Supplementary => "Z",
    }
}

/// Strip the 2-letter country code prefix from a VAT ID (e.g. "CZ12345678" -> "12345678").
/// Returns an owned String to avoid borrow issues with temporary uppercase conversions.
pub fn strip_country_prefix(dic: &str) -> String {
    let dic = dic.trim();
    if dic.len() > 2 {
        let prefix = &dic[..2];
        if prefix.bytes().all(|b| b.is_ascii_uppercase()) {
            return dic[2..].to_string();
        }
    }
    dic.to_string()
}

/// Write the XML declaration header.
pub fn xml_header() -> &'static str {
    "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_to_whole_czk_rounded() {
        assert_eq!(to_whole_czk_rounded(Amount::from_halere(10050)), 101);
        assert_eq!(to_whole_czk_rounded(Amount::from_halere(10049)), 100);
        assert_eq!(to_whole_czk_rounded(Amount::from_halere(-10050)), -101);
        assert_eq!(to_whole_czk_rounded(Amount::from_halere(0)), 0);
    }

    #[test]
    fn test_to_whole_czk_truncated() {
        assert_eq!(to_whole_czk_truncated(Amount::from_halere(10050)), 100);
        assert_eq!(to_whole_czk_truncated(Amount::from_halere(10099)), 100);
        assert_eq!(to_whole_czk_truncated(Amount::from_halere(-10050)), -100);
        assert_eq!(to_whole_czk_truncated(Amount::from_halere(0)), 0);
    }

    #[test]
    fn test_dph_filing_type_code() {
        assert_eq!(dph_filing_type_code(&FilingType::Regular), "B");
        assert_eq!(dph_filing_type_code(&FilingType::Corrective), "O");
        assert_eq!(dph_filing_type_code(&FilingType::Supplementary), "D");
    }

    #[test]
    fn test_control_statement_filing_type_code() {
        assert_eq!(
            control_statement_filing_type_code(&FilingType::Regular),
            "R"
        );
        assert_eq!(
            control_statement_filing_type_code(&FilingType::Corrective),
            "N"
        );
        assert_eq!(
            control_statement_filing_type_code(&FilingType::Supplementary),
            "O"
        );
    }

    #[test]
    fn test_vies_filing_type_code() {
        assert_eq!(vies_filing_type_code(&FilingType::Regular), "B");
        assert_eq!(vies_filing_type_code(&FilingType::Corrective), "O");
        assert_eq!(vies_filing_type_code(&FilingType::Supplementary), "N");
    }

    #[test]
    fn test_dpfo_filing_type_code() {
        assert_eq!(dpfo_filing_type_code(&FilingType::Regular), "B");
        assert_eq!(dpfo_filing_type_code(&FilingType::Corrective), "O");
        assert_eq!(dpfo_filing_type_code(&FilingType::Supplementary), "D");
    }

    #[test]
    fn test_cssz_filing_type_code() {
        assert_eq!(cssz_filing_type_code(&FilingType::Regular), "N");
        assert_eq!(cssz_filing_type_code(&FilingType::Corrective), "O");
        assert_eq!(cssz_filing_type_code(&FilingType::Supplementary), "Z");
    }

    #[test]
    fn test_strip_country_prefix() {
        assert_eq!(strip_country_prefix("CZ12345678"), "12345678".to_string());
        assert_eq!(strip_country_prefix("DE999999999"), "999999999".to_string());
        assert_eq!(strip_country_prefix("12345678"), "12345678".to_string());
        assert_eq!(strip_country_prefix(""), "".to_string());
        assert_eq!(
            strip_country_prefix("  CZ12345678  "),
            "12345678".to_string()
        );
    }
}
