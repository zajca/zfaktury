//! Types for ISDOC generation.

/// Supplier (OSVC) details needed for ISDOC generation.
/// These are loaded from application settings.
pub struct SupplierInfo {
    pub company_name: String,
    pub ico: String,
    pub dic: String,
    pub street: String,
    pub city: String,
    pub zip: String,
    pub email: String,
    pub phone: String,
    pub bank_account: String,
    pub bank_code: String,
    pub iban: String,
    pub swift: String,
}
