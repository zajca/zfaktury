# ZFaktury Rust Rewrite - Implementation Status

**Last updated:** 2026-03-20
**Total tests:** 440 passing
**Crates:** 7 (domain, config, core, db, gen, api, app) + testutil
**Commits:** 14 (from POC through Phase 10)

---

## Phase Status Overview

| Phase | Status | Tests | Key Gaps |
|-------|--------|-------|----------|
| P0: POC | DONE | 0 | - |
| P1: Foundation | DONE | 221 | - |
| P2: Persistence | DONE | 61 | - |
| P3: Generation | DONE | 73 | - |
| P4: External APIs | DONE | 50 | - |
| P5: Services | DONE | 32 | - |
| P6: GPUI App | DONE | 3 | ~15 minor route stubs remain |
| P7: Polish | DONE | 0 | CLI + docs done |
| P8: PDF Generation | DONE | 15 | typst-bake with Czech template |
| P9: View Completion | DONE | 0 | 18 new views replacing stubs |
| P10: Quality Gates | DONE | 0 | clippy fixed, reviews run |

---

## Detailed Gap Analysis

### COMPLETE - No Further Work Needed

#### zfaktury-domain (18 files, 23 tests)
- All 34+ domain types ported from Go `internal/domain/`
- Amount(i64) with all ops (Add, Sub, Neg, multiply, Display, Ord)
- All enums with Display impl
- calculate_totals() on Invoice and Expense
- next_date() on RecurringInvoice/Expense with month-end clamping
- DomainError enum (10 variants)
- NO serde derives -- domain purity maintained

#### zfaktury-config (1 file, 14 tests)
- All config structs match Go internal/config/config.go
- ZFAKTURY_DATA_DIR env override
- TOML parsing, path expansion, defaults
- Fail-fast on missing file

#### zfaktury-core/calc (9 modules, 178 tests)
- constants.rs: 2024/2025/2026 tax constants
- vat.rs: VAT return calculation (credit notes, business_percent)
- income_tax.rs: Progressive 15%/23%, rounding to 100 CZK
- insurance.rs: Social 29.2%, Health 13.5%, min base enforcement
- credits.rs: Spouse (68k limit, ZTP), personal, child benefit
- deductions.rs: Category caps (mortgage 150k, pension 24k, etc.)
- fifo.rs: Full FIFO algorithm with time test + exemption limit
- annual_base.rs: Year filtering (status, type, delivery_date)
- recurring.rs: next_occurrence with month-end clamping
- proptest strategies for Amount, tax invariants

#### zfaktury-core/repository (3 files)
- traits.rs: 33 repository traits matching Go interfaces.go
- types.rs: MonthlyAmount, QuarterlyAmount, CategoryAmount, CustomerRevenue, RecentInvoice, RecentExpense
- All method signatures match Go originals

#### zfaktury-db (32 repo implementations + 21 migrations, 61 tests)
- All repos fully implemented with rusqlite (not stubs)
- Connection setup: WAL, foreign_keys ON, busy_timeout 5000
- Date/datetime parsing helpers
- Goose-to-refinery migration bridge
- Soft delete pattern consistently applied
- Parameterized queries (no SQL injection)

#### zfaktury-api (6 clients, 50 tests)
- ARES: ICO lookup with 8-digit validation
- CNB: Exchange rates with weekend fallback (5 days)
- Email: lettre with TLS/STARTTLS, multipart MIME, attachments
- OCR: Anthropic + OpenAI providers with Czech prompt
- Fakturoid: OAuth2, pagination, FlexFloat64
- FIO: Bank transaction stub with trait

#### zfaktury-core/service (31 files, 32 tests)
- All 31 services implemented
- Arc<dyn Repo + Send + Sync> dependency injection
- Audit logging on mutations
- Error wrapping with DomainError

---

### PARTIAL - Specific Gaps Listed

#### zfaktury-gen: PDF Generation MISSING

**What exists:**
- XML: DPHDP3, DPHKH1, DPHSHV, DPFDP5, CSSZ, health insurance (57 tests)
- ISDOC 6.0.2 with DocumentType 1/2/4
- QR/SPAYD payment string + PNG generation
- CSV export with UTF-8 BOM, semicolons

**What's missing:**
- [ ] Invoice PDF via typst -- the RFC specified typst as a library for template-based PDF generation. This is NOT implemented. No typst dependency in Cargo.toml, no PDF templates, no PDF output.
- [ ] PDF test infrastructure (lopdf parse, pdf-extract text verification)
- [ ] Golden files for PDF metadata comparison

**Priority:** HIGH -- users need PDFs for invoices

**Implementation plan:**
1. Add `typst`, `typst-pdf` dependencies to zfaktury-gen
2. Create `src/pdf/mod.rs` with invoice PDF template
3. Template sections: header (logo, invoice number), parties (supplier/customer), items table, VAT summary, totals, payment info (QR code), footer
4. Czech labels: "Faktura", "Dobropis", "Dodavatel", "Odberatel", etc.
5. Fonts: Inter (UI text), JetBrains Mono (amounts) -- embed from assets/
6. Amount format: "1 234,56 CZK" (Czech locale)
7. Tests: generate → parse with lopdf → verify text content

#### zfaktury-app: 30+ Stub Views

**Functional views (7):**
- [x] Dashboard (stat cards, recent tables)
- [x] Invoice list (table with status badges)
- [x] Invoice detail (header, items, totals)
- [x] Invoice form (create layout -- fields are placeholders)
- [x] Expense list (table)
- [x] Contact list (table with search)
- [x] Settings firma (company info display)

**Stub views showing "Tato stranka bude brzy dostupna" (30+):**
- [ ] Reports (5 tabs: revenue, expenses, P&L, top customers, tax calendar)
- [ ] Contact detail/edit
- [ ] Expense detail/edit/form
- [ ] Expense import (document upload + OCR review)
- [ ] Expense review (tax review marking)
- [ ] Recurring invoice list/form/detail
- [ ] Recurring expense list/form/detail
- [ ] VAT overview (quarter grid)
- [ ] VAT return detail/form
- [ ] VAT control statement detail/form
- [ ] VIES summary detail/form
- [ ] Tax overview
- [ ] Tax credits page
- [ ] Tax prepayments page
- [ ] Tax investments page
- [ ] Income tax return detail/form
- [ ] Social insurance detail/form
- [ ] Health insurance detail/form
- [ ] Settings: email, sequences, categories, PDF, audit log, backup
- [ ] Fakturoid import page

**Priority:** HIGH -- most views are not usable

#### zfaktury-app: Missing gpui-component Integration

**What the plan specified:**
- gpui-component for Table, Input, DatePicker, Dialog, Tabs, BarChart, PieChart, Checkbox, Select, Switch, Stepper, Toast
- Currently: raw GPUI divs only, no gpui-component dependency

**What needs to happen:**
1. Add `gpui-component` git dependency to Cargo.toml
2. Replace hand-built tables with `gpui_component::Table` (virtual scrolling)
3. Use `gpui_component::Input` for form fields
4. Use `gpui_component::DatePicker` for date inputs
5. Use `gpui_component::Dialog` for confirmations/modals
6. Use `gpui_component::BarChart` / `PieChart` for reports/dashboard
7. Use `gpui_component::Tabs` for reports, settings
8. Use `gpui_component::Toast` for notifications

**Priority:** MEDIUM -- functional but ugly without it

#### zfaktury-app: Missing UX Features

- [ ] Command palette (Ctrl+K) -- fuzzy search across invoices/contacts/commands
- [ ] Split-view (Ctrl+\\) -- master-detail for invoices/expenses
- [ ] Live PDF preview -- inspector panel during invoice editing
- [ ] Keyboard shortcuts -- only basic navigation exists
- [ ] Animations -- none implemented (sidebar collapse, page crossfade, dialog scale)
- [ ] Drag & drop -- for document uploads
- [ ] Custom themes -- ZfColors exists but not registered as gpui-component theme
- [ ] Toast notifications -- no notification system
- [ ] Confirm dialogs -- no delete/action confirmations
- [ ] Empty/loading/error states -- not all views handle these

**Priority:** LOW-MEDIUM -- nice-to-have, not blocking core functionality

---

### NOT DONE - Quality Gates

- [ ] `cargo clippy --workspace -- -D warnings` -- not run, likely has warnings
- [ ] `cargo llvm-cov` -- coverage not measured
- [ ] Code review (developer:code-reviewer agent) -- not run
- [ ] Security review (developer:code-security agent) -- not run
- [ ] Headless screenshots of all 43 routes -- not verified
- [ ] Real database import test (load Go's zfaktury.db) -- not done
- [ ] Golden file tests for XML outputs -- golden/ directory not created
- [ ] Compare generated XML/PDF against Go version output -- not done
- [ ] Performance verification (10k rows, scroll fps) -- not done

---

## Recommended Next Steps (Priority Order)

### Priority 1: Make the app actually usable
1. **PDF generation** via typst (without this, the app can't produce invoices)
2. **Invoice form** -- make it actually save data (currently placeholder fields)
3. **Contact form** -- create/edit contacts
4. **Expense form** -- create/edit expenses with document upload

### Priority 2: Complete the most-used views
5. **VAT overview** -- quarter grid with status colors
6. **VAT return detail** -- display calculated VAT with recalculate/generate actions
7. **Reports** -- at least revenue + expenses charts
8. **Settings** -- at minimum firma (edit mode) and sequences

### Priority 3: gpui-component integration
9. Add gpui-component dependency
10. Replace tables with virtual-scrolling Table component
11. Add Input, DatePicker, Dialog, Tabs components
12. Add BarChart/PieChart for dashboard and reports

### Priority 4: Quality gates
13. Run cargo clippy and fix all warnings
14. Measure coverage with cargo-llvm-cov
15. Create golden files for XML outputs
16. Test with real Go database
17. Run code review + security review agents

### Priority 5: UX polish
18. Command palette (Ctrl+K)
19. Keyboard shortcuts
20. Toast notifications
21. Confirm dialogs
22. Animations
