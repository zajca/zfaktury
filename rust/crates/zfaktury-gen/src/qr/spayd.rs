//! SPAYD (Short Payment Descriptor) generation and QR code rendering.
//!
//! SPD format: `SPD*1.0*ACC:{IBAN}*AM:{amount}*CC:CZK*X-VS:{vs}*MSG:{message}`

use zfaktury_domain::Amount;

use crate::Result;

/// Generate a SPAYD payment string.
///
/// The amount is formatted as CZK (halere / 100). Whole amounts have no decimals,
/// fractional amounts have 2 decimal places.
pub fn generate_spayd_string(iban: &str, amount: Amount, vs: &str, message: &str) -> String {
    let mut parts = vec![
        "SPD*1.0".to_string(),
        format!("ACC:{iban}"),
        format!("AM:{}", format_spayd_amount(amount)),
        "CC:CZK".to_string(),
    ];

    if !vs.is_empty() {
        parts.push(format!("X-VS:{vs}"));
    }

    if !message.is_empty() {
        parts.push(format!("MSG:{message}"));
    }

    parts.join("*")
}

/// Generate a QR code PNG image from any string data.
pub fn generate_qr_png(data: &str, size: u32) -> Result<Vec<u8>> {
    use image::Luma;
    use qrcode::QrCode;
    use std::io::Cursor;

    let code = QrCode::new(data.as_bytes())
        .map_err(|e| crate::GenError::InvalidInput(format!("QR code generation failed: {e}")))?;

    let image = code
        .render::<Luma<u8>>()
        .quiet_zone(true)
        .min_dimensions(size, size)
        .build();

    let mut buf = Cursor::new(Vec::new());
    image.write_to(&mut buf, image::ImageFormat::Png)?;

    Ok(buf.into_inner())
}

/// Format an Amount for SPAYD: whole amounts as integer, fractional with 2 decimals.
fn format_spayd_amount(amount: Amount) -> String {
    let halere = amount.halere();
    let whole = halere / 100;
    let fraction = (halere % 100).abs();
    if fraction == 0 {
        format!("{whole}")
    } else {
        format!("{whole}.{fraction:02}")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_spayd_basic() {
        let result = generate_spayd_string(
            "CZ6508000000192000145399",
            Amount::new(15_000, 0),
            "20240001",
            "FV20240001",
        );
        assert!(result.starts_with("SPD*1.0*"));
        assert!(result.contains("ACC:CZ6508000000192000145399"));
        assert!(result.contains("AM:15000"));
        assert!(result.contains("CC:CZK"));
        assert!(result.contains("X-VS:20240001"));
        assert!(result.contains("MSG:FV20240001"));
    }

    #[test]
    fn test_spayd_fractional_amount() {
        let result =
            generate_spayd_string("CZ6508000000192000145399", Amount::new(100, 50), "", "");
        assert!(result.contains("AM:100.50"));
    }

    #[test]
    fn test_spayd_whole_amount_no_decimals() {
        let result =
            generate_spayd_string("CZ6508000000192000145399", Amount::new(1000, 0), "", "");
        assert!(result.contains("AM:1000"));
        // Should NOT have "AM:1000.00".
        assert!(!result.contains("AM:1000.00"));
    }

    #[test]
    fn test_spayd_empty_optional_fields() {
        let result = generate_spayd_string("CZ6508000000192000145399", Amount::new(500, 0), "", "");
        assert!(!result.contains("X-VS:"));
        assert!(!result.contains("MSG:"));
    }

    #[test]
    fn test_qr_png_valid() {
        let png_data =
            generate_qr_png("SPD*1.0*ACC:CZ6508000000192000145399*AM:1000*CC:CZK", 200).unwrap();

        // PNG magic bytes.
        assert!(png_data.len() > 8);
        assert_eq!(&png_data[0..4], &[0x89, 0x50, 0x4E, 0x47]);
    }

    #[test]
    fn test_qr_png_empty_data() {
        // Even empty data should produce a valid QR.
        let png_data = generate_qr_png("test", 100).unwrap();
        assert!(!png_data.is_empty());
        assert_eq!(&png_data[0..4], &[0x89, 0x50, 0x4E, 0x47]);
    }
}
