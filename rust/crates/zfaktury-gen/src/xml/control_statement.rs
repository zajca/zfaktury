//! DPHKH1 VAT control statement XML generation.
//!
//! Generates EPO XML for Czech VAT control statements (Kontrolni hlaseni).

use quick_xml::Writer;
use quick_xml::events::{BytesDecl, BytesEnd, BytesStart, BytesText, Event};
use std::io::Cursor;

use zfaktury_domain::{ControlSection, VATControlStatement, VATControlStatementLine};

use super::common::{
    control_statement_filing_type_code, strip_country_prefix, to_whole_czk_rounded,
};
use crate::Result;

/// Generate DPHKH1 XML bytes from a control statement, its lines, and the taxpayer DIC.
pub fn generate_control_statement_xml(
    cs: &VATControlStatement,
    lines: &[VATControlStatementLine],
    dic: &str,
) -> Result<Vec<u8>> {
    let dic_num = strip_country_prefix(&dic.to_uppercase());

    let filing_code = control_statement_filing_type_code(&cs.filing_type);

    let mut writer = Writer::new_with_indent(Cursor::new(Vec::new()), b' ', 2);

    // XML declaration.
    writer.write_event(Event::Decl(BytesDecl::new("1.0", Some("UTF-8"), None)))?;
    writer.write_event(Event::Text(BytesText::new("\n")))?;

    // <Pisemnost xmlns="http://adis.mfcr.cz/rozhranni/">
    let mut pisemnost = BytesStart::new("Pisemnost");
    pisemnost.push_attribute(("xmlns", "http://adis.mfcr.cz/rozhranni/"));
    writer.write_event(Event::Start(pisemnost))?;

    // <DPHKH1>
    writer.write_event(Event::Start(BytesStart::new("DPHKH1")))?;

    // <VetaD .../>
    let mut veta_d = BytesStart::new("VetaD");
    veta_d.push_attribute(("d_typ", filing_code));
    veta_d.push_attribute(("rok", cs.period.year.to_string().as_str()));
    veta_d.push_attribute(("mesic", cs.period.month.to_string().as_str()));
    veta_d.push_attribute(("dokdphkh", "KH"));
    veta_d.push_attribute(("khdph_forma", filing_code));
    writer.write_event(Event::Empty(veta_d))?;

    // <VetaP .../>
    let mut veta_p = BytesStart::new("VetaP");
    veta_p.push_attribute(("dic", dic_num.as_str()));
    veta_p.push_attribute(("typ_ds", "P"));
    writer.write_event(Event::Empty(veta_p))?;

    // Write line elements sorted by section.
    for line in lines {
        match &line.section {
            ControlSection::A4 => write_veta_a4(&mut writer, line)?,
            ControlSection::A5 => write_veta_a5(&mut writer, line)?,
            ControlSection::B2 => write_veta_b2(&mut writer, line)?,
            ControlSection::B3 => write_veta_b3(&mut writer, line)?,
        }
    }

    // </DPHKH1>
    writer.write_event(Event::End(BytesEnd::new("DPHKH1")))?;

    // </Pisemnost>
    writer.write_event(Event::End(BytesEnd::new("Pisemnost")))?;

    Ok(writer.into_inner().into_inner())
}

/// Format a date string from YYYY-MM-DD to DD.MM.YYYY for the XML.
fn format_dppd(dppd: &str) -> String {
    if let Ok(date) = chrono::NaiveDate::parse_from_str(dppd, "%Y-%m-%d") {
        date.format("%d.%m.%Y").to_string()
    } else {
        dppd.to_string()
    }
}

/// Add VAT rate attributes to an element (zakl_dane1/dan1 for 21%, zakl_dane2/dan2 for 12%).
fn add_vat_rate_attrs(elem: &mut BytesStart, line: &VATControlStatementLine) {
    let base = to_whole_czk_rounded(line.base);
    let vat = to_whole_czk_rounded(line.vat);

    match line.vat_rate_percent {
        21 => {
            if base != 0 {
                elem.push_attribute(("zakl_dane1", base.to_string().as_str()));
            }
            if vat != 0 {
                elem.push_attribute(("dan1", vat.to_string().as_str()));
            }
        }
        12 => {
            if base != 0 {
                elem.push_attribute(("zakl_dane2", base.to_string().as_str()));
            }
            if vat != 0 {
                elem.push_attribute(("dan2", vat.to_string().as_str()));
            }
        }
        _ => {}
    }
}

fn write_veta_a4(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    line: &VATControlStatementLine,
) -> crate::Result<()> {
    let mut elem = BytesStart::new("VetaA4");
    elem.push_attribute(("c_evid_dd", line.document_number.as_str()));
    elem.push_attribute(("dppd", format_dppd(&line.dppd).as_str()));
    let dic_odb = strip_country_prefix(&line.partner_dic.to_uppercase());
    elem.push_attribute(("dic_odb", dic_odb.as_str()));
    elem.push_attribute(("kod_rezim_pln", "0"));
    add_vat_rate_attrs(&mut elem, line);
    writer.write_event(Event::Empty(elem))?;
    Ok(())
}

fn write_veta_a5(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    line: &VATControlStatementLine,
) -> crate::Result<()> {
    let mut elem = BytesStart::new("VetaA5");
    elem.push_attribute(("kod_rezim_pln", "0"));
    add_vat_rate_attrs(&mut elem, line);
    writer.write_event(Event::Empty(elem))?;
    Ok(())
}

fn write_veta_b2(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    line: &VATControlStatementLine,
) -> crate::Result<()> {
    let mut elem = BytesStart::new("VetaB2");
    elem.push_attribute(("c_evid_dd", line.document_number.as_str()));
    elem.push_attribute(("dppd", format_dppd(&line.dppd).as_str()));
    let dic_dod = strip_country_prefix(&line.partner_dic.to_uppercase());
    elem.push_attribute(("dic_dod", dic_dod.as_str()));
    elem.push_attribute(("kod_rezim_pln", "0"));
    add_vat_rate_attrs(&mut elem, line);
    writer.write_event(Event::Empty(elem))?;
    Ok(())
}

fn write_veta_b3(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    line: &VATControlStatementLine,
) -> crate::Result<()> {
    let mut elem = BytesStart::new("VetaB3");
    elem.push_attribute(("kod_rezim_pln", "0"));
    add_vat_rate_attrs(&mut elem, line);
    writer.write_event(Event::Empty(elem))?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use zfaktury_domain::{Amount, ControlSection, FilingType, TaxPeriod};

    fn test_control_statement() -> VATControlStatement {
        let dt = chrono::NaiveDate::from_ymd_opt(2024, 1, 1)
            .unwrap()
            .and_hms_opt(0, 0, 0)
            .unwrap();
        VATControlStatement {
            id: 1,
            period: TaxPeriod {
                year: 2024,
                month: 1,
                quarter: 0,
            },
            filing_type: FilingType::Regular,
            xml_data: Vec::new(),
            status: zfaktury_domain::FilingStatus::Draft,
            filed_at: None,
            created_at: dt,
            updated_at: dt,
        }
    }

    fn test_lines() -> Vec<VATControlStatementLine> {
        let make_line = |section: ControlSection, rate: i32| VATControlStatementLine {
            id: 1,
            control_statement_id: 1,
            section,
            partner_dic: "CZ87654321".to_string(),
            document_number: "FV20240001".to_string(),
            dppd: "2024-01-15".to_string(),
            base: Amount::new(50_000, 0),
            vat: Amount::new(10_500, 0),
            vat_rate_percent: rate,
            invoice_id: Some(1),
            expense_id: None,
        };

        vec![
            make_line(ControlSection::A4, 21),
            make_line(ControlSection::B2, 21),
        ]
    }

    #[test]
    fn test_generate_valid_xml() {
        let cs = test_control_statement();
        let lines = test_lines();
        let result = generate_control_statement_xml(&cs, &lines, "CZ12345678").unwrap();
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
        assert!(xml_str.contains("xmlns=\"http://adis.mfcr.cz/rozhranni/\""));
        assert!(xml_str.contains("<DPHKH1>"));
    }

    #[test]
    fn test_root_attributes() {
        let cs = test_control_statement();
        let lines = test_lines();
        let result = generate_control_statement_xml(&cs, &lines, "CZ12345678").unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("d_typ=\"R\""));
        assert!(xml_str.contains("dokdphkh=\"KH\""));
        assert!(xml_str.contains("dic=\"12345678\""));
    }

    #[test]
    fn test_amounts_whole_czk() {
        let cs = test_control_statement();
        let lines = test_lines();
        let result = generate_control_statement_xml(&cs, &lines, "CZ12345678").unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        // base = 50000 CZK, vat = 10500 CZK
        assert!(xml_str.contains("zakl_dane1=\"50000\""));
        assert!(xml_str.contains("dan1=\"10500\""));
    }

    #[test]
    fn test_date_formatting() {
        let cs = test_control_statement();
        let lines = test_lines();
        let result = generate_control_statement_xml(&cs, &lines, "CZ12345678").unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        // YYYY-MM-DD should be converted to DD.MM.YYYY.
        assert!(xml_str.contains("dppd=\"15.01.2024\""));
    }

    #[test]
    fn test_sections_a4_b2() {
        let cs = test_control_statement();
        let lines = test_lines();
        let result = generate_control_statement_xml(&cs, &lines, "CZ12345678").unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("<VetaA4"));
        assert!(xml_str.contains("<VetaB2"));
        assert!(xml_str.contains("dic_odb=\"87654321\""));
        assert!(xml_str.contains("dic_dod=\"87654321\""));
    }

    #[test]
    fn test_filing_type_codes() {
        for (filing_type, expected) in [
            (FilingType::Regular, "R"),
            (FilingType::Corrective, "N"),
            (FilingType::Supplementary, "O"),
        ] {
            let mut cs = test_control_statement();
            cs.filing_type = filing_type;
            let result = generate_control_statement_xml(&cs, &[], "CZ12345678").unwrap();
            let xml_str = String::from_utf8(result).unwrap();
            assert!(
                xml_str.contains(&format!("d_typ=\"{expected}\"")),
                "Expected {expected}"
            );
        }
    }
}
