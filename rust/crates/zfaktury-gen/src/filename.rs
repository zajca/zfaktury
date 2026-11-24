//! Names of generated files, matching the Go `internal/filename` package:
//! lowercase ASCII, hyphen-separated, `<kind>-<period>.<ext>`.

/// Lowercases an invoice number, replaces every character outside `[a-z0-9-]`
/// with a hyphen, collapses repeated hyphens and trims them.
pub fn sanitize_invoice_number(number: &str) -> String {
    let mut out = String::with_capacity(number.len());
    let mut last_hyphen = true;
    for c in number.chars() {
        if c.is_ascii_alphanumeric() {
            out.push(c.to_ascii_lowercase());
            last_hyphen = false;
        } else if !last_hyphen {
            out.push('-');
            last_hyphen = true;
        }
    }
    out.trim_end_matches('-').to_string()
}

fn invoice_name(number: &str, ext: &str) -> String {
    let sanitized = sanitize_invoice_number(number);
    if sanitized.is_empty() {
        format!("faktura.{ext}")
    } else {
        format!("faktura-{sanitized}.{ext}")
    }
}

/// Invoice PDF name, e.g. `faktura-2025001.pdf`.
pub fn invoice_pdf(number: &str) -> String {
    invoice_name(number, "pdf")
}

/// Invoice ISDOC name, e.g. `faktura-2025001.isdoc`.
pub fn invoice_isdoc(number: &str) -> String {
    invoice_name(number, "isdoc")
}

/// Yearly invoice CSV name, e.g. `faktury-2025.csv`.
pub fn invoices_csv(year: i32) -> String {
    format!("faktury-{year:04}.csv")
}

/// Yearly expense CSV name, e.g. `vydaje-2025.csv`.
pub fn expenses_csv(year: i32) -> String {
    format!("vydaje-{year:04}.csv")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn invoice_names() {
        assert_eq!(invoice_pdf("2025001"), "faktura-2025001.pdf");
        assert_eq!(invoice_isdoc("FV/2025/001"), "faktura-fv-2025-001.isdoc");
        assert_eq!(invoice_pdf("  ../x--y "), "faktura-x-y.pdf");
        assert_eq!(invoice_pdf("///"), "faktura.pdf");
    }

    #[test]
    fn csv_names() {
        assert_eq!(invoices_csv(2025), "faktury-2025.csv");
        assert_eq!(expenses_csv(2025), "vydaje-2025.csv");
    }
}
