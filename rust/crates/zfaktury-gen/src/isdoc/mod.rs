//! ISDOC 6.0.2 XML invoice generation.

mod generator;
mod types;

pub use generator::generate_isdoc;
pub use types::SupplierInfo;
