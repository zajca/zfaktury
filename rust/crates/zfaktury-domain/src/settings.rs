/// PDF rendering settings.
#[derive(Debug, Clone, Default)]
pub struct PDFSettings {
    pub accent_color: Option<String>,
    pub font_size: Option<f32>,
    pub footer_text: Option<String>,
    pub logo_path: Option<String>,
    pub show_bank_details: bool,
    pub show_qr: bool,
}

// Setting key constants.
pub const SETTING_BANK_ACCOUNT: &str = "bank_account";
pub const SETTING_BANK_CODE: &str = "bank_code";
pub const SETTING_CITY: &str = "city";
pub const SETTING_COMPANY_NAME: &str = "company_name";
pub const SETTING_CSSZ_CODE: &str = "cssz_code";
pub const SETTING_DIC: &str = "dic";
pub const SETTING_EMAIL: &str = "email";
pub const SETTING_EMAIL_ATTACH_ISDOC: &str = "email_attach_isdoc";
pub const SETTING_EMAIL_ATTACH_PDF: &str = "email_attach_pdf";
pub const SETTING_EMAIL_BODY_TPL: &str = "email_body_tpl";
pub const SETTING_EMAIL_SUBJECT_TPL: &str = "email_subject_tpl";
pub const SETTING_FINANCNI_URAD_CODE: &str = "financni_urad_code";
pub const SETTING_FIRST_NAME: &str = "first_name";
pub const SETTING_HEALTH_INS_CODE: &str = "health_ins_code";
pub const SETTING_HOUSE_NUMBER: &str = "house_number";
pub const SETTING_IBAN: &str = "iban";
pub const SETTING_ICO: &str = "ico";
pub const SETTING_LAST_NAME: &str = "last_name";
pub const SETTING_OKEC: &str = "okec";
pub const SETTING_PDF_ACCENT_COLOR: &str = "pdf_accent_color";
pub const SETTING_PDF_FONT_SIZE: &str = "pdf_font_size";
pub const SETTING_PDF_FOOTER_TEXT: &str = "pdf_footer_text";
pub const SETTING_PDF_LOGO_PATH: &str = "pdf_logo_path";
pub const SETTING_PDF_SHOW_BANK_DETAILS: &str = "pdf_show_bank_details";
pub const SETTING_PDF_SHOW_QR: &str = "pdf_show_qr";
pub const SETTING_PHONE: &str = "phone";
pub const SETTING_PRAC_UFO: &str = "prac_ufo";
pub const SETTING_STREET: &str = "street";
pub const SETTING_SWIFT: &str = "swift";
pub const SETTING_UFO_CODE: &str = "ufo_code";
pub const SETTING_VAT_REGISTERED: &str = "vat_registered";
pub const SETTING_ZIP: &str = "zip";
