use chrono::NaiveDateTime;
use std::fmt;

/// Contact type enum.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ContactType {
    Company,
    Individual,
}

impl fmt::Display for ContactType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ContactType::Company => write!(f, "company"),
            ContactType::Individual => write!(f, "individual"),
        }
    }
}

/// EU member state country codes for DIC validation.
const EU_COUNTRY_CODES: &[&str] = &[
    "AT", "BE", "BG", "HR", "CY", "DE", "DK", "EE", "ES", "FI", "FR", "GR", "EL", "HU", "IE",
    "IT", "LT", "LU", "LV", "MT", "NL", "PL", "PT", "RO", "SE", "SI", "SK",
];

/// A business contact (customer or vendor).
#[derive(Debug, Clone)]
pub struct Contact {
    pub id: i64,
    pub contact_type: ContactType,
    pub name: String,

    // Czech business identifiers
    pub ico: String,
    pub dic: String,

    // Address
    pub street: String,
    pub city: String,
    pub zip: String,
    pub country: String,

    // Contact info
    pub email: String,
    pub phone: String,
    pub web: String,

    // Bank details
    pub bank_account: String,
    pub bank_code: String,
    pub iban: String,
    pub swift: String,

    // Settings
    pub payment_terms_days: i32,
    pub tags: String,
    pub notes: String,

    // Flags
    pub is_favorite: bool,

    /// Set when the contact is flagged as an unreliable VAT payer.
    pub vat_unreliable_at: Option<NaiveDateTime>,

    // Timestamps
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
    pub deleted_at: Option<NaiveDateTime>,
}

impl Contact {
    /// Returns the 2-letter country prefix from the DIC (e.g. "CZ" from "CZ12345678").
    /// Returns empty string if DIC is too short.
    pub fn dic_country_code(&self) -> &str {
        if self.dic.len() < 2 { "" } else { &self.dic[..2] }
    }

    /// Returns true if the contact has a non-CZ EU DIC.
    pub fn is_eu_partner(&self) -> bool {
        let code = self.dic_country_code();
        if code.is_empty() || code == "CZ" {
            return false;
        }
        EU_COUNTRY_CODES.contains(&code)
    }

    /// Returns true if the contact has a Czech DIC.
    pub fn has_cz_dic(&self) -> bool {
        self.dic_country_code() == "CZ"
    }
}

/// Filtering options for listing contacts.
#[derive(Debug, Clone, Default)]
pub struct ContactFilter {
    pub search: String,
    pub contact_type: Option<ContactType>,
    pub favorite: Option<bool>,
    pub limit: i32,
    pub offset: i32,
}
