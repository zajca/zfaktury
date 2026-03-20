//! Health insurance overview XML generation (placeholder).
//!
//! Health insurance providers (VZP, OZP, etc.) each have slightly different
//! submission formats. This module provides a simplified XML export that
//! covers the common fields.

use quick_xml::Writer;
use quick_xml::events::{BytesDecl, BytesEnd, BytesStart, BytesText, Event};
use std::collections::HashMap;
use std::io::Cursor;

use zfaktury_domain::HealthInsuranceOverview;

use super::common::to_whole_czk_truncated;
use crate::Result;

/// Generate a health insurance overview XML.
///
/// Required settings keys:
/// - `health_ins_code` - Health insurance company code
/// - `taxpayer_first_name`, `taxpayer_last_name`
/// - `taxpayer_birth_number`
pub fn generate_health_insurance_xml(
    hio: &HealthInsuranceOverview,
    settings: &HashMap<String, String>,
) -> Result<Vec<u8>> {
    let mut writer = Writer::new_with_indent(Cursor::new(Vec::new()), b' ', 2);

    writer.write_event(Event::Decl(BytesDecl::new("1.0", Some("UTF-8"), None)))?;
    writer.write_event(Event::Text(BytesText::new("\n")))?;

    // <PrehledOSVC>
    let mut root = BytesStart::new("PrehledOSVC");
    root.push_attribute(("xmlns", "http://zfaktury.cz/health-insurance"));
    root.push_attribute(("version", "1.0"));
    writer.write_event(Event::Start(root))?;

    // <Metadata>
    writer.write_event(Event::Start(BytesStart::new("Metadata")))?;
    write_text_element(&mut writer, "Year", &hio.year.to_string())?;
    write_text_element(
        &mut writer,
        "InsuranceCode",
        &get_setting(settings, "health_ins_code"),
    )?;
    write_text_element(&mut writer, "Software", "ZFaktury")?;
    writer.write_event(Event::End(BytesEnd::new("Metadata")))?;

    // <Taxpayer>
    writer.write_event(Event::Start(BytesStart::new("Taxpayer")))?;
    write_text_element(
        &mut writer,
        "FirstName",
        &get_setting(settings, "taxpayer_first_name"),
    )?;
    write_text_element(
        &mut writer,
        "LastName",
        &get_setting(settings, "taxpayer_last_name"),
    )?;
    write_text_element(
        &mut writer,
        "BirthNumber",
        &get_setting(settings, "taxpayer_birth_number"),
    )?;
    writer.write_event(Event::End(BytesEnd::new("Taxpayer")))?;

    // <Calculation>
    writer.write_event(Event::Start(BytesStart::new("Calculation")))?;
    write_text_element(
        &mut writer,
        "TotalRevenue",
        &to_whole_czk_truncated(hio.total_revenue).to_string(),
    )?;
    write_text_element(
        &mut writer,
        "TotalExpenses",
        &to_whole_czk_truncated(hio.total_expenses).to_string(),
    )?;
    write_text_element(
        &mut writer,
        "TaxBase",
        &to_whole_czk_truncated(hio.tax_base).to_string(),
    )?;
    write_text_element(
        &mut writer,
        "AssessmentBase",
        &to_whole_czk_truncated(hio.assessment_base).to_string(),
    )?;
    write_text_element(
        &mut writer,
        "MinAssessmentBase",
        &to_whole_czk_truncated(hio.min_assessment_base).to_string(),
    )?;
    write_text_element(
        &mut writer,
        "FinalAssessmentBase",
        &to_whole_czk_truncated(hio.final_assessment_base).to_string(),
    )?;
    write_text_element(
        &mut writer,
        "TotalInsurance",
        &to_whole_czk_truncated(hio.total_insurance).to_string(),
    )?;
    write_text_element(
        &mut writer,
        "Prepayments",
        &to_whole_czk_truncated(hio.prepayments).to_string(),
    )?;
    write_text_element(
        &mut writer,
        "Difference",
        &to_whole_czk_truncated(hio.difference).to_string(),
    )?;
    write_text_element(
        &mut writer,
        "NewMonthlyPrepay",
        &to_whole_czk_truncated(hio.new_monthly_prepay).to_string(),
    )?;
    writer.write_event(Event::End(BytesEnd::new("Calculation")))?;

    // </PrehledOSVC>
    writer.write_event(Event::End(BytesEnd::new("PrehledOSVC")))?;

    Ok(writer.into_inner().into_inner())
}

fn get_setting(settings: &HashMap<String, String>, key: &str) -> String {
    settings.get(key).cloned().unwrap_or_default()
}

fn write_text_element(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    tag: &str,
    text: &str,
) -> crate::Result<()> {
    writer.write_event(Event::Start(BytesStart::new(tag)))?;
    writer.write_event(Event::Text(BytesText::new(text)))?;
    writer.write_event(Event::End(BytesEnd::new(tag)))?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use zfaktury_domain::{Amount, FilingType};

    fn test_settings() -> HashMap<String, String> {
        let mut m = HashMap::new();
        m.insert("health_ins_code".to_string(), "111".to_string());
        m.insert("taxpayer_first_name".to_string(), "Jan".to_string());
        m.insert("taxpayer_last_name".to_string(), "Novak".to_string());
        m.insert(
            "taxpayer_birth_number".to_string(),
            "8001011234".to_string(),
        );
        m
    }

    fn test_health_insurance() -> HealthInsuranceOverview {
        let dt = chrono::NaiveDate::from_ymd_opt(2024, 1, 1)
            .unwrap()
            .and_hms_opt(0, 0, 0)
            .unwrap();
        HealthInsuranceOverview {
            id: 1,
            year: 2024,
            filing_type: FilingType::Regular,
            total_revenue: Amount::new(1_000_000, 0),
            total_expenses: Amount::new(600_000, 0),
            tax_base: Amount::new(400_000, 0),
            assessment_base: Amount::new(200_000, 0),
            min_assessment_base: Amount::new(120_000, 0),
            final_assessment_base: Amount::new(200_000, 0),
            insurance_rate: 135,
            total_insurance: Amount::new(27_000, 0),
            prepayments: Amount::new(20_000, 0),
            difference: Amount::new(7_000, 0),
            new_monthly_prepay: Amount::new(2_250, 0),
            xml_data: Vec::new(),
            status: zfaktury_domain::FilingStatus::Draft,
            filed_at: None,
            created_at: dt,
            updated_at: dt,
        }
    }

    #[test]
    fn test_generate_valid_xml() {
        let hio = test_health_insurance();
        let settings = test_settings();
        let result = generate_health_insurance_xml(&hio, &settings).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        // Verify valid XML by parsing.
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

        assert!(xml_str.contains("<PrehledOSVC"));
        assert!(xml_str.contains("<Year>2024</Year>"));
    }

    #[test]
    fn test_amounts_whole_czk() {
        let hio = test_health_insurance();
        let settings = test_settings();
        let result = generate_health_insurance_xml(&hio, &settings).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("<TotalRevenue>1000000</TotalRevenue>"));
        assert!(xml_str.contains("<TotalInsurance>27000</TotalInsurance>"));
        assert!(xml_str.contains("<Difference>7000</Difference>"));
    }
}
