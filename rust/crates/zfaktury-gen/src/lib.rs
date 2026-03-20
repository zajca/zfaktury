//! Document generation crate for ZFaktury.
//!
//! Covers:
//! - VAT XML (DPHDP3, DPHKH1, DPHSHV)
//! - Income tax XML (DPFDP5)
//! - Social insurance XML (CSSZ)
//! - Health insurance XML (placeholder)
//! - ISDOC 6.0.2 XML
//! - QR/SPAYD payment codes
//! - CSV export

pub mod csv;
pub mod isdoc;
pub mod qr;
pub mod xml;

use thiserror::Error;

/// Errors produced by document generation.
#[derive(Debug, Error)]
pub enum GenError {
    #[error("invalid input: {0}")]
    InvalidInput(String),

    #[error("XML serialization error: {0}")]
    Xml(#[from] quick_xml::Error),

    #[error("XML encoding error: {0}")]
    XmlAttr(#[from] quick_xml::events::attributes::AttrError),

    #[error("CSV error: {0}")]
    Csv(#[from] ::csv::Error),

    #[error("image encoding error: {0}")]
    Image(#[from] image::ImageError),

    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
}

pub type Result<T> = std::result::Result<T, GenError>;
