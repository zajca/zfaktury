# Phase 6: GPUI Desktop Application

**Crate:** `zfaktury-app`
**Estimated duration:** 4-6 weeks
**Status:** RFC Draft
**Dependencies:** Phase 1 (domain), Phase 2 (persistence), Phase 3 (services), Phase 4 (PDF/XML gen), Phase 5 (CLI)

---

## 1. Executive Summary

Phase 6 builds the full native desktop application using Zed's GPUI framework and the `gpui-component` widget library. It replaces the SvelteKit + Wails v3 frontend with a single Rust binary that renders at 120fps via GPU, eliminates all web overhead, and provides a keyboard-first experience with command palette, split-view, and live PDF preview.

The existing SvelteKit frontend has 45 route pages (43 unique screens after deduplication). Every screen will be ported to a native GPUI view, preserving all Czech UI text, status labels, color semantics, and business logic.

---

## 2. Goals and Non-Goals

### Goals

- Port all 43 route pages from SvelteKit to native GPUI views
- Implement custom ZFaktury Dark/Light theme pair
- Command palette with diacritics-insensitive fuzzy search (Ctrl+K)
- Split-view master-detail for invoice and expense lists (Ctrl+\\)
- Live PDF preview in inspector panel while editing invoices
- 120fps virtual scrolling for tables with 10,000+ rows
- Keyboard-first navigation with discoverable shortcuts
- GPU-rendered charts (bar, doughnut) for dashboard and reports
- Native OS integration: file dialogs, clipboard, drag & drop, system notifications
- All monetary values rendered in JetBrains Mono with tabular-nums

### Non-Goals

- Mobile/responsive layout (desktop-only, minimum 1024x768)
- Web/server mode (handled by separate `zfaktury-server` binary from Phase 5)
- Theming API for user-created themes (only Dark/Light shipped)
- Plugin/extension system
- Multi-window support (single window with panels)
- Internationalization beyond Czech (all UI text is Czech)

---

## 3. Architecture Overview

### 3.1 Crate Structure

```
crates/zfaktury-app/
├── Cargo.toml
├── assets/
│   ├── fonts/
│   │   ├── Inter-Variable.ttf
│   │   └── JetBrainsMono-Variable.ttf
│   └── icons/               # Lucide SVG icons (sidebar, actions)
├── src/
│   ├── main.rs              # Entry point, app bootstrap
│   ├── app.rs               # ZFakturyApp: window creation, service wiring
│   ├── theme.rs             # Custom theme definitions (Dark/Light)
│   ├── navigation.rs        # Route enum, NavigationState, history
│   ├── root.rs              # Root view: sidebar + content + inspector layout
│   ├── sidebar.rs           # Collapsible sidebar navigation
│   ├── title_bar.rs         # Title bar: search input, theme toggle
│   ├── command_palette.rs   # Ctrl+K overlay with fuzzy search
│   ├── toast.rs             # Toast notification system
│   ├── views/
│   │   ├── mod.rs
│   │   ├── dashboard.rs
│   │   ├── reports.rs
│   │   ├── invoices/
│   │   │   ├── mod.rs
│   │   │   ├── list.rs
│   │   │   ├── detail.rs
│   │   │   ├── form.rs
│   │   │   └── items_editor.rs
│   │   ├── expenses/
│   │   │   ├── mod.rs
│   │   │   ├── list.rs
│   │   │   ├── detail.rs
│   │   │   ├── form.rs
│   │   │   ├── import.rs
│   │   │   └── review.rs
│   │   ├── contacts/
│   │   │   ├── mod.rs
│   │   │   ├── list.rs
│   │   │   └── detail.rs
│   │   ├── recurring/
│   │   │   ├── mod.rs
│   │   │   ├── invoices_list.rs
│   │   │   ├── invoice_form.rs
│   │   │   ├── expenses_list.rs
│   │   │   └── expense_form.rs
│   │   ├── vat/
│   │   │   ├── mod.rs
│   │   │   ├── overview.rs
│   │   │   ├── return_detail.rs
│   │   │   ├── return_form.rs
│   │   │   ├── control_detail.rs
│   │   │   ├── control_form.rs
│   │   │   ├── vies_detail.rs
│   │   │   └── vies_form.rs
│   │   ├── tax/
│   │   │   ├── mod.rs
│   │   │   ├── overview.rs
│   │   │   ├── credits.rs
│   │   │   ├── prepayments.rs
│   │   │   ├── investments.rs
│   │   │   ├── income_detail.rs
│   │   │   ├── income_form.rs
│   │   │   ├── social_detail.rs
│   │   │   ├── social_form.rs
│   │   │   ├── health_detail.rs
│   │   │   └── health_form.rs
│   │   └── settings/
│   │       ├── mod.rs
│   │       ├── firma.rs
│   │       ├── email.rs
│   │       ├── sequences.rs
│   │       ├── categories.rs
│   │       ├── pdf.rs
│   │       ├── audit_log.rs
│   │       ├── backup.rs
│   │       └── fakturoid_import.rs
│   ├── components/
│   │   ├── mod.rs
│   │   ├── date_input.rs
│   │   ├── currency_selector.rs
│   │   ├── category_picker.rs
│   │   ├── contact_picker.rs
│   │   ├── document_upload.rs
│   │   ├── status_badge.rs
│   │   ├── status_timeline.rs
│   │   ├── bar_chart.rs
│   │   ├── doughnut_chart.rs
│   │   ├── email_dialog.rs
│   │   ├── credit_note_dialog.rs
│   │   ├── ocr_review_dialog.rs
│   │   ├── confirm_dialog.rs
│   │   └── split_view.rs
│   └── util/
│       ├── mod.rs
│       ├── format.rs         # Czech number/date/currency formatting
│       ├── fuzzy.rs          # Diacritics-insensitive fuzzy search
│       └── keyboard.rs       # Keyboard shortcut registry
```

### 3.2 Dependency Graph

```
zfaktury-app
├── zfaktury-service   (Arc<XxxService> for all business logic)
├── zfaktury-domain    (domain types, Amount, errors)
├── zfaktury-gen       (PDF generation, XML export)
├── zfaktury-persist   (database, migrations - startup only)
├── gpui               (UI framework)
├── gpui-component     (widget library: Table, Button, Input, etc.)
├── lucide-icons-gpui  (icon set, if available, else inline SVG paths)
└── nucleo             (fuzzy matching for command palette)
```

### 3.3 Service Layer Integration

All services from `zfaktury-service` are wrapped in `Arc<T>` and shared across views. Views never touch the database directly.

```rust
pub struct AppServices {
    pub invoices: Arc<InvoiceService>,
    pub expenses: Arc<ExpenseService>,
    pub contacts: Arc<ContactService>,
    pub recurring_invoices: Arc<RecurringInvoiceService>,
    pub recurring_expenses: Arc<RecurringExpenseService>,
    pub vat_returns: Arc<VatReturnService>,
    pub control_statements: Arc<ControlStatementService>,
    pub vies_declarations: Arc<ViesDeclarationService>,
    pub income_tax: Arc<IncomeTaxService>,
    pub social_insurance: Arc<SocialInsuranceService>,
    pub health_insurance: Arc<HealthInsuranceService>,
    pub tax_credits: Arc<TaxCreditService>,
    pub tax_prepayments: Arc<TaxPrepaymentService>,
    pub tax_investments: Arc<TaxInvestmentService>,
    pub settings: Arc<SettingsService>,
    pub sequences: Arc<SequenceService>,
    pub categories: Arc<CategoryService>,
    pub documents: Arc<DocumentService>,
    pub dashboard: Arc<DashboardService>,
    pub reports: Arc<ReportService>,
    pub audit_log: Arc<AuditLogService>,
    pub backup: Arc<BackupService>,
    pub import: Arc<ImportService>,
    pub email: Arc<EmailService>,
}
```

### 3.4 Threading Model

```
Main Thread (GPU/UI)
├── Renders all views at 120fps
├── Handles input events, layout, paint
├── NEVER blocks on I/O
│
Background Executor (thread pool)
├── Database queries (via service layer)
├── PDF generation (zfaktury-gen)
├── XML generation (zfaktury-gen)
├── CNB exchange rate fetch (HTTP)
├── ARES company lookup (HTTP)
├── SMTP email sending
├── Fuzzy search scoring (large datasets)
├── File I/O (document upload, backup)
└── OCR processing
```

Every operation that touches the database, network, or filesystem runs on `cx.background_executor()`. The pattern:

```rust
fn load_data(&mut self, cx: &mut Context<Self>) {
    self.loading = true;
    cx.notify();

    let service = self.services.invoices.clone();
    let filter = self.filter.clone();

    cx.spawn(|this, mut cx| async move {
        let result = cx.background_executor().spawn(async move {
            service.list(&filter)
        }).await;

        this.update(&mut cx, |this, cx| {
            match result {
                Ok(data) => {
                    this.invoices = data;
                    this.error = None;
                }
                Err(e) => {
                    this.error = Some(format!("Nepodařilo se načíst faktury: {e}"));
                }
            }
            this.loading = false;
            cx.notify();
        }).ok();
    }).detach();
}
```

---

## 4. Window Layout

### 4.1 Main Window Structure

```
+--------------------------------------------------------------------+
| Title Bar (native, draggable)  "ZFaktury"  | [search] | [theme]    |
+----------+-------------------------------------------------+-------+
|          | Breadcrumb / Context Bar (36px)  + action btns   |       |
| Sidebar  +-------------------------------------------------+ Insp. |
| (52-     | Main Content Area                               | Panel |
|  208px)  | (scrollable, centered, max-w 960-1200px)        | (280  |
|          |                                                  |  px,  |
|          |                                                  | togg) |
+----------+-------------------------------------------------+-------+
```

### 4.2 Title Bar (48px)

- Native window decorations on Linux (GTK CSD)
- Draggable title area with "ZFaktury" text
- Global search input (Ctrl+F to focus) -- triggers command palette on complex queries
- Theme toggle button (sun/moon icon)
- Window controls: minimize, maximize, close (native)

### 4.3 Sidebar

Ported from the existing `Layout.svelte` navigation structure:

**Top-level items (no section header):**
| Icon | Label | Route |
|------|-------|-------|
| Home | Dashboard | `Dashboard` |
| BarChart | Přehledy | `Reports` |

**Section: (no header, direct items)**
| Icon | Label | Route | Actions |
|------|-------|-------|---------|
| FileText | Faktury | `InvoiceList` | + Nová faktura |
| RefreshCw | Šablony faktur | `RecurringList` | + Nová šablona |
| Wallet | Náklady | `ExpenseList` | + Přidat náklad, + Import dokladů, + Opakovaný náklad |
| Users | Kontakty | `ContactList` | + Nový kontakt |

**Section: Účetnictví**
| Icon | Label | Route | Children |
|------|-------|-------|----------|
| Calculator | DPH | `VatOverview` | -- |
| Receipt | Daň z příjmů | `TaxOverview` | Slevy a odpočty, Zálohy, Investice |

**Section: Nastavení**
| Icon | Label | Route |
|------|-------|-------|
| Building | Firma | `SettingsFirma` |
| Mail | Email | `SettingsEmail` |
| Hash | Číselné řady | `SettingsSequences` |
| Tag | Kategorie | `SettingsCategories` |
| FileText | PDF šablona | `SettingsPdf` |
| Upload | Import z Fakturoid | `ImportFakturoid` |
| ClipboardCheck | Audit log | `SettingsAuditLog` |
| Database | Zálohy | `SettingsBackup` |

**Sidebar behavior:**
- Expanded width: 208px
- Collapsed width: 52px (icons only)
- Toggle: Ctrl+Shift+L or collapse button
- Hover expansion: when collapsed, hovering shows full sidebar as floating overlay with `shadow-xl`
- Active item: highlighted with accent background
- Active detection: longest prefix match (same logic as existing `isActive()` in Layout.svelte)
- State persisted to config file (not localStorage)
- Version text in footer when expanded: "ZFaktury v{VERSION}"
- Collapsed: section dividers shown as thin horizontal lines
- Expanded: section headers shown as uppercase muted text labels

### 4.4 Context Bar (36px)

Sits between the title bar and main content area. Contains:
- Breadcrumb trail (e.g., "Faktury > FV-2026-0042")
- Page-specific action buttons (right-aligned):
  - Invoice detail: "Odeslat", "Uhrazeno", "Duplikovat", "Dobropis", "PDF", "ISDOC"
  - Expense detail: "Upravit", "Smazat", "PDF"
  - List views: "Nová faktura" / "Přidat náklad" / etc.
- Action buttons use `gpui_component::Button` with appropriate variants

### 4.5 Inspector Panel (280px, toggleable)

Right-side panel, toggled via:
- Ctrl+P on invoice detail/edit (shows PDF preview)
- Double-click column header on list views (shows column stats)
- Not shown by default

Contents vary by context:
- **Invoice edit:** Live PDF preview (re-rendered on 500ms debounce after form changes)
- **Invoice detail:** PDF preview of current invoice
- **List views:** Column statistics, filter summary

### 4.6 Main Content Area

- Scrollable vertically
- Horizontally centered with max-width constraint:
  - Dashboard, reports: 1200px
  - Forms: 960px
  - Lists: full width (no max constraint, table fills available space)
- Padding: 20px on all sides

---

## 5. Navigation & Routing

### 5.1 Route Enum

```rust
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum Route {
    // Top level
    Dashboard,
    Reports,

    // Invoices
    InvoiceList,
    InvoiceNew,
    InvoiceDetail(i64),

    // Recurring invoices
    RecurringList,
    RecurringNew,
    RecurringDetail(i64),

    // Expenses
    ExpenseList,
    ExpenseNew,
    ExpenseDetail(i64),
    ExpenseImport,
    ExpenseReview,

    // Recurring expenses
    RecurringExpenseList,
    RecurringExpenseNew,
    RecurringExpenseDetail(i64),

    // Contacts
    ContactList,
    ContactDetail(i64),

    // VAT
    VatOverview,
    VatReturnNew,
    VatReturnDetail(i64),
    ControlStatementNew,
    ControlStatementDetail(i64),
    ViesNew,
    ViesDetail(i64),

    // Tax
    TaxOverview,
    TaxCredits,
    TaxPrepayments,
    TaxInvestments,
    IncomeTaxNew,
    IncomeTaxDetail(i64),
    SocialInsuranceNew,
    SocialInsuranceDetail(i64),
    HealthInsuranceNew,
    HealthInsuranceDetail(i64),

    // Settings
    SettingsFirma,
    SettingsEmail,
    SettingsSequences,
    SettingsCategories,
    SettingsPdf,
    SettingsAuditLog,
    SettingsBackup,

    // Import
    ImportFakturoid,
}
```

### 5.2 Route Metadata

Each route has associated metadata used by the breadcrumb, sidebar highlighting, and page title:

```rust
impl Route {
    /// Czech display name for breadcrumb and page title
    pub fn label(&self) -> &'static str { ... }

    /// Parent route for breadcrumb chain
    pub fn parent(&self) -> Option<Route> { ... }

    /// Sidebar section this route belongs to (for active highlighting)
    pub fn sidebar_section(&self) -> SidebarSection { ... }

    /// Full breadcrumb chain from root to this route
    pub fn breadcrumbs(&self) -> Vec<(Route, &'static str)> { ... }
}
```

Example breadcrumb chains:
- `Dashboard` -> `["Dashboard"]`
- `InvoiceDetail(42)` -> `["Faktury", "FV-2026-0042"]` (number fetched from data)
- `TaxCredits` -> `["Daň z příjmů", "Slevy a odpočty"]`

### 5.3 NavigationState

```rust
pub struct NavigationState {
    current: Route,
    history: Vec<Route>,
    forward_stack: Vec<Route>,
}

impl NavigationState {
    pub fn new() -> Self {
        Self {
            current: Route::Dashboard,
            history: Vec::new(),
            forward_stack: Vec::new(),
        }
    }

    pub fn navigate(&mut self, route: Route) {
        if route == self.current {
            return;
        }
        self.history.push(self.current.clone());
        self.current = route;
        self.forward_stack.clear();
    }

    pub fn back(&mut self) -> bool {
        if let Some(prev) = self.history.pop() {
            self.forward_stack.push(self.current.clone());
            self.current = prev;
            true
        } else {
            false
        }
    }

    pub fn forward(&mut self) -> bool {
        if let Some(next) = self.forward_stack.pop() {
            self.history.push(self.current.clone());
            self.current = next;
            true
        } else {
            false
        }
    }

    pub fn current(&self) -> &Route { &self.current }
    pub fn can_go_back(&self) -> bool { !self.history.is_empty() }
    pub fn can_go_forward(&self) -> bool { !self.forward_stack.is_empty() }
}
```

### 5.4 Navigation Actions (GPUI)

```rust
actions!(
    zfaktury,
    [
        NavigateBack,
        NavigateForward,
        NavigateTo,
        NavigateDashboard,
        NavigateInvoiceList,
        NavigateNewInvoice,
        NavigateExpenseList,
        NavigateNewExpense,
        NavigateContactList,
        NavigateSettings,
        ToggleSidebar,
        ToggleInspector,
        ToggleCommandPalette,
        ToggleSplitView,
        FocusSearch,
        SaveCurrent,
    ]
);
```

---

## 6. Theme System

### 6.1 ZFaktury Dark (Default)

The dark theme uses warm-tinted dark backgrounds to reduce eye strain during long accounting sessions.

```rust
pub fn zfaktury_dark() -> Theme {
    Theme {
        name: "ZFaktury Dark".into(),

        // Backgrounds
        base: rgb(0x1e1d21),           // Main background
        surface: rgb(0x1e1e22),         // Panels, cards, sidebar
        elevated: rgb(0x26262a),        // Raised surfaces, table headers
        hover: rgb(0x2e2e33),           // Hover state
        active: rgb(0x36363b),          // Active/pressed state

        // Text
        text_primary: rgb(0xececef),    // Primary text
        text_secondary: rgb(0xa0a0a8),  // Secondary text, labels
        text_muted: rgb(0x6b6b73),      // Muted text, placeholders
        text_on_accent: rgb(0xffffff),  // Text on accent-colored backgrounds

        // Accent (indigo)
        accent: rgb(0x5e6ad2),
        accent_hover: rgb(0x6e7ae2),
        accent_muted: rgba(0x5e6ad2, 0.15), // Background for active nav items

        // Semantic colors
        success: rgb(0x4ade80),
        success_bg: rgba(0x4ade80, 0.12),
        warning: rgb(0xfbbf24),
        warning_bg: rgba(0xfbbf24, 0.12),
        danger: rgb(0xf87171),
        danger_bg: rgba(0xf87171, 0.12),
        info: rgb(0x60a5fa),
        info_bg: rgba(0x60a5fa, 0.12),

        // Borders
        border: rgb(0x2a2a2f),
        border_subtle: rgba(0xffffff, 0.06), // Section dividers

        // Buttons
        primary_button_bg: linear_gradient(rgb(0x5e6ad2), rgb(0x7c6fe3)),
        primary_button_hover: linear_gradient(rgb(0x6e7ae2), rgb(0x8c7ff3)),

        // Scrollbar
        scrollbar_track: rgba(0xffffff, 0.04),
        scrollbar_thumb: rgba(0xffffff, 0.12),
        scrollbar_thumb_hover: rgba(0xffffff, 0.20),
    }
}
```

### 6.2 ZFaktury Light

```rust
pub fn zfaktury_light() -> Theme {
    Theme {
        name: "ZFaktury Light".into(),

        // Backgrounds
        base: rgb(0xffffff),
        surface: rgb(0xf8f9fa),
        elevated: rgb(0xf1f3f5),
        hover: rgb(0xe9ecef),
        active: rgb(0xdee2e6),

        // Text
        text_primary: rgb(0x1a1a1a),
        text_secondary: rgb(0x555560),
        text_muted: rgb(0x8b8b96),
        text_on_accent: rgb(0xffffff),

        // Accent (deeper indigo for contrast on white)
        accent: rgb(0x4f5bd5),
        accent_hover: rgb(0x5f6be5),
        accent_muted: rgba(0x4f5bd5, 0.10),

        // Semantic colors
        success: rgb(0x16a34a),
        success_bg: rgba(0x16a34a, 0.10),
        warning: rgb(0xd97706),
        warning_bg: rgba(0xd97706, 0.10),
        danger: rgb(0xdc2626),
        danger_bg: rgba(0xdc2626, 0.10),
        info: rgb(0x2563eb),
        info_bg: rgba(0x2563eb, 0.10),

        // Borders
        border: rgb(0xe5e7eb),
        border_subtle: rgb(0xf0f0f2),

        // Buttons
        primary_button_bg: linear_gradient(rgb(0x4f5bd5), rgb(0x6c61d8)),
        primary_button_hover: linear_gradient(rgb(0x5f6be5), rgb(0x7c71e8)),

        // Scrollbar
        scrollbar_track: rgba(0x000000, 0.04),
        scrollbar_thumb: rgba(0x000000, 0.15),
        scrollbar_thumb_hover: rgba(0x000000, 0.25),
    }
}
```

### 6.3 Fonts

- **UI font:** Inter (loaded from embedded assets, fallback to system-ui)
- **Monospace font:** JetBrains Mono (loaded from embedded assets)
- JetBrains Mono is used for ALL monetary values, invoice numbers, percentage values, and any tabular numeric data
- Font sizes follow the existing scale: 11px (xs/muted), 13px (sm/body), 14px (base), 16px (section headers), 18px (stat numbers), 20px (page title)

### 6.4 Theme Registration

```rust
fn register_themes(cx: &mut AppContext) {
    let registry = ThemeRegistry::global(cx);
    registry.register(zfaktury_dark());
    registry.register(zfaktury_light());

    // Load preference from config, default to dark
    let preferred = load_theme_preference();
    cx.set_theme(match preferred {
        ThemePreference::Dark => zfaktury_dark(),
        ThemePreference::Light => zfaktury_light(),
    });
}
```

Theme preference is persisted to the config file (`~/.zfaktury/config.toml`) under `[app] theme = "dark"`.

---

## 7. Screen Specifications

All 43 screens are ported from the existing SvelteKit pages. This section specifies the GPUI implementation details for each.

### 7.1 Dashboard (`Route::Dashboard`)

**Data source:** `DashboardService::get()`
**Layout:** Centered, max-width 1200px

**Components:**
1. **Page header**: "ZFaktury" h1 + "Přehled vašeho podnikání" subtitle
2. **4 stat cards** (horizontal grid, equal width):

| Card | Value field | Value style | Secondary |
|------|------------|-------------|-----------|
| Příjmy tento měsíc | `revenue_current_month` | JetBrains Mono 18px, text-primary | -- |
| Náklady tento měsíc | `expenses_current_month` | JetBrains Mono 18px, text-primary | -- |
| Neuhrazené faktury | `unpaid_count` | JetBrains Mono 18px, text-warning | `unpaid_total` in JetBrains Mono 12px text-muted |
| Faktury po splatnosti | `overdue_count` | JetBrains Mono 18px, text-danger | `overdue_total` in JetBrains Mono 12px text-muted |

3. **Bar chart**: "Příjmy vs Náklady" -- 12-month grouped bar chart
   - Labels: Led, Úno, Bře, Dub, Kvě, Čer, Čvc, Srp, Zář, Říj, Lis, Pro
   - Green bars: revenue, Red bars: expenses
   - Height: 300px
   - Tooltip on hover showing exact amount in CZK format

4. **Two recent tables** (2-column grid on wide screens):
   - **Poslední faktury** (5 rows): Číslo, Stav (badge), Částka, Datum
   - **Poslední náklady** (5 rows): Popis, Kategorie, Částka, Datum
   - Clickable rows navigate to detail
   - Empty state: "Zatím žádné faktury." / "Zatím žádné náklady."

5. **Tax calendar strip** (below charts): next 3 upcoming tax deadlines
   - Each deadline: date (JetBrains Mono), description, countdown badge

**Loading state:** Centered spinner with "Načítání..." sr-only text
**Error state:** Red alert box with error message

### 7.2 Invoice List (`Route::InvoiceList`)

**Data source:** `InvoiceService::list(filter)`
**Layout:** Full width (no max-width constraint for the table)

**Components:**
1. **Page header**: "Faktury" h1 + "Nová faktura" primary button (right-aligned)
2. **Filter bar**:
   - Search input (instant, debounced 200ms, searches number + customer name)
   - Status dropdown: Vše, Koncept, Odeslaná, Uhrazená, Po splatnosti, Stornovaná
   - Date range picker: Od / Do
   - Type filter: Vše, Faktura, Zálohová faktura, Dobropis
3. **DataTable** with virtual scrolling:
   - Columns:

| Column | Width | Align | Font | Content |
|--------|-------|-------|------|---------|
| Číslo | 140px | left | Inter medium | invoice_number |
| Zákazník | flex | left | Inter | customer.name |
| Datum | 100px | right | Inter | issue_date (DD.MM.YYYY) |
| Splatnost | 100px | right | Inter | due_date (DD.MM.YYYY) |
| Částka | 120px | right | JetBrains Mono | total_amount (CZK format) |
| Stav | 110px | center | Inter | StatusBadge component |

   - Column headers: clickable for sort (ascending/descending/none cycle)
   - Column dividers: draggable for resize
   - Fixed header during scroll
   - Row height: 40px
   - Hover: bg-hover
   - Click: navigate to `InvoiceDetail(id)`
   - Keyboard: arrow keys to move selection, Enter to open

4. **Status badges** (colors match existing `statusColors` from `invoice.ts`):
   - Koncept: bg-elevated text-secondary
   - Odeslaná: bg-info-bg text-info
   - Uhrazená: bg-success-bg text-success
   - Po splatnosti: bg-danger-bg text-danger
   - Stornovaná: bg-elevated text-muted

5. **Empty state**: Centered illustration + "Zatím žádné faktury." + "Vytvořit první fakturu" link
6. **Pagination**: "Zobrazeno X-Y z Z" label + page size selector (25, 50, 100)

**Split-view mode (Ctrl+\\):**
- Table shrinks to left panel (300-500px, resizable divider)
- Selected invoice detail renders in right panel
- Clicking rows updates right panel without full navigation

### 7.3 Invoice Detail (`Route::InvoiceDetail(id)`)

**Data source:** `InvoiceService::get(id)`
**Layout:** Centered, max-width 960px

**Components:**
1. **Header section**:
   - Invoice number (h1, JetBrains Mono)
   - Type badge (Faktura / Zálohová faktura / Dobropis)
   - Status badge (current status)
   - "Upravit" toggle button (switches to edit mode)

2. **Context bar actions**:
   - "Odeslat" (opens email dialog)
   - "Uhrazeno" (mark as paid, confirm dialog)
   - "Duplikovat" (creates new invoice with same data)
   - "Dobropis" (opens credit note dialog)
   - "PDF" dropdown: Stáhnout PDF, Stáhnout ISDOC, Zobrazit QR
   - "Smazat" (danger, confirm dialog -- only for drafts)

3. **Info sections** (card-based layout):
   - **Customer info**: Name, ICO, DIC, address (clickable, navigates to contact)
   - **Dates**: Issue date, Due date, Delivery date, Paid at (all DD.MM.YYYY format)
   - **Payment**: Variable symbol, Constant symbol, Payment method, Bank account, IBAN
   - **Items table**: Description, Qty, Unit, Unit price, VAT rate, VAT amount, Total
   - **Totals**: Subtotal, VAT breakdown by rate, Total (highlighted, JetBrains Mono 18px)
   - **Notes**: Public notes, Internal notes (muted)
   - **Documents**: List of attached files with download links
   - **Related invoices**: If credit note or proforma, show related invoices

4. **Status timeline** (StatusTimeline component):
   - Visual timeline showing: Created -> Sent -> Paid (with dates)
   - Current status highlighted

5. **View/edit toggle**: Smooth crossfade animation (150ms) between read-only and form views

### 7.4 Invoice Create/Edit (`Route::InvoiceNew` / edit mode in detail)

**Data sources:** `ContactService::list()`, `SequenceService::list()`, `SettingsService::get()`
**Layout:** Centered, max-width 960px

**Form sections:**

1. **Invoice metadata**:
   - Type selector: Faktura / Zálohová faktura
   - Sequence picker (dropdown, filters by invoice type)
   - Invoice number (auto-generated from sequence, read-only)
   - Variable symbol (auto-filled from number, editable)

2. **Customer**:
   - Contact picker (searchable dropdown with typeahead)
   - "Nový kontakt" quick-add button
   - Once selected: shows name, ICO, DIC, address (read-only summary)

3. **Dates**:
   - Issue date (DateInput with calendar picker)
   - Due date (DateInput, with preset buttons: +7, +14, +30, +60 days from issue date)
   - Delivery date (DateInput, defaults to issue date)

4. **Currency & payment**:
   - CurrencySelector: CZK (default), EUR, USD, GBP + custom
   - Exchange rate input (auto-fetched from CNB when non-CZK selected)
   - Payment method: Bankovní převod / Hotovost / Karta
   - Bank account fields (pre-filled from settings)

5. **Invoice items** (InvoiceItemsEditor):
   - Inline editable table rows
   - Columns: Popis, Množství, Jednotka, Cena za jednotku, DPH %, DPH, Celkem
   - "Přidat položku" button at bottom
   - Drag handle for reordering
   - Delete (X) button per row
   - VAT rate selector per row: 0%, 12%, 21%
   - Live calculation: changing quantity/price/vat immediately updates totals
   - All monetary values in JetBrains Mono

6. **Totals** (auto-calculated, read-only):
   - Základ daně (subtotal)
   - DPH 12% / DPH 21% (broken down by rate, only shown if items use that rate)
   - Celkem (total, highlighted)

7. **Notes**:
   - Poznámka (public, shown on invoice)
   - Interní poznámka (private, not shown on invoice)

8. **Actions**:
   - "Uložit jako koncept" (primary)
   - "Uložit a odeslat" (secondary)
   - "Zrušit" (ghost, navigates back)

**Validation:**
- At least one item required
- Customer required
- Issue date required
- Due date required and >= issue date
- All item quantities > 0
- All item unit prices > 0
- Errors shown inline below relevant fields with red text

**Inspector panel (Ctrl+P):**
- Shows live PDF preview
- 500ms debounce: after any form field change, regenerate PDF in background
- Zoom with scroll wheel
- "Stáhnout PDF" button at bottom

### 7.5 Expense List (`Route::ExpenseList`)

**Data source:** `ExpenseService::list(filter)`
**Layout:** Full width

**DataTable columns:**

| Column | Width | Content |
|--------|-------|---------|
| Číslo | 120px | expense number or "---" |
| Kategorie | 120px | CategoryBadge (color dot + name) |
| Popis | flex | description |
| Dodavatel | 150px | vendor name or contact name |
| Datum | 100px | issue_date (DD.MM.YYYY) |
| Částka | 120px | total_amount (JetBrains Mono, CZK) |
| DPH | 80px | vat_amount (JetBrains Mono, CZK) |

**Filters:** search, category dropdown, date range, vendor, tax reviewed (toggle)

**Category badges:** colored dot (matching `ExpenseCategory.color`) + category name

### 7.6 Expense Detail/Create/Edit

Similar structure to invoice, with expense-specific fields:

- **Category picker**: color-coded dropdown with all categories
- **Business percent slider**: 0-100% slider with numeric input (determines tax-deductible portion)
- **Tax deductible toggle**: boolean
- **Vendor**: contact picker or free-text vendor name
- **Document upload**: drag & drop zone + file browser button
  - Shows list of uploaded documents with thumbnails
  - Delete button per document
  - OCR review button (if OCR configured in settings)
- **Expense items**: same InvoiceItemsEditor pattern (shared component logic)

### 7.7 Expense Import (`Route::ExpenseImport`)

**Layout:** Centered, max-width 960px

1. **Upload zone**: Large drag & drop area for PDFs/images
   - "Přetáhněte soubory sem" text + file browser button
   - Accepts: PDF, PNG, JPG, JPEG
   - Multiple files allowed
   - Progress bar per file during upload

2. **Uploaded documents list**: Shows each uploaded document with:
   - Thumbnail preview
   - Filename, file size
   - OCR status indicator (pending / processing / done / error)
   - "Vytvořit náklad" button (navigates to pre-filled expense form)
   - "Smazat" button

### 7.8 Expense Review (`Route::ExpenseReview`)

**Layout:** Full width, two-column

- Left: document viewer (PDF/image preview, zoomable)
- Right: expense form pre-filled from OCR data
- OCR fields highlighted with confidence indicators (green/yellow/red border)
- "Potvrdit a uložit" button saves expense and marks document as reviewed

### 7.9 Contact List (`Route::ContactList`)

**Data source:** `ContactService::list(filter)`

**DataTable columns:**

| Column | Content |
|--------|---------|
| Jméno / Název | name (bold if favorite, star icon) |
| IČO | ico (JetBrains Mono) |
| DIČ | dic (JetBrains Mono) |
| Město | city |
| Email | email |
| Typ | "Firma" / "Osoba" badge |

**Filters:** search (name, ICO, email), type toggle (Vše / Firma / Osoba), favorites only toggle

**Actions:** "Nový kontakt" button, bulk actions (future)

### 7.10 Contact Detail (`Route::ContactDetail(id)`)

**Layout:** Two-column (info left, related data right)

**Left column (contact info):**
- Name (h1), type badge
- ICO, DIČ (with ARES lookup button for Czech companies)
- Address: street, city, zip, country
- Email, phone, web
- Bank: account, bank code, IBAN, SWIFT
- Payment terms: X days
- Tags (comma-separated badges)
- Notes
- Edit/Delete buttons
- Favorite toggle (star icon)
- VAT unreliable warning (if `vat_unreliable_at` set)

**Right column (related data):**
- **Faktury pro tento kontakt**: last 10 invoices table (click to navigate)
- **Náklady od tohoto kontaktu**: last 10 expenses table (click to navigate)
- **Souhrnné statistiky**: total invoiced, total paid, total expenses, outstanding

**ARES lookup:** Button next to ICO field. When clicked:
- Spawns background task to query ARES API
- On success: fills name, address fields automatically
- On error: toast notification with error

### 7.11 Recurring Invoice List (`Route::RecurringList`)

**DataTable columns:** Name/label, Customer, Frequency (badge), Next issue date, Amount, Active (toggle)

**Frequency badges:** Týdně (purple), Měsíčně (blue), Čtvrtletně (green), Ročně (orange)

### 7.12 Recurring Invoice Form (`Route::RecurringNew` / `Route::RecurringDetail(id)`)

Same as invoice form, plus:
- **Frequency selector**: Týdně / Měsíčně / Čtvrtletně / Ročně
- **Start date / End date** (optional end)
- **Active toggle**
- **Next issue date** (read-only, calculated)
- **Generated invoices list** (for existing templates, shows history)

### 7.13 Recurring Expense List/Form

Same pattern as recurring invoices, adapted for expense fields.

### 7.14 VAT Overview (`Route::VatOverview`)

**Layout:** Centered, max-width 1200px

1. **Year selector**: dropdown at top right (current year default)

2. **Quarter grid**: 4 quarter cards in a row
   - Each card shows 3 months
   - Each month cell: color-coded border
     - Green: filed
     - Blue: ready (calculated, not filed)
     - Dashed gray: draft / not started
   - Month cell shows: period label, total VAT amount

3. **Quick action buttons**:
   - "Nové přiznání k DPH" -> `VatReturnNew`
   - "Nové kontrolní hlášení" -> `ControlStatementNew`
   - "Nové hlášení VIES" -> `ViesNew`

4. **Filing list**: DataTable of all filings for selected year
   - Columns: Typ (Přiznání / KH / VIES), Období, Stav, Datum podání, Akce
   - Click to navigate to detail

### 7.15 VAT Return Detail (`Route::VatReturnDetail(id)`)

**Layout:** Centered, max-width 960px

1. **Summary card**: Period, Status badge, Total VAT (JetBrains Mono 18px)
2. **VAT breakdown table**:
   - Rows grouped by rate: 21%, 12%, 0%
   - Columns: Základ daně, DPH
   - Separate sections for: Výstupy (outputs), Vstupy (inputs), Rozdíl
3. **Linked invoices**: DataTable of invoices included in this return
4. **Linked expenses**: DataTable of expenses included in this return
5. **Actions**:
   - "Přepočítat" (recalculate from current data)
   - "Generovat XML" (generate XML for submission)
   - "Označit jako podané" (mark as filed)
   - "Stáhnout XML" (download generated XML file)
6. **Status transitions**: Draft -> Ready (after recalculate) -> Filed

### 7.16 VAT Return Form (`Route::VatReturnNew`)

- Period selector (quarter/month picker)
- Tax office code
- "Vypočítat" button: triggers background calculation, shows progress
- After calculation: shows preview of return data (same as detail summary)
- "Uložit" saves as draft

### 7.17 Control Statement Detail/Form

Similar to VAT return, with control statement-specific fields:
- Section A1-A5, B1-B3 breakdown
- Transaction-level detail table
- XML generation for EPO submission

### 7.18 VIES Declaration Detail/Form

- EU trade partner list (by country)
- Amounts per partner
- XML generation

### 7.19 Tax Overview (`Route::TaxOverview`)

**Layout:** Centered, max-width 1200px

1. **Year selector**
2. **Summary cards**:
   - Příjmy celkem (total revenue)
   - Náklady celkem (total expenses)
   - Základ daně (tax base)
   - Daň 15% / 23% (calculated tax)
   - Slevy na dani (total credits)
   - Daň po slevách (tax after credits)

3. **Filing sections** (3 expandable cards):
   - **Daň z příjmů**: list of income tax filings + "Nové přiznání" button
   - **Sociální pojištění**: list of social insurance filings + "Nový přehled" button
   - **Zdravotní pojištění**: list of health insurance filings + "Nový přehled" button

4. **Tax calendar**: upcoming deadlines with countdown

### 7.20 Tax Credits (`Route::TaxCredits`)

**Layout:** Centered, max-width 960px

1. **Year selector** at top
2. **4 card sections**:
   - **Sleva na poplatníka** (personal credit): toggle + amount (read-only, statutory)
   - **Sleva na manžela/ku** (spouse credit): toggle + income field + amount
   - **Slevy na děti** (children credits): list with add/edit/delete
     - Per child: name, birth date, disability status, ZTP holder, months
     - Auto-calculated amount per child
   - **Odpočty** (deductions): list with add/edit/delete
     - Types: pension insurance, life insurance, mortgage interest, donations, union dues, exams
     - Per deduction: type, amount, document reference
     - Document upload per deduction
3. **Total summary**: sum of all credits and deductions

### 7.21 Tax Prepayments (`Route::TaxPrepayments`)

**Layout:** Centered, max-width 960px

- Year selector
- Prepayment schedule table: Date, Amount, Status (paid/unpaid), Payment date
- "Přidat zálohu" button
- Toggle paid status per row
- Summary: total prepaid, remaining

### 7.22 Tax Investments (`Route::TaxInvestments`)

**Layout:** Centered, max-width 960px

1. **3 tabs** (gpui-component TabBar):
   - **Dokumenty**: uploaded broker statements, annual summaries
   - **Kapitálové příjmy (ss 8)**: capital income from dividends, interest
   - **Ostatní příjmy (ss 10)**: security transactions (buy/sell with FIFO)

2. **Investment summary card**: Year totals (gross income, costs, net)

3. **FIFO recalculation**: "Přepočítat FIFO" button
   - Background calculation
   - Shows transaction-level FIFO matching results

4. **Document upload**: drag & drop for broker statements
   - OCR extraction for recognized formats

### 7.23 Income Tax Detail/Form

**Tax filing wizard** (4-step Stepper from gpui-component):

**Step 1 - Příprava dat:**
- Summary of all data sources (revenue, expenses, credits, prepayments, investments)
- Missing data warnings (e.g., "Nemáte vyplněné slevy na dani")
- "Pokračovat" button

**Step 2 - Kontrola:**
- Full tax calculation preview
- Line-by-line review with expandable details
- Warnings for mismatches (e.g., prepayments vs. calculated tax)
- "Zpět" / "Pokračovat" buttons

**Step 3 - Generování XML:**
- Progress indicator during XML generation
- Preview of generated data
- "Generovat" button
- On success: shows download link

**Step 4 - Podání:**
- "Označit jako podané" button
- Filing date selector
- "Stáhnout XML" button
- Status: Koncept -> Vygenerováno -> Podáno

### 7.24 Social Insurance Detail/Form

Similar wizard structure to income tax, with social insurance-specific fields:
- Assessment base calculation
- Insurance rate (29.2%)
- Monthly prepayment calculation
- ČSSZ XML format

### 7.25 Health Insurance Detail/Form

Similar wizard structure:
- Assessment base (50% of tax base)
- Insurance rate (13.5%)
- Monthly prepayment calculation
- VZP/other insurer XML format

### 7.26 Settings - Firma (`Route::SettingsFirma`)

**Layout:** Two-column settings page (nav list left 200px, form right)

Settings nav (left column, shared across all settings pages):
- Firma (active)
- Email
- Číselné řady
- Kategorie
- PDF šablona
- Import z Fakturoid
- Audit log
- Zálohy

**Form fields:**
- Název firmy / Jméno
- IČO (with ARES lookup button)
- DIČ
- Ulice, Město, PSČ, Země
- Email, Telefon, Web
- Bankovní účet, Kód banky
- IBAN, SWIFT
- Logo upload (drag & drop, preview)

"Uložit" button at bottom. Toast notification on success.

### 7.27 Settings - Email (`Route::SettingsEmail`)

- SMTP server, port, username, password (masked)
- From address, From name
- TLS toggle
- "Odeslat testovací email" button (sends to from address)
- Email templates section (invoice sent, payment reminder)

### 7.28 Settings - Sequences (`Route::SettingsSequences`)

- List of number sequences (DataTable)
- Columns: Název, Předpona, Aktuální číslo, Formát, Typ (faktura/náklad)
- "Nová řada" button opens dialog
- Edit/delete per row
- Dialog fields: name, prefix, current number, format pattern, reset period (yearly/never)

### 7.29 Settings - Categories (`Route::SettingsCategories`)

- Reorderable list of expense categories
- Per category: name, color (color picker), description
- Drag handle for reorder
- "Nová kategorie" button
- Delete with confirmation (only if no expenses use it, else show count)

### 7.30 Settings - PDF (`Route::SettingsPdf`)

- Logo upload with preview
- Accent color picker (used in PDF header)
- Footer text (custom text at bottom of invoices)
- QR payment code toggle (show/hide QR on invoices)
- Font size selector (small/medium/large)
- "Náhled" button: generates sample PDF and shows in inspector panel

### 7.31 Settings - Audit Log (`Route::SettingsAuditLog`)

- Searchable DataTable with virtual scrolling
- Columns: Datum, Uživatel, Entita, Akce, Detail
- Filters: entity type dropdown, action type dropdown, date range
- Entity types: Faktura, Náklad, Kontakt, Nastavení, etc.
- Action types: Vytvořeno, Upraveno, Smazáno, Odesláno, etc.
- Click row to expand and show full JSON diff

### 7.32 Settings - Backup (`Route::SettingsBackup`)

- **Local backups**: list of backup files with date, size
  - "Vytvořit zálohu" button (creates SQLite backup)
  - "Obnovit" button per backup (with confirmation dialog, warns about data loss)
  - "Stáhnout" button per backup (native file save dialog)
- **S3 configuration** (optional):
  - Endpoint, bucket, access key, secret key
  - "Otestovat připojení" button
  - Auto-backup schedule: Off / Denně / Týdně
- **Import**: "Nahrát zálohu" button (native file open dialog)

### 7.33 Import from Fakturoid (`Route::ImportFakturoid`)

- API key input
- Account slug input
- "Připojit" button (validates credentials)
- Import options checkboxes: Kontakty, Faktury, Náklady
- "Spustit import" button
- Progress indicator: X/Y contacts, X/Y invoices, X/Y expenses
- Completion summary with any errors/warnings

### 7.34 Reports (`Route::Reports`)

**Layout:** Centered, max-width 1200px

**5 tabs** (gpui-component TabBar):

1. **Příjmy** (Revenue):
   - Year selector
   - BarChart: monthly revenue (12 bars)
   - Summary: total, average monthly, best month, worst month

2. **Náklady** (Expenses):
   - Year selector
   - BarChart: monthly expenses (12 bars)
   - DoughnutChart: expenses by category
   - Summary: total, average monthly, top category

3. **Zisk a ztráta** (Profit/Loss):
   - Year selector
   - BarChart: monthly revenue vs expenses (grouped)
   - Line overlay: cumulative profit
   - Summary table: monthly breakdown

4. **Top zákazníci** (Top Customers):
   - Year selector
   - DataTable: rank, customer name, total invoiced, invoice count, average invoice
   - Top 20 customers sorted by total invoiced

5. **Daňový kalendář** (Tax Calendar):
   - Year selector
   - Timeline view of all tax deadlines
   - Each deadline: date, description, status (upcoming/past/filed)
   - Color coding: green (filed), blue (upcoming), red (overdue), gray (past)

---

## 8. Shared Components

### 8.1 StatusBadge

```rust
pub struct StatusBadge {
    label: SharedString,
    variant: StatusVariant,
}

pub enum StatusVariant {
    Default,  // bg-elevated text-secondary (draft)
    Info,     // bg-info-bg text-info (sent)
    Success,  // bg-success-bg text-success (paid)
    Danger,   // bg-danger-bg text-danger (overdue)
    Warning,  // bg-warning-bg text-warning
    Muted,    // bg-elevated text-muted (cancelled)
}
```

Renders as a rounded pill with padding, matching the existing Svelte `statusColors` map exactly.

### 8.2 DateInput

- Text input with DD.MM.YYYY format mask
- Calendar popup on click/focus (month grid)
- Arrow keys navigate calendar
- Escape closes calendar
- Preset buttons (configurable): +7d, +14d, +30d, +60d
- Validates date on blur, shows error for invalid dates

### 8.3 CurrencySelector

- Dropdown with common currencies: CZK, EUR, USD, GBP
- Custom currency input option
- When non-CZK selected: auto-fetches exchange rate from CNB on background thread
- Exchange rate input (editable, pre-filled from CNB)
- Rate date shown (CNB publishes daily)

### 8.4 CategoryPicker

- Dropdown showing all expense categories
- Each option: colored dot + category name
- Search/filter within dropdown
- "Nová kategorie" quick-add at bottom

### 8.5 ContactPicker

- Searchable dropdown/combobox
- Shows: name, ICO, city in each option
- Fuzzy search (diacritics-insensitive)
- "Nový kontakt" quick-add button
- Selected contact: shows summary card below picker

### 8.6 DocumentUpload

- Drag & drop zone (dashed border, changes color on dragover)
- "Vybrat soubory" button (native file open dialog)
- File list with: name, size, type icon, delete button
- Upload progress per file
- Accepted types configurable (PDF, images, etc.)

### 8.7 InvoiceItemsEditor

Shared between invoice and expense forms.

- Inline editable table
- Columns: Popis (text), Množství (number), Jednotka (text), Cena/j (money), DPH % (select), DPH (computed, read-only), Celkem (computed, read-only)
- Add row button at bottom
- Delete (X) button per row
- Drag handle for reorder
- Tab navigates between cells
- Enter in last cell creates new row
- All monetary columns use JetBrains Mono
- Live total calculation on every keystroke

### 8.8 BarChart

GPU-rendered bar chart using GPUI primitives.

```rust
pub struct BarChart {
    labels: Vec<SharedString>,
    datasets: Vec<ChartDataset>,
    height: Pixels,
    show_legend: bool,
    show_tooltip: bool,
}

pub struct ChartDataset {
    label: SharedString,
    data: Vec<f64>,
    color: Hsla,
}
```

Features:
- Gradient fills on bars
- Hover tooltip showing exact value (formatted as CZK)
- Legend at top (dataset labels with colored indicators)
- Y-axis labels (auto-scaled)
- X-axis labels (from `labels` vec)
- Staggered entrance animation on data load (400ms total, 30ms stagger per bar)
- Smooth transition when data changes

### 8.9 DoughnutChart

GPU-rendered doughnut/pie chart.

```rust
pub struct DoughnutChart {
    segments: Vec<ChartSegment>,
    size: Pixels,
    show_legend: bool,
}

pub struct ChartSegment {
    label: SharedString,
    value: f64,
    color: Hsla,
}
```

Features:
- Hover: segment expands slightly (2px), tooltip shows label + value + percentage
- Legend beside chart (right side)
- Center text: total value
- Smooth arc animation on data load

### 8.10 ConfirmDialog

Modal dialog for destructive actions.

- Title, message, confirm button (danger variant), cancel button
- Focus trapped within dialog
- Escape closes (cancels)
- Enter confirms
- Backdrop: semi-transparent dark overlay

### 8.11 EmailDialog

Modal for sending invoices via email.

- To field (pre-filled from contact email)
- Subject (pre-filled: "Faktura {number}")
- Body (template with placeholders)
- Attach PDF toggle (default: on)
- "Odeslat" / "Zrušit" buttons
- Sending progress indicator

### 8.12 CreditNoteDialog

Modal for creating credit notes from existing invoices.

- Shows original invoice summary
- Reason field (text)
- Items to credit (checkboxes, pre-selected all)
- Partial credit amount per item
- "Vytvořit dobropis" button

### 8.13 SplitView

Container component for master-detail layouts.

```rust
pub struct SplitView {
    left_panel: AnyView,
    right_panel: AnyView,
    divider_position: Pixels,  // draggable
    min_left: Pixels,          // 300px
    min_right: Pixels,         // 400px
}
```

- Draggable divider (3px wide, hover: 6px visual width)
- Double-click divider: reset to default position
- Ctrl+\\ toggles between split and full-width

### 8.14 StatusTimeline

Visual timeline for entity lifecycle.

```rust
pub struct StatusTimeline {
    steps: Vec<TimelineStep>,
    current_index: usize,
}

pub struct TimelineStep {
    label: SharedString,
    date: Option<NaiveDate>,
    status: TimelineStepStatus,
}

pub enum TimelineStepStatus {
    Completed,
    Current,
    Upcoming,
}
```

Renders as horizontal dots connected by lines. Completed steps: accent color. Current: pulsing accent. Upcoming: muted.

### 8.15 Toast

Non-modal notification system.

```rust
pub enum ToastLevel {
    Success,
    Error,
    Warning,
    Info,
}

pub struct Toast {
    message: SharedString,
    level: ToastLevel,
    duration: Duration, // default 4s, errors 8s
    action: Option<(SharedString, Box<dyn Fn()>)>, // optional action button
}
```

- Stacks from bottom-right
- Auto-dismiss after duration
- Hover pauses auto-dismiss
- Click X to dismiss manually
- Entrance: slide in from right (200ms)
- Exit: fade out (150ms)

---

## 9. Command Palette

### 9.1 Activation

- **Ctrl+K**: opens command palette
- **Escape**: closes
- Frosted glass (glassmorphism) backdrop overlay

### 9.2 Search Behavior

The command palette provides a unified search across multiple sources:

1. **Commands** (always available):
   - "Nová faktura" -> navigates to InvoiceNew
   - "Nový náklad" -> navigates to ExpenseNew
   - "Nový kontakt" -> navigates to ContactNew
   - "Nastavení" -> navigates to SettingsFirma
   - "Přepnout téma" -> toggles dark/light
   - "Zavřít panel" -> closes inspector
   - All keyboard shortcuts shown as hints

2. **Invoices** (searched by number):
   - Prefix "f:" or "faktura:" to force invoice search
   - Shows: number, customer, status badge, amount
   - Enter: navigates to InvoiceDetail

3. **Contacts** (searched by name, ICO):
   - Prefix "k:" or "kontakt:" to force contact search
   - Shows: name, ICO, city
   - Enter: navigates to ContactDetail

4. **Expenses** (searched by description, vendor):
   - Prefix "n:" or "naklad:" to force expense search
   - Shows: description, vendor, amount, date
   - Enter: navigates to ExpenseDetail

### 9.3 Fuzzy Search

Uses the `nucleo` crate for fuzzy matching with Czech diacritics normalization:

```rust
/// Normalize Czech diacritics for fuzzy matching.
/// č->c, ř->r, ž->z, š->s, ď->d, ť->t, ň->n, ú->u, ů->u, á->a, é->e, í->i, ó->o, ý->y, ě->e
pub fn normalize_czech(input: &str) -> String { ... }
```

Matching process:
1. Normalize both query and candidate
2. Run nucleo fuzzy match on normalized strings
3. Score and rank results
4. Group by source type (commands first, then invoices, contacts, expenses)

### 9.4 Recent Items

Below the search input, show 5 most recently visited items (persisted in memory, not to disk):
- Format: icon + label + route description
- Click or Enter to navigate

### 9.5 UI Layout

```
+-------------------------------------------+
| [search icon] Search ZFaktury...    [Esc]  |
+-------------------------------------------+
| Naposledy navštívené                       |
|   Faktura FV-2026-0042        Faktury      |
|   Kontakt ABC s.r.o.          Kontakty     |
|   Nastavení firmy             Nastavení    |
+-------------------------------------------+
| Příkazy                                    |
|   Nová faktura                 Ctrl+N      |
|   Nový náklad                  Ctrl+Shift+N|
|   Přepnout téma                            |
+-------------------------------------------+
| Faktury                                    |
|   FV-2026-0042  ABC s.r.o.  ■ Uhrazená    |
|   FV-2026-0041  XYZ a.s.   ■ Odeslaná     |
+-------------------------------------------+
```

---

## 10. Keyboard Shortcuts

### 10.1 Global Shortcuts

| Shortcut | Action | Context |
|----------|--------|---------|
| Ctrl+K | Open command palette | Always |
| Ctrl+N | New invoice | Always |
| Ctrl+Shift+N | New expense | Always |
| Ctrl+, | Open settings | Always |
| Ctrl+Shift+L | Toggle sidebar collapse | Always |
| Ctrl+\\ | Toggle split view | List views |
| Ctrl+F | Focus search / filter input | Always |
| Alt+Left | Navigate back | Always |
| Alt+Right | Navigate forward | Always |

### 10.2 List View Shortcuts

| Shortcut | Action |
|----------|--------|
| Up/Down | Move row selection |
| Enter | Open selected item |
| Delete | Delete selected (with confirmation) |
| Home | Select first row |
| End | Select last row |
| Page Up/Down | Scroll one page |

### 10.3 Detail/Form Shortcuts

| Shortcut | Action |
|----------|--------|
| Ctrl+S | Save current form |
| Ctrl+P | Toggle PDF preview (invoices) |
| Ctrl+E | Toggle edit mode |
| Escape | Cancel edit / close dialog |
| Tab | Next form field |
| Shift+Tab | Previous form field |

### 10.4 Dialog Shortcuts

| Shortcut | Action |
|----------|--------|
| Escape | Close / cancel |
| Enter | Confirm (when confirm button focused) |
| Tab | Cycle through dialog fields |

### 10.5 Implementation

```rust
pub struct KeyboardShortcutRegistry {
    shortcuts: HashMap<Keystroke, Box<dyn Action>>,
}

impl KeyboardShortcutRegistry {
    pub fn register_global_shortcuts(cx: &mut AppContext) {
        cx.bind_keys([
            KeyBinding::new("ctrl-k", ToggleCommandPalette, None),
            KeyBinding::new("ctrl-n", NavigateNewInvoice, None),
            KeyBinding::new("ctrl-shift-n", NavigateNewExpense, None),
            KeyBinding::new("ctrl-,", NavigateSettings, None),
            KeyBinding::new("ctrl-shift-l", ToggleSidebar, None),
            KeyBinding::new("ctrl-\\", ToggleSplitView, None),
            KeyBinding::new("ctrl-f", FocusSearch, None),
            KeyBinding::new("alt-left", NavigateBack, None),
            KeyBinding::new("alt-right", NavigateForward, None),
        ]);
    }
}
```

Shortcuts are context-sensitive: form shortcuts only active when a form view has focus, list shortcuts only when a table has focus.

---

## 11. Virtual Scrolling

### 11.1 Architecture

All list views use gpui-component's `Table` with virtual row rendering. Only visible rows (plus a small overscan buffer) are rendered at any time.

```rust
pub struct VirtualTable<T: 'static> {
    items: Vec<T>,
    visible_range: Range<usize>,
    row_height: Pixels,       // fixed 40px
    overscan: usize,          // 10 rows above/below viewport
    scroll_offset: Pixels,
    selected_index: Option<usize>,
    sort_column: Option<usize>,
    sort_direction: SortDirection,
    columns: Vec<ColumnDef<T>>,
}
```

### 11.2 Performance Targets

- **Smooth scrolling**: 120fps with 10,000+ rows
- **Instant filter**: search results appear within 100ms for 10,000 rows
- **Column resize**: real-time during drag, no layout thrash

### 11.3 Background Filtering

When the user types in the search/filter input:
1. Debounce 200ms
2. Clone filter params and dataset reference
3. Spawn on `background_executor()`
4. Perform fuzzy/exact match on background thread
5. Return filtered indices to main thread
6. Main thread updates `visible_items` and calls `cx.notify()`

This keeps the main thread free for rendering during large dataset filtering.

### 11.4 Scroll Performance

- `scroll_offset` is tracked as a continuous `Pixels` value (not per-row)
- Visible range is computed: `start = (offset / row_height).floor()`, `end = start + viewport_rows + overscan`
- Only rows in `[start-overscan, end+overscan]` are rendered
- Scroll events update offset and recompute visible range
- No full re-render on scroll, only element translation

---

## 12. Live PDF Preview

### 12.1 Trigger

- Ctrl+P on invoice detail or edit view opens the inspector panel with PDF preview
- On invoice edit: PDF regenerates on every form change (500ms debounce)
- On invoice detail: PDF renders once on panel open

### 12.2 Rendering Pipeline

```
Form change detected
    |
    v (500ms debounce)
Collect current form state into InvoiceForPdf struct
    |
    v (background_executor)
Call gen::generate_invoice_pdf(invoice, settings, items)
    |
    v (returns Vec<u8> PDF bytes)
Render PDF pages to images (pdf_to_image on background thread)
    |
    v (returns Vec<ImageData>)
Update inspector panel with rendered pages
    |
    v (main thread, cx.notify())
Display in scrollable image list
```

### 12.3 PDF Viewer Features

- Scrollable page list (vertical)
- Zoom: Ctrl+scroll wheel (50% to 200%, default 100%)
- Fit-to-width mode (default)
- "Stáhnout PDF" button at bottom
- Page indicator: "Strana 1 z 2"

### 12.4 File Download

When user clicks "Stáhnout PDF" or "Stáhnout XML":
1. Generate file on background thread
2. Open native file save dialog with suggested filename
3. Write file to selected path
4. Show success toast

---

## 13. Animations

### 13.1 Animation Principles

- All animations use easing curves, never linear
- Monetary values NEVER animate (instant update for all numbers)
- Animations are subtle and functional, not decorative
- Respect `prefers-reduced-motion` OS setting (disable all non-essential animations)

### 13.2 Animation Catalog

| Element | Trigger | Duration | Easing | Description |
|---------|---------|----------|--------|-------------|
| Sidebar collapse | Toggle | 200ms | ease-out | Width transition 208px <-> 52px |
| Sidebar hover expand | Mouse enter/leave | 150ms | ease-out | Width + shadow |
| Page transition | Route change | 150ms | ease-in-out | Crossfade (opacity 0->1) |
| Dialog open | Action | 200ms | ease-out | Scale 0.95->1.0 + fade |
| Dialog close | Dismiss | 150ms | ease-in | Scale 1.0->0.95 + fade |
| Toast enter | Show | 200ms | ease-out | Slide from right |
| Toast exit | Dismiss | 150ms | ease-in | Fade out |
| Chart bar enter | Data load | 400ms total | ease-out | Staggered height from 0 (30ms stagger) |
| Chart segment enter | Data load | 400ms total | ease-out | Staggered arc sweep |
| Command palette open | Ctrl+K | 200ms | ease-out | Scale 0.98->1.0 + backdrop fade |
| Split view divider | Drag | 0ms | -- | Immediate position follow |
| Table row hover | Mouse | 0ms | -- | Immediate color change |
| Status badge | Appear | 0ms | -- | Immediate (no animation) |
| Inspector panel | Toggle | 250ms | ease-out | Width 0->280px slide |

### 13.3 Implementation

```rust
fn animate_sidebar(&mut self, cx: &mut Context<Self>) {
    let target_width = if self.collapsed { px(52.0) } else { px(208.0) };
    cx.animate(
        &self.sidebar_width,
        target_width,
        Animation::new(Duration::from_millis(200))
            .with_easing(Easing::EaseOut),
    );
}
```

---

## 14. Native OS Integration

### 14.1 File Dialogs

Use GPUI's built-in file dialog API (backed by native dialogs on each platform).

```rust
// Save dialog
let path = cx.prompt_for_new_path(&PathPromptOptions {
    default_name: Some(format!("{}.pdf", invoice.number)),
    filters: vec![("PDF files", &["pdf"])],
}).await;

// Open dialog
let paths = cx.prompt_for_paths(&PathPromptOptions {
    allow_multiple: true,
    filters: vec![
        ("Documents", &["pdf", "png", "jpg", "jpeg"]),
        ("All files", &["*"]),
    ],
}).await;
```

### 14.2 Clipboard

```rust
// Copy to clipboard
cx.write_to_clipboard(ClipboardItem::new_string(invoice.number.clone()));

// Show toast confirmation
self.show_toast(cx, "Číslo faktury zkopírováno", ToastLevel::Info);
```

Used for: copy invoice number, copy IBAN, copy variable symbol, copy contact ICO.

### 14.3 Drag & Drop

Document upload zones accept file drops from the OS file manager:

```rust
fn handle_external_drop(&mut self, paths: &[PathBuf], cx: &mut Context<Self>) {
    let valid_extensions = ["pdf", "png", "jpg", "jpeg"];
    let valid_paths: Vec<_> = paths.iter()
        .filter(|p| p.extension()
            .and_then(|e| e.to_str())
            .is_some_and(|e| valid_extensions.contains(&e.to_lowercase().as_str())))
        .cloned()
        .collect();

    if valid_paths.is_empty() {
        self.show_toast(cx, "Nepodporovaný formát souboru", ToastLevel::Warning);
        return;
    }

    for path in valid_paths {
        self.upload_document(path, cx);
    }
}
```

### 14.4 System Notifications

For long-running background operations (backup, import, large PDF generation):

```rust
cx.show_notification("ZFaktury", "Záloha byla úspěšně vytvořena");
```

---

## 15. Czech Formatting Utilities

### 15.1 Money Formatting

```rust
/// Format Amount (i64, halere) as Czech CZK string.
/// Examples: 123456 -> "1 234,56 Kč", -50000 -> "-500,00 Kč", 0 -> "0,00 Kč"
pub fn format_czk(amount: Amount) -> String { ... }

/// Format Amount with explicit currency code.
/// Examples: (12345, "EUR") -> "123,45 EUR"
pub fn format_currency(amount: Amount, currency: &str) -> String { ... }
```

Thousands separator: non-breaking space (U+00A0).
Decimal separator: comma.
Currency symbol: after the number.

### 15.2 Date Formatting

```rust
/// Format date as Czech DD.MM.YYYY.
/// Example: 2026-03-20 -> "20.03.2026"
pub fn format_date(date: NaiveDate) -> String { ... }

/// Format date as relative ("dnes", "včera", "před 3 dny", "20.03.2026").
pub fn format_date_relative(date: NaiveDate) -> String { ... }

/// Format datetime as "20.03.2026 14:30".
pub fn format_datetime(dt: NaiveDateTime) -> String { ... }
```

### 15.3 Number Formatting

```rust
/// Format number with Czech locale (comma decimal, space thousands).
/// Example: 1234567.89 -> "1 234 567,89"
pub fn format_number(value: f64, decimals: usize) -> String { ... }

/// Format percentage.
/// Example: 21.0 -> "21 %"
pub fn format_percent(value: f64) -> String { ... }
```

### 15.4 Czech Month Names

```rust
pub const MONTH_LABELS_SHORT: [&str; 12] = [
    "Led", "Úno", "Bře", "Dub", "Kvě", "Čer",
    "Čvc", "Srp", "Zář", "Říj", "Lis", "Pro",
];

pub const MONTH_LABELS_FULL: [&str; 12] = [
    "Leden", "Únor", "Březen", "Duben", "Květen", "Červen",
    "Červenec", "Srpen", "Září", "Říjen", "Listopad", "Prosinec",
];
```

---

## 16. App Bootstrap & Service Wiring

### 16.1 Entry Point (`main.rs`)

```rust
fn main() {
    // 1. Load config from ~/.zfaktury/config.toml
    let config = Config::load().unwrap_or_else(|e| {
        eprintln!("Failed to load config: {e}");
        std::process::exit(1);
    });

    // 2. Initialize logging
    init_logging(&config.log);

    // 3. Open database
    let db = Database::open(&config.database_path())
        .unwrap_or_else(|e| {
            eprintln!("Failed to open database: {e}");
            std::process::exit(1);
        });

    // 4. Run migrations
    db.migrate().unwrap_or_else(|e| {
        eprintln!("Failed to run migrations: {e}");
        std::process::exit(1);
    });

    // 5. Wire services
    let services = AppServices::new(db);

    // 6. Launch GPUI app
    gpui::App::new().run(move |cx| {
        register_themes(cx);
        register_actions(cx);
        register_key_bindings(cx);
        load_fonts(cx);

        cx.open_window(
            WindowOptions {
                title: Some("ZFaktury".into()),
                bounds: WindowBounds::Windowed(Bounds::centered(
                    None, size(px(1280.0), px(800.0)), cx,
                )),
                ..Default::default()
            },
            |window, cx| {
                cx.new(|cx| RootView::new(services, cx))
            },
        ).unwrap();
    });
}
```

### 16.2 RootView

The root view manages the top-level layout and routing.

```rust
pub struct RootView {
    services: Arc<AppServices>,
    navigation: NavigationState,
    sidebar: Sidebar,
    title_bar: TitleBar,
    command_palette: Option<CommandPalette>,
    inspector: Option<InspectorPanel>,
    toast_manager: ToastManager,
    active_view: AnyView,
    split_view_enabled: bool,
}

impl Render for RootView {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        div()
            .size_full()
            .flex()
            .flex_col()
            .bg(cx.theme().base)
            .child(self.title_bar.render(window, cx))
            .child(
                div()
                    .flex_1()
                    .flex()
                    .child(self.sidebar.render(window, cx))
                    .child(
                        div()
                            .flex_1()
                            .flex()
                            .child(self.render_context_bar(window, cx))
                            .child(self.render_main_content(window, cx))
                            .children(self.inspector.as_ref().map(|i| i.render(window, cx)))
                    )
            )
            .children(self.command_palette.as_ref().map(|cp| cp.render(window, cx)))
            .child(self.toast_manager.render(window, cx))
    }
}
```

### 16.3 View Lifecycle

When navigation changes:
1. `NavigationState::navigate(route)` updates current route
2. `RootView` constructs the appropriate view for the new route
3. New view calls `load_data()` in its constructor (spawns background task)
4. Old view is dropped (GPUI handles cleanup)
5. Crossfade animation during transition (150ms)

```rust
fn create_view_for_route(
    &self,
    route: &Route,
    cx: &mut Context<Self>,
) -> AnyView {
    match route {
        Route::Dashboard => cx.new(|cx|
            DashboardView::new(self.services.clone(), cx)
        ).into(),
        Route::InvoiceList => cx.new(|cx|
            InvoiceListView::new(self.services.clone(), cx)
        ).into(),
        Route::InvoiceDetail(id) => cx.new(|cx|
            InvoiceDetailView::new(*id, self.services.clone(), cx)
        ).into(),
        // ... all 43 routes
    }
}
```

---

## 17. Error Handling

### 17.1 Error Display Strategy

| Error type | Display method |
|-----------|---------------|
| Page load failure | Full-page error card with retry button |
| Form validation | Inline red text below each invalid field |
| Save/update failure | Toast (error level, 8s duration) |
| Background task failure | Toast with retry action button |
| Network error (ARES, CNB) | Toast with "Zkusit znovu" action |
| File I/O error | Toast with details |

### 17.2 Error Card Component

```rust
pub struct ErrorCard {
    message: SharedString,
    retry_action: Option<Box<dyn Fn(&mut WindowContext)>>,
}
```

Rendered as: red-bordered card with error icon, message text, and optional "Zkusit znovu" button.

### 17.3 Form Validation Pattern

```rust
pub struct FormField<T> {
    value: T,
    error: Option<SharedString>,
    touched: bool,
}

impl<T> FormField<T> {
    pub fn set(&mut self, value: T) {
        self.value = value;
        self.touched = true;
        self.error = None;
    }

    pub fn set_error(&mut self, msg: impl Into<SharedString>) {
        self.error = Some(msg.into());
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }
}
```

Validation runs on blur (per-field) and on submit (all fields).

---

## 18. State Management

### 18.1 Per-View State

Each view owns its own state. There is no global state store (no Redux/Vuex equivalent). Views receive `Arc<AppServices>` and call service methods to load/save data.

```rust
pub struct InvoiceListView {
    services: Arc<AppServices>,
    invoices: Vec<Invoice>,
    loading: bool,
    error: Option<String>,
    filter: InvoiceFilter,
    selected_index: Option<usize>,
    sort: SortState,
}
```

### 18.2 Cross-View Communication

When one view modifies data that another view might display (e.g., creating an invoice then navigating to the list), the target view simply reloads its data on mount. There is no global event bus needed because:
- Views are created fresh on navigation
- Data is always loaded from the service layer (source of truth is the database)

For the split-view case where both list and detail are visible simultaneously:
- List view holds a `selected_id: Option<i64>`
- Detail view is a child of the list view
- When detail saves, it notifies parent list to reload

### 18.3 Preferences State

User preferences (sidebar collapsed, theme, split-view default, etc.) are persisted to the config file and loaded at startup:

```rust
pub struct AppPreferences {
    pub sidebar_collapsed: bool,
    pub theme: ThemePreference,
    pub split_view_default: bool,
    pub default_page_size: usize, // 25, 50, 100
    pub inspector_width: f32,
}
```

---

## 19. Implementation Plan

### 19.1 Week 1: Foundation

| Task | Files | Est. |
|------|-------|------|
| Crate setup, Cargo.toml, dependencies | `Cargo.toml` | 2h |
| Theme definitions (dark + light) | `theme.rs` | 4h |
| Navigation state + Route enum | `navigation.rs` | 3h |
| Font loading (Inter, JetBrains Mono) | `app.rs` | 2h |
| Czech formatting utilities | `util/format.rs` | 4h |
| Fuzzy search with diacritics | `util/fuzzy.rs` | 3h |
| Keyboard shortcut registry | `util/keyboard.rs` | 2h |
| Root view skeleton (sidebar + content) | `root.rs` | 4h |
| Title bar | `title_bar.rs` | 3h |
| Toast notification system | `toast.rs` | 3h |
| App bootstrap + service wiring | `main.rs`, `app.rs` | 3h |
| **Total** | | **33h** |

### 19.2 Week 2: Core Components + Dashboard

| Task | Files | Est. |
|------|-------|------|
| Sidebar (collapsible, all nav items) | `sidebar.rs` | 6h |
| StatusBadge component | `components/status_badge.rs` | 2h |
| BarChart (GPU-rendered) | `components/bar_chart.rs` | 8h |
| DoughnutChart (GPU-rendered) | `components/doughnut_chart.rs` | 6h |
| ConfirmDialog | `components/confirm_dialog.rs` | 3h |
| DateInput with calendar | `components/date_input.rs` | 6h |
| Dashboard view (complete) | `views/dashboard.rs` | 6h |
| **Total** | | **37h** |

### 19.3 Week 3: Invoice Screens

| Task | Files | Est. |
|------|-------|------|
| VirtualTable wrapper | Based on gpui-component Table | 6h |
| Invoice list (virtual scroll, filters) | `views/invoices/list.rs` | 8h |
| Invoice detail (view mode) | `views/invoices/detail.rs` | 6h |
| InvoiceItemsEditor | `views/invoices/items_editor.rs` | 8h |
| ContactPicker | `components/contact_picker.rs` | 4h |
| CurrencySelector | `components/currency_selector.rs` | 3h |
| Invoice form (create/edit) | `views/invoices/form.rs` | 8h |
| SplitView component | `components/split_view.rs` | 4h |
| Split-view for invoice list | Integration | 3h |
| **Total** | | **50h** |

### 19.4 Week 4: Expenses, Contacts, Recurring

| Task | Files | Est. |
|------|-------|------|
| CategoryPicker | `components/category_picker.rs` | 3h |
| DocumentUpload (drag & drop) | `components/document_upload.rs` | 5h |
| Expense list | `views/expenses/list.rs` | 4h |
| Expense detail | `views/expenses/detail.rs` | 4h |
| Expense form | `views/expenses/form.rs` | 5h |
| Expense import | `views/expenses/import.rs` | 4h |
| Expense review (2-col) | `views/expenses/review.rs` | 5h |
| Contact list | `views/contacts/list.rs` | 4h |
| Contact detail (2-col) | `views/contacts/detail.rs` | 5h |
| Recurring invoice list/form | `views/recurring/*.rs` | 6h |
| Recurring expense list/form | `views/recurring/*.rs` | 4h |
| **Total** | | **49h** |

### 19.5 Week 5: VAT, Tax, Reports

| Task | Files | Est. |
|------|-------|------|
| VAT overview (quarter grid) | `views/vat/overview.rs` | 5h |
| VAT return detail/form | `views/vat/return_*.rs` | 6h |
| Control statement detail/form | `views/vat/control_*.rs` | 5h |
| VIES detail/form | `views/vat/vies_*.rs` | 4h |
| Tax overview | `views/tax/overview.rs` | 4h |
| Tax credits | `views/tax/credits.rs` | 5h |
| Tax prepayments | `views/tax/prepayments.rs` | 3h |
| Tax investments (3 tabs) | `views/tax/investments.rs` | 5h |
| Income tax wizard (4 steps) | `views/tax/income_*.rs` | 6h |
| Social insurance wizard | `views/tax/social_*.rs` | 4h |
| Health insurance wizard | `views/tax/health_*.rs` | 4h |
| Reports (5 tabs, charts) | `views/reports.rs` | 8h |
| **Total** | | **59h** |

### 19.6 Week 6: Settings, Polish, Testing

| Task | Files | Est. |
|------|-------|------|
| Settings - Firma | `views/settings/firma.rs` | 4h |
| Settings - Email | `views/settings/email.rs` | 3h |
| Settings - Sequences | `views/settings/sequences.rs` | 3h |
| Settings - Categories (reorder) | `views/settings/categories.rs` | 4h |
| Settings - PDF | `views/settings/pdf.rs` | 3h |
| Settings - Audit log | `views/settings/audit_log.rs` | 4h |
| Settings - Backup | `views/settings/backup.rs` | 4h |
| Import from Fakturoid | `views/settings/fakturoid_import.rs` | 4h |
| Command palette (complete) | `command_palette.rs` | 8h |
| Live PDF preview | Inspector panel integration | 6h |
| EmailDialog | `components/email_dialog.rs` | 3h |
| CreditNoteDialog | `components/credit_note_dialog.rs` | 3h |
| StatusTimeline | `components/status_timeline.rs` | 3h |
| Animation polish pass | All views | 4h |
| GPUI view tests | `#[gpui::test]` across all views | 8h |
| Unit tests (format, fuzzy, nav) | `util/`, `navigation.rs` | 4h |
| Integration testing pass | Manual full-app testing | 6h |
| **Total** | | **74h** |

### 19.7 Total Estimates

| Week | Focus | Hours |
|------|-------|-------|
| 1 | Foundation | 33h |
| 2 | Components + Dashboard | 37h |
| 3 | Invoice screens | 50h |
| 4 | Expenses, Contacts, Recurring | 49h |
| 5 | VAT, Tax, Reports | 59h |
| 6 | Settings, Polish, Testing | 74h |
| **Total** | | **302h** |

At 40h/week, this is approximately 7.5 working weeks. The 4-6 week estimate assumes parallel development of independent view groups.

---

## 20. Testing Strategy

### 20.1 GPUI View Tests

Every view is tested using `#[gpui::test]` with `TestAppContext`:

```rust
#[gpui::test]
async fn test_dashboard_loads_data(cx: &mut TestAppContext) {
    let services = create_test_services().await;
    let view = cx.add_window(|window, cx| {
        cx.new(|cx| DashboardView::new(Arc::new(services), cx))
    });

    // Verify loading state
    view.update(cx, |view, _| {
        assert!(view.loading);
    });

    // Wait for background task
    cx.background_executor().run_until_parked();

    // Verify data loaded
    view.update(cx, |view, _| {
        assert!(!view.loading);
        assert!(view.data.is_some());
    });
}
```

### 20.2 Unit Test Coverage

| Module | Target | Scope |
|--------|--------|-------|
| `util/format.rs` | 100% | All Czech formatting functions |
| `util/fuzzy.rs` | 100% | Diacritics normalization, fuzzy matching |
| `navigation.rs` | 100% | Route metadata, NavigationState methods |
| `theme.rs` | Compile-time | Theme struct completeness verified by type system |

### Headless Visual Verification (Tier 2)

Every screen must be verified via headless screenshots during the Phase 6 quality gate:

**Setup:**
- Build app with test fixture database containing sample data (contacts, invoices, expenses, VAT returns, etc.)
- Use `rust/scripts/headless-screenshot.sh` wrapper (cage + WLR_BACKENDS=headless + grim)
- All deps available via `nix develop`

**Process for each of the 43 routes:**
1. `./rust/scripts/headless-screenshot.sh "./target/debug/zfaktury-app --route /path --db tests/fixtures/test.db --exit-after 5" /tmp/screenshots/path.png 4`
2. Agent reads the screenshot image and verifies:
   - Sidebar visible with correct navigation items
   - Main content area shows expected data
   - Czech labels correct (not English, not garbled)
   - Status badges show correct colors
   - Tables have data rows (not empty)
   - No blank areas or rendering artifacts
   - Dark theme applied (dark background, light text)
3. Screenshot saved to `rust/tests/screenshots/` for user review

**CLI arguments the app MUST implement:**
- `--route <path>` -- navigate directly to this screen on launch (e.g., "/invoices", "/vat", "/settings/firma")
- `--exit-after <seconds>` -- automatically close after N seconds
- `--db <path>` -- use this SQLite database file instead of default

**Test fixture database:**
Create `rust/tests/fixtures/test.db` with:
- 5 contacts (mix of company/individual, CZ/EU)
- 10 invoices (draft, sent, paid, overdue, cancelled states)
- 8 expenses (various categories, some tax-reviewed)
- 2 recurring invoices, 1 recurring expense
- 1 VAT return (draft), 1 control statement
- Tax credits (spouse, 2 children, personal)
- Settings (company info, bank details, PDF settings)

### 20.3 Manual Test Protocol

Before release, verify each of these manually:
1. App launches and shows dashboard with real data
2. Navigate every route via sidebar
3. Navigate via command palette
4. CRUD: create, read, update, delete an invoice
5. CRUD: create, read, update, delete an expense
6. CRUD: create, read, update, delete a contact
7. Invoice PDF preview in inspector
8. File drag & drop for document upload
9. Theme toggle (dark/light)
10. Keyboard: Ctrl+K, Ctrl+N, Ctrl+S, arrows in tables, Enter to open
11. Split-view on invoice list
12. Virtual scroll with large dataset (import 10,000 test invoices)
13. All settings pages save correctly
14. Backup create + restore cycle

---

## 21. Acceptance Criteria

1. App launches and displays dashboard with real data from existing SQLite database
2. All 43 routes are navigable and functional (verified by clicking through every sidebar item)
3. CRUD operations work for all entities: invoices, expenses, contacts, recurring templates, VAT returns, control statements, VIES declarations, tax filings, categories, sequences
4. Command palette (Ctrl+K) searches across invoices, contacts, expenses, and commands with diacritics-insensitive fuzzy matching
5. Split-view (Ctrl+\\) works for invoice list and expense list
6. PDF preview renders in inspector panel when editing an invoice and updates live with form changes (500ms debounce)
7. All keyboard shortcuts listed in Section 10 are functional and do not conflict
8. Dark/light theme toggle works and preference persists across restarts
9. Virtual scrolling is smooth (no visible frame drops) with 10,000+ rows in any list view
10. Charts render with correct data on dashboard and reports
11. File drag & drop works for document uploads in expense forms
12. Native file dialogs (save/open) work for PDF download, backup export, document import
13. App supports `--route`, `--exit-after`, and `--db` CLI arguments for headless testing
14. Headless screenshots (via cage + grim) produce valid PNGs for all 43 routes
15. Agent-verified screenshots show correct layout, Czech labels, and theme colors

---

## 22. Review Checklist

Before marking Phase 6 complete:

- [ ] All Czech UI text is correct (labels, status names, menu items, error messages, empty states)
- [ ] JetBrains Mono used for ALL monetary values, invoice numbers, ICO/DIC, percentages
- [ ] Numbers never animate (instant update for all monetary and numeric values)
- [ ] Keyboard shortcuts do not conflict with each other or with OS shortcuts
- [ ] All database queries, PDF generation, XML generation, HTTP calls run on `background_executor()`
- [ ] `Arc<XxxService>` shared correctly across views (no `Clone` of service structs, only `Arc::clone`)
- [ ] Theme colors match specification in Section 6 exactly
- [ ] Sidebar collapse/expand animation is smooth (200ms ease-out)
- [ ] Dialog focus management correct: focus trapped within dialog, focus returns to trigger element on close
- [ ] Empty states shown for all list views ("Zatim zadne faktury", etc.)
- [ ] Loading states shown during all async operations (spinner or skeleton)
- [ ] Error states shown with descriptive Czech messages for all failure modes
- [ ] Form validation shows inline errors below relevant fields
- [ ] Tab order is logical in all forms
- [ ] All context bar actions work (send, mark paid, duplicate, credit note, download)
- [ ] Status badges use exact same color mapping as existing SvelteKit frontend
- [ ] Date formatting is DD.MM.YYYY throughout
- [ ] Currency formatting uses Czech locale (comma decimal, space thousands, "Kc" suffix)
- [ ] --route, --exit-after, --db CLI args work correctly
- [ ] Headless screenshots pass for all 43 routes
- [ ] Test fixture database created with representative sample data

---

## 23. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| gpui-component missing needed widgets | High | Medium | Implement custom GPUI components; gpui-component is actively developed so check latest API |
| PDF-to-image rendering for preview | Medium | Low | Use `pdfium` or `mupdf` bindings; fallback: show "PDF generated" message without preview |
| Chart rendering complexity in GPUI | Medium | Medium | Start with simple rect-based bars; iterate on gradients and animations |
| 120fps scroll performance with complex rows | Medium | Low | Profile early, simplify row rendering if needed, reduce overscan |
| Font embedding size | Low | Low | Subset fonts to needed glyphs (Latin + Czech diacritics) |
| Native file dialog platform differences | Low | Low | GPUI abstracts this; test on Linux (GTK) |
| Theme color tuning | Low | High | Plan for iterative color adjustment after first full render |

---

## 24. Dependencies on Other Phases

| Phase | What is needed | Status |
|-------|---------------|--------|
| Phase 1 (domain) | All domain types, Amount, error types | Required before start |
| Phase 2 (persistence) | All repository implementations, migrations | Required before start |
| Phase 3 (services) | All service layer methods | Required before start |
| Phase 4 (gen) | `generate_invoice_pdf()`, XML export functions | Required for PDF preview, download actions |
| Phase 5 (CLI) | Config loading, database setup code (shared) | Required for app bootstrap |

Phase 6 can begin UI skeleton work (theme, navigation, sidebar, components) before Phases 3-5 are complete, using mock data. However, full integration requires all prior phases.

---

## 25. Future Considerations (Out of Scope)

These are explicitly NOT part of Phase 6 but noted for future phases:

- **Multi-window**: open invoice in separate window
- **Undo/redo**: global undo stack for form edits
- **Offline mode**: queue operations when DB is locked
- **Print**: native print dialog integration
- **Auto-update**: check for new versions on startup
- **Telemetry**: opt-in usage analytics
- **Custom themes**: user-defined color themes via TOML
- **Localization**: languages beyond Czech
