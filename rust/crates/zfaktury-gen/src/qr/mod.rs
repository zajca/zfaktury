//! QR code and SPAYD payment string generation.

mod spayd;

pub use spayd::{generate_qr_png, generate_spayd_string};
