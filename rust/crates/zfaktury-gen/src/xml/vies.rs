//! DPHSHV VIES summary XML generation.
//!
//! Generates EPO XML for VIES recapitulative statements (Souhrnne hlaseni).

use quick_xml::Writer;
use quick_xml::events::{BytesDecl, BytesEnd, BytesStart, BytesText, Event};
use std::io::Cursor;

use zfaktury_domain::{VIESSummary, VIESSummaryLine};

use super::common::{strip_country_prefix, to_whole_czk_rounded, vies_filing_type_code};
use crate::Result;

/// Generate DPHSHV XML bytes from a VIES summary, its lines, and the filer DIC.
pub fn generate_vies_xml(
    vs: &VIESSummary,
    lines: &[VIESSummaryLine],
    dic: &str,
) -> Result<Vec<u8>> {
    let filer_dic = strip_country_prefix(&dic.to_uppercase());
    let filing_code = vies_filing_type_code(&vs.filing_type);

    let mut writer = Writer::new_with_indent(Cursor::new(Vec::new()), b' ', 2);

    // XML declaration.
    writer.write_event(Event::Decl(BytesDecl::new("1.0", Some("UTF-8"), None)))?;
    writer.write_event(Event::Text(BytesText::new("\n")))?;

    // <Pisemnost xmlns="http://adis.mfcr.cz/rozhranni/">
    let mut pisemnost = BytesStart::new("Pisemnost");
    pisemnost.push_attribute(("xmlns", "http://adis.mfcr.cz/rozhranni/"));
    writer.write_event(Event::Start(pisemnost))?;

    // <DPHSHV>
    writer.write_event(Event::Start(BytesStart::new("DPHSHV")))?;

    // <VetaD .../>
    let mut veta_d = BytesStart::new("VetaD");
    veta_d.push_attribute(("k_daph", filing_code));
    veta_d.push_attribute(("rok", vs.period.year.to_string().as_str()));
    veta_d.push_attribute(("ctvrt", vs.period.quarter.to_string().as_str()));
    veta_d.push_attribute(("dic_odb", filer_dic.as_str()));
    writer.write_event(Event::Empty(veta_d))?;

    // <VetaP .../> for each line
    for line in lines {
        let mut veta_p = BytesStart::new("VetaP");
        veta_p.push_attribute(("k_stat", line.country_code.as_str()));
        let partner_dic = strip_country_prefix(&line.partner_dic.to_uppercase());
        veta_p.push_attribute(("dic_odbe", partner_dic.as_str()));
        veta_p.push_attribute(("k_plneni", line.service_code.as_str()));
        let obrat = to_whole_czk_rounded(line.total_amount);
        veta_p.push_attribute(("obrat", obrat.to_string().as_str()));
        writer.write_event(Event::Empty(veta_p))?;
    }

    // </DPHSHV>
    writer.write_event(Event::End(BytesEnd::new("DPHSHV")))?;

    // </Pisemnost>
    writer.write_event(Event::End(BytesEnd::new("Pisemnost")))?;

    Ok(writer.into_inner().into_inner())
}

#[cfg(test)]
mod tests {
    use super::*;
    use zfaktury_domain::{Amount, FilingType, TaxPeriod};

    fn test_vies_summary() -> VIESSummary {
        let dt = chrono::NaiveDate::from_ymd_opt(2024, 1, 1)
            .unwrap()
            .and_hms_opt(0, 0, 0)
            .unwrap();
        VIESSummary {
            id: 1,
            period: TaxPeriod {
                year: 2024,
                month: 0,
                quarter: 1,
            },
            filing_type: FilingType::Regular,
            xml_data: Vec::new(),
            status: zfaktury_domain::FilingStatus::Draft,
            filed_at: None,
            created_at: dt,
            updated_at: dt,
        }
    }

    fn test_lines() -> Vec<VIESSummaryLine> {
        vec![VIESSummaryLine {
            id: 1,
            vies_summary_id: 1,
            partner_dic: "DE999999999".to_string(),
            country_code: "DE".to_string(),
            total_amount: Amount::new(150_000, 0),
            service_code: "3".to_string(),
        }]
    }

    #[test]
    fn test_generate_valid_xml() {
        let vs = test_vies_summary();
        let lines = test_lines();
        let result = generate_vies_xml(&vs, &lines, "CZ12345678").unwrap();
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
        assert!(xml_str.contains("<DPHSHV>"));
    }

    #[test]
    fn test_vies_attributes() {
        let vs = test_vies_summary();
        let lines = test_lines();
        let result = generate_vies_xml(&vs, &lines, "CZ12345678").unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("k_daph=\"B\""));
        assert!(xml_str.contains("dic_odb=\"12345678\""));
        assert!(xml_str.contains("ctvrt=\"1\""));
    }

    #[test]
    fn test_partner_lines() {
        let vs = test_vies_summary();
        let lines = test_lines();
        let result = generate_vies_xml(&vs, &lines, "CZ12345678").unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("k_stat=\"DE\""));
        assert!(xml_str.contains("dic_odbe=\"999999999\""));
        assert!(xml_str.contains("k_plneni=\"3\""));
        assert!(xml_str.contains("obrat=\"150000\""));
    }

    #[test]
    fn test_filing_type_codes() {
        for (filing_type, expected) in [
            (FilingType::Regular, "B"),
            (FilingType::Corrective, "O"),
            (FilingType::Supplementary, "N"),
        ] {
            let mut vs = test_vies_summary();
            vs.filing_type = filing_type;
            let result = generate_vies_xml(&vs, &[], "CZ12345678").unwrap();
            let xml_str = String::from_utf8(result).unwrap();
            assert!(
                xml_str.contains(&format!("k_daph=\"{expected}\"")),
                "Expected {expected}"
            );
        }
    }
}
