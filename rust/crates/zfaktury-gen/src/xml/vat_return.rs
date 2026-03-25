//! DPHDP3 VAT return XML generation.
//!
//! Generates EPO XML for Czech VAT returns (Priznani k DPH).

use quick_xml::Writer;
use quick_xml::events::{BytesDecl, BytesEnd, BytesStart, BytesText, Event};
use std::io::Cursor;

use zfaktury_domain::VATReturn;

use super::common::{TaxpayerInfo, dph_filing_type_code, to_whole_czk_rounded};
use crate::{GenError, Result};

/// Generate DPHDP3 XML bytes from a VATReturn and taxpayer info.
pub fn generate_vat_return_xml(vr: &VATReturn, taxpayer: &TaxpayerInfo) -> Result<Vec<u8>> {
    if taxpayer.dic.is_empty() {
        return Err(GenError::InvalidInput(
            "DIC is required for XML generation".to_string(),
        ));
    }

    let filing_code = dph_filing_type_code(&vr.filing_type);

    // Convert amounts to whole CZK (rounded).
    let obrat23 = to_whole_czk_rounded(vr.output_vat_base_21);
    let dan23 = to_whole_czk_rounded(vr.output_vat_amount_21);
    let obrat5 = to_whole_czk_rounded(vr.output_vat_base_12);
    let dan5 = to_whole_czk_rounded(vr.output_vat_amount_12);

    let pln23 = to_whole_czk_rounded(vr.input_vat_base_21);
    let odp_tuz23_nar = to_whole_czk_rounded(vr.input_vat_amount_21);
    let pln5 = to_whole_czk_rounded(vr.input_vat_base_12);
    let odp_tuz5_nar = to_whole_czk_rounded(vr.input_vat_amount_12);

    // Total deductions.
    let odp_sum_nar = odp_tuz23_nar + odp_tuz5_nar;
    // Total output tax.
    let dan_zocelk = dan23 + dan5;
    // Total input deductions.
    let odp_zocelk = odp_sum_nar;
    // Net VAT.
    let net_vat = dan_zocelk - odp_zocelk;

    // Determine trans: "A" if any amounts, "N" otherwise.
    let trans = if obrat23 != 0
        || dan23 != 0
        || obrat5 != 0
        || dan5 != 0
        || pln23 != 0
        || odp_tuz23_nar != 0
        || pln5 != 0
        || odp_tuz5_nar != 0
    {
        "A"
    } else {
        "N"
    };

    // Determine dano_da / dano_no.
    let (dano_da, dano_no) = if net_vat > 0 {
        (net_vat as f64, "0".to_string())
    } else if net_vat < 0 {
        (0.0, format!("{:.1}", (-net_vat) as f64))
    } else {
        (0.0, "0".to_string())
    };

    // Submission date: use provided or empty (caller responsible for formatting).
    let submission_date = if taxpayer.submission_date.is_empty() {
        chrono::Local::now().format("%d.%m.%Y").to_string()
    } else {
        taxpayer.submission_date.clone()
    };

    let mut writer = Writer::new_with_indent(Cursor::new(Vec::new()), b' ', 2);

    // XML declaration.
    writer.write_event(Event::Decl(BytesDecl::new("1.0", Some("UTF-8"), None)))?;
    writer.write_event(Event::Text(BytesText::new("\n")))?;

    // <Pisemnost nazevSW="ZFaktury">
    let mut pisemnost = BytesStart::new("Pisemnost");
    pisemnost.push_attribute(("nazevSW", "ZFaktury"));
    writer.write_event(Event::Start(pisemnost))?;

    // <DPHDP3 verzePis="01.02.16">
    let mut dphdp3 = BytesStart::new("DPHDP3");
    dphdp3.push_attribute(("verzePis", "01.02.16"));
    writer.write_event(Event::Start(dphdp3))?;

    // <VetaD .../>
    let mut veta_d = BytesStart::new("VetaD");
    veta_d.push_attribute(("dokument", "DP3"));
    veta_d.push_attribute(("k_uladis", "DPH"));
    veta_d.push_attribute(("dapdph_forma", filing_code));
    veta_d.push_attribute(("typ_platce", "P"));
    veta_d.push_attribute(("trans", trans));
    if !taxpayer.okec.is_empty() {
        veta_d.push_attribute(("c_okec", taxpayer.okec.as_str()));
    }
    veta_d.push_attribute(("d_poddp", submission_date.as_str()));
    veta_d.push_attribute(("rok", vr.period.year.to_string().as_str()));
    if vr.period.month > 0 {
        veta_d.push_attribute(("mesic", vr.period.month.to_string().as_str()));
    }
    if vr.period.quarter > 0 {
        veta_d.push_attribute(("ctvrt", vr.period.quarter.to_string().as_str()));
    }
    writer.write_event(Event::Empty(veta_d))?;

    // <VetaP .../>
    let mut veta_p = BytesStart::new("VetaP");
    if !taxpayer.prac_ufo.is_empty() {
        veta_p.push_attribute(("c_pracufo", taxpayer.prac_ufo.as_str()));
    }
    if !taxpayer.ufo_code.is_empty() {
        veta_p.push_attribute(("c_ufo", taxpayer.ufo_code.as_str()));
    }
    veta_p.push_attribute(("dic", taxpayer.dic.as_str()));
    if !taxpayer.email.is_empty() {
        veta_p.push_attribute(("email", taxpayer.email.as_str()));
    }
    if !taxpayer.phone.is_empty() {
        veta_p.push_attribute(("c_telef", taxpayer.phone.as_str()));
    }
    if !taxpayer.street.is_empty() {
        veta_p.push_attribute(("ulice", taxpayer.street.as_str()));
    }
    if !taxpayer.city.is_empty() {
        veta_p.push_attribute(("naz_obce", taxpayer.city.as_str()));
    }
    if !taxpayer.zip.is_empty() {
        veta_p.push_attribute(("psc", taxpayer.zip.as_str()));
    }
    veta_p.push_attribute(("stat", "ČESKÁ REPUBLIKA"));
    if !taxpayer.house_num.is_empty() {
        veta_p.push_attribute(("c_pop", taxpayer.house_num.as_str()));
    }
    veta_p.push_attribute(("c_orient", ""));
    if !taxpayer.first_name.is_empty() {
        veta_p.push_attribute(("jmeno", taxpayer.first_name.as_str()));
    }
    if !taxpayer.last_name.is_empty() {
        veta_p.push_attribute(("prijmeni", taxpayer.last_name.as_str()));
    }
    veta_p.push_attribute(("titul", ""));
    veta_p.push_attribute(("typ_ds", "F"));
    writer.write_event(Event::Empty(veta_p))?;

    // <Veta1 .../>
    let mut veta1 = BytesStart::new("Veta1");
    veta1.push_attribute(("obrat23", format_f64(obrat23 as f64).as_str()));
    veta1.push_attribute(("dan23", format_f64(dan23 as f64).as_str()));
    veta1.push_attribute(("obrat5", format_f64(obrat5 as f64).as_str()));
    veta1.push_attribute(("dan5", format_f64(dan5 as f64).as_str()));
    veta1.push_attribute(("p_sl23_e", format_f64(0.0).as_str()));
    veta1.push_attribute(("dan_psl23_e", format_f64(0.0).as_str()));
    veta1.push_attribute(("p_sl5_e", format_f64(0.0).as_str()));
    veta1.push_attribute(("dan_psl5_e", format_f64(0.0).as_str()));
    veta1.push_attribute(("p_sl23_z", format_f64(0.0).as_str()));
    veta1.push_attribute(("dan_psl23_z", format_f64(0.0).as_str()));
    veta1.push_attribute(("p_sl5_z", format_f64(0.0).as_str()));
    veta1.push_attribute(("dan_psl5_z", format_f64(0.0).as_str()));
    veta1.push_attribute(("rez_pren23", format_f64(0.0).as_str()));
    veta1.push_attribute(("dan_rpren23", format_f64(0.0).as_str()));
    veta1.push_attribute(("rez_pren5", format_f64(0.0).as_str()));
    veta1.push_attribute(("dan_rpren5", format_f64(0.0).as_str()));
    writer.write_event(Event::Empty(veta1))?;

    // <Veta2 .../>
    let mut veta2 = BytesStart::new("Veta2");
    veta2.push_attribute(("dod_zb", format_f64(0.0).as_str()));
    veta2.push_attribute(("pln_sluzby", format_f64(0.0).as_str()));
    veta2.push_attribute(("pln_rez_pren", format_f64(0.0).as_str()));
    veta2.push_attribute(("pln_zaslani", format_f64(0.0).as_str()));
    veta2.push_attribute(("pln_ost", format_f64(0.0).as_str()));
    writer.write_event(Event::Empty(veta2))?;

    // <Veta4 .../>
    let mut veta4 = BytesStart::new("Veta4");
    veta4.push_attribute(("pln23", format_f64(pln23 as f64).as_str()));
    veta4.push_attribute(("odp_tuz23_nar", format_f64(odp_tuz23_nar as f64).as_str()));
    veta4.push_attribute(("pln5", format_f64(pln5 as f64).as_str()));
    veta4.push_attribute(("odp_tuz5_nar", format_f64(odp_tuz5_nar as f64).as_str()));
    veta4.push_attribute(("nar_zdp23", format_f64(0.0).as_str()));
    veta4.push_attribute(("od_zdp23", format_f64(0.0).as_str()));
    veta4.push_attribute(("nar_zdp5", format_f64(0.0).as_str()));
    veta4.push_attribute(("od_zdp5", format_f64(0.0).as_str()));
    veta4.push_attribute(("odp_sum_kr", "0"));
    veta4.push_attribute(("odp_sum_nar", format_f64(odp_sum_nar as f64).as_str()));
    writer.write_event(Event::Empty(veta4))?;

    // <Veta6 .../>
    let mut veta6 = BytesStart::new("Veta6");
    veta6.push_attribute(("dano", "0"));
    veta6.push_attribute(("dano_no", dano_no.as_str()));
    veta6.push_attribute(("dano_da", format_f64(dano_da).as_str()));
    veta6.push_attribute(("dan_zocelk", format_f64(dan_zocelk as f64).as_str()));
    veta6.push_attribute(("odp_zocelk", format_f64(odp_zocelk as f64).as_str()));
    writer.write_event(Event::Empty(veta6))?;

    // </DPHDP3>
    writer.write_event(Event::End(BytesEnd::new("DPHDP3")))?;

    // </Pisemnost>
    writer.write_event(Event::End(BytesEnd::new("Pisemnost")))?;

    Ok(writer.into_inner().into_inner())
}

/// Format a float for XML attribute output (always one decimal place to match EPO/Fakturoid format).
fn format_f64(v: f64) -> String {
    format!("{:.1}", v)
}

#[cfg(test)]
mod tests {
    use super::*;
    use zfaktury_domain::{Amount, FilingType, TaxPeriod};

    fn test_taxpayer() -> TaxpayerInfo {
        TaxpayerInfo {
            dic: "12345678".to_string(),
            first_name: "Jan".to_string(),
            last_name: "Novak".to_string(),
            street: "Hlavni 1".to_string(),
            house_num: "1".to_string(),
            zip: "11000".to_string(),
            city: "Praha".to_string(),
            phone: "+420123456789".to_string(),
            email: "jan@example.com".to_string(),
            ufo_code: "451".to_string(),
            prac_ufo: "2001".to_string(),
            okec: "62010".to_string(),
            submission_date: "25.01.2024".to_string(),
        }
    }

    fn test_vat_return() -> VATReturn {
        VATReturn {
            id: 1,
            period: TaxPeriod {
                year: 2024,
                month: 1,
                quarter: 0,
            },
            filing_type: FilingType::Regular,
            output_vat_base_21: Amount::new(100_000, 0),
            output_vat_amount_21: Amount::new(21_000, 0),
            output_vat_base_12: Amount::new(50_000, 0),
            output_vat_amount_12: Amount::new(6_000, 0),
            output_vat_base_0: Amount::ZERO,
            reverse_charge_base_21: Amount::ZERO,
            reverse_charge_amount_21: Amount::ZERO,
            reverse_charge_base_12: Amount::ZERO,
            reverse_charge_amount_12: Amount::ZERO,
            input_vat_base_21: Amount::new(30_000, 0),
            input_vat_amount_21: Amount::new(6_300, 0),
            input_vat_base_12: Amount::new(10_000, 0),
            input_vat_amount_12: Amount::new(1_200, 0),
            total_output_vat: Amount::new(27_000, 0),
            total_input_vat: Amount::new(7_500, 0),
            net_vat: Amount::new(19_500, 0),
            xml_data: Vec::new(),
            status: zfaktury_domain::FilingStatus::Draft,
            filed_at: None,
            created_at: chrono::NaiveDate::from_ymd_opt(2024, 1, 1)
                .unwrap()
                .and_hms_opt(0, 0, 0)
                .unwrap(),
            updated_at: chrono::NaiveDate::from_ymd_opt(2024, 1, 1)
                .unwrap()
                .and_hms_opt(0, 0, 0)
                .unwrap(),
        }
    }

    #[test]
    fn test_generate_valid_xml() {
        let vr = test_vat_return();
        let tp = test_taxpayer();
        let result = generate_vat_return_xml(&vr, &tp).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        // Verify it's valid XML by parsing.
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

        // Verify root element.
        assert!(xml_str.contains("<Pisemnost"));
        assert!(xml_str.contains("nazevSW=\"ZFaktury\""));
    }

    #[test]
    fn test_amounts_are_whole_czk() {
        let vr = test_vat_return();
        let tp = test_taxpayer();
        let result = generate_vat_return_xml(&vr, &tp).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        // Output VAT base 21% = 100000 CZK.
        assert!(xml_str.contains("obrat23=\"100000.0\""));
        // Output VAT amount 21% = 21000 CZK.
        assert!(xml_str.contains("dan23=\"21000.0\""));
    }

    #[test]
    fn test_filing_type_codes() {
        let tp = test_taxpayer();

        for (filing_type, expected_code) in [
            (FilingType::Regular, "B"),
            (FilingType::Corrective, "O"),
            (FilingType::Supplementary, "D"),
        ] {
            let mut vr = test_vat_return();
            vr.filing_type = filing_type;
            let result = generate_vat_return_xml(&vr, &tp).unwrap();
            let xml_str = String::from_utf8(result).unwrap();
            assert!(
                xml_str.contains(&format!("dapdph_forma=\"{expected_code}\"")),
                "Expected filing code {expected_code}"
            );
        }
    }

    #[test]
    fn test_empty_dic_returns_error() {
        let vr = test_vat_return();
        let mut tp = test_taxpayer();
        tp.dic = String::new();
        let result = generate_vat_return_xml(&vr, &tp);
        assert!(result.is_err());
    }

    #[test]
    fn test_zero_amounts_trans_n() {
        let tp = test_taxpayer();
        let mut vr = test_vat_return();
        vr.output_vat_base_21 = Amount::ZERO;
        vr.output_vat_amount_21 = Amount::ZERO;
        vr.output_vat_base_12 = Amount::ZERO;
        vr.output_vat_amount_12 = Amount::ZERO;
        vr.input_vat_base_21 = Amount::ZERO;
        vr.input_vat_amount_21 = Amount::ZERO;
        vr.input_vat_base_12 = Amount::ZERO;
        vr.input_vat_amount_12 = Amount::ZERO;

        let result = generate_vat_return_xml(&vr, &tp).unwrap();
        let xml_str = String::from_utf8(result).unwrap();
        assert!(xml_str.contains("trans=\"N\""));
    }

    #[test]
    fn test_negative_net_vat_dano_no() {
        let tp = test_taxpayer();
        let mut vr = test_vat_return();
        // Make input VAT exceed output VAT.
        vr.input_vat_amount_21 = Amount::new(50_000, 0);
        vr.input_vat_amount_12 = Amount::new(10_000, 0);

        let result = generate_vat_return_xml(&vr, &tp).unwrap();
        let xml_str = String::from_utf8(result).unwrap();
        // Net VAT = (21000 + 6000) - (50000 + 10000) = -33000
        // dano_da should be 0, dano_no should be "33000.0".
        assert!(xml_str.contains("dano_da=\"0.0\""));
        assert!(xml_str.contains("dano_no=\"33000.0\""));
    }
}
