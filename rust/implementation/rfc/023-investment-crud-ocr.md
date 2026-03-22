# RFC-023: Investment CRUD Forms & Document OCR

**Status:** Draft
**Date:** 2026-03-21
**Crate:** `zfaktury-app` (primary), `zfaktury-app/Cargo.toml`, `app.rs`, `root.rs`

---

## 1. Problem Statement

The `TaxInvestmentsView` currently renders investment data in read-only mode. Users can see capital income entries, security transactions, and a year summary, but they cannot:

- **Add, edit, or delete** capital income entries (section 8).
- **Add, edit, or delete** security transactions (section 10).
- **Upload** broker statement documents (PDF/CSV).
- **Trigger OCR extraction** from uploaded documents to auto-populate entries.
- **Recalculate FIFO** cost basis for security transactions.

All backend services are fully implemented (`InvestmentIncomeService`, `InvestmentDocumentService`, `OcrProvider`), but the UI does not expose these capabilities. The view is 518 lines of pure display code with no form components, no mutation handlers, and no document management.

Additionally, the `OCRService` / `OcrProvider` is not wired into `AppServices`, so even if the UI existed, document extraction would have no provider to call.

---

## 2. Goals and Non-Goals

### Goals

- Add a 3-tab interface: Documents, Capital Income (section 8), Security Transactions (section 10).
- Implement full CRUD for capital income entries with inline forms.
- Implement full CRUD for security transactions with inline forms.
- Implement document upload via native file picker (`rfd` crate).
- Wire `OcrProvider` into `AppServices` so extraction can be triggered from the UI.
- Add "Extrahovat" (Extract) button per document that calls the extraction pipeline.
- Add "Prepocitat FIFO" (Recalculate FIFO) button for security transactions.
- Show computed/read-only fields (net_amount, cost_basis, computed_gain, time_test_exempt, exempt_amount).
- Keep the summary card always visible at the top.

### Non-Goals

- Custom OCR prompt tuning for investment documents (reuse existing `OcrProvider`).
- Drag-and-drop file upload (native file picker is sufficient).
- Bulk import from CSV without OCR (future RFC).
- Multi-year FIFO spanning across tax years (existing service logic handles this).
- Document preview/viewer (download to system viewer instead).

---

## 3. Proposed Solution

### 3.1 High-Level Architecture

```
TaxInvestmentsView (enhanced)
  |-- Summary card (always visible at top)
  |-- Tab bar: Dokumenty | Kapitalove prijmy (p.8) | Obchody s CP (p.10)
  |
  |-- [Tab: Dokumenty]
  |     |-- Upload button (rfd file picker)
  |     |-- Platform selector (Select)
  |     |-- Document list table
  |     |-- Per-row: status badge, "Extrahovat", "Stahnout", "Smazat" buttons
  |
  |-- [Tab: Kapitalove prijmy]
  |     |-- "Pridat zaznam" button -> inline form
  |     |-- Entries table
  |     |-- Per-row: "Upravit", "Smazat" buttons
  |
  |-- [Tab: Obchody s CP]
        |-- "Pridat transakci" button + "Prepocitat FIFO" button
        |-- Transactions table
        |-- Per-row: "Upravit", "Smazat" buttons
```

### 3.2 View State Changes

The `TaxInvestmentsView` struct grows from 7 fields to approximately 30+:

```rust
pub struct TaxInvestmentsView {
    // Services
    income_service: Arc<InvestmentIncomeService>,
    document_service: Arc<InvestmentDocumentService>,
    ocr_provider: Option<Arc<dyn OcrProvider>>,

    // Display state
    year: i32,
    loading: bool,
    error: Option<String>,
    active_tab: InvestmentTab,
    summary: Option<InvestmentYearSummary>,

    // Data
    capital_entries: Vec<CapitalIncomeEntry>,
    security_transactions: Vec<SecurityTransaction>,
    documents: Vec<InvestmentDocument>,

    // Capital income form state
    capital_form_visible: bool,
    capital_editing_id: Option<i64>,
    capital_category: Entity<Select>,
    capital_description: Entity<TextInput>,
    capital_income_date: Entity<DateInput>,
    capital_gross_amount: Entity<NumberInput>,
    capital_withheld_tax_cz: Entity<NumberInput>,
    capital_withheld_tax_foreign: Entity<NumberInput>,
    capital_country_code: Entity<TextInput>,
    capital_needs_declaring: Entity<Checkbox>,

    // Security transaction form state
    security_form_visible: bool,
    security_editing_id: Option<i64>,
    security_asset_type: Entity<Select>,
    security_asset_name: Entity<TextInput>,
    security_isin: Entity<TextInput>,
    security_transaction_type: Entity<Select>,
    security_transaction_date: Entity<DateInput>,
    security_quantity: Entity<NumberInput>,
    security_unit_price: Entity<NumberInput>,
    security_total_amount: Entity<NumberInput>,
    security_fees: Entity<NumberInput>,
    security_currency_code: Entity<Select>,
    security_exchange_rate: Entity<NumberInput>,

    // Document upload state
    document_platform: Entity<Select>,
    uploading: bool,
    extracting_doc_id: Option<i64>,

    // Confirm dialog
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    saving: bool,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum InvestmentTab {
    Documents,
    CapitalIncome,
    SecurityTransactions,
}
```

---

## 4. Tab System

### 4.1 Tab Bar Rendering

Render a horizontal tab bar below the header/year selector and above the content. Three tabs:

| Tab | Label | Internal variant |
|-----|-------|-----------------|
| 1 | Dokumenty | `InvestmentTab::Documents` |
| 2 | Kapitalove prijmy (p.8) | `InvestmentTab::CapitalIncome` |
| 3 | Obchody s CP (p.10) | `InvestmentTab::SecurityTransactions` |

Default active tab: `CapitalIncome` (matches the Go original's behavior of showing income data first).

Tab bar styling follows the existing GPUI patterns:

```rust
fn render_tab_bar(&self, cx: &mut Context<Self>) -> Div {
    div()
        .flex()
        .gap_1()
        .border_b_1()
        .border_color(rgb(ZfColors::BORDER))
        .pb_1()
        .child(self.render_tab("Dokumenty", InvestmentTab::Documents, cx))
        .child(self.render_tab("Kapitalove prijmy (p.8)", InvestmentTab::CapitalIncome, cx))
        .child(self.render_tab("Obchody s CP (p.10)", InvestmentTab::SecurityTransactions, cx))
}

fn render_tab(&self, label: &str, tab: InvestmentTab, cx: &mut Context<Self>) -> Div {
    let is_active = self.active_tab == tab;
    let (bg, text_color, border) = if is_active {
        (ZfColors::SURFACE_ACTIVE, ZfColors::TEXT_PRIMARY, ZfColors::ACCENT)
    } else {
        (ZfColors::SURFACE, ZfColors::TEXT_MUTED, ZfColors::BORDER)
    };

    div()
        .id(SharedString::from(format!("tab-{:?}", tab)))
        .cursor_pointer()
        .px_4()
        .py_2()
        .bg(rgb(bg))
        .border_b_2()
        .border_color(rgb(border))
        .rounded_t_md()
        .text_sm()
        .text_color(rgb(text_color))
        .child(label.to_string())
        .on_click(cx.listener(move |this, _: &ClickEvent, _w, cx| {
            this.active_tab = tab;
            cx.notify();
        }))
}
```

### 4.2 Content Dispatch

In the `Render` impl, after the summary card and tab bar, dispatch to the active tab's render method:

```rust
match self.active_tab {
    InvestmentTab::Documents => content = content.child(self.render_documents_tab(cx)),
    InvestmentTab::CapitalIncome => content = content.child(self.render_capital_tab(cx)),
    InvestmentTab::SecurityTransactions => content = content.child(self.render_securities_tab(cx)),
}
```

---

## 5. Documents Tab

### 5.1 Upload Flow

1. User clicks "Nahrat dokument" button.
2. `rfd::FileDialog::new()` opens a native file picker filtered to `*.pdf`, `*.csv`, `*.jpg`, `*.png`.
3. On selection, read file bytes and determine content type from extension.
4. Call `document_service.create_record(...)` with year, platform, filename, content_type.
5. Write file bytes to `{data_dir}/investment_documents/{id}_{filename}`.
6. Reload document list.

The file picker must run on a background thread since `rfd` blocks. Use `cx.background_executor().spawn(...)`:

```rust
fn upload_document(&mut self, cx: &mut Context<Self>) {
    let platform = self.document_platform.read(cx)
        .selected_value().unwrap_or("other").to_string();
    let year = self.year;
    let doc_svc = self.document_service.clone();

    self.uploading = true;
    cx.notify();

    cx.spawn(async move |this, cx| {
        let file_result = cx.background_executor().spawn(async move {
            let file = rfd::FileDialog::new()
                .add_filter("Documents", &["pdf", "csv", "jpg", "png"])
                .set_title("Vyberte dokument")
                .pick_file();
            file
        }).await;

        if let Some(path) = file_result {
            let filename = path.file_name()
                .map(|n| n.to_string_lossy().to_string())
                .unwrap_or_default();
            let content_type = match path.extension().and_then(|e| e.to_str()) {
                Some("pdf") => "application/pdf",
                Some("csv") => "text/csv",
                Some("jpg") | Some("jpeg") => "image/jpeg",
                Some("png") => "image/png",
                _ => "application/octet-stream",
            };

            let data = std::fs::read(&path).ok();
            if let Some(data) = data {
                let result = cx.background_executor().spawn(async move {
                    let mut doc = InvestmentDocument {
                        id: 0,
                        year,
                        platform: parse_platform(&platform),
                        filename,
                        content_type: content_type.to_string(),
                        storage_path: String::new(),
                        size: data.len() as i64,
                        extraction_status: ExtractionStatus::Pending,
                        extraction_error: String::new(),
                        created_at: chrono::Local::now().naive_local(),
                        updated_at: chrono::Local::now().naive_local(),
                    };
                    doc_svc.create_record(&mut doc)?;
                    // Write file to storage
                    let storage_dir = format!("{}/investment_documents", doc_svc.data_dir());
                    std::fs::create_dir_all(&storage_dir).ok();
                    let storage_path = format!("{}/{}_{}", storage_dir, doc.id, doc.filename);
                    std::fs::write(&storage_path, &data).ok();
                    Ok::<(), DomainError>(())
                }).await;

                this.update(cx, |this, cx| {
                    this.uploading = false;
                    if let Err(e) = result {
                        this.error = Some(format!("Chyba pri nahravani: {e}"));
                    }
                    this.load_data(cx);
                    cx.notify();
                }).ok();
            }
        } else {
            this.update(cx, |this, cx| {
                this.uploading = false;
                cx.notify();
            }).ok();
        }
    }).detach();
}
```

### 5.2 Platform Selector

A `Select` component with options:

| Value | Label |
|-------|-------|
| `portu` | Portu |
| `zonky` | Zonky |
| `trading212` | Trading 212 |
| `revolut` | Revolut |
| `other` | Ostatni |

### 5.3 Document List Table

Columns:

| Column | Width | Content |
|--------|-------|---------|
| Nazev souboru | flex-1 | `doc.filename` |
| Platforma | w-28 | Platform label |
| Stav | w-24 | Status badge (pending/extracted/failed) |
| Velikost | w-20 | File size formatted |
| Akce | w-48 | Action buttons |

Status badge colors:
- `Pending` -> `STATUS_YELLOW` / `STATUS_YELLOW_BG`, label "Cekajici"
- `Extracted` -> `STATUS_GREEN` / `STATUS_GREEN_BG`, label "Extrahovano"
- `Failed` -> `STATUS_RED` / `STATUS_RED_BG`, label "Chyba"

### 5.4 Per-Document Actions

Three buttons per row:

1. **Extrahovat** (Extract) -- Visible only when status is `Pending` or `Failed`. Triggers OCR extraction.
2. **Stahnout** (Download) -- Opens the file using `open::that()` or copies to a user-chosen path.
3. **Smazat** (Delete) -- Shows `ConfirmDialog`, then calls `document_service.delete(id)`. This also cascades deletes to linked capital/security entries.

### 5.5 Extraction Flow

```rust
fn extract_document(&mut self, doc_id: i64, cx: &mut Context<Self>) {
    let doc_svc = self.document_service.clone();
    let ocr = self.ocr_provider.clone();
    self.extracting_doc_id = Some(doc_id);
    cx.notify();

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            let ocr = ocr.ok_or(DomainError::InvalidInput)?; // OCR not configured
            let doc = doc_svc.get_by_id(doc_id)?;
            let file_path = format!("{}/investment_documents/{}_{}",
                doc_svc.data_dir(), doc.id, doc.filename);
            let data = std::fs::read(&file_path)
                .map_err(|_| DomainError::NotFound)?;

            let ocr_result = ocr.process_image(&data, &doc.content_type)
                .map_err(|e| DomainError::ExternalError(format!("{e}")))?;

            // The extraction service would parse OCR results and create
            // CapitalIncomeEntry / SecurityTransaction records linked to doc_id.
            // For now, update the document status.
            // Real implementation needs an InvestmentExtractionService
            // that maps OCR output to investment domain types.

            Ok::<(), DomainError>(())
        }).await;

        this.update(cx, |this, cx| {
            this.extracting_doc_id = None;
            if let Err(e) = result {
                this.error = Some(format!("Chyba pri extrakci: {e}"));
            }
            this.load_data(cx);
            cx.notify();
        }).ok();
    }).detach();
}
```

**Note:** The existing `OcrProvider` is designed for invoice extraction (returns `OCRResult` with vendor_name, invoice_number, etc.). Investment document extraction needs a **new system prompt** and **new response type** tailored to broker statements. This is covered in Section 8 (OCR Wiring).

---

## 6. Capital Income Tab (Section 8)

### 6.1 Table Layout

Columns (same as existing, plus action column):

| Column | Width | Content |
|--------|-------|---------|
| Popis | flex-1 | description |
| Kategorie | w-24 | Category label (Czech) |
| Datum | w-24 | income_date formatted |
| Brutto | w-28 | gross_amount (right-aligned) |
| Dan CZ | w-24 | withheld_tax_cz (right-aligned) |
| Dan zahr. | w-24 | withheld_tax_foreign (right-aligned) |
| Netto | w-28 | net_amount (right-aligned, bold) |
| Zeme | w-12 | country_code |
| Akce | w-24 | Edit/Delete buttons |

### 6.2 Add Form

Triggered by "Pridat zaznam" button. Sets `capital_form_visible = true` and `capital_editing_id = None`.

The form renders inline above the table (inside the same card):

```
+----------------------------------------------+
| Novy kapitalovy prijem                        |
|                                               |
| [Kategorie v]  [Popis____________]            |
| [Datum____]  [Brutto____]  [Dan CZ____]       |
| [Dan zahr.____]  [Zeme__]  [x] Pridat do DP   |
|                                               |
|              [Zrusit]  [Ulozit]               |
+----------------------------------------------+
```

Form fields:

| Field | Component | Validation |
|-------|-----------|------------|
| `category` | `Select` | Required. Options: Dividenda CZ, Dividenda zahranicni, Urok, Kupon, Fond, Ostatni |
| `description` | `TextInput` | Required, non-empty |
| `income_date` | `DateInput` | Required, valid date |
| `gross_amount` | `NumberInput` | Required, > 0 |
| `withheld_tax_cz` | `NumberInput` | Optional, defaults to 0 |
| `withheld_tax_foreign` | `NumberInput` | Optional, defaults to 0 |
| `country_code` | `TextInput` | Optional, 2-letter ISO code |
| `needs_declaring` | `Checkbox` | Default: true. Label: "Pridat do danoveho priznani" |

Category select options:

| Value | Label |
|-------|-------|
| `dividend_cz` | Dividenda CZ |
| `dividend_foreign` | Dividenda zahranicni |
| `interest` | Urok |
| `coupon` | Kupon |
| `fund_distribution` | Distribuce fondu |
| `other` | Ostatni |

### 6.3 Save Capital Entry

```rust
fn save_capital_entry(&mut self, cx: &mut Context<Self>) {
    // 1. Read all form fields
    let category = parse_capital_category(
        self.capital_category.read(cx).selected_value().unwrap_or("")
    );
    let description = self.capital_description.read(cx).value().to_string();
    let income_date = self.capital_income_date.read(cx).iso_value().to_string();
    let gross = self.capital_gross_amount.read(cx).to_amount().unwrap_or(Amount::ZERO);
    let tax_cz = self.capital_withheld_tax_cz.read(cx).to_amount().unwrap_or(Amount::ZERO);
    let tax_foreign = self.capital_withheld_tax_foreign.read(cx).to_amount().unwrap_or(Amount::ZERO);
    let country = self.capital_country_code.read(cx).value().to_string();
    let needs_declaring = self.capital_needs_declaring.read(cx).is_checked();

    // 2. Validate
    if description.trim().is_empty() {
        self.error = Some("Zadejte popis.".into());
        cx.notify();
        return;
    }
    let parsed_date = NaiveDate::parse_from_str(&income_date, "%Y-%m-%d");
    if parsed_date.is_err() {
        self.error = Some("Neplatne datum.".into());
        cx.notify();
        return;
    }
    if gross == Amount::ZERO {
        self.error = Some("Zadejte castku brutto.".into());
        cx.notify();
        return;
    }

    // 3. Build domain struct
    let now = chrono::Local::now().naive_local();
    let mut entry = CapitalIncomeEntry {
        id: self.capital_editing_id.unwrap_or(0),
        year: self.year,
        document_id: None,
        category,
        description,
        income_date: parsed_date.unwrap(),
        gross_amount: gross,
        withheld_tax_cz: tax_cz,
        withheld_tax_foreign: tax_foreign,
        country_code: country,
        needs_declaring,
        net_amount: Amount::ZERO, // computed by service
        created_at: now,
        updated_at: now,
    };

    // 4. Save via service
    self.saving = true;
    self.error = None;
    cx.notify();

    let service = self.income_service.clone();
    let is_edit = self.capital_editing_id.is_some();

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            if is_edit {
                service.update_capital_entry(&mut entry)?;
            } else {
                service.create_capital_entry(&mut entry)?;
            }
            Ok::<(), DomainError>(())
        }).await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => {
                    this.capital_form_visible = false;
                    this.capital_editing_id = None;
                    this.load_data(cx);
                }
                Err(e) => {
                    this.error = Some(format!("Chyba pri ukladani: {e}"));
                }
            }
            cx.notify();
        }).ok();
    }).detach();
}
```

### 6.4 Edit Capital Entry

Clicking "Upravit" on a row:
1. Sets `capital_form_visible = true`.
2. Sets `capital_editing_id = Some(entry.id)`.
3. Populates all form fields from the existing entry.

```rust
fn edit_capital_entry(&mut self, entry: &CapitalIncomeEntry, cx: &mut Context<Self>) {
    self.capital_form_visible = true;
    self.capital_editing_id = Some(entry.id);

    self.capital_category.update(cx, |s, cx| {
        s.set_selected_value(&entry.category.to_string(), cx);
    });
    self.capital_description.update(cx, |t, cx| {
        t.set_value(&entry.description, cx);
    });
    self.capital_income_date.update(cx, |d, cx| {
        d.set_iso_value(&entry.income_date.to_string(), cx);
    });
    self.capital_gross_amount.update(cx, |n, cx| {
        n.set_amount(entry.gross_amount, cx);
    });
    self.capital_withheld_tax_cz.update(cx, |n, cx| {
        n.set_amount(entry.withheld_tax_cz, cx);
    });
    self.capital_withheld_tax_foreign.update(cx, |n, cx| {
        n.set_amount(entry.withheld_tax_foreign, cx);
    });
    self.capital_country_code.update(cx, |t, cx| {
        t.set_value(&entry.country_code, cx);
    });
    self.capital_needs_declaring.update(cx, |c, cx| {
        c.set_checked(entry.needs_declaring, cx);
    });

    cx.notify();
}
```

### 6.5 Delete Capital Entry

1. Click "Smazat" on a row.
2. Show `ConfirmDialog` with message: "Opravdu chcete smazat tento zaznam?"
3. On confirm, call `income_service.delete_capital_entry(id)` on background executor.
4. On success, reload data.

---

## 7. Security Transactions Tab (Section 10)

### 7.1 Table Layout

| Column | Width | Content |
|--------|-------|---------|
| Nazev | flex-1 | asset_name |
| Typ | w-16 | Asset type label |
| Operace | w-16 | Buy/Sell label |
| Datum | w-24 | transaction_date |
| Pocet | w-20 | quantity / 10000 formatted |
| Jedn. cena | w-24 | unit_price (right-aligned) |
| Celkem | w-24 | total_amount (right-aligned) |
| Poplatky | w-20 | fees (right-aligned) |
| Naklady | w-24 | cost_basis (right-aligned, computed, read-only) |
| Zisk | w-24 | computed_gain (right-aligned, computed) |
| Osv. | w-12 | time_test_exempt (checkmark or dash) |
| Akce | w-24 | Edit/Delete buttons |

### 7.2 Add Form

Triggered by "Pridat transakci" button. Renders inline above the table.

```
+------------------------------------------------------+
| Nova transakce                                        |
|                                                       |
| [Typ aktiva v]  [Nazev___________]  [ISIN________]   |
| [Nakup/Prodej v]  [Datum____]  [Pocet____]            |
| [Jedn. cena____]  [Celkem____]  [Poplatky____]        |
| [Mena v]  [Kurz____]                                  |
|                                                       |
|                       [Zrusit]  [Ulozit]              |
+------------------------------------------------------+
```

Form fields:

| Field | Component | Validation |
|-------|-----------|------------|
| `asset_type` | `Select` | Required. Options: Akcie, ETF, Dluhopis, Fond, Krypto, Ostatni |
| `asset_name` | `TextInput` | Required, non-empty |
| `isin` | `TextInput` | Optional |
| `transaction_type` | `Select` | Required. Options: Nakup, Prodej |
| `transaction_date` | `DateInput` | Required |
| `quantity` | `NumberInput` | Required, > 0. Entered as decimal (e.g. 1.5), stored as quantity * 10000 |
| `unit_price` | `NumberInput` | Required, >= 0 |
| `total_amount` | `NumberInput` | Required, >= 0. Auto-calculated from quantity * unit_price if empty |
| `fees` | `NumberInput` | Optional, defaults to 0 |
| `currency_code` | `Select` | Required. Options: CZK, EUR, USD, GBP |
| `exchange_rate` | `NumberInput` | Required. Default: 1.0000. Entered as decimal, stored as rate * 10000 |

Asset type select options:

| Value | Label |
|-------|-------|
| `stock` | Akcie |
| `etf` | ETF |
| `bond` | Dluhopis |
| `fund` | Fond |
| `crypto` | Krypto |
| `other` | Ostatni |

Transaction type select options:

| Value | Label |
|-------|-------|
| `buy` | Nakup |
| `sell` | Prodej |

### 7.3 Save Security Transaction

Similar pattern to capital entry save. Key differences:

- Quantity conversion: user enters `1.5`, stored as `15000` (multiply by 10000).
- Exchange rate conversion: user enters `25.450`, stored as `254500` (multiply by 10000).
- `total_amount` auto-calculation: if `total_amount` is zero/empty but `quantity` and `unit_price` are set, compute `total_amount = quantity_raw * unit_price` (where `quantity_raw` is the actual float value).
- `cost_basis`, `computed_gain`, `time_test_exempt`, `exempt_amount` are computed server-side (by FIFO recalculation), so they are NOT part of the form. They display as read-only in the table.

```rust
fn save_security_transaction(&mut self, cx: &mut Context<Self>) {
    // Read form fields
    let asset_type = parse_asset_type(
        self.security_asset_type.read(cx).selected_value().unwrap_or("")
    );
    let asset_name = self.security_asset_name.read(cx).value().to_string();
    let isin = self.security_isin.read(cx).value().to_string();
    let tx_type = parse_transaction_type(
        self.security_transaction_type.read(cx).selected_value().unwrap_or("buy")
    );
    let tx_date_str = self.security_transaction_date.read(cx).iso_value().to_string();

    // Quantity: user enters decimal, we multiply by 10000
    let qty_str = self.security_quantity.read(cx).value().to_string();
    let qty_float: f64 = qty_str.replace(',', ".").parse().unwrap_or(0.0);
    let quantity = (qty_float * 10000.0).round() as i64;

    let unit_price = self.security_unit_price.read(cx).to_amount().unwrap_or(Amount::ZERO);
    let total = self.security_total_amount.read(cx).to_amount().unwrap_or(Amount::ZERO);
    let fees = self.security_fees.read(cx).to_amount().unwrap_or(Amount::ZERO);

    let currency = self.security_currency_code.read(cx)
        .selected_value().unwrap_or("CZK").to_string();

    // Exchange rate: user enters decimal, multiply by 10000
    let rate_str = self.security_exchange_rate.read(cx).value().to_string();
    let rate_float: f64 = rate_str.replace(',', ".").parse().unwrap_or(1.0);
    let exchange_rate = (rate_float * 10000.0).round() as i64;

    // Validate
    if asset_name.trim().is_empty() {
        self.error = Some("Zadejte nazev aktiva.".into());
        cx.notify();
        return;
    }
    let parsed_date = NaiveDate::parse_from_str(&tx_date_str, "%Y-%m-%d");
    if parsed_date.is_err() {
        self.error = Some("Neplatne datum.".into());
        cx.notify();
        return;
    }
    if quantity <= 0 {
        self.error = Some("Zadejte pocet.".into());
        cx.notify();
        return;
    }

    // Auto-calculate total if not provided
    let final_total = if total == Amount::ZERO && unit_price != Amount::ZERO {
        // total = (quantity / 10000) * unit_price
        Amount::from_raw(
            (quantity as f64 / 10000.0 * unit_price.raw() as f64).round() as i64
        )
    } else {
        total
    };

    let now = chrono::Local::now().naive_local();
    let mut tx = SecurityTransaction {
        id: self.security_editing_id.unwrap_or(0),
        year: self.year,
        document_id: None,
        asset_type,
        asset_name,
        isin,
        transaction_type: tx_type,
        transaction_date: parsed_date.unwrap(),
        quantity,
        unit_price,
        total_amount: final_total,
        fees,
        currency_code: currency,
        exchange_rate,
        cost_basis: Amount::ZERO,       // computed by FIFO
        computed_gain: Amount::ZERO,     // computed by FIFO
        time_test_exempt: false,         // computed by FIFO
        exempt_amount: Amount::ZERO,     // computed by FIFO
        created_at: now,
        updated_at: now,
    };

    self.saving = true;
    self.error = None;
    cx.notify();

    let service = self.income_service.clone();
    let is_edit = self.security_editing_id.is_some();

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            if is_edit {
                service.update_security_transaction(&mut tx)?;
            } else {
                service.create_security_transaction(&mut tx)?;
            }
            Ok::<(), DomainError>(())
        }).await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => {
                    this.security_form_visible = false;
                    this.security_editing_id = None;
                    this.load_data(cx);
                }
                Err(e) => {
                    this.error = Some(format!("Chyba pri ukladani: {e}"));
                }
            }
            cx.notify();
        }).ok();
    }).detach();
}
```

### 7.4 FIFO Recalculation

The "Prepocitat FIFO" button triggers a full FIFO cost-basis recalculation for all sell transactions in the selected year.

```rust
fn recalculate_fifo(&mut self, cx: &mut Context<Self>) {
    let service = self.income_service.clone();
    let year = self.year;
    self.loading = true;
    self.error = None;
    cx.notify();

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            // Get all sell transactions for the year
            let sells = service.list_security_transactions(year)?;
            // For each sell, compute FIFO cost basis using buys
            // This requires the SecurityTransactionRepo::list_buys_for_fifo
            // and SecurityTransactionRepo::update_fifo_results methods.
            //
            // NOTE: The InvestmentIncomeService does not currently expose
            // a recalculate_fifo method. This needs to be added to the service.
            // See Section 10 (Modified Files) for details.
            Ok::<(), DomainError>(())
        }).await;

        this.update(cx, |this, cx| {
            this.loading = false;
            if let Err(e) = result {
                this.error = Some(format!("Chyba pri prepoctu FIFO: {e}"));
            }
            this.load_data(cx);
            cx.notify();
        }).ok();
    }).detach();
}
```

**Service gap:** `InvestmentIncomeService` currently lacks a `recalculate_fifo(year)` method. The repo trait has `list_buys_for_fifo(asset_name, asset_type)` and `update_fifo_results(id, cost_basis, computed_gain, exempt_amount, time_test_exempt)`, so the building blocks exist. A new service method must be added:

```rust
// In InvestmentIncomeService:
pub fn recalculate_fifo(&self, year: i32) -> Result<(), DomainError> {
    let sells = self.security_repo.list_sells_by_year(year)?;
    for sell in &sells {
        let buys = self.security_repo.list_buys_for_fifo(
            &sell.asset_name, &sell.asset_type.to_string()
        )?;
        let (cost_basis, exempt) = compute_fifo_cost_basis(sell, &buys);
        let gain = sell.total_amount - sell.fees - cost_basis;
        let time_exempt = is_time_test_exempt(sell, &buys);
        self.security_repo.update_fifo_results(
            sell.id, cost_basis, gain, exempt, time_exempt
        )?;
    }
    Ok(())
}
```

### 7.5 Computed Fields Display

For each security transaction row, the following fields are read-only (not editable) and computed by the FIFO engine:

| Field | Display |
|-------|---------|
| `cost_basis` | Amount, right-aligned, `TEXT_MUTED` color |
| `computed_gain` | Amount, right-aligned, green if positive, red if negative |
| `time_test_exempt` | Checkmark icon if true, dash if false |
| `exempt_amount` | Amount, right-aligned (shown if `time_test_exempt` is true) |

---

## 8. OCR Wiring

### 8.1 Current State

- `OcrProvider` trait and implementations (`AnthropicProvider`, `OpenAIProvider`) exist in `zfaktury-api`.
- `OcrConfig` exists in `zfaktury-config` with fields: `provider`, `api_key`, `model`, `base_url`.
- `AppServices` does **not** create or hold an `OcrProvider` instance.
- The `ImportService` receives `None` for OCR in `app.rs` line 358-362.

### 8.2 Changes to `app.rs`

Add OCR provider construction based on config:

```rust
// In AppServices::new(), after existing service wiring:

// OCR provider (optional, based on config)
let ocr_provider: Option<Arc<dyn OcrProvider>> = config.ocr.as_ref().and_then(|ocr_cfg| {
    let api_key = ocr_cfg.api_key.as_deref().unwrap_or("");
    if api_key.is_empty() {
        return None;
    }
    let model = ocr_cfg.model.as_deref().unwrap_or("");
    match ocr_cfg.provider.as_deref().unwrap_or("") {
        "anthropic" | "claude" => {
            let mut p = AnthropicProvider::new(api_key, model);
            if let Some(ref url) = ocr_cfg.base_url {
                p = p.with_base_url(url);
            }
            Some(Arc::new(p) as Arc<dyn OcrProvider>)
        }
        "openai" => {
            let mut p = OpenAIProvider::new_openai(api_key, model);
            if let Some(ref url) = ocr_cfg.base_url {
                p = p.with_base_url(url);
            }
            Some(Arc::new(p) as Arc<dyn OcrProvider>)
        }
        "openrouter" => {
            let mut p = OpenAIProvider::new_openrouter(api_key, model);
            if let Some(ref url) = ocr_cfg.base_url {
                p = p.with_base_url(url);
            }
            Some(Arc::new(p) as Arc<dyn OcrProvider>)
        }
        _ => None,
    }
});
```

`AppServices` struct gets a new field:

```rust
pub ocr_provider: Option<Arc<dyn OcrProvider + Send + Sync>>,
```

**Note:** `AppServices::new()` must accept `config: &Config` as an additional parameter (currently it only takes `db_path` and `data_dir`). This is a minor signature change.

### 8.3 Investment-Specific OCR

The existing `OcrProvider` extracts invoice data (vendor, amounts, VAT). For investment documents (broker statements), we need a different extraction flow:

**Option A (recommended):** Create an `InvestmentExtractionService` in `zfaktury-core` that:
1. Reads file bytes from storage.
2. Calls `OcrProvider::process_image()` with a specialized system prompt for broker statements.
3. Parses the response into `Vec<CapitalIncomeEntry>` and `Vec<SecurityTransaction>`.
4. Creates the entries via `InvestmentIncomeService`.
5. Updates document status via `InvestmentDocumentRepo::update_extraction()`.

**Option B (simpler, less accurate):** Reuse the generic OCR prompt and manually map the resulting `OCRResult` fields to investment entries. This loses accuracy since the prompt is optimized for invoices, not broker statements.

This RFC recommends **Option A**. The `InvestmentExtractionService` can be a follow-up implementation. For the initial UI work, the extraction button should:
1. Check if OCR is configured (`ocr_provider.is_some()`).
2. If not, show an error toast: "OCR neni nakonfigurovano. Nastavte [ocr] v config.toml."
3. If configured, call the extraction pipeline and update the document status.

### 8.4 OCR Config Availability in UI

The extraction button should be disabled (grayed out) when `self.ocr_provider.is_none()`, with a tooltip or label explaining that OCR is not configured. This prevents confusion when the user clicks "Extrahovat" and nothing happens.

---

## 9. Summary Card

The summary card remains always visible at the top, regardless of active tab. No changes to the existing rendering logic except:

- It should reload whenever data changes (already handled by `load_data()`).
- The net_amount for capital entries is computed server-side by `InvestmentIncomeService`.
- Summary data comes from `get_year_summary(year)` which aggregates both capital and security data.

Current summary layout (preserved):

```
+-------------------------------------------------------------------+
| Souhrn investicnich prijmu                                        |
|                                                                   |
| Kapitalove prijmy (brutto)  |  Srazena dan  |  Kapit. prijem (n)  |
| 123 456,78 Kc               |  12 345,67 Kc |  111 111,11 Kc      |
| -----------------------------------------------------------       |
| Ostatni prijmy (brutto) | Naklady | Osvobozeno | Zaklad dane (p10)|
| 456 789,00 Kc           | 100 000 | 50 000     | 306 789,00 Kc    |
+-------------------------------------------------------------------+
```

---

## 10. Modified Files

### 10.1 `zfaktury-app/src/views/tax_investments.rs` (MAJOR rewrite)

Current: 518 lines, read-only display.
Expected: ~1200-1500 lines with tabs, forms, upload, extraction.

Changes:
- Add `InvestmentTab` enum.
- Expand struct with form state fields.
- Accept `InvestmentDocumentService` and `Option<Arc<dyn OcrProvider>>` in constructor.
- Add methods: `render_tab_bar()`, `render_documents_tab()`, `render_capital_tab()`, `render_securities_tab()`.
- Add methods: `save_capital_entry()`, `edit_capital_entry()`, `delete_capital_entry()`.
- Add methods: `save_security_transaction()`, `edit_security_transaction()`, `delete_security_transaction()`.
- Add methods: `upload_document()`, `extract_document()`, `delete_document()`.
- Add method: `recalculate_fifo()`.
- Add form reset helpers: `reset_capital_form()`, `reset_security_form()`.
- Update `load_data()` to also load documents.

### 10.2 `zfaktury-app/src/app.rs`

- Change `AppServices::new()` signature to accept `config: &Config`.
- Add `ocr_provider: Option<Arc<dyn OcrProvider + Send + Sync>>` field to `AppServices`.
- Construct OCR provider from `config.ocr`.
- Add `use zfaktury_api::ocr::{AnthropicProvider, OpenAIProvider, OcrProvider};` import.

### 10.3 `zfaktury-app/src/root.rs`

- Update `Route::TaxInvestments` handler to pass `services.investment_documents` and `services.ocr_provider` to `TaxInvestmentsView::new()`.

```rust
Route::TaxInvestments => {
    let income_svc = services.investment_income.clone();
    let doc_svc = services.investment_documents.clone();
    let ocr = services.ocr_provider.clone();
    ContentView::TaxInvestments(cx.new(|cx| {
        TaxInvestmentsView::new(income_svc, doc_svc, ocr, cx)
    }))
}
```

### 10.4 `zfaktury-app/Cargo.toml`

Add `rfd` dependency for native file picker:

```toml
rfd = "0.15"
```

Add `zfaktury-api` dependency for `OcrProvider`:

```toml
zfaktury-api = { path = "../zfaktury-api" }
```

### 10.5 `zfaktury-core/src/service/investment_income_svc.rs`

Add `recalculate_fifo(year: i32)` method that:
1. Lists all sell transactions for the year.
2. For each sell, fetches matching buys via `list_buys_for_fifo`.
3. Computes FIFO cost basis, gain, time-test exemption.
4. Updates results via `update_fifo_results`.

### 10.6 `zfaktury-app/src/main.rs` (minor)

If `AppServices::new()` signature changes to accept `&Config`, the caller in `main.rs` must pass it.

---

## 11. Data Flow

### 11.1 Create Capital Income Entry

```
User clicks "Pridat zaznam"
  -> capital_form_visible = true, cx.notify()
  -> User fills form, clicks "Ulozit"
  -> save_capital_entry() validates form fields
  -> Builds CapitalIncomeEntry struct
  -> cx.spawn() -> background_executor
     -> income_service.create_capital_entry(&mut entry)
        -> Service computes net_amount
        -> capital_repo.create(&entry)
        -> audit.log(...)
     <- Ok(())
  -> this.update() -> capital_form_visible = false
  -> load_data(cx) reloads all data + summary
  -> cx.notify()
```

### 11.2 Delete Security Transaction

```
User clicks "Smazat" on row
  -> confirm_dialog = Some(ConfirmDialog::new(...))
  -> cx.notify() (dialog renders)
  -> User clicks "Potvrdit"
  -> cx.spawn() -> background_executor
     -> income_service.delete_security_transaction(id)
        -> security_repo.delete(id)
        -> audit.log(...)
     <- Ok(())
  -> this.update() -> confirm_dialog = None
  -> load_data(cx) reloads all data + summary
  -> cx.notify()
```

### 11.3 Upload + Extract Document

```
User selects platform, clicks "Nahrat dokument"
  -> upload_document() -> cx.spawn()
     -> background: rfd::FileDialog::pick_file()
     <- Some(path)
     -> background: read file bytes
     -> background: document_service.create_record(...)
     -> background: write file to storage
     <- Ok(())
  -> this.update() -> uploading = false
  -> load_data(cx)

User clicks "Extrahovat" on document row
  -> extract_document(doc_id) -> cx.spawn()
     -> background: read file from storage
     -> background: ocr_provider.process_image(data, content_type)
     <- OCRResult
     -> background: parse OCR results into CapitalIncomeEntry / SecurityTransaction
     -> background: create entries via services
     -> background: document_service repo.update_extraction(id, "extracted", "")
     <- Ok(())
  -> this.update() -> extracting_doc_id = None
  -> load_data(cx)
```

### 11.4 FIFO Recalculation

```
User clicks "Prepocitat FIFO"
  -> recalculate_fifo() -> cx.spawn()
     -> background: income_service.recalculate_fifo(year)
        -> For each sell in year:
           -> list_buys_for_fifo(asset_name, asset_type)
           -> FIFO algorithm: match sell quantity against buy queue
           -> Compute cost_basis, gain, time_test_exempt, exempt_amount
           -> update_fifo_results(sell.id, ...)
     <- Ok(())
  -> this.update()
  -> load_data(cx) reloads transactions with updated computed fields
```

---

## 12. Error Handling

### 12.1 Service Errors

All service calls return `Result<T, DomainError>`. The view catches errors and stores them in `self.error: Option<String>`, which renders as a red banner at the top (existing pattern).

Error types and their UI messages:

| DomainError variant | UI message |
|---------------------|------------|
| `InvalidInput` | "Neplatny vstup. Zkontrolujte formular." |
| `NotFound` | "Zaznam nebyl nalezen." |
| `ExternalError(msg)` | "Chyba externi sluzby: {msg}" (OCR failures) |
| Other | "Neocekavana chyba: {error}" |

### 12.2 File I/O Errors

- File read failure during upload: "Nelze precist soubor: {path}"
- File write failure: "Nelze ulozit soubor: {path}"
- Missing file during extraction: "Soubor nenalezen v ulozisti."

### 12.3 OCR Not Configured

When user clicks "Extrahovat" but `self.ocr_provider.is_none()`:
- Set `self.error = Some("OCR neni nakonfigurovano. Pridejte sekci [ocr] do config.toml.")`.
- Do not attempt extraction.

### 12.4 Validation Errors

Form validation errors are shown in the same error banner. The first failing validation sets `self.error` and returns early from the save method. The banner clears on the next successful operation or when the user starts editing again.

---

## 13. Form Component Initialization

All form components are created in the `TaxInvestmentsView::new()` constructor (same pattern as `ExpenseFormView`). They are reused across add/edit operations by resetting their values.

```rust
impl TaxInvestmentsView {
    pub fn new(
        income_service: Arc<InvestmentIncomeService>,
        document_service: Arc<InvestmentDocumentService>,
        ocr_provider: Option<Arc<dyn OcrProvider>>,
        cx: &mut Context<Self>,
    ) -> Self {
        let year = chrono::Local::now().date_naive().year();

        // Capital income form components
        let capital_category = cx.new(|_cx| {
            Select::new("cap-category", "Vyberte kategorii...", capital_category_options())
        });
        let capital_description = cx.new(|cx| {
            TextInput::new("cap-description", "Popis prijmu...", cx)
        });
        let capital_income_date = cx.new(|cx| DateInput::new("cap-date", cx));
        let capital_gross_amount = cx.new(|cx| NumberInput::new("cap-gross", "0,00", cx));
        let capital_withheld_tax_cz = cx.new(|cx| NumberInput::new("cap-tax-cz", "0,00", cx));
        let capital_withheld_tax_foreign = cx.new(|cx| NumberInput::new("cap-tax-foreign", "0,00", cx));
        let capital_country_code = cx.new(|cx| TextInput::new("cap-country", "CZ", cx));
        let capital_needs_declaring = cx.new(|_cx| {
            Checkbox::new("cap-declaring", "Pridat do danoveho priznani", true)
        });

        // Security transaction form components
        let security_asset_type = cx.new(|_cx| {
            Select::new("sec-asset-type", "Typ aktiva...", asset_type_options())
        });
        let security_asset_name = cx.new(|cx| TextInput::new("sec-name", "Nazev aktiva...", cx));
        let security_isin = cx.new(|cx| TextInput::new("sec-isin", "ISIN...", cx));
        let security_transaction_type = cx.new(|_cx| {
            Select::new("sec-tx-type", "Operace...", transaction_type_options())
        });
        let security_transaction_date = cx.new(|cx| DateInput::new("sec-date", cx));
        let security_quantity = cx.new(|cx| NumberInput::new("sec-qty", "0", cx));
        let security_unit_price = cx.new(|cx| NumberInput::new("sec-price", "0,00", cx));
        let security_total_amount = cx.new(|cx| NumberInput::new("sec-total", "0,00", cx));
        let security_fees = cx.new(|cx| NumberInput::new("sec-fees", "0,00", cx));
        let security_currency_code = cx.new(|_cx| {
            Select::new("sec-currency", "Mena", currency_options())
        });
        let security_exchange_rate = cx.new(|cx| {
            NumberInput::new("sec-rate", "1,0000", cx).with_value("1,0000")
        });

        // Document upload components
        let document_platform = cx.new(|_cx| {
            Select::new("doc-platform", "Platforma...", platform_options())
        });

        let mut view = Self {
            income_service,
            document_service,
            ocr_provider,
            year,
            loading: true,
            error: None,
            active_tab: InvestmentTab::CapitalIncome,
            summary: None,
            capital_entries: Vec::new(),
            security_transactions: Vec::new(),
            documents: Vec::new(),
            capital_form_visible: false,
            capital_editing_id: None,
            capital_category,
            capital_description,
            capital_income_date,
            capital_gross_amount,
            capital_withheld_tax_cz,
            capital_withheld_tax_foreign,
            capital_country_code,
            capital_needs_declaring,
            security_form_visible: false,
            security_editing_id: None,
            security_asset_type,
            security_asset_name,
            security_isin,
            security_transaction_type,
            security_transaction_date,
            security_quantity,
            security_unit_price,
            security_total_amount,
            security_fees,
            security_currency_code,
            security_exchange_rate,
            document_platform,
            uploading: false,
            extracting_doc_id: None,
            confirm_dialog: None,
            saving: false,
        };
        view.load_data(cx);
        view
    }

    fn reset_capital_form(&mut self, cx: &mut Context<Self>) {
        self.capital_editing_id = None;
        self.capital_category.update(cx, |s, cx| s.clear_selection(cx));
        self.capital_description.update(cx, |t, cx| t.set_value("", cx));
        self.capital_income_date.update(cx, |d, cx| {
            d.set_iso_value(&chrono::Local::now().date_naive().to_string(), cx);
        });
        self.capital_gross_amount.update(cx, |n, cx| n.set_value("".to_string(), cx));
        self.capital_withheld_tax_cz.update(cx, |n, cx| n.set_value("".to_string(), cx));
        self.capital_withheld_tax_foreign.update(cx, |n, cx| n.set_value("".to_string(), cx));
        self.capital_country_code.update(cx, |t, cx| t.set_value("CZ", cx));
        self.capital_needs_declaring.update(cx, |c, cx| c.set_checked(true, cx));
    }

    fn reset_security_form(&mut self, cx: &mut Context<Self>) {
        self.security_editing_id = None;
        self.security_asset_type.update(cx, |s, cx| s.clear_selection(cx));
        self.security_asset_name.update(cx, |t, cx| t.set_value("", cx));
        self.security_isin.update(cx, |t, cx| t.set_value("", cx));
        self.security_transaction_type.update(cx, |s, cx| s.clear_selection(cx));
        self.security_transaction_date.update(cx, |d, cx| {
            d.set_iso_value(&chrono::Local::now().date_naive().to_string(), cx);
        });
        self.security_quantity.update(cx, |n, cx| n.set_value("".to_string(), cx));
        self.security_unit_price.update(cx, |n, cx| n.set_value("".to_string(), cx));
        self.security_total_amount.update(cx, |n, cx| n.set_value("".to_string(), cx));
        self.security_fees.update(cx, |n, cx| n.set_value("".to_string(), cx));
        self.security_currency_code.update(cx, |s, cx| s.set_selected_value("CZK", cx));
        self.security_exchange_rate.update(cx, |n, cx| n.set_value("1,0000".to_string(), cx));
    }
}
```

### Helper Functions

```rust
fn capital_category_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "dividend_cz".into(), label: "Dividenda CZ".into() },
        SelectOption { value: "dividend_foreign".into(), label: "Dividenda zahranicni".into() },
        SelectOption { value: "interest".into(), label: "Urok".into() },
        SelectOption { value: "coupon".into(), label: "Kupon".into() },
        SelectOption { value: "fund_distribution".into(), label: "Distribuce fondu".into() },
        SelectOption { value: "other".into(), label: "Ostatni".into() },
    ]
}

fn asset_type_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "stock".into(), label: "Akcie".into() },
        SelectOption { value: "etf".into(), label: "ETF".into() },
        SelectOption { value: "bond".into(), label: "Dluhopis".into() },
        SelectOption { value: "fund".into(), label: "Fond".into() },
        SelectOption { value: "crypto".into(), label: "Krypto".into() },
        SelectOption { value: "other".into(), label: "Ostatni".into() },
    ]
}

fn transaction_type_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "buy".into(), label: "Nakup".into() },
        SelectOption { value: "sell".into(), label: "Prodej".into() },
    ]
}

fn platform_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "portu".into(), label: "Portu".into() },
        SelectOption { value: "zonky".into(), label: "Zonky".into() },
        SelectOption { value: "trading212".into(), label: "Trading 212".into() },
        SelectOption { value: "revolut".into(), label: "Revolut".into() },
        SelectOption { value: "other".into(), label: "Ostatni".into() },
    ]
}

fn currency_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "CZK".into(), label: "CZK".into() },
        SelectOption { value: "EUR".into(), label: "EUR".into() },
        SelectOption { value: "USD".into(), label: "USD".into() },
        SelectOption { value: "GBP".into(), label: "GBP".into() },
    ]
}
```

---

## 14. Updated `load_data` Method

The `load_data` method must now also fetch documents:

```rust
fn load_data(&mut self, cx: &mut Context<Self>) {
    let income_svc = self.income_service.clone();
    let doc_svc = self.document_service.clone();
    let year = self.year;

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            let summary = income_svc.get_year_summary(year)?;
            let capital = income_svc.list_capital_entries(year)?;
            let securities = income_svc.list_security_transactions(year)?;
            let documents = doc_svc.list_by_year(year)?;
            Ok::<(
                InvestmentYearSummary,
                Vec<CapitalIncomeEntry>,
                Vec<SecurityTransaction>,
                Vec<InvestmentDocument>,
            ), DomainError>((summary, capital, securities, documents))
        }).await;

        this.update(cx, |this, cx| {
            this.loading = false;
            match result {
                Ok((summary, capital, securities, documents)) => {
                    this.summary = Some(summary);
                    this.capital_entries = capital;
                    this.security_transactions = securities;
                    this.documents = documents;
                }
                Err(e) => {
                    this.error = Some(format!("Chyba pri nacitani investic: {e}"));
                }
            }
            cx.notify();
        }).ok();
    }).detach();
}
```

---

## 15. Risks and Open Questions

### 15.1 `rfd` on Linux/Wayland

The `rfd` crate uses GTK file dialogs on Linux. On Wayland-only systems without X11, there may be compatibility issues. The GPUI framework already uses GTK3/WebKitGTK for desktop builds, so GTK should be available. If `rfd` file picker fails, the error should be caught and displayed as a toast.

**Mitigation:** `rfd` is already proven to work with GPUI apps (it is the same approach used by Zed for file dialogs). If there are issues, fall back to a simple file path text input.

### 15.2 OCR Prompt for Investment Documents

The existing OCR prompt is designed for invoice extraction. Investment documents (broker statements) have different structure -- tables of transactions, dividend reports, annual tax summaries. The extraction quality will be poor with the current prompt.

**Recommendation:** Create an investment-specific system prompt in a future RFC. For now, the extraction button should work end-to-end but may produce incomplete results.

### 15.3 FIFO Algorithm Correctness

The FIFO recalculation depends on correct implementation of:
- Matching sell transactions against buy transactions for the same asset.
- Handling partial lot sales.
- Czech 3-year time test for capital gains exemption.
- Currency conversion at historical exchange rates.

These are business-critical calculations for tax compliance. The algorithm should be unit-tested thoroughly.

**Recommendation:** Implement the FIFO algorithm in `zfaktury-core` with comprehensive test coverage (unit tests + property tests) before wiring to the UI.

### 15.4 View File Size

The enhanced view will be 1200-1500 lines. Consider splitting into sub-modules if it grows beyond 1500:

```
views/
  tax_investments/
    mod.rs              // TaxInvestmentsView struct, tab dispatch
    documents_tab.rs    // Document upload, list, extraction
    capital_tab.rs      // Capital income CRUD
    securities_tab.rs   // Security transaction CRUD, FIFO
```

This is optional -- the existing codebase keeps views in single files (e.g., `expense_form.rs` at 810 lines).

### 15.5 `AppServices::new()` Signature Change

Adding `config: &Config` parameter to `AppServices::new()` is a breaking change. All callers (currently just `main.rs`) must be updated. This is low-risk since there is only one call site.

---

## 16. Implementation Order

Recommended implementation phases:

1. **Tab system + layout** -- Add `InvestmentTab` enum, tab bar rendering, content dispatch. No forms yet. (~100 lines)

2. **Capital Income CRUD** -- Add form components, save/edit/delete handlers. Test with real data. (~400 lines)

3. **Security Transaction CRUD** -- Add form components, save/edit/delete handlers. Quantity/rate conversions. (~450 lines)

4. **Document upload** -- Add `rfd` dependency, file picker, document list. (~200 lines)

5. **OCR wiring** -- Wire `OcrProvider` in `app.rs`, extraction button, status updates. (~150 lines)

6. **FIFO recalculation** -- Add service method, UI button, reload. (~100 lines)

Each phase can be reviewed and merged independently.

---

## 17. Testing

### 17.1 Unit Tests

- `parse_capital_category()`, `parse_asset_type()`, `parse_transaction_type()` -- round-trip string to enum.
- Quantity conversion: `1.5` -> `15000`, `0.0001` -> `1`.
- Exchange rate conversion: `25.450` -> `254500`.
- Total amount auto-calculation: `qty=1.5, price=100.00` -> `total=150.00`.

### 17.2 Integration Tests

- `InvestmentIncomeService::recalculate_fifo()` -- with real SQLite, create buys/sells, verify cost basis.
- FIFO with partial lots: buy 10 at 100, sell 3 at 150 -> cost_basis = 300, gain = 150.
- Time test exemption: buy > 3 years ago, sell today -> `time_test_exempt = true`.

### 17.3 Manual UI Testing

- Upload a PDF, verify it appears in document list with "Cekajici" status.
- Add a capital income entry, verify summary updates.
- Edit a security transaction, verify changes persist after reload.
- Delete an entry, verify confirmation dialog appears.
- Switch years, verify data reloads for the new year.
- Click "Prepocitat FIFO", verify computed fields update.
