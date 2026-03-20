//! Invoice PDF generation using typst-bake.
//!
//! Compiles an embedded typst template with invoice data to produce a PDF.

mod formatting;

use std::collections::BTreeMap;

use typst_bake::{IntoDict, IntoValue};
use zfaktury_domain::{Amount, Invoice, InvoiceStatus, InvoiceType};

use crate::Result;
use crate::qr::{generate_qr_png, generate_spayd_string};

/// Supplier information for PDF generation.
pub struct SupplierInfo {
    pub name: String,
    pub ico: String,
    pub dic: String,
    pub vat_registered: bool,
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

/// PDF customization settings.
pub struct PdfRenderSettings {
    pub accent_color: String,
    pub footer_text: String,
    pub show_qr: bool,
    pub show_bank_details: bool,
}

impl Default for PdfRenderSettings {
    fn default() -> Self {
        Self {
            accent_color: "#2563eb".to_string(),
            footer_text: String::new(),
            show_qr: true,
            show_bank_details: true,
        }
    }
}

/// A single line item passed to the typst template.
#[derive(IntoValue)]
struct TypstItem {
    index: String,
    description: String,
    quantity: String,
    unit: String,
    unit_price: String,
    vat_rate: String,
    vat_amount: String,
    total: String,
}

/// A VAT group row passed to the typst template.
#[derive(IntoValue)]
struct TypstVatGroup {
    rate: String,
    base: String,
    vat: String,
}

/// All data passed to the typst template via `sys.inputs`.
#[derive(IntoValue, IntoDict)]
struct TypstInputs {
    // Header
    type_label: String,
    invoice_number: String,
    status_label: String,

    // Dates
    issue_date: String,
    due_date: String,
    delivery_date: String,

    // Accent color
    accent_color: String,

    // Supplier
    supplier_name: String,
    supplier_ico: String,
    supplier_dic: String,
    supplier_street: String,
    supplier_city: String,
    supplier_zip: String,
    supplier_email: String,
    supplier_phone: String,

    // Customer
    has_customer: String,
    customer_name: String,
    customer_ico: String,
    customer_dic: String,
    customer_street: String,
    customer_city: String,
    customer_zip: String,

    // Items
    items: Vec<TypstItem>,

    // VAT groups
    vat_groups: Vec<TypstVatGroup>,

    // Totals
    subtotal_czk: String,
    vat_czk: String,
    total_czk: String,

    // Payment
    show_bank_details: String,
    bank_account_full: String,
    iban: String,
    swift: String,
    variable_symbol: String,
    constant_symbol: String,

    // QR code
    has_qr: String,
    qr_png: Vec<u8>,

    // Footer
    vat_note: String,
    footer_text: String,
}

/// Generate an invoice PDF as bytes.
///
/// Compiles the embedded typst template with invoice data and returns the raw PDF.
pub fn generate_invoice_pdf(
    invoice: &Invoice,
    supplier: &SupplierInfo,
    settings: &PdfRenderSettings,
) -> Result<Vec<u8>> {
    let inputs = build_inputs(invoice, supplier, settings)?;

    let pdf = typst_bake::document!("invoice.typ")
        .with_inputs(inputs)
        .to_pdf()
        .map_err(|e| crate::GenError::InvalidInput(format!("typst compilation failed: {e}")))?;

    Ok(pdf)
}

/// Build the TypstInputs struct from domain data.
fn build_inputs(
    invoice: &Invoice,
    supplier: &SupplierInfo,
    settings: &PdfRenderSettings,
) -> Result<TypstInputs> {
    let type_label = invoice_type_label(&invoice.invoice_type);
    let status_label = status_label(&invoice.status);

    // Format dates as DD.MM.YYYY.
    let issue_date = invoice.issue_date.format("%d.%m.%Y").to_string();
    let due_date = invoice.due_date.format("%d.%m.%Y").to_string();
    let delivery_date = invoice.delivery_date.format("%d.%m.%Y").to_string();

    // Build items.
    let items: Vec<TypstItem> = invoice
        .items
        .iter()
        .enumerate()
        .map(|(i, item)| {
            let item_subtotal =
                Amount::from_halere(item.quantity.halere() * item.unit_price.halere() / 100);
            let item_total = item_subtotal + item.vat_amount;

            TypstItem {
                index: format!("{}", i + 1),
                description: item.description.clone(),
                quantity: formatting::format_quantity(item.quantity),
                unit: item.unit.clone(),
                unit_price: formatting::format_czech_amount(item.unit_price),
                vat_rate: format!("{}%", item.vat_rate_percent),
                vat_amount: formatting::format_czech_amount(item.vat_amount),
                total: formatting::format_czech_amount(item_total),
            }
        })
        .collect();

    // Build VAT groups, sorted by rate.
    let mut vat_map: BTreeMap<i32, (Amount, Amount)> = BTreeMap::new();
    for item in &invoice.items {
        let item_subtotal =
            Amount::from_halere(item.quantity.halere() * item.unit_price.halere() / 100);
        let entry = vat_map
            .entry(item.vat_rate_percent)
            .or_insert((Amount::ZERO, Amount::ZERO));
        entry.0 += item_subtotal;
        entry.1 += item.vat_amount;
    }

    let vat_groups: Vec<TypstVatGroup> = vat_map
        .iter()
        .map(|(rate, (base, vat))| TypstVatGroup {
            rate: format!("{}%", rate),
            base: formatting::format_czech_amount(*base),
            vat: formatting::format_czech_amount(*vat),
        })
        .collect();

    // Determine bank details (invoice overrides supplier).
    let iban = if invoice.iban.is_empty() {
        &supplier.iban
    } else {
        &invoice.iban
    };
    let swift = if invoice.swift.is_empty() {
        &supplier.swift
    } else {
        &invoice.swift
    };
    let bank_account = if invoice.bank_account.is_empty() {
        &supplier.bank_account
    } else {
        &invoice.bank_account
    };
    let bank_code = if invoice.bank_code.is_empty() {
        &supplier.bank_code
    } else {
        &invoice.bank_code
    };

    let bank_account_full = if !bank_account.is_empty() {
        if !bank_code.is_empty() {
            format!("{}/{}", bank_account, bank_code)
        } else {
            bank_account.to_string()
        }
    } else {
        String::new()
    };

    // Generate QR code if enabled and IBAN available.
    let (has_qr, qr_png) = if settings.show_qr && !iban.is_empty() {
        let spayd = generate_spayd_string(
            iban,
            invoice.total_amount,
            &invoice.variable_symbol,
            &invoice.invoice_number,
        );
        match generate_qr_png(&spayd, 200) {
            Ok(png) => ("true".to_string(), png),
            Err(_) => ("false".to_string(), Vec::new()),
        }
    } else {
        ("false".to_string(), Vec::new())
    };

    // Customer info.
    let (has_customer, cust_name, cust_ico, cust_dic, cust_street, cust_city, cust_zip) =
        if let Some(ref c) = invoice.customer {
            (
                "true".to_string(),
                c.name.clone(),
                c.ico.clone(),
                c.dic.clone(),
                c.street.clone(),
                c.city.clone(),
                c.zip.clone(),
            )
        } else {
            (
                "false".to_string(),
                String::new(),
                String::new(),
                String::new(),
                String::new(),
                String::new(),
                String::new(),
            )
        };

    // VAT note for non-VAT-registered suppliers.
    let vat_note = if !supplier.vat_registered {
        "Subjekt není plátce DPH.".to_string()
    } else {
        String::new()
    };

    Ok(TypstInputs {
        type_label,
        invoice_number: invoice.invoice_number.clone(),
        status_label,

        issue_date,
        due_date: due_date.clone(),
        delivery_date,

        accent_color: settings.accent_color.clone(),

        supplier_name: supplier.name.clone(),
        supplier_ico: supplier.ico.clone(),
        supplier_dic: supplier.dic.clone(),
        supplier_street: supplier.street.clone(),
        supplier_city: supplier.city.clone(),
        supplier_zip: supplier.zip.clone(),
        supplier_email: supplier.email.clone(),
        supplier_phone: supplier.phone.clone(),

        has_customer,
        customer_name: cust_name,
        customer_ico: cust_ico,
        customer_dic: cust_dic,
        customer_street: cust_street,
        customer_city: cust_city,
        customer_zip: cust_zip,

        items,
        vat_groups,

        subtotal_czk: formatting::format_czech_amount_czk(invoice.subtotal_amount),
        vat_czk: formatting::format_czech_amount_czk(invoice.vat_amount),
        total_czk: formatting::format_czech_amount_czk(invoice.total_amount),

        show_bank_details: if settings.show_bank_details {
            "true".to_string()
        } else {
            "false".to_string()
        },
        bank_account_full,
        iban: iban.to_string(),
        swift: swift.to_string(),
        variable_symbol: invoice.variable_symbol.clone(),
        constant_symbol: invoice.constant_symbol.clone(),

        has_qr,
        qr_png,

        vat_note,
        footer_text: settings.footer_text.clone(),
    })
}

/// Czech label for invoice type.
fn invoice_type_label(t: &InvoiceType) -> String {
    match t {
        InvoiceType::Regular => "Faktura".to_string(),
        InvoiceType::Proforma => "Proforma faktura".to_string(),
        InvoiceType::CreditNote => "Dobropis".to_string(),
    }
}

/// Czech label for invoice status.
fn status_label(s: &InvoiceStatus) -> String {
    match s {
        InvoiceStatus::Draft => "Koncept".to_string(),
        InvoiceStatus::Sent => "Odeslaná".to_string(),
        InvoiceStatus::Paid => "Uhrazená".to_string(),
        InvoiceStatus::Overdue => "Po splatnosti".to_string(),
        InvoiceStatus::Cancelled => "Stornovaná".to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::pdf::formatting::format_czech_amount;
    use zfaktury_domain::{InvoiceItem, InvoiceStatus, InvoiceType};
    use zfaktury_testutil::builders::{ContactBuilder, InvoiceBuilder, InvoiceItemBuilder};

    fn test_supplier() -> SupplierInfo {
        SupplierInfo {
            name: "Jan Novák".to_string(),
            ico: "12345678".to_string(),
            dic: "CZ12345678".to_string(),
            vat_registered: true,
            street: "Hlavní 1".to_string(),
            city: "Praha".to_string(),
            zip: "11000".to_string(),
            email: "jan@example.com".to_string(),
            phone: "+420123456789".to_string(),
            bank_account: "123456789".to_string(),
            bank_code: "0100".to_string(),
            iban: "CZ6508000000192000145399".to_string(),
            swift: "GIBACZPX".to_string(),
        }
    }

    fn test_settings() -> PdfRenderSettings {
        PdfRenderSettings::default()
    }

    fn test_invoice_with_items() -> Invoice {
        let items = vec![
            InvoiceItemBuilder::new()
                .description("Webová stránka")
                .quantity(Amount::new(1, 0))
                .unit_price(Amount::new(15_000, 0))
                .vat_rate(21)
                .build(),
            InvoiceItemBuilder::new()
                .description("Hosting (12 měsíců)")
                .quantity(Amount::new(12, 0))
                .unit_price(Amount::new(200, 0))
                .vat_rate(21)
                .build(),
        ];
        InvoiceBuilder::new().with_items(items).build()
    }

    #[test]
    fn test_generate_basic_invoice_pdf() {
        let mut invoice = test_invoice_with_items();
        invoice.customer = Some(ContactBuilder::new().name("Acme s.r.o.").build());

        let pdf = generate_invoice_pdf(&invoice, &test_supplier(), &test_settings()).unwrap();

        // PDF magic bytes: %PDF
        assert!(
            pdf.len() > 1024,
            "PDF should be at least 1KB, got {} bytes",
            pdf.len()
        );
        assert!(
            pdf.len() < 1_000_000,
            "PDF should be under 1MB, got {} bytes",
            pdf.len()
        );
        assert_eq!(&pdf[0..5], b"%PDF-", "should start with PDF magic bytes");
    }

    #[test]
    fn test_generate_credit_note_pdf() {
        let items = vec![
            InvoiceItemBuilder::new()
                .description("Opravná položka")
                .unit_price(Amount::new(5_000, 0))
                .vat_rate(21)
                .build(),
        ];
        let invoice = InvoiceBuilder::new()
            .invoice_type(InvoiceType::CreditNote)
            .with_items(items)
            .build();

        let pdf = generate_invoice_pdf(&invoice, &test_supplier(), &test_settings()).unwrap();

        assert_eq!(&pdf[0..5], b"%PDF-");
        assert!(pdf.len() > 1024);
    }

    #[test]
    fn test_generate_proforma_pdf() {
        let items = vec![
            InvoiceItemBuilder::new()
                .description("Záloha za služby")
                .unit_price(Amount::new(10_000, 0))
                .vat_rate(21)
                .build(),
        ];
        let invoice = InvoiceBuilder::new()
            .invoice_type(InvoiceType::Proforma)
            .with_items(items)
            .build();

        let pdf = generate_invoice_pdf(&invoice, &test_supplier(), &test_settings()).unwrap();

        assert_eq!(&pdf[0..5], b"%PDF-");
        assert!(pdf.len() > 1024);
    }

    #[test]
    fn test_generate_with_qr_code() {
        let mut invoice = test_invoice_with_items();
        invoice.iban = "CZ6508000000192000145399".to_string();

        let settings_with_qr = PdfRenderSettings {
            show_qr: true,
            ..PdfRenderSettings::default()
        };
        let settings_without_qr = PdfRenderSettings {
            show_qr: false,
            ..PdfRenderSettings::default()
        };

        let pdf_with = generate_invoice_pdf(&invoice, &test_supplier(), &settings_with_qr).unwrap();
        let pdf_without =
            generate_invoice_pdf(&invoice, &test_supplier(), &settings_without_qr).unwrap();

        assert_eq!(&pdf_with[0..5], b"%PDF-");
        assert_eq!(&pdf_without[0..5], b"%PDF-");
        // QR code adds an embedded PNG image, so the PDF should be notably larger.
        assert!(
            pdf_with.len() > pdf_without.len(),
            "PDF with QR ({} bytes) should be larger than without ({} bytes)",
            pdf_with.len(),
            pdf_without.len()
        );
    }

    #[test]
    fn test_generate_without_customer() {
        let items = vec![
            InvoiceItemBuilder::new()
                .description("Konzultace")
                .unit_price(Amount::new(2_000, 0))
                .vat_rate(21)
                .build(),
        ];
        let invoice = InvoiceBuilder::new().with_items(items).build();
        // invoice.customer is None by default.

        let pdf = generate_invoice_pdf(&invoice, &test_supplier(), &test_settings()).unwrap();

        assert_eq!(&pdf[0..5], b"%PDF-");
        assert!(pdf.len() > 1024);
    }

    #[test]
    fn test_generate_non_vat_registered() {
        let mut supplier = test_supplier();
        supplier.vat_registered = false;

        let items = vec![
            InvoiceItemBuilder::new()
                .description("Služba")
                .unit_price(Amount::new(1_000, 0))
                .vat_rate(0)
                .build(),
        ];
        let invoice = InvoiceBuilder::new().with_items(items).build();

        let pdf = generate_invoice_pdf(&invoice, &supplier, &test_settings()).unwrap();

        assert_eq!(&pdf[0..5], b"%PDF-");
        assert!(pdf.len() > 1024);
    }

    #[test]
    fn test_czech_amount_formatting() {
        assert_eq!(format_czech_amount(Amount::new(1234, 56)), "1\u{a0}234,56");
        assert_eq!(format_czech_amount(Amount::new(0, 99)), "0,99");
        assert_eq!(
            format_czech_amount(Amount::new(1_000_000, 0)),
            "1\u{a0}000\u{a0}000,00"
        );
        assert_eq!(format_czech_amount(Amount::new(0, 0)), "0,00");
        assert_eq!(format_czech_amount(Amount::new(999, 0)), "999,00");
        assert_eq!(format_czech_amount(Amount::new(1000, 0)), "1\u{a0}000,00");
    }

    #[test]
    fn test_invoice_type_labels() {
        assert_eq!(invoice_type_label(&InvoiceType::Regular), "Faktura");
        assert_eq!(
            invoice_type_label(&InvoiceType::Proforma),
            "Proforma faktura"
        );
        assert_eq!(invoice_type_label(&InvoiceType::CreditNote), "Dobropis");
    }

    #[test]
    fn test_status_labels() {
        assert_eq!(status_label(&InvoiceStatus::Draft), "Koncept");
        assert_eq!(status_label(&InvoiceStatus::Sent), "Odeslaná");
        assert_eq!(status_label(&InvoiceStatus::Paid), "Uhrazená");
        assert_eq!(status_label(&InvoiceStatus::Overdue), "Po splatnosti");
        assert_eq!(status_label(&InvoiceStatus::Cancelled), "Stornovaná");
    }
}
