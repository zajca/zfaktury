//! DPFDP5 income tax return XML generation.
//!
//! Generates EPO XML for Czech personal income tax returns
//! (Priznani k dani z prijmu fyzickych osob).

use quick_xml::Writer;
use quick_xml::events::{BytesDecl, BytesEnd, BytesStart, BytesText, Event};
use std::collections::HashMap;
use std::io::Cursor;

use zfaktury_domain::IncomeTaxReturn;

use super::common::{dpfo_filing_type_code, to_whole_czk_truncated};
use crate::Result;

/// Generate DPFDP5 XML bytes from an IncomeTaxReturn and settings map.
///
/// Required settings keys:
/// - `financni_urad_code` - Financial office code
/// - `taxpayer_first_name`, `taxpayer_last_name`
/// - `taxpayer_birth_number`
/// - `dic` - Tax identification number
/// - `taxpayer_street`, `taxpayer_house_number`, `taxpayer_city`, `taxpayer_postal_code`
pub fn generate_income_tax_xml(
    itr: &IncomeTaxReturn,
    settings: &HashMap<String, String>,
) -> Result<Vec<u8>> {
    let filing_code = dpfo_filing_type_code(&itr.filing_type);
    let tax_base_rounded = to_whole_czk_truncated(itr.tax_base_rounded);

    let mut writer = Writer::new_with_indent(Cursor::new(Vec::new()), b' ', 2);

    // XML declaration.
    writer.write_event(Event::Decl(BytesDecl::new("1.0", Some("UTF-8"), None)))?;
    writer.write_event(Event::Text(BytesText::new("\n")))?;

    // <Pisemnost nazevSW="ZFaktury" verzeSW="1.0">
    let mut pisemnost = BytesStart::new("Pisemnost");
    pisemnost.push_attribute(("nazevSW", "ZFaktury"));
    pisemnost.push_attribute(("verzeSW", "1.0"));
    writer.write_event(Event::Start(pisemnost))?;

    // <DPFDP5 verzePis="05.01">
    let mut dpfdp5 = BytesStart::new("DPFDP5");
    dpfdp5.push_attribute(("verzePis", "05.01"));
    writer.write_event(Event::Start(dpfdp5))?;

    // <VetaD .../>
    let mut veta_d = BytesStart::new("VetaD");
    veta_d.push_attribute(("dokument", "DP5"));
    veta_d.push_attribute(("k_uladis", "DPF"));
    veta_d.push_attribute(("rok", itr.year.to_string().as_str()));
    veta_d.push_attribute(("dap_typ", filing_code));
    veta_d.push_attribute((
        "c_ufo_cil",
        get_setting(settings, "financni_urad_code").as_str(),
    ));
    veta_d.push_attribute(("pln_moc", "A"));
    veta_d.push_attribute(("audit", "N"));

    // Section 7 - business income.
    veta_d.push_attribute((
        "kc_zd7",
        to_whole_czk_truncated(itr.tax_base).to_string().as_str(),
    ));
    veta_d.push_attribute((
        "pr_zd7",
        to_whole_czk_truncated(itr.total_revenue)
            .to_string()
            .as_str(),
    ));
    veta_d.push_attribute((
        "vy_zd7",
        to_whole_czk_truncated(itr.used_expenses)
            .to_string()
            .as_str(),
    ));

    // Tax calculation.
    veta_d.push_attribute(("kc_zakldan23", tax_base_rounded.to_string().as_str()));
    veta_d.push_attribute(("kc_zakldan", tax_base_rounded.to_string().as_str()));
    veta_d.push_attribute(("kc_zdzaokr", tax_base_rounded.to_string().as_str()));
    veta_d.push_attribute((
        "da_slezap",
        to_whole_czk_truncated(itr.total_tax).to_string().as_str(),
    ));

    // Tax credits.
    veta_d.push_attribute((
        "sleva_rp",
        to_whole_czk_truncated(itr.credit_basic)
            .to_string()
            .as_str(),
    ));
    veta_d.push_attribute((
        "uhrn_slevy35ba",
        to_whole_czk_truncated(itr.total_credits)
            .to_string()
            .as_str(),
    ));
    veta_d.push_attribute((
        "da_slevy35ba",
        to_whole_czk_truncated(itr.tax_after_credits)
            .to_string()
            .as_str(),
    ));

    // Child benefit.
    veta_d.push_attribute((
        "kc_dazvyhod",
        to_whole_czk_truncated(itr.child_benefit)
            .to_string()
            .as_str(),
    ));
    veta_d.push_attribute((
        "da_slevy35c",
        to_whole_czk_truncated(itr.tax_after_benefit)
            .to_string()
            .as_str(),
    ));

    // Prepayments and result.
    veta_d.push_attribute((
        "kc_zalpred",
        to_whole_czk_truncated(itr.prepayments).to_string().as_str(),
    ));
    veta_d.push_attribute((
        "kc_zbyvpred",
        to_whole_czk_truncated(itr.tax_due).to_string().as_str(),
    ));
    writer.write_event(Event::Empty(veta_d))?;

    // <VetaP .../>
    let mut veta_p = BytesStart::new("VetaP");
    veta_p.push_attribute((
        "jmeno",
        get_setting(settings, "taxpayer_first_name").as_str(),
    ));
    veta_p.push_attribute((
        "prijmeni",
        get_setting(settings, "taxpayer_last_name").as_str(),
    ));
    veta_p.push_attribute((
        "rod_c",
        get_setting(settings, "taxpayer_birth_number").as_str(),
    ));
    veta_p.push_attribute(("dic", get_setting(settings, "dic").as_str()));
    veta_p.push_attribute(("ulice", get_setting(settings, "taxpayer_street").as_str()));
    veta_p.push_attribute((
        "c_pop",
        get_setting(settings, "taxpayer_house_number").as_str(),
    ));
    veta_p.push_attribute(("naz_obce", get_setting(settings, "taxpayer_city").as_str()));
    veta_p.push_attribute((
        "psc",
        get_setting(settings, "taxpayer_postal_code").as_str(),
    ));
    veta_p.push_attribute(("k_stat", "CZ"));
    veta_p.push_attribute(("stat", "\u{010c}ESK\u{00c1} REPUBLIKA"));
    writer.write_event(Event::Empty(veta_p))?;

    // </DPFDP5>
    writer.write_event(Event::End(BytesEnd::new("DPFDP5")))?;

    // </Pisemnost>
    writer.write_event(Event::End(BytesEnd::new("Pisemnost")))?;

    Ok(writer.into_inner().into_inner())
}

/// Get a setting value, returning empty string if missing.
fn get_setting(settings: &HashMap<String, String>, key: &str) -> String {
    settings.get(key).cloned().unwrap_or_default()
}

#[cfg(test)]
mod tests {
    use super::*;
    use zfaktury_domain::{Amount, FilingType};

    fn test_settings() -> HashMap<String, String> {
        let mut m = HashMap::new();
        m.insert("financni_urad_code".to_string(), "451".to_string());
        m.insert("taxpayer_first_name".to_string(), "Jan".to_string());
        m.insert("taxpayer_last_name".to_string(), "Novak".to_string());
        m.insert(
            "taxpayer_birth_number".to_string(),
            "8001011234".to_string(),
        );
        m.insert("dic".to_string(), "CZ12345678".to_string());
        m.insert("taxpayer_street".to_string(), "Hlavni 1".to_string());
        m.insert("taxpayer_house_number".to_string(), "1".to_string());
        m.insert("taxpayer_city".to_string(), "Praha".to_string());
        m.insert("taxpayer_postal_code".to_string(), "11000".to_string());
        m
    }

    fn test_income_tax_return() -> IncomeTaxReturn {
        let dt = chrono::NaiveDate::from_ymd_opt(2024, 1, 1)
            .unwrap()
            .and_hms_opt(0, 0, 0)
            .unwrap();
        IncomeTaxReturn {
            id: 1,
            year: 2024,
            filing_type: FilingType::Regular,
            total_revenue: Amount::new(1_000_000, 0),
            actual_expenses: Amount::new(300_000, 0),
            flat_rate_percent: 60,
            flat_rate_amount: Amount::new(600_000, 0),
            used_expenses: Amount::new(600_000, 0),
            tax_base: Amount::new(400_000, 0),
            total_deductions: Amount::ZERO,
            tax_base_rounded: Amount::new(400_000, 0),
            tax_at_15: Amount::new(60_000, 0),
            tax_at_23: Amount::ZERO,
            total_tax: Amount::new(60_000, 0),
            credit_basic: Amount::new(30_840, 0),
            credit_spouse: Amount::ZERO,
            credit_disability: Amount::ZERO,
            credit_student: Amount::ZERO,
            total_credits: Amount::new(30_840, 0),
            tax_after_credits: Amount::new(29_160, 0),
            child_benefit: Amount::ZERO,
            tax_after_benefit: Amount::new(29_160, 0),
            prepayments: Amount::new(10_000, 0),
            tax_due: Amount::new(19_160, 0),
            capital_income_gross: Amount::ZERO,
            capital_income_tax: Amount::ZERO,
            capital_income_net: Amount::ZERO,
            other_income_gross: Amount::ZERO,
            other_income_expenses: Amount::ZERO,
            other_income_exempt: Amount::ZERO,
            other_income_net: Amount::ZERO,
            xml_data: Vec::new(),
            status: zfaktury_domain::FilingStatus::Draft,
            filed_at: None,
            created_at: dt,
            updated_at: dt,
        }
    }

    #[test]
    fn test_generate_valid_xml() {
        let itr = test_income_tax_return();
        let settings = test_settings();
        let result = generate_income_tax_xml(&itr, &settings).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        // Verify valid XML.
        let mut reader = quick_xml::Reader::from_str(&xml_str);
        reader.config_mut().trim_text(true);
        let mut buf = Vec::new();
        loop {
            match reader.read_event_into(&mut buf) {
                Ok(quick_xml::events::Event::Eof) => break,
                Err(e) => panic!("XML parse error: {e}"),
                _ => {}
            }
            buf.clear();
        }

        assert!(xml_str.contains("<Pisemnost"));
        assert!(xml_str.contains("nazevSW=\"ZFaktury\""));
        assert!(xml_str.contains("<DPFDP5"));
    }

    #[test]
    fn test_amounts_whole_czk() {
        let itr = test_income_tax_return();
        let settings = test_settings();
        let result = generate_income_tax_xml(&itr, &settings).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("kc_zd7=\"400000\""));
        assert!(xml_str.contains("pr_zd7=\"1000000\""));
        assert!(xml_str.contains("vy_zd7=\"600000\""));
    }

    #[test]
    fn test_filing_type_codes() {
        let settings = test_settings();

        for (filing_type, expected_code) in [
            (FilingType::Regular, "B"),
            (FilingType::Corrective, "O"),
            (FilingType::Supplementary, "D"),
        ] {
            let mut itr = test_income_tax_return();
            itr.filing_type = filing_type;
            let result = generate_income_tax_xml(&itr, &settings).unwrap();
            let xml_str = String::from_utf8(result).unwrap();
            assert!(
                xml_str.contains(&format!("dap_typ=\"{expected_code}\"")),
                "Expected filing code {expected_code}"
            );
        }
    }

    #[test]
    fn test_taxpayer_info() {
        let itr = test_income_tax_return();
        let settings = test_settings();
        let result = generate_income_tax_xml(&itr, &settings).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("jmeno=\"Jan\""));
        assert!(xml_str.contains("prijmeni=\"Novak\""));
        assert!(xml_str.contains("rod_c=\"8001011234\""));
        assert!(xml_str.contains("k_stat=\"CZ\""));
    }
}
