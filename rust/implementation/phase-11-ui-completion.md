# Phase 11-19: GPUI UI Completion to 100% SvelteKit Parity

**Scope:** Make every GPUI view fully functional -- forms, actions, navigation, data persistence
**Estimated duration:** 8-10 weeks across 9 phases
**Prerequisites:** Phases 0-10 complete (all backend crates production-ready)
**Status:** RFC

---

## 1. Executive Summary

The Rust backend (domain, config, core, db, gen, api) is production-ready with 440 tests. The GPUI desktop app has 27 view files but they are all **read-only displays or stubs**. No form can save data, no action button works, no list row is clickable.

This RFC defines 9 phases (11-19) to bring the UI to 100% feature parity with the SvelteKit frontend. Phase 11 is infrastructure that blocks all others; Phases 12-19 can be partially parallelized.

---

## 2. Current State

### What exists in zfaktury-app

**Services wired (12):** dashboard, invoices, expenses, contacts, settings, categories, sequences, audit, recurring_invoices, recurring_expenses, vat_returns, reports

**Services NOT wired (18):** BackupService, DocumentService, HealthInsuranceService, IncomeTaxReturnService, ImportService, InvestmentDocumentService, InvestmentIncomeService, InvoiceDocumentService, OCRService, OverdueService, ReminderService, SocialInsuranceService, TaxCalendarService, TaxCreditsService, TaxDeductionService, TaxYearSettingsService, VATControlStatementService, VIESSummaryService

**Components (1):** StatusBadge only

**Routes (44 defined, 4 missing):**
- Missing: ContactNew, ContactEdit(id), InvoiceEdit(id), ExpenseEdit(id)
- 19 routes fall through to StubView

**Navigation:** Only sidebar emits NavigateEvent. No row clicks, no "New" buttons, no "Back" buttons.

### What the SvelteKit frontend can do (target)

See `rust/implementation/STATUS.md` section "SvelteKit Feature Parity Reference" for the full feature list.

---

## 3. Architecture Decisions

### Form Components Strategy

**Option A: gpui-component crate** (https://github.com/longbridgeapp/gpui-component)
- Provides: Input, DatePicker, Select, Table, Modal, Tabs, Toast, Button, Checkbox, NumberInput
- Risk: May not compile with the exact GPUI revision we depend on

**Option B: Custom minimal components**
- Build from scratch using GPUI primitives (div, on_click, on_key_down)
- More work but guaranteed compatibility

**Recommendation:** Try Option A first. If it doesn't compile, fall back to Option B. The components needed are well-defined regardless.

### Navigation Pattern

Views implement `EventEmitter<NavigateEvent>`. RootView subscribes to content view events (same pattern as sidebar). This enables:
- List row click → detail page
- "New" button → form page
- "Back" button → previous page (NavigationState::go_back)
- Form save → detail page

### Form Save Pattern

```rust
// In form view:
fn save(&mut self, cx: &mut Context<Self>) {
    let svc = self.service.clone();
    let data = self.build_domain_struct();
    cx.spawn(|this, mut cx| async move {
        let result = cx.background_executor()
            .spawn(async move { svc.create(&data) })
            .await;
        this.update(&mut cx, |this, cx| {
            match result {
                Ok(entity) => {
                    cx.emit(NavigateEvent(Route::InvoiceDetail(entity.id)));
                }
                Err(e) => {
                    this.error = Some(e.to_string());
                    cx.notify();
                }
            }
        })
    }).detach();
}
```

### File Download Pattern

```rust
// For PDF/ISDOC/QR downloads:
fn download_pdf(&mut self, cx: &mut Context<Self>) {
    let invoice = self.invoice.clone();
    let settings_svc = self.settings.clone();
    cx.spawn(|_this, mut cx| async move {
        let pdf_bytes = cx.background_executor().spawn(async move {
            let supplier = load_supplier(&settings_svc)?;
            let settings = load_pdf_settings(&settings_svc)?;
            zfaktury_gen::pdf::generate_invoice_pdf(&invoice, &supplier, &settings)
        }).await?;
        // Use GPUI file save dialog
        cx.update(|_cx| {
            // write pdf_bytes to chosen path
        })
    }).detach();
}
```

---

## 4. Phase Details

### Phase 11: UI Infrastructure

**BLOCKS ALL OTHER PHASES.** Must be completed first.

#### 11A: Form Components

Create in `zfaktury-app/src/components/`:

| Component | File | Purpose |
|---|---|---|
| TextInput | `text_input.rs` | Single-line editable text with placeholder, focus |
| NumberInput | `number_input.rs` | Numeric input aware of Amount (halere) |
| TextArea | `text_area.rs` | Multi-line text |
| Select | `select.rs` | Dropdown with option list overlay |
| DateInput | `date_input.rs` | Date parsing (dd.mm.yyyy Czech format) |
| Checkbox | `checkbox.rs` | Boolean toggle |
| Button | `button.rs` | Styled button with loading/disabled states |
| ConfirmDialog | `confirm_dialog.rs` | Modal overlay for destructive actions |
| Toast | `toast.rs` | Success/error notification |

#### 11B: Navigation

Modify `root.rs` to subscribe to content view NavigateEvent:
```rust
// After creating content view entity:
if let ContentView::InvoiceList(entity) = &self.content {
    cx.subscribe(entity, |this, _view, event: &NavigateEvent, cx| {
        this.navigate_to(event.0.clone(), cx);
    }).detach();
}
```

#### 11C: Missing Routes

Add to `navigation.rs`:
```rust
ContactNew,
ContactEdit(i64),
InvoiceEdit(i64),
ExpenseEdit(i64),
```

Add path parsing and labels.

#### 11D: Wire Missing Services

Add 18 services to AppServices in `app.rs`. Each needs its own DB connection. Example:
```rust
pub struct AppServices {
    // ... existing 12 ...
    pub backup: Arc<BackupService>,
    pub documents: Arc<DocumentService>,
    pub health_insurance: Arc<HealthInsuranceService>,
    pub income_tax: Arc<IncomeTaxReturnService>,
    // ... etc ...
}
```

---

### Phase 12: Invoice CRUD

**Files to modify:** `invoice_form.rs` (rewrite), `invoice_detail.rs`, `invoice_list.rs`
**Files to create:** `components/invoice_items_editor.rs`

#### Invoice Items Editor (critical shared component)

Reusable component for editing invoice line items:
- Row per item: description (text), quantity (number), unit (text), unit_price (number), vat_rate (select: 0/12/21%)
- "Add item" button, delete button per row
- Live calculation: vat_amount = unit_price * quantity * vat_rate / 100, total = unit_price * quantity + vat_amount
- Footer: subtotal, VAT breakdown by rate, grand total

#### Invoice Form

State:
```rust
struct InvoiceFormView {
    mode: FormMode, // Create / Edit(i64)
    customer_id: Option<i64>,
    invoice_type: InvoiceType,
    issue_date: String,
    due_date: String,
    delivery_date: String,
    variable_symbol: String,
    constant_symbol: String,
    currency_code: String,
    notes: String,
    internal_notes: String,
    items: Vec<InvoiceItemForm>,
    // services
    invoices: Arc<InvoiceService>,
    contacts: Arc<ContactService>,
    sequences: Arc<SequenceService>,
    // state
    contacts_list: Vec<Contact>,
    saving: bool,
    error: Option<String>,
}
```

#### Invoice Detail Actions

| Button | Action |
|---|---|
| Upravit | Navigate to InvoiceEdit(id) |
| Smazat | ConfirmDialog → invoices.delete(id) → InvoiceList |
| Odeslano | invoices.mark_as_sent(id) → reload |
| Uhrazeno | Dialog(amount, date) → invoices.mark_as_paid(id, amount, date) → reload |
| Duplikovat | invoices.duplicate(id) → InvoiceDetail(new_id) |
| Dobropis | Dialog(reason) → invoices.create_credit_note(id, ...) → InvoiceDetail(cn_id) |
| Vyuctovat | invoices.settle_proforma(id) → reload |
| PDF | generate_invoice_pdf → save file dialog |
| ISDOC | generate_isdoc → save file dialog |
| QR | generate_qr_png → save file dialog |
| Email | Dialog(to, subject, body, attach_pdf, attach_isdoc) → EmailSender.send |

---

### Phase 13: Expense+Contact CRUD

**Files to modify:** `expense_form.rs` (rewrite), `expense_detail.rs`, `expense_list.rs`, `contact_detail.rs`, `contact_list.rs`
**Files to create:** `contact_form.rs`

#### Contact Form with ARES Lookup

Key feature: when user enters ICO and clicks "Vyhledat v ARES", call `AresClient::lookup_ico(ico)` on background thread, then auto-fill name, DIC, street, city, zip from the response.

#### Expense Document Upload

Use `cx.open_file_dialog()` for native file picker. Upload via DocumentService. Display list with download/delete buttons.

---

### Phase 14: Settings CRUD

**Files to modify:** All settings_*.rs views
**Files to create:** `settings_pdf.rs`

Key pattern: toggle between read-only and edit mode:
```rust
struct SettingsFirmaView {
    editing: bool,
    // original values (for cancel)
    original: HashMap<String, String>,
    // current edited values
    values: HashMap<String, String>,
    // ...
}
```

---

### Phase 15: Recurring Templates

**Files to create:** `recurring_invoice_detail.rs`, `recurring_invoice_form.rs`, `recurring_expense_detail.rs`, `recurring_expense_form.rs`
**Files to modify:** `recurring_invoice_list.rs`, `recurring_expense_list.rs`

Reuse the invoice items editor from Phase 12 for recurring invoice form.

---

### Phase 16: VAT Management

**Files to create:** `vat_return_form.rs`, `vat_control_detail.rs`, `vat_control_form.rs`, `vies_detail.rs`, `vies_form.rs`
**Files to modify:** `vat_overview.rs`, `vat_return_detail.rs`

All VAT views follow the same pattern: list/overview → create form → detail with recalculate/XML/file actions.

---

### Phase 17: Tax Filing

**Files to create:** `tax_income_detail.rs`, `tax_income_form.rs`, `tax_social_detail.rs`, `tax_social_form.rs`, `tax_health_detail.rs`, `tax_health_form.rs`
**Files to modify:** `tax_overview.rs`, `tax_credits.rs`, `tax_prepayments.rs`, `tax_investments.rs`

---

### Phase 18: Reports+Dashboard

**Files to modify:** `reports.rs`, `dashboard.rs`

Charts: Use simple colored `div().w(px(value * scale))` horizontal bars. Not gpui-component charts.

Reports tabs: Revenue, Expenses, Profit & Loss, Top Customers, Tax Calendar.

---

### Phase 19: Import+UX Polish

**Files to create:** `expense_import.rs`, `expense_review.rs`
**Files to modify:** `import_fakturoid.rs`, all other views for UX polish

---

## 5. Files Summary

### New Files to Create (17)

```
components/text_input.rs
components/number_input.rs
components/text_area.rs
components/select.rs
components/date_input.rs
components/checkbox.rs
components/button.rs
components/confirm_dialog.rs
components/toast.rs
components/invoice_items_editor.rs
views/contact_form.rs
views/settings_pdf.rs
views/recurring_invoice_detail.rs
views/recurring_invoice_form.rs
views/recurring_expense_detail.rs
views/recurring_expense_form.rs
views/vat_return_form.rs
views/vat_control_detail.rs
views/vat_control_form.rs
views/vies_detail.rs
views/vies_form.rs
views/tax_income_detail.rs
views/tax_income_form.rs
views/tax_social_detail.rs
views/tax_social_form.rs
views/tax_health_detail.rs
views/tax_health_form.rs
views/expense_import.rs
views/expense_review.rs
```

### Files to Modify (always by lead)

```
app.rs              -- wire 18 missing services
root.rs             -- add ContentView variants, create_content_view arms, event subscriptions
navigation.rs       -- add 4 missing routes + path parsing
components/mod.rs   -- register all new components
views/mod.rs        -- register all new view modules
Cargo.toml          -- add gpui-component if used
```

### Files to Modify (by teammates)

```
views/invoice_form.rs      -- Phase 12: complete rewrite
views/invoice_detail.rs    -- Phase 12: add actions
views/invoice_list.rs      -- Phase 12: add navigation + filters
views/expense_form.rs      -- Phase 13: complete rewrite
views/expense_detail.rs    -- Phase 13: add actions
views/expense_list.rs      -- Phase 13: add navigation + filters
views/contact_detail.rs    -- Phase 13: add actions
views/contact_list.rs      -- Phase 13: add navigation
views/settings_firma.rs    -- Phase 14: edit mode
views/settings_email.rs    -- Phase 14: edit mode + test
views/settings_sequences.rs -- Phase 14: CRUD
views/settings_categories.rs -- Phase 14: CRUD
views/settings_backup.rs   -- Phase 14: wire service
views/settings_audit.rs    -- Phase 14: filters
views/recurring_invoice_list.rs  -- Phase 15: navigation + actions
views/recurring_expense_list.rs  -- Phase 15: navigation + actions
views/vat_overview.rs      -- Phase 16: wire services
views/vat_return_detail.rs -- Phase 16: add actions
views/tax_overview.rs      -- Phase 17: wire services
views/tax_credits.rs       -- Phase 17: complete rewrite
views/tax_prepayments.rs   -- Phase 17: wire data
views/tax_investments.rs   -- Phase 17: wire data + CRUD
views/reports.rs           -- Phase 18: expand tabs
views/dashboard.rs         -- Phase 18: quick actions + charts
views/import_fakturoid.rs  -- Phase 19: complete rewrite
```

---

## 6. Route Coverage After Completion

After all phases, every route maps to a functional view:

| Route | Phase | View Type |
|---|---|---|
| Dashboard | P18 | Enhanced |
| Reports | P18 | Enhanced (5 tabs) |
| InvoiceList | P12 | Enhanced (search/filter/pagination) |
| InvoiceNew | P12 | Form (create) |
| InvoiceEdit(id) | P11+P12 | Form (edit) |
| InvoiceDetail(id) | P12 | Detail + 11 action buttons |
| RecurringInvoiceList | P15 | Enhanced |
| RecurringInvoiceNew | P15 | Form |
| RecurringInvoiceDetail(id) | P15 | Detail + actions |
| ExpenseList | P13 | Enhanced |
| ExpenseNew | P13 | Form (create) |
| ExpenseEdit(id) | P11+P13 | Form (edit) |
| ExpenseDetail(id) | P13 | Detail + actions |
| ExpenseImport | P19 | File picker + OCR |
| ExpenseReview | P19 | Bulk review |
| RecurringExpenseList | P15 | Enhanced |
| RecurringExpenseNew | P15 | Form |
| RecurringExpenseDetail(id) | P15 | Detail + actions |
| ContactList | P13 | Enhanced |
| ContactNew | P11+P13 | Form (create) |
| ContactEdit(id) | P11+P13 | Form (edit) |
| ContactDetail(id) | P13 | Detail + actions |
| VATOverview | P16 | Enhanced (real data) |
| VATReturnNew | P16 | Form |
| VATReturnDetail(id) | P16 | Detail + actions |
| VATControlNew | P16 | Form |
| VATControlDetail(id) | P16 | Detail + actions |
| VIESNew | P16 | Form |
| VIESDetail(id) | P16 | Detail + actions |
| TaxOverview | P17 | Enhanced (real data) |
| TaxCredits | P17 | Full CRUD |
| TaxPrepayments | P17 | Editable table |
| TaxInvestments | P17 | Full CRUD + FIFO |
| TaxIncomeNew | P17 | Form |
| TaxIncomeDetail(id) | P17 | Detail + actions |
| TaxSocialNew | P17 | Form |
| TaxSocialDetail(id) | P17 | Detail + actions |
| TaxHealthNew | P17 | Form |
| TaxHealthDetail(id) | P17 | Detail + actions |
| SettingsFirma | P14 | Editable form |
| SettingsEmail | P14 | Editable + test |
| SettingsSequences | P14 | CRUD list |
| SettingsCategories | P14 | CRUD list |
| SettingsPdf | P14 | Config form |
| SettingsAuditLog | P14 | Filterable list |
| SettingsBackup | P14 | Create/list/delete |
| ImportFakturoid | P19 | Import wizard |

**Total: 47 routes, 0 StubView**

---

## 7. Verification Protocol

### Per-Phase Verification

```bash
cd rust && nix develop
cargo build --workspace
cargo test --workspace
cargo clippy --workspace -- -D warnings
```

### Final Verification (after Phase 19)

1. **Route check:** Launch app with `--route <path>` for each of 47 routes. Verify no StubView.
2. **Workflow test:** Create contact → Create invoice with items → Send → Mark paid → Download PDF
3. **Tax workflow:** Create VAT return → Recalculate → Generate XML → Mark filed
4. **Settings test:** Edit company info → Save → Verify persisted
5. **Headless screenshots:** Screenshot all 47 routes, review each for layout + Czech labels + data
