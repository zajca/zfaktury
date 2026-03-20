//! CSSZ social insurance overview XML generation.
//!
//! Generates XML for the Czech Social Security Administration (CSSZ)
//! annual overview for self-employed persons (Prehled OSVC).

use quick_xml::Writer;
use quick_xml::events::{BytesDecl, BytesEnd, BytesStart, BytesText, Event};
use std::collections::HashMap;
use std::io::Cursor;

use zfaktury_domain::SocialInsuranceOverview;

use super::common::{cssz_filing_type_code, to_whole_czk_truncated};
use crate::Result;

/// Generate CSSZ OSVC overview XML from a SocialInsuranceOverview and settings map.
///
/// Required settings keys:
/// - `cssz_code` - CSSZ office code
/// - `taxpayer_first_name`, `taxpayer_last_name`
/// - `taxpayer_birth_number`, `taxpayer_birth_date`
/// - `taxpayer_street`, `taxpayer_house_number`, `taxpayer_city`, `taxpayer_postal_code`
/// - `flat_rate_expenses` - "true" if using flat-rate expenses
pub fn generate_social_insurance_xml(
    sio: &SocialInsuranceOverview,
    settings: &HashMap<String, String>,
) -> Result<Vec<u8>> {
    let rok = sio.year.to_string();
    let filing_code = cssz_filing_type_code(&sio.filing_type);

    let assessment_base = to_whole_czk_truncated(sio.assessment_base);
    let min_assessment_base = to_whole_czk_truncated(sio.min_assessment_base);
    let final_assessment_base = to_whole_czk_truncated(sio.final_assessment_base);
    let total_insurance = to_whole_czk_truncated(sio.total_insurance);
    let prepayments = to_whole_czk_truncated(sio.prepayments);
    let difference = to_whole_czk_truncated(sio.difference);
    let total_revenue = to_whole_czk_truncated(sio.total_revenue);
    let total_expenses = to_whole_czk_truncated(sio.total_expenses);
    let new_monthly_prepay = to_whole_czk_truncated(sio.new_monthly_prepay);

    let pau_flag = if get_setting(settings, "flat_rate_expenses") == "true" {
        "A"
    } else {
        "N"
    };

    let mut writer = Writer::new_with_indent(Cursor::new(Vec::new()), b' ', 2);

    // XML declaration.
    writer.write_event(Event::Decl(BytesDecl::new("1.0", Some("UTF-8"), None)))?;
    writer.write_event(Event::Text(BytesText::new("\n")))?;

    // <OSVC xmlns="http://schemas.cssz.cz/OSVC2025" version="1.0">
    let mut osvc = BytesStart::new("OSVC");
    osvc.push_attribute(("xmlns", "http://schemas.cssz.cz/OSVC2025"));
    osvc.push_attribute(("version", "1.0"));
    writer.write_event(Event::Start(osvc))?;

    // <VENDOR productName="ZFaktury" productVersion="1.0"/>
    let mut vendor = BytesStart::new("VENDOR");
    vendor.push_attribute(("productName", "ZFaktury"));
    vendor.push_attribute(("productVersion", "1.0"));
    writer.write_event(Event::Empty(vendor))?;

    // <SENDER .../>
    let mut sender = BytesStart::new("SENDER");
    sender.push_attribute(("EmailNotifikace", ""));
    sender.push_attribute(("ISDSreport", "3"));
    sender.push_attribute(("VerzeProtokolu", "1"));
    writer.write_event(Event::Empty(sender))?;

    // <prehledosvc ...>
    let mut prehled = BytesStart::new("prehledosvc");
    prehled.push_attribute(("for", get_setting(settings, "cssz_code").as_str()));
    prehled.push_attribute(("dep", "122"));
    prehled.push_attribute(("rok", rok.as_str()));
    prehled.push_attribute(("typ", filing_code));
    prehled.push_attribute(("vsdp", ""));
    prehled.push_attribute(("dat", ""));
    writer.write_event(Event::Start(prehled))?;

    // <client>
    writer.write_event(Event::Start(BytesStart::new("client")))?;

    // <name fir="..." sur="..." tit=""/>
    let mut name = BytesStart::new("name");
    name.push_attribute(("fir", get_setting(settings, "taxpayer_first_name").as_str()));
    name.push_attribute(("sur", get_setting(settings, "taxpayer_last_name").as_str()));
    name.push_attribute(("tit", ""));
    writer.write_event(Event::Empty(name))?;

    // <birth bno="..." den="..."/>
    let mut birth = BytesStart::new("birth");
    birth.push_attribute((
        "bno",
        get_setting(settings, "taxpayer_birth_number").as_str(),
    ));
    birth.push_attribute(("den", get_setting(settings, "taxpayer_birth_date").as_str()));
    writer.write_event(Event::Empty(birth))?;

    // <adr str="..." num="..." pnu="..." cit="..." cnt="CZ"/>
    let mut adr = BytesStart::new("adr");
    adr.push_attribute(("str", get_setting(settings, "taxpayer_street").as_str()));
    adr.push_attribute((
        "num",
        get_setting(settings, "taxpayer_house_number").as_str(),
    ));
    adr.push_attribute((
        "pnu",
        get_setting(settings, "taxpayer_postal_code").as_str(),
    ));
    adr.push_attribute(("cit", get_setting(settings, "taxpayer_city").as_str()));
    adr.push_attribute(("cnt", "CZ"));
    writer.write_event(Event::Empty(adr))?;

    // <idds/>
    writer.write_event(Event::Start(BytesStart::new("idds")))?;
    writer.write_event(Event::End(BytesEnd::new("idds")))?;

    // <email/>
    writer.write_event(Event::Start(BytesStart::new("email")))?;
    writer.write_event(Event::End(BytesEnd::new("email")))?;

    // <tel/>
    writer.write_event(Event::Start(BytesStart::new("tel")))?;
    writer.write_event(Event::End(BytesEnd::new("tel")))?;

    // <druc>H</druc>
    write_text_element(&mut writer, "druc", "H")?;

    // <hlavc><m1>A</m1><m2/>...<m13/></hlavc>
    writer.write_event(Event::Start(BytesStart::new("hlavc")))?;
    write_text_element(&mut writer, "m1", "A")?;
    for i in 2..=13 {
        let tag = format!("m{i}");
        write_text_element(&mut writer, &tag, "")?;
    }
    writer.write_event(Event::End(BytesEnd::new("hlavc")))?;

    // <vedc><m1/>...<m13/><zam/><duchod/><pdite/><ppm/><pece/><ndite/></vedc>
    writer.write_event(Event::Start(BytesStart::new("vedc")))?;
    for i in 1..=13 {
        let tag = format!("m{i}");
        write_text_element(&mut writer, &tag, "")?;
    }
    for tag in ["zam", "duchod", "pdite", "ppm", "pece", "ndite"] {
        write_text_element(&mut writer, tag, "")?;
    }
    writer.write_event(Event::End(BytesEnd::new("vedc")))?;

    // <narok><m1/>...<m13/></narok>
    writer.write_event(Event::Start(BytesStart::new("narok")))?;
    for i in 1..=13 {
        let tag = format!("m{i}");
        write_text_element(&mut writer, &tag, "")?;
    }
    writer.write_event(Event::End(BytesEnd::new("narok")))?;

    // <sleva><m1>n</m1><m2/>...<m13/></sleva>
    writer.write_event(Event::Start(BytesStart::new("sleva")))?;
    write_text_element(&mut writer, "m1", "n")?;
    for i in 2..=13 {
        let tag = format!("m{i}");
        write_text_element(&mut writer, &tag, "")?;
    }
    writer.write_event(Event::End(BytesEnd::new("sleva")))?;

    // </client>
    writer.write_event(Event::End(BytesEnd::new("client")))?;

    // <pvv pri="1">
    let mut pvv = BytesStart::new("pvv");
    pvv.push_attribute(("pri", "1"));
    writer.write_event(Event::Start(pvv))?;

    // <mesc h="1" v=""/>
    let mut mesc = BytesStart::new("mesc");
    mesc.push_attribute(("h", "1"));
    mesc.push_attribute(("v", ""));
    writer.write_event(Event::Empty(mesc))?;

    // <mesv h="1" v=""/>
    let mut mesv = BytesStart::new("mesv");
    mesv.push_attribute(("h", "1"));
    mesv.push_attribute(("v", ""));
    writer.write_event(Event::Empty(mesv))?;

    // <mesp>1</mesp>
    write_text_element(&mut writer, "mesp", "1")?;

    // <rdza h="..." v="..."/>
    let mut rdza = BytesStart::new("rdza");
    rdza.push_attribute(("h", total_revenue.to_string().as_str()));
    rdza.push_attribute(("v", total_expenses.to_string().as_str()));
    writer.write_event(Event::Empty(rdza))?;

    // <vvz h="..." v="0"/>
    let mut vvz = BytesStart::new("vvz");
    vvz.push_attribute(("h", assessment_base.to_string().as_str()));
    vvz.push_attribute(("v", "0"));
    writer.write_event(Event::Empty(vvz))?;

    // <dvz h="0" v="0"/>
    let mut dvz = BytesStart::new("dvz");
    dvz.push_attribute(("h", "0"));
    dvz.push_attribute(("v", "0"));
    writer.write_event(Event::Empty(dvz))?;

    // <mvz>...</mvz>
    write_text_element(&mut writer, "mvz", &min_assessment_base.to_string())?;
    // <uvz>...</uvz>
    write_text_element(&mut writer, "uvz", &final_assessment_base.to_string())?;
    // <vzza/>
    write_text_element(&mut writer, "vzza", "")?;
    // <vzsu>...</vzsu>
    write_text_element(&mut writer, "vzsu", &final_assessment_base.to_string())?;
    // <vzsvc>...</vzsvc>
    write_text_element(&mut writer, "vzsvc", &final_assessment_base.to_string())?;
    // <poj>...</poj>
    write_text_element(&mut writer, "poj", &total_insurance.to_string())?;
    // <slev/>
    write_text_element(&mut writer, "slev", "")?;
    // <pojposlev/>
    write_text_element(&mut writer, "pojposlev", "")?;
    // <zal>...</zal>
    write_text_element(&mut writer, "zal", &prepayments.to_string())?;
    // <ned>...</ned>
    write_text_element(&mut writer, "ned", &difference.to_string())?;

    // </pvv>
    writer.write_event(Event::End(BytesEnd::new("pvv")))?;

    // <prihldp/>
    write_text_element(&mut writer, "prihldp", "")?;

    // <zal pau="..." vz="..." dp="..." np="0" duch=""/>
    let mut zal = BytesStart::new("zal");
    zal.push_attribute(("ved", ""));
    zal.push_attribute(("pau", pau_flag));
    zal.push_attribute(("vz", final_assessment_base.to_string().as_str()));
    zal.push_attribute(("dp", new_monthly_prepay.to_string().as_str()));
    zal.push_attribute(("np", "0"));
    zal.push_attribute(("duch", ""));
    writer.write_event(Event::Empty(zal))?;

    // <pre vra="0" kam="">
    let mut pre = BytesStart::new("pre");
    pre.push_attribute(("vra", "0"));
    pre.push_attribute(("kam", ""));
    writer.write_event(Event::Start(pre))?;
    write_text_element(&mut writer, "rok", "")?;
    write_text_element(&mut writer, "iban", "")?;
    // <bs pu="" cu="" kb="" ss="" vs=""/>
    let mut bs = BytesStart::new("bs");
    bs.push_attribute(("pu", ""));
    bs.push_attribute(("cu", ""));
    bs.push_attribute(("kb", ""));
    bs.push_attribute(("ss", ""));
    bs.push_attribute(("vs", ""));
    writer.write_event(Event::Empty(bs))?;
    // <adr cit="" cnt="" num="" pnu="" str=""/>
    let mut pre_adr = BytesStart::new("adr");
    pre_adr.push_attribute(("cit", ""));
    pre_adr.push_attribute(("cnt", ""));
    pre_adr.push_attribute(("num", ""));
    pre_adr.push_attribute(("pnu", ""));
    pre_adr.push_attribute(("str", ""));
    writer.write_event(Event::Empty(pre_adr))?;
    writer.write_event(Event::End(BytesEnd::new("pre")))?;

    // <prizn>
    writer.write_event(Event::Start(BytesStart::new("prizn")))?;
    write_text_element(&mut writer, "pau", "H")?;
    write_text_element(&mut writer, "pov", "A")?;
    write_text_element(&mut writer, "elektr", "N")?;
    write_text_element(&mut writer, "por", "N")?;
    write_text_element(&mut writer, "meldat", "")?;
    writer.write_event(Event::End(BytesEnd::new("prizn")))?;

    // <spo bno="" den="">
    let mut spo = BytesStart::new("spo");
    spo.push_attribute(("bno", ""));
    spo.push_attribute(("den", ""));
    writer.write_event(Event::Start(spo))?;
    // <name sur="" fir="" tit=""/>
    let mut spo_name = BytesStart::new("name");
    spo_name.push_attribute(("sur", ""));
    spo_name.push_attribute(("fir", ""));
    spo_name.push_attribute(("tit", ""));
    writer.write_event(Event::Empty(spo_name))?;
    // <adr cit="" cnt="" num="" pnu="" str=""/>
    let mut spo_adr = BytesStart::new("adr");
    spo_adr.push_attribute(("cit", ""));
    spo_adr.push_attribute(("cnt", ""));
    spo_adr.push_attribute(("num", ""));
    spo_adr.push_attribute(("pnu", ""));
    spo_adr.push_attribute(("str", ""));
    writer.write_event(Event::Empty(spo_adr))?;
    writer.write_event(Event::End(BytesEnd::new("spo")))?;

    // <dat dre=""/>
    let mut dat = BytesStart::new("dat");
    dat.push_attribute(("dre", ""));
    writer.write_event(Event::Empty(dat))?;

    // </prehledosvc>
    writer.write_event(Event::End(BytesEnd::new("prehledosvc")))?;

    // </OSVC>
    writer.write_event(Event::End(BytesEnd::new("OSVC")))?;

    Ok(writer.into_inner().into_inner())
}

/// Get a setting value, returning empty string if missing.
fn get_setting(settings: &HashMap<String, String>, key: &str) -> String {
    settings.get(key).cloned().unwrap_or_default()
}

/// Write a simple text element: <tag>text</tag>.
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
        m.insert("cssz_code".to_string(), "PSSZ".to_string());
        m.insert("taxpayer_first_name".to_string(), "Jan".to_string());
        m.insert("taxpayer_last_name".to_string(), "Novak".to_string());
        m.insert(
            "taxpayer_birth_number".to_string(),
            "8001011234".to_string(),
        );
        m.insert("taxpayer_birth_date".to_string(), "01.01.1980".to_string());
        m.insert("taxpayer_street".to_string(), "Hlavni 1".to_string());
        m.insert("taxpayer_house_number".to_string(), "1".to_string());
        m.insert("taxpayer_city".to_string(), "Praha".to_string());
        m.insert("taxpayer_postal_code".to_string(), "11000".to_string());
        m.insert("flat_rate_expenses".to_string(), "true".to_string());
        m
    }

    fn test_social_insurance() -> SocialInsuranceOverview {
        let dt = chrono::NaiveDate::from_ymd_opt(2024, 1, 1)
            .unwrap()
            .and_hms_opt(0, 0, 0)
            .unwrap();
        SocialInsuranceOverview {
            id: 1,
            year: 2024,
            filing_type: FilingType::Regular,
            total_revenue: Amount::new(1_000_000, 0),
            total_expenses: Amount::new(600_000, 0),
            tax_base: Amount::new(400_000, 0),
            assessment_base: Amount::new(200_000, 0),
            min_assessment_base: Amount::new(120_000, 0),
            final_assessment_base: Amount::new(200_000, 0),
            insurance_rate: 292,
            total_insurance: Amount::new(58_400, 0),
            prepayments: Amount::new(40_000, 0),
            difference: Amount::new(18_400, 0),
            new_monthly_prepay: Amount::new(4_867, 0),
            xml_data: Vec::new(),
            status: zfaktury_domain::FilingStatus::Draft,
            filed_at: None,
            created_at: dt,
            updated_at: dt,
        }
    }

    #[test]
    fn test_generate_valid_xml() {
        let sio = test_social_insurance();
        let settings = test_settings();
        let result = generate_social_insurance_xml(&sio, &settings).unwrap();
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

        assert!(xml_str.contains("<OSVC"));
        assert!(xml_str.contains("xmlns=\"http://schemas.cssz.cz/OSVC2025\""));
    }

    #[test]
    fn test_root_element_and_vendor() {
        let sio = test_social_insurance();
        let settings = test_settings();
        let result = generate_social_insurance_xml(&sio, &settings).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("productName=\"ZFaktury\""));
        assert!(xml_str.contains("productVersion=\"1.0\""));
    }

    #[test]
    fn test_amounts_whole_czk() {
        let sio = test_social_insurance();
        let settings = test_settings();
        let result = generate_social_insurance_xml(&sio, &settings).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        // revenue = 1_000_000, expenses = 600_000
        assert!(xml_str.contains(r#"<rdza h="1000000" v="600000"/>"#));
    }

    #[test]
    fn test_filing_type_codes() {
        let settings = test_settings();

        for (filing_type, expected) in [
            (FilingType::Regular, "N"),
            (FilingType::Corrective, "O"),
            (FilingType::Supplementary, "Z"),
        ] {
            let mut sio = test_social_insurance();
            sio.filing_type = filing_type;
            let result = generate_social_insurance_xml(&sio, &settings).unwrap();
            let xml_str = String::from_utf8(result).unwrap();
            assert!(
                xml_str.contains(&format!("typ=\"{expected}\"")),
                "Expected {expected}"
            );
        }
    }

    #[test]
    fn test_flat_rate_flag() {
        let sio = test_social_insurance();

        // With flat_rate_expenses = "true"
        let settings = test_settings();
        let result = generate_social_insurance_xml(&sio, &settings).unwrap();
        let xml_str = String::from_utf8(result).unwrap();
        assert!(xml_str.contains("pau=\"A\""));

        // Without flat_rate_expenses
        let mut settings_no_flat = test_settings();
        settings_no_flat.remove("flat_rate_expenses");
        let result = generate_social_insurance_xml(&sio, &settings_no_flat).unwrap();
        let xml_str = String::from_utf8(result).unwrap();
        assert!(xml_str.contains("pau=\"N\""));
    }
}
