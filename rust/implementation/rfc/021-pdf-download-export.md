# RFC-021: PDF Download & Export Features

## Status: Draft
## Author: Claude
## Date: 2026-03-22

---

## 1. Problem Statement

The Rust GPUI app cannot:
- Download invoice PDFs (generator exists in `zfaktury-gen` but no UI button)
- Export CSV files for invoices/expenses
- Export ISDOC XML for invoices
- Download tax filing XML documents
- Customize PDF settings (Route::SettingsPdf maps to StubView)

All generation code exists in `zfaktury-gen`:
- `generate_invoice_pdf()` -> `Vec<u8>` (Typst-based)
- `export_invoices_csv()` / `export_expenses_csv()` -> `Vec<u8>`
- ISDOC XML generation
- QR payment code generation (SPAYD format)
- Tax XML generation (VAT, income tax, social/health insurance, VIES)

## 2. Goals

- Add PDF download button to invoice detail view
- Add ISDOC export button to invoice detail view
- Add QR payment code display to invoice detail view
- Create SettingsPdfView for PDF customization
- Add CSV export buttons to reports view
- Add XML download buttons to tax detail views
- Use `rfd` crate for native file save dialogs

## 3. File Save Strategy

Since GPUI has no built-in save dialog, use `rfd` crate:

```rust
cx.spawn(async move |this, cx| {
    // Generate PDF bytes
    let pdf_bytes = generate_invoice_pdf(&invoice, &supplier, &settings)?;

    // Native save dialog
    let file = rfd::AsyncFileDialog::new()
        .set_file_name(&format!("faktura-{}.pdf", invoice.invoice_number))
        .add_filter("PDF", &["pdf"])
        .set_title("Ulozit fakturu jako PDF")
        .save_file()
        .await;

    if let Some(file) = file {
        std::fs::write(file.path(), &pdf_bytes)?;
        this.update(&mut cx, |this, cx| {
            this.success_message = Some("PDF ulozeno".to_string());
            cx.notify();
        })?;
    }
}).detach();
```

Alternative: Save to temp file and open with system viewer:
```rust
let tmp = std::env::temp_dir().join(format!("faktura-{}.pdf", number));
std::fs::write(&tmp, &pdf_bytes)?;
std::process::Command::new("xdg-open").arg(&tmp).spawn()?;
```

## 4. Invoice Detail PDF Button

### 4.1 Changes to `views/invoice_detail.rs`

Add to existing action bar (alongside Odeslat, Uhradit, etc.):

```rust
// New fields in InvoiceDetailView
pdf_generating: bool,

// In render action bar
Button::new("download-pdf", "Stahnout PDF")
    .variant(ButtonVariant::Secondary)
    .disabled(self.pdf_generating)
    .on_click(cx.listener(|this, _, _, cx| {
        this.download_pdf(cx);
    }))
```

**download_pdf method:**
1. Set `pdf_generating = true`, notify
2. Spawn async task:
   a. Load supplier info from `settings_service.get_supplier_info()`
   b. Load PDF settings from `settings_service.get_pdf_settings()`
   c. Call `zfaktury_gen::pdf::generate_invoice_pdf(&invoice, &supplier, &pdf_settings)`
   d. Open `rfd::AsyncFileDialog` save dialog with default filename
   e. Write bytes to selected path
3. On success: set `pdf_generating = false`, show toast
4. On error: set `pdf_generating = false`, show error

### 4.2 ISDOC Export Button

Same pattern as PDF but with ISDOC generator:

```rust
Button::new("export-isdoc", "ISDOC")
    .variant(ButtonVariant::Ghost)
    .on_click(cx.listener(|this, _, _, cx| {
        this.export_isdoc(cx);
    }))
```

Generate ISDOC XML -> save dialog with `.isdoc` extension.

### 4.3 QR Payment Code

Display inline in invoice detail (below payment info section):

```rust
// Generate QR code
let spayd = generate_spayd_string(&iban, amount, &vs, &invoice_number);
let qr_png = generate_qr_png(&spayd, 200)?;

// Render as GPUI image
// Note: GPUI can render images via gpui::img() or SharedString data URL
// May need to convert PNG bytes to a renderable format
```

If inline image rendering is too complex in GPUI, alternative: "Zobrazit QR" button that saves QR PNG to temp file and opens with system viewer.

## 5. New File: `views/settings_pdf.rs`

```rust
pub struct SettingsPdfView {
    settings_service: Arc<SettingsService>,

    // Form fields
    accent_color: Entity<TextInput>,      // Hex color "#2563eb"
    footer_text: Entity<TextInput>,       // Custom footer
    show_qr: Entity<Checkbox>,            // Show QR payment code
    show_bank_details: Entity<Checkbox>,  // Show bank details

    // State
    editing: bool,
    saving: bool,
    error: Option<String>,
    success: Option<String>,

    // Current values (for view mode)
    current_settings: PdfRenderSettings,
}
```

**UI Layout:**
1. Header: "Nastaveni PDF sablony"
2. View mode: Display current settings as key-value pairs
3. Edit mode (toggle with "Upravit" button):
   - Accent color: TextInput with hex value + color preview div
   - Footer text: TextInput
   - Show QR: Checkbox
   - Show bank details: Checkbox
4. Action buttons:
   - "Ulozit" -> `settings_service.save_pdf_settings(settings)`
   - "Nahled" -> generate sample PDF and open with xdg-open
   - "Zrusit" -> revert to saved values

**Save flow:**
1. Read input values
2. Validate hex color format
3. Call `settings_service.save_pdf_settings(PdfRenderSettings { ... })`
4. Reload settings, exit edit mode
5. Show success toast

## 6. Reports CSV Export

### 6.1 Changes to `views/reports.rs`

Add export buttons to reports header area:

```rust
// In render method, header section
div().flex().gap_2().children([
    Button::new("export-invoices-csv", "Export faktur (CSV)")
        .variant(ButtonVariant::Secondary)
        .on_click(cx.listener(|this, _, _, cx| {
            this.export_invoices_csv(cx);
        })),
    Button::new("export-expenses-csv", "Export nakladu (CSV)")
        .variant(ButtonVariant::Secondary)
        .on_click(cx.listener(|this, _, _, cx| {
            this.export_expenses_csv(cx);
        })),
])
```

**export_invoices_csv method:**
1. Load all invoices for selected year from `invoice_service.list(year)`
2. Call `zfaktury_gen::csv::export_invoices_csv(&invoices)`
3. Save dialog with filename `faktury-{year}.csv`
4. Write bytes

Same pattern for expenses.

## 7. Tax XML Download

### 7.1 Changes to Tax Detail Views

Add "Stahnout XML" button to these existing views:
- `views/tax_income_detail.rs` - Income tax return XML
- `views/tax_social_detail.rs` - Social insurance overview XML
- `views/tax_health_detail.rs` - Health insurance overview XML
- `views/vat_return_detail.rs` - VAT return XML
- `views/vat_control_detail.rs` - VAT control statement XML
- `views/vies_detail.rs` - VIES summary XML

Pattern for each:

```rust
Button::new("download-xml", "Stahnout XML")
    .variant(ButtonVariant::Secondary)
    .disabled(self.xml_data.is_none())
    .on_click(cx.listener(|this, _, _, cx| {
        this.download_xml(cx);
    }))
```

**download_xml method:**
1. Get XML bytes from entity (already generated by "Generovat XML" action)
2. Save dialog with appropriate filename (e.g., `dpfo-2025.xml`, `dph-2025-01.xml`)
3. Write bytes

## 8. Modified Files Summary

| File | Change |
|------|--------|
| `views/settings_pdf.rs` | NEW - PDF settings form |
| `views/mod.rs` | Add `pub mod settings_pdf;` |
| `root.rs` | Map `Route::SettingsPdf` to SettingsPdfView, add ContentView variant |
| `views/invoice_detail.rs` | Add PDF/ISDOC/QR buttons and handler methods |
| `views/reports.rs` | Add CSV export buttons |
| `views/tax_income_detail.rs` | Add XML download button |
| `views/tax_social_detail.rs` | Add XML download button |
| `views/tax_health_detail.rs` | Add XML download button |
| `views/vat_return_detail.rs` | Add XML download button |
| `views/vat_control_detail.rs` | Add XML download button |
| `views/vies_detail.rs` | Add XML download button |
| `Cargo.toml` (zfaktury-app) | Add `rfd = "0.15"` dependency |

## 9. Dependencies

```toml
# Cargo.toml additions for zfaktury-app
[dependencies]
rfd = "0.15"                    # Native file dialogs
zfaktury-gen = { path = "../zfaktury-gen" }  # Already exists, ensure PDF + CSV + ISDOC modules available
```

## 10. Service Dependencies

Invoice detail needs access to:
- `InvoiceService` (already has)
- `SettingsService` (for supplier info + PDF settings) - may need to add to InvoiceDetailView

Reports view needs:
- `InvoiceService` (for CSV export data)
- `ExpenseService` (for CSV export data)

Settings PDF view needs:
- `SettingsService` (already available in AppServices)

## 11. Error Handling

| Error | Handling |
|-------|----------|
| PDF generation failure | Error toast "Chyba pri generovani PDF" |
| File save dialog cancelled | No action, return silently |
| File write failure | Error toast with OS error message |
| Settings save failure | Error toast, keep form data |
| Missing supplier info | Warning: "Nejdrive vyplnte firemni udaje v Nastaveni" |
| Missing IBAN for QR | QR button disabled, tooltip "Vyplnte IBAN v nastaveni" |

## 12. Implementation Order

1. Add `rfd` dependency to Cargo.toml
2. Create `settings_pdf.rs` and wire route
3. Add PDF download button to `invoice_detail.rs`
4. Add ISDOC button to `invoice_detail.rs`
5. Add CSV export to `reports.rs`
6. Add XML download to tax detail views
7. QR code display (may require image rendering research)

## 13. Future Enhancements

- PDF preview within GPUI (requires image rendering of PDF pages)
- Email sending with PDF attachment (requires SMTP client)
- Batch PDF generation (all invoices for a period)
- Logo upload for PDF customization
- Print dialog integration
