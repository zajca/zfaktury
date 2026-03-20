# Phase 7: Polish

**Scope:** CLI subcommands, drag & drop, animations, accessibility audit, final test pass, real database import verification, documentation (ARCHITECTURE.md, GUI-DEVELOPMENT.md, CLAUDE.md).

**Estimated duration:** 1-2 weeks

**Prerequisites:** Phases 1-6 complete and merged. All core features (invoices, expenses, VAT returns, tax returns, PDF/XML generation, GPUI desktop) working.

---

## 1. CLI Subcommands (clap derive)

The main binary supports both desktop mode (default) and CLI subcommands. When no subcommand is provided, the GPUI desktop app launches. CLI commands share the same config loading and database path resolution as the desktop app.

### Command Structure

```
zfaktury                        # Launch GPUI desktop app (default)
zfaktury backup                 # Create backup (DB + documents archive)
zfaktury backup --list          # List available backups with timestamps and sizes
zfaktury backup --restore <id>  # Restore from a specific backup
zfaktury backup --delete <id>   # Delete a specific backup
zfaktury migrate                # Run all pending database migrations
zfaktury migrate --status       # Show current migration version and pending migrations
zfaktury version                # Print version, build date, git commit hash
```

### Implementation

```rust
use clap::{Parser, Subcommand, Args};

#[derive(Parser)]
#[command(name = "zfaktury", version, about = "Invoicing and tax management for Czech sole proprietors")]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    /// Manage database and document backups
    Backup(BackupArgs),
    /// Run or inspect database migrations
    Migrate(MigrateArgs),
    /// Show version information
    Version,
}

#[derive(Args)]
struct BackupArgs {
    /// List all available backups
    #[arg(long)]
    list: bool,
    /// Restore from a specific backup ID
    #[arg(long, value_name = "ID")]
    restore: Option<String>,
    /// Delete a specific backup ID
    #[arg(long, value_name = "ID")]
    delete: Option<String>,
}

#[derive(Args)]
struct MigrateArgs {
    /// Show migration status without running anything
    #[arg(long)]
    status: bool,
}
```

Entry point logic in `main.rs`:

```rust
fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();
    let config = load_config()?;

    match cli.command {
        None => launch_desktop(config),
        Some(Commands::Backup(args)) => run_backup(config, args),
        Some(Commands::Migrate(args)) => run_migrate(config, args),
        Some(Commands::Version) => print_version(),
    }
}
```

### Backup Details

- **Create:** Copies the SQLite database file (with WAL checkpoint first) and archives all document files from the data directory into a timestamped `.tar.zst` file stored in `<data_dir>/backups/`.
- **Backup naming:** `zfaktury-backup-<YYYYMMDD-HHMMSS>.tar.zst`
- **List output:** Table with columns: ID (timestamp portion), date, size, database version.
- **Restore:** Extracts archive to a temporary location, verifies integrity (SQLite `PRAGMA integrity_check`), then replaces current database and documents. Creates a pre-restore backup automatically.
- **Delete:** Removes the archive file after confirmation prompt.

### Migrate Details

- **Run:** Applies all pending embedded migrations (same migration system as desktop startup).
- **Status output:** Shows current version number, total available migrations, and lists any pending migrations by name.
- Migrations are embedded in the binary (no external SQL files at runtime).

### Config Sharing

CLI commands use the exact same `load_config()` path as the desktop app:
1. Check `ZFAKTURY_DATA_DIR` env var
2. Fall back to `~/.zfaktury/`
3. Load `config.toml` from data directory
4. Database path from config or default `<data_dir>/zfaktury.db`

---

## 2. Drag & Drop for Documents

Document uploads via drag & drop for: expenses, invoices, investments, tax deductions.

### Accepted Files

| Extension | MIME type |
|-----------|-----------|
| `.pdf` | `application/pdf` |
| `.png` | `image/png` |
| `.jpg` / `.jpeg` | `image/jpeg` |

### Behavior

1. **Drop zone:** A designated area within each entity's detail/edit view. Shows dashed border with muted text ("Pretahnte soubory sem" / "Drag files here").
2. **Drag over:** When files hover over the drop zone, the border becomes solid with the accent color, background lightens. Visual feedback must appear within one frame.
3. **Validation on drop:**
   - Check file extension against allowed list.
   - Check file size <= 20 MB per file.
   - Reject with inline error message if invalid (red border, error text below drop zone).
4. **Multiple files:** Process sequentially. Show a progress indicator (file N of M) and individual file name during upload.
5. **Success:** File saved to `<data_dir>/documents/<entity_type>/<entity_id>/`. Refresh the document list in the UI.
6. **Error handling:** If save fails, show error for that specific file and continue with remaining files.

### GPUI Implementation

```rust
fn render_drop_zone(&self, cx: &mut Context<Self>) -> impl IntoElement {
    div()
        .id("document-drop-zone")
        .border_2()
        .border_dashed()
        .border_color(if self.drag_active {
            cx.theme().accent
        } else {
            cx.theme().border_muted
        })
        .bg(if self.drag_active {
            cx.theme().accent.opacity(0.05)
        } else {
            cx.theme().bg_surface
        })
        .rounded_lg()
        .p_4()
        .on_drag_over(cx.listener(|this, _, _, cx| {
            this.drag_active = true;
            cx.notify();
        }))
        .on_drag_leave(cx.listener(|this, _, _, cx| {
            this.drag_active = false;
            cx.notify();
        }))
        .on_drop(cx.listener(|this, paths: &Vec<PathBuf>, _, cx| {
            this.drag_active = false;
            this.handle_dropped_files(paths, cx);
        }))
        .child(self.render_drop_zone_content(cx))
}
```

### File Storage

Documents are stored on disk, not in the database. Database stores metadata (filename, content type, size, upload timestamp, entity reference). File path is deterministic from entity type + entity ID + filename, so backup/restore preserves the structure.

---

## 3. Animations

All animations use GPUI's built-in animation system. Performance requirement: animations must not cause frame drops below 60fps on the target hardware.

### Animation Catalog

| Element | Trigger | Duration | Easing | Description |
|---------|---------|----------|--------|-------------|
| Sidebar | Toggle button / shortcut | 200ms | ease-out | Width 208px <-> 52px. Content crossfades. |
| Page transition | Route change | 150ms | ease-out | Outgoing page fades to opacity 0, incoming fades from 0 to 1. |
| Dialog open | Open action | 200ms | ease-out | Scale 0.95 + opacity 0 -> scale 1.0 + opacity 1.0. Backdrop fades in simultaneously. |
| Dialog close | Close/Escape/backdrop click | 150ms | ease-in | Reverse of open. Focus returns after animation completes. |
| Chart stagger | Data load / tab switch | ~400ms total | ease-out | Each bar/slice animates in sequence with 50ms stagger delay between elements. |
| Toast slide-in | Notification trigger | 200ms | ease-out | translateY(100%) -> translateY(0). Auto-dismiss after 5000ms with 300ms fade-out. |
| Status badge pulse | Status change | 600ms (1 cycle) | ease-in-out | Subtle scale 1.0 -> 1.1 -> 1.0 with opacity variation. Single cycle only. |

### Rules

- **NEVER animate monetary values.** All Amount fields update instantly. No counting-up effects, no transitions on currency displays.
- Animations must respect system reduced-motion preference. When reduced motion is enabled, all animations complete instantly (duration = 0).
- Sidebar animation must not cause layout reflow in the main content area -- use fixed layout with the sidebar width as an animated property.

### Implementation Pattern

```rust
use gpui::animation::{Animation, AnimationExt};
use std::time::Duration;

impl Sidebar {
    fn render(&self, cx: &mut Context<Self>) -> impl IntoElement {
        let target_width = if self.collapsed { px(52.) } else { px(208.) };

        div()
            .id("sidebar")
            .w(target_width)
            .with_animation(
                "sidebar-width",
                Animation::new(Duration::from_millis(200))
                    .with_easing(Easing::EaseOut),
            )
            .overflow_hidden()
            .child(self.render_sidebar_content(cx))
    }
}

impl Dialog {
    fn render_overlay(&self, cx: &mut Context<Self>) -> impl IntoElement {
        let (scale, opacity) = if self.visible {
            (1.0, 1.0)
        } else {
            (0.95, 0.0)
        };

        div()
            .id("dialog")
            .scale(scale)
            .opacity(opacity)
            .with_animation(
                "dialog-appear",
                Animation::new(Duration::from_millis(
                    if self.visible { 200 } else { 150 }
                ))
                .with_easing(if self.visible {
                    Easing::EaseOut
                } else {
                    Easing::EaseIn
                }),
            )
            .child(self.render_dialog_content(cx))
    }
}
```

---

## 4. Accessibility Audit

### Focus Management

| Requirement | Implementation |
|-------------|---------------|
| Tab order follows visual layout | Use GPUI's default focus order; verify no `tabindex` hacks break flow. |
| Focus trap in dialogs | When dialog is open, Tab/Shift+Tab cycle only within dialog children. Implement via GPUI `FocusScope`. |
| Focus return on dialog close | Store the previously focused element ID before opening dialog. Restore focus after close animation completes. |
| Skip-to-content link | Hidden link as first focusable element, becomes visible on focus, jumps to main content area. |
| Visible focus ring | 2px solid ring using accent color with 2px offset on all interactive elements. Must be visible in both light and dark themes. |

### Keyboard Navigation

| Key | Context | Action |
|-----|---------|--------|
| Tab / Shift+Tab | Global | Move focus forward/backward |
| Enter | Focused button/link/row | Activate element |
| Space | Checkbox/switch | Toggle |
| Escape | Dialog/modal/dropdown | Close/cancel |
| Arrow Up/Down | Table | Move row selection |
| Arrow Left/Right | Table (with horizontal scroll) | Scroll horizontally |
| Ctrl+K / Cmd+K | Global | Open command palette |
| Ctrl+N / Cmd+N | List views | Create new entity |
| Ctrl+S / Cmd+S | Edit views | Save |
| Ctrl+Backspace | Edit views | Cancel/discard |

Every feature must be fully operable via keyboard alone. No mouse-only interactions.

### Screen Reader Support

Where GPUI provides accessibility primitives:
- Set appropriate roles on interactive elements (button, link, checkbox, dialog, table, row, cell).
- Provide accessible names for icon-only buttons via tooltip text or explicit label.
- Announce async operation results: "Faktura ulozena" (invoice saved), "Chyba pri ukladani" (error saving).
- Associate table column headers with cells.
- Associate form labels with inputs (every input must have a label, even if visually hidden).

### Contrast Requirements

| Element | Minimum Ratio | Standard |
|---------|---------------|----------|
| Body text | 4.5:1 | WCAG AA |
| Large text (18px+ or 14px bold) | 3:1 | WCAG AA |
| UI components and borders | 3:1 | WCAG AA |
| Focus ring against background | 3:1 | WCAG AA |

Status indicators must not rely solely on color. Each status must also have a distinct text label or icon:
- Paid: green + check icon + "Zaplaceno"
- Unpaid: yellow + clock icon + "Nezaplaceno"
- Overdue: red + warning icon + "Po splatnosti"
- Cancelled: gray + cross icon + "Stornováno"

### Audit Checklist

- [ ] All buttons, links, and interactive elements are focusable via Tab
- [ ] All forms have associated labels (visible or `aria-label`)
- [ ] Error messages are announced to screen readers
- [ ] Loading states are communicated (spinner + text)
- [ ] Dialog focus trap works correctly (Tab does not escape)
- [ ] Dialog focus returns to trigger on close
- [ ] No information conveyed only by color
- [ ] Arrow key navigation works in all tables
- [ ] Escape closes all overlays (dialogs, dropdowns, command palette)
- [ ] Contrast ratios meet WCAG AA in both light and dark themes
- [ ] Reduced motion preference disables all animations

---

## 5. Final Test Pass and Coverage Enforcement

### Coverage Thresholds

| Scope | Threshold | Measurement |
|-------|-----------|-------------|
| `zfaktury-calc` crate (all calculation logic) | 100% line + branch | `cargo llvm-cov` |
| `domain::amount` module | 100% line + branch | `cargo llvm-cov` |
| `zfaktury-gen` crate (PDF/XML generation) | 100% line | `cargo llvm-cov` |
| All other crates | 90%+ line | `cargo llvm-cov` |

Coverage is measured with `cargo-llvm-cov`:

```bash
# Install (one-time)
cargo install cargo-llvm-cov

# Generate LCOV report
cargo llvm-cov --workspace --lcov --output-path lcov.info

# Per-crate coverage check
cargo llvm-cov -p zfaktury-calc --show-lines
cargo llvm-cov -p zfaktury-gen --show-lines
cargo llvm-cov --workspace --summary-only
```

### Full Test Matrix

| Check | Command | Pass Condition |
|-------|---------|----------------|
| All unit + integration tests | `cargo test --workspace` | All pass, zero failures |
| Clippy (strict) | `cargo clippy --workspace -- -D warnings` | Zero warnings |
| Format check | `cargo fmt --check` | No formatting differences |
| calc crate coverage | `cargo llvm-cov -p zfaktury-calc` | 100% line + branch |
| gen crate coverage | `cargo llvm-cov -p zfaktury-gen` | 100% line |
| Workspace coverage | `cargo llvm-cov --workspace` | 90%+ line |
| Golden file tests | `cargo test golden` | All golden files match |
| XSD validation tests | `cargo test xsd` | All generated XML valid against XSD schemas |
| Doc tests | `cargo test --doc --workspace` | All pass |

### CI Enforcement

These checks run in CI on every push and PR. A failure in any check blocks merge. The coverage thresholds are enforced via a script that parses `cargo llvm-cov` output and exits non-zero if below threshold.

---

## 6. Import Test with Real Database

The most critical verification in this phase: loading an existing SQLite database created by the Go version of ZFaktury and confirming all data is intact and all generation outputs match.

### Test Fixture

A real `zfaktury.db` file (from `~/.zfaktury/`) is copied to `tests/fixtures/real_zfaktury.db`. This file is gitignored (contains real business data) but must be present for local test runs. CI skips this test when the fixture is absent.

### Verification Steps

1. **Open database** with the Rust migration system. The migration bridge must detect the Go-era schema version and apply only Rust-specific migrations (if any).
2. **Contacts:** Load all contacts. Verify count matches expected. Spot-check fields (name, ICO, DIC, address).
3. **Invoices:** Load all invoices with items and customer references. Verify:
   - Invoice numbers, dates, due dates, payment dates
   - Item descriptions, quantities, unit prices, VAT rates
   - Computed totals match stored totals (recalculate with Rust calc)
   - Customer reference resolves to correct contact
4. **Expenses:** Load all expenses. Verify documents references, amounts, categories.
5. **VAT returns:** Load VAT returns with linked invoices and expenses. Verify section totals.
6. **Tax returns:** Load tax returns with all fields populated.
7. **Settings:** Verify all settings key-value pairs are readable.
8. **Audit log:** Verify audit log entries are parseable.
9. **PDF comparison:** Generate a PDF for a known invoice using both Go and Rust. Extract text content and compare. Layout differences are acceptable; data content must match exactly.
10. **XML comparison:** Generate XML for a known VAT return. Compare element-by-element. Must be identical (XML is deterministic).

### Automated Test

```rust
#[cfg(test)]
mod real_db_tests {
    use std::path::Path;

    #[test]
    fn test_real_database_import() {
        let db_path = Path::new("tests/fixtures/real_zfaktury.db");
        if !db_path.exists() {
            eprintln!("Skipping real database import test: fixture not found at {}", db_path.display());
            return;
        }

        // Open database with migration bridge
        let conn = open_and_migrate(db_path).expect("Failed to open real database");

        // Verify contacts
        let contacts = list_all_contacts(&conn).expect("Failed to load contacts");
        assert!(!contacts.is_empty(), "Expected contacts in real database");

        // Verify invoices with items
        let invoices = list_all_invoices_with_items(&conn).expect("Failed to load invoices");
        assert!(!invoices.is_empty(), "Expected invoices in real database");
        for invoice in &invoices {
            assert!(!invoice.items.is_empty(), "Invoice {} has no items", invoice.number);
            // Recalculate totals with Rust calc and compare
            let recalculated = calculate_invoice_totals(&invoice.items);
            assert_eq!(
                recalculated.total, invoice.total,
                "Total mismatch for invoice {}: calc={} stored={}",
                invoice.number, recalculated.total, invoice.total
            );
        }

        // Verify expenses
        let expenses = list_all_expenses(&conn).expect("Failed to load expenses");
        assert!(!expenses.is_empty(), "Expected expenses in real database");

        // Verify VAT returns
        let vat_returns = list_all_vat_returns(&conn).expect("Failed to load VAT returns");
        // May be empty if user hasn't filed yet, so no assert on count

        // Verify settings
        let settings = load_all_settings(&conn).expect("Failed to load settings");
        assert!(!settings.is_empty(), "Expected settings in real database");

        // PDF generation comparison (if reference PDF exists)
        let reference_pdf_path = Path::new("tests/fixtures/reference_invoice.pdf");
        if reference_pdf_path.exists() && !invoices.is_empty() {
            let generated_pdf = generate_invoice_pdf(&invoices[0]);
            let generated_text = extract_pdf_text(&generated_pdf);
            let reference_text = extract_pdf_text_from_file(reference_pdf_path);
            assert_eq!(generated_text, reference_text, "PDF text content mismatch");
        }

        // XML generation comparison (if reference XML exists)
        let reference_xml_path = Path::new("tests/fixtures/reference_vat_return.xml");
        if reference_xml_path.exists() && !vat_returns.is_empty() {
            let generated_xml = generate_vat_return_xml(&vat_returns[0]);
            let reference_xml = std::fs::read_to_string(reference_xml_path).unwrap();
            assert_eq!(
                normalize_xml(&generated_xml),
                normalize_xml(&reference_xml),
                "VAT return XML content mismatch"
            );
        }
    }
}
```

---

## 7. Performance Verification

Manual verification (not automated in CI, but documented and performed before release):

| Scenario | Target | How to verify |
|----------|--------|---------------|
| Scroll 10,000 invoices | 120fps, no frame drops | Load test fixture with 10k rows, scroll with trackpad, observe frame counter |
| Instant search | < 100ms to show results | Type in search field, measure time to first result render |
| PDF preview | < 2 seconds | Open invoice detail with PDF preview tab |
| App startup | < 1 second (cold start) | `time ./zfaktury --version` as proxy; desktop launch measured with stopwatch |
| Page navigation | < 50ms | Click between sidebar items, observe transition (should feel instant) |
| Command palette | < 50ms to open | Press Ctrl+K, measure time to palette visible |

---

## 8. Final Cleanup

### Code Quality

- [ ] Remove all `TODO` and `FIXME` comments. Each must be either implemented or converted to a tracked issue.
- [ ] Remove all `dbg!()` and `println!()` debug statements. Only `tracing` macros in production code.
- [ ] Verify all `unwrap()` calls are justified (test code only, or provably infallible). Production code must use proper error handling.
- [ ] Verify all error messages shown to users are in Czech and helpful (no raw SQL errors, no stack traces, no Rust panic messages).
- [ ] Verify all log messages (tracing) are in English and structured.

### Cross-Platform

- [ ] All file paths use `std::path::PathBuf` (no hardcoded `/` separators).
- [ ] Home directory resolved via `dirs::home_dir()` or equivalent.
- [ ] Config and data directories follow platform conventions (Linux: `~/.zfaktury`, macOS: `~/.zfaktury` for consistency with Go version).
- [ ] File permissions set correctly on created directories (0o700 for data dir).

### Metadata

- [ ] `Cargo.toml` workspace metadata complete: name, version, description, license (MIT), authors, repository URL.
- [ ] `rust-toolchain.toml` at workspace root pinning MSRV (minimum supported Rust version) for reproducible builds.
- [ ] All crate-level `Cargo.toml` files have correct dependency versions (no `*` versions).
- [ ] `deny.toml` configured for `cargo-deny`: no duplicate deps, license check, advisory audit.

### Config Compatibility

- [ ] Rust version reads the same `config.toml` format as the Go version.
- [ ] All existing config keys are supported. Unknown keys are ignored with a warning log (not a hard error).
- [ ] Database path resolution identical to Go version.

---

## 9. Documentation Deliverables

Phase 7 includes writing 3 documentation files that capture architecture and GUI development knowledge:

### 1. `rust/docs/ARCHITECTURE.md`
System architecture reference covering:
- Cargo workspace structure and crate dependency graph (with diagram)
- 3-layer architecture: GPUI View → Service (Arc<T>) → Repository (Box<dyn Trait>) → rusqlite
- Data flow for reads and writes (including background_executor threading)
- Amount(i64) money system -- why i64, conversion methods, display formatting
- Database conventions: TEXT dates (ISO 8601), INTEGER amounts (halere), INTEGER booleans, soft deletes
- Error handling: thiserror for typed errors in crates, anyhow for app-level errors, DomainError enum
- Threading model: main thread = GPUI render loop, background_executor = DB/network/PDF/heavy compute
- Config system: TOML file at ~/.zfaktury/config.toml, ZFAKTURY_DATA_DIR env override, fail-fast on missing
- Migration strategy: refinery with embedded SQL, goose→refinery bridge for existing Go databases
- Service wiring: AppServices container, Arc<T> sharing, dependency construction order

### 2. `rust/docs/GUI-DEVELOPMENT.md`
GPUI GUI development guide covering:
- GPUI core concepts used: Views (Render trait), Context (App/Window/View), Elements (div, svg), Actions, KeyBindings
- Custom theme system: ThemeRegistry, ZFaktury Dark/Light color tokens, font setup (Inter + JetBrains Mono)
- Navigation/routing: Route enum, NavigationState with history/forward stacks, navigate/back/forward methods
- Data loading pattern: cx.spawn + background_executor + this.update + cx.notify (with code example)
- Component inventory: every shared component, what it does, when to use it (table format)
- Sidebar: collapsible architecture, sections, Lucide icons, Ctrl+Shift+L shortcut, localStorage persistence
- Split-view: master-detail for invoices/expenses, resizable divider, Ctrl+\\ toggle
- Command palette: fuzzy search algorithm, diacritics normalization for Czech, command registry
- Virtual scrolling: gpui-component Table setup, background filtering, fixed header
- Animation patterns: which elements animate, timing values, easing -- and the rule that numbers NEVER animate
- Headless testing: cage + grim setup, headless-screenshot.sh usage, --route/--exit-after args, Tier 2 workflow
- How to add a new screen (step-by-step): 1) Add Route variant, 2) Create view struct implementing Render, 3) Add to sidebar, 4) Wire in root.rs routing match, 5) Add data loading, 6) Add #[gpui::test], 7) Add headless screenshot test
- How to add a new form: state fields pattern, validation flow, save handler with background_executor, optimistic update, error display

### 3. `rust/CLAUDE.md`
Claude Code project instructions for the Rust codebase:
- Project overview (what ZFaktury-Rust is, GPUI desktop app, SQLite)
- Build & run commands: `nix develop`, `cargo build/test/clippy/fmt`, headless screenshot
- Architecture summary (brief, links to docs/ARCHITECTURE.md)
- Coding standards: Amount for money (never float), no serde on domain types, error wrapping with context, Czech UI text accuracy
- Test conventions: rstest for tables, proptest for properties, golden files (UPDATE_GOLDEN=1), #[gpui::test] for UI, wiremock for APIs
- File naming: snake_case.rs for Rust, migrations V{N}__{name}.sql
- Key dependencies table
- Adding new features checklist (domain → repo trait → repo impl → service → GPUI view → tests → screenshot)
- GUI testing: how to use headless-screenshot.sh, how to verify screenshots

**Important:** These docs must be written from the ACTUAL implemented code, not copied from RFC files. The agent writing them should read the real source files.

---

## 10. Acceptance Criteria

All of the following must be true before Phase 7 is considered complete:

1. `zfaktury backup` creates a valid backup archive containing database and documents.
2. `zfaktury backup --restore <id>` restores database and documents correctly.
3. `zfaktury migrate` runs pending migrations; `--status` shows current state.
4. `zfaktury version` prints version, commit hash, and build date.
5. Drag & drop uploads documents with proper validation and visual feedback.
6. All seven animation types implemented and smooth (sidebar, page, dialog, chart, toast, badge).
7. Animations respect system reduced-motion preference.
8. Monetary values never animate.
9. Focus management correct in all dialogs (trap + return).
10. Full keyboard navigation for all features (no mouse-only interactions).
11. Focus ring visible on all interactive elements in both themes.
12. WCAG AA contrast ratios met in both light and dark themes.
13. `cargo test --workspace` -- all tests pass.
14. `cargo clippy --workspace -- -D warnings` -- zero warnings.
15. `cargo fmt --check` -- all code formatted.
16. Coverage: calc + amount at 100%, gen at 100%, workspace at 90%+.
17. Golden file tests pass.
18. XSD validation tests pass.
19. Real database import test passes (when fixture present).
20. Generated PDF text content matches Go version for same invoice.
21. Generated XML matches Go version for same VAT return.
22. No `TODO`/`FIXME` comments in production code.
23. No `dbg!()`/`println!()` in production code.
24. All user-facing error messages in Czech.
25. Cross-platform file paths (no hardcoded separators).
26. `rust/docs/ARCHITECTURE.md` covers workspace structure, 3-layer architecture, threading model, and all conventions.
27. `rust/docs/GUI-DEVELOPMENT.md` covers GPUI patterns, component inventory, navigation, and step-by-step guides.
28. `rust/CLAUDE.md` provides accurate build commands, coding standards, and feature addition checklist.
29. All three documentation files are written from actual implemented code, not copied from RFC files.

---

## 11. Review Checklist

Final review before merge:

- [ ] CLI commands share config loading with desktop mode (single `load_config()`)
- [ ] Backup archive includes both database file and documents directory
- [ ] Restore creates automatic pre-restore backup
- [ ] Restore verifies database integrity before replacing
- [ ] Animations do not affect monetary value displays
- [ ] Focus returns to trigger element after every dialog close
- [ ] Tab order is logical in all forms (top-to-bottom, left-to-right)
- [ ] High contrast mode (both themes) is readable and meets WCAG AA
- [ ] No TODO/FIXME comments remain in production code
- [ ] Error messages are helpful and in Czech where user-facing
- [ ] Log messages are in English and use structured tracing
- [ ] Performance acceptable on reasonable hardware (see section 7)
- [ ] `rust-toolchain.toml` pins toolchain version
- [ ] `cargo-deny` passes (licenses, advisories, duplicates)
- [ ] Real database import test documented and passing locally
- [ ] `rust/docs/ARCHITECTURE.md` written from actual source code, not RFC files
- [ ] `rust/docs/GUI-DEVELOPMENT.md` includes component inventory matching real codebase
- [ ] `rust/CLAUDE.md` build commands verified to work
- [ ] Documentation (ARCHITECTURE.md, GUI-DEVELOPMENT.md, CLAUDE.md) cross-referenced and consistent
