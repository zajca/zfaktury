# RFC-020: Expense OCR & Document Import

## Status: Draft
## Author: Claude
## Date: 2026-03-22

---

## 1. Problem Statement

The Rust GPUI app has two routes (`Route::ExpenseImport`, `Route::ExpenseReview`) that map to `StubView`. Users cannot:
- Upload expense documents (receipts, invoices)
- Run AI OCR to extract structured data from documents
- Review and confirm OCR results before creating expenses

The entire backend infrastructure exists:
- `OCRService` with `OCRProvider` trait (supports OpenAI, Anthropic, Gemini, Mistral, OpenRouter)
- `DocumentService` for file management (create, list, delete)
- `ImportService` to create skeleton expenses from documents
- Domain types: `OCRResult`, `OCRItem`, `ExpenseDocument`

**Critical gap:** `OCRService` is NOT wired into `AppServices`.

## 2. Goals

- Implement `ExpenseImportView` for document upload
- Implement `ExpenseReviewView` for OCR result review and expense creation
- Wire OCR provider into application services
- Enable native file picker via `rfd` crate (GPUI has no built-in file dialog)

## 3. Non-Goals

- Batch document upload (single file per import)
- Document versioning
- Drag-and-drop support (GPUI limitation)

## 4. Proposed Solution

### 4.1 File Picker Strategy

GPUI has no built-in file dialog. Use the `rfd` (Rusty File Dialogs) crate which provides native OS file dialogs on Linux (GTK), macOS, and Windows.

```rust
// Cargo.toml
[dependencies]
rfd = "0.15"
```

```rust
// Usage pattern in GPUI async context
cx.spawn(async move |this, cx| {
    let file = rfd::AsyncFileDialog::new()
        .add_filter("Documents", &["pdf", "jpg", "jpeg", "png", "webp"])
        .set_title("Vyberte doklad")
        .pick_file()
        .await;

    if let Some(file) = file {
        let path = file.path().to_path_buf();
        let data = std::fs::read(&path)?;
        let filename = path.file_name().unwrap().to_string_lossy().to_string();
        let content_type = mime_guess::from_path(&path)
            .first_or_octet_stream()
            .to_string();
        // ... process file
    }
}).detach();
```

### 4.2 New File: `views/expense_import.rs`

```rust
pub struct ExpenseImportView {
    // Services
    import_service: Arc<ImportService>,
    document_service: Arc<DocumentService>,
    ocr_service: Option<Arc<OCRService>>,

    // State
    loading: bool,
    processing: bool,
    error: Option<String>,
    success_message: Option<String>,

    // Recent imports (list of documents uploaded in this session)
    recent_imports: Vec<ImportResult>,
}
```

**UI Layout:**
1. Header: "Import dokladu" with back button
2. Upload area:
   - "Vybrat soubor" button (triggers `rfd::AsyncFileDialog`)
   - Supported formats note: "PDF, JPG, PNG, WebP (max 20 MB)"
   - Processing spinner when uploading/OCR running
3. OCR status indicator:
   - Green: "OCR aktivni (provider: openai)"
   - Yellow: "OCR neni nastaveno - doklad bude importovan bez rozpoznani"
4. Recent imports list:
   - Each row: filename, date, amount (if OCR succeeded), status badge
   - Click row -> navigate to ExpenseReview or ExpenseDetail

**Upload Flow:**
1. User clicks "Vybrat soubor" -> `rfd::AsyncFileDialog`
2. File selected -> validate size (<=20MB) and type
3. Call `import_service.create_skeleton_expense(filename)` -> creates expense
4. Call `document_service.create_record(expense_id, filename, content_type, data)` -> stores file
5. If OCR configured: call `ocr_service.process_document(document_id)` -> get OCRResult
6. If OCR succeeded: navigate to `Route::ExpenseReview` with expense_id + OCR data
7. If OCR failed or not configured: navigate to `Route::ExpenseDetail(expense_id)` for manual edit

### 4.3 New File: `views/expense_review.rs`

```rust
pub struct ExpenseReviewView {
    // Services
    expense_service: Arc<ExpenseService>,
    contact_service: Arc<ContactService>,
    category_service: Arc<CategoryService>,

    // Data from OCR
    expense_id: i64,
    ocr_result: OCRResult,
    confidence: f64,

    // Editable form fields (pre-filled from OCR)
    vendor_name: Entity<TextInput>,
    vendor_ico: Entity<TextInput>,
    description: Entity<TextInput>,
    invoice_number: Entity<TextInput>,
    issue_date: Entity<DateInput>,
    total_amount: Entity<NumberInput>,
    vat_amount: Entity<NumberInput>,
    vat_rate: Entity<Select>,
    currency: Entity<Select>,
    category: Entity<Select>,

    // State
    saving: bool,
    error: Option<String>,
    categories: Vec<Category>,
    contacts: Vec<Contact>,
}
```

**UI Layout:**
1. Header: "Kontrola rozpoznanych udaju" with confidence badge (green >80%, yellow 50-80%, red <50%)
2. Two-column form:
   - Left: OCR-extracted fields as editable inputs (pre-populated from OCRResult)
   - Right: Document preview placeholder (future: PDF/image rendering)
3. Items table (read-only from OCR):
   - Description, quantity, unit_price, vat_rate, total per item
4. Action buttons:
   - "Potvrdit a ulozit" -> updates expense with confirmed values
   - "Upravit rucne" -> navigate to ExpenseForm for full manual edit
   - "Zahodit" -> delete skeleton expense, navigate to ExpenseList

**Confirm Flow:**
1. Read all input values
2. Find or create contact by vendor_name/ICO
3. Update expense: `expense_service.update(expense_id, ...)`
4. Navigate to `Route::ExpenseDetail(expense_id)`
5. Show toast "Naklad ulozen"

### 4.4 Confidence Display Component

Inline confidence badge using existing `StatusBadge` or custom rendering:

```rust
fn render_confidence(confidence: f64) -> impl IntoElement {
    let (color, label) = match confidence {
        c if c >= 0.8 => (ZfColors::STATUS_PAID, "Vysoka"),
        c if c >= 0.5 => (ZfColors::STATUS_SENT, "Stredni"),
        _ => (ZfColors::STATUS_OVERDUE, "Nizka"),
    };
    div()
        .px_2().py_0p5().rounded_sm()
        .bg(color)
        .text_color(gpui::white())
        .text_xs()
        .child(format!("{:.0}% - {}", confidence * 100.0, label))
}
```

## 5. OCR Provider Wiring

### 5.1 AppServices Changes (`app.rs`)

Add `ocr_service` field to `AppServices`:

```rust
pub struct AppServices {
    // ... existing 30 services ...
    pub ocr_service: Option<Arc<OCRService>>,
}
```

### 5.2 OCR Configuration

Read from config (already defined in domain):

```toml
[ocr]
provider = "openai"  # openai|claude|gemini|mistral|openrouter
api_key = "sk-..."
model = ""           # optional, uses provider default
base_url = ""        # optional override
```

Construction in `app.rs`:

```rust
let ocr_service = if let Some(ocr_config) = &config.ocr {
    if !ocr_config.api_key.is_empty() {
        let provider = create_ocr_provider(
            &ocr_config.provider,
            &ocr_config.api_key,
            ocr_config.model.as_deref(),
            ocr_config.base_url.as_deref(),
        )?;
        Some(Arc::new(OCRService::new(provider, document_repo.clone())))
    } else {
        None
    }
} else {
    None
};
```

## 6. Modified Files

| File | Change |
|------|--------|
| `views/expense_import.rs` | NEW - Upload view |
| `views/expense_review.rs` | NEW - OCR review view |
| `views/mod.rs` | Add `pub mod expense_import; pub mod expense_review;` |
| `app.rs` | Wire OCRService, add to AppServices |
| `root.rs` | Map `Route::ExpenseImport` and `Route::ExpenseReview` to new views, add ContentView variants |
| `Cargo.toml` (zfaktury-app) | Add `rfd = "0.15"` dependency |

## 7. Error Handling

| Error | Handling |
|-------|----------|
| File too large (>20MB) | Inline error message, don't upload |
| Unsupported file type | Inline error message with supported types |
| OCR not configured | Warning banner, import without OCR, redirect to expense detail |
| OCR API failure | Warning toast, expense still created, redirect to manual edit |
| Network error during OCR | Same as API failure |
| File read error | Error toast, no expense created |
| Save failure | Error toast, keep form data |

## 8. Navigation Flow

```
ExpenseList -> "Import dokladu" button -> ExpenseImportView
  -> File selected + OCR success -> ExpenseReviewView -> Confirm -> ExpenseDetail
  -> File selected + OCR fail    -> ExpenseDetail (manual edit)
  -> File selected + no OCR      -> ExpenseDetail (manual edit)
```

## 9. Future Enhancements

- Document preview (PDF/image rendering in GPUI)
- Batch upload (multiple files at once)
- Drag-and-drop (when GPUI supports it)
- OCR for invoice documents (not just expenses)
- Re-run OCR button on existing documents
