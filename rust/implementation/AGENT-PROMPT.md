# ZFaktury Rust Rewrite - Agent Orchestrator Prompt

## Mission

You are implementing a complete rewrite of ZFaktury (Czech invoicing app) from Go to Rust. The rewrite preserves all existing features with a new GPUI desktop UI and significantly improved test coverage.

**Working directory:** `rust/` folder in the ZFaktury repository.

**Reference code:** Go source in `internal/`, frontend in `frontend/src/`.

**Implementation RFCs:** `rust/implementation/phase-{1-7}-*.md` and `rust/implementation/phase-11-ui-completion.md` - These are your detailed specifications. Follow them meticulously.

## Progress Tracking with Tasks

You MUST use the task system to track all work. This is critical for visibility and resumability.

### Phase-Level Tasks

At the start of the project, create a top-level task for each phase:

```
TaskCreate: "Phase 0: POC - GPUI + Headless Screenshot" (status: pending)
TaskCreate: "Phase 1: Foundation" (status: pending)
TaskCreate: "Phase 2: Persistence" (status: pending)
TaskCreate: "Phase 3: Generation" (status: pending)
TaskCreate: "Phase 4: External APIs" (status: pending)
TaskCreate: "Phase 5: Services" (status: pending)
TaskCreate: "Phase 6: GPUI App" (status: pending)
TaskCreate: "Phase 7: Polish" (status: pending)
```

### Sub-Tasks Per Phase

When starting a phase, break it into granular sub-tasks. Example for Phase 1:

```
TaskCreate: "P1: Amount newtype + arithmetic ops + tests" (status: pending)
TaskCreate: "P1: Contact, Invoice, Expense domain types" (status: pending)
TaskCreate: "P1: Tax, Investment, Recurring domain types" (status: pending)
TaskCreate: "P1: DomainError enum" (status: pending)
TaskCreate: "P1: Config crate (TOML loading, env override)" (status: pending)
TaskCreate: "P1: calc/constants.rs (2024-2026)" (status: pending)
TaskCreate: "P1: calc/vat.rs + tests" (status: pending)
TaskCreate: "P1: calc/income_tax.rs + tests" (status: pending)
TaskCreate: "P1: calc/insurance.rs + tests" (status: pending)
TaskCreate: "P1: calc/credits.rs + deductions.rs + tests" (status: pending)
TaskCreate: "P1: calc/fifo.rs + tests" (status: pending)
TaskCreate: "P1: calc/annual_base.rs + recurring.rs + tests" (status: pending)
TaskCreate: "P1: Test utilities (builders, golden helpers, test DB)" (status: pending)
TaskCreate: "P1: Quality gates (build, clippy, coverage, reviews)" (status: pending)
```

### Task State Transitions

- `pending` -- not started yet
- `in_progress` -- actively being worked on (use TaskUpdate when starting)
- `completed` -- done and verified (use TaskUpdate when finished)

**Rules:**
- Update task to `in_progress` BEFORE starting work
- Update task to `completed` AFTER verifying it works (build + tests pass)
- NEVER leave a task `in_progress` when moving to something else -- either complete it or note the blocker
- When a teammate agent finishes its work, update the corresponding tasks to `completed`
- Phase-level task is `completed` only after ALL sub-tasks AND quality gates pass

## Team-Based Parallelization with TeamCreate

For parallel implementation within each phase, use **TeamCreate** to spawn teammate agents that work in isolated worktrees. This is the primary mechanism for parallelization.

### How to Use TeamCreate

```
TeamCreate with teammates:
  - name: "domain-types"
    prompt: "Implement all domain types in zfaktury-domain crate. [detailed instructions...]"
  - name: "config-crate"
    prompt: "Implement zfaktury-config crate. [detailed instructions...]"
  - name: "calc-tax"
    prompt: "Implement calc/vat.rs, calc/income_tax.rs, calc/insurance.rs with 100% coverage. [detailed instructions...]"
  - name: "calc-credits"
    prompt: "Implement calc/credits.rs, calc/deductions.rs, calc/fifo.rs, calc/annual_base.rs, calc/recurring.rs with 100% coverage. [detailed instructions...]"
```

### Teammate Rules

1. Each teammate works in an **isolated worktree** (automatic with TeamCreate)
2. Each teammate owns specific files -- **NO overlapping file ownership**
3. Each teammate MUST run `cargo build` and `cargo test` on their code before reporting done
4. The lead (you) merges teammate outputs into the main branch
5. Shared files are edited ONLY by the lead after all teammates finish

### Shared Files (Lead Merges Only)

These files are touched by multiple teammates and must be integrated by the lead:

- `rust/Cargo.toml` -- workspace member list
- `rust/crates/*/Cargo.toml` -- cross-crate dependencies
- `rust/crates/zfaktury-core/src/lib.rs` -- module re-exports
- `rust/crates/zfaktury-domain/src/lib.rs` -- module re-exports

### Communication with Teammates

Use **SendMessage** to communicate with running teammates:
- Ask for status updates
- Provide API contracts they need (e.g., "here are the trait signatures to implement against")
- Coordinate when one teammate's output is needed by another

## Execution Protocol

### Before Each Phase

1. **Read the RFC** for the current phase completely
2. **Read relevant Go source files** referenced in the RFC
3. **Create sub-tasks** (TaskCreate) for all work items in the phase
4. **Update phase task** to `in_progress`
5. **Identify shared files** that only the lead should edit
6. **Identify teammate split** -- which files each teammate owns

### During Each Phase

1. **Create a team** (TeamCreate) with isolated worktree teammates for parallel tasks
2. **Update sub-tasks** to `in_progress` as teammates start working
3. Each teammate runs `cargo build` and `cargo test` on its own code before reporting done
4. **Update sub-tasks** to `completed` as teammates finish
5. Lead merges teammate outputs, integrates shared files (Cargo.toml workspace, lib.rs re-exports)
6. After merge, run full workspace build: `cargo build --workspace`
7. Run full test suite: `cargo test --workspace`
8. Fix any failures before proceeding to reviews

### After Each Phase - Quality Gates (ALL MUST PASS)

**Gate 1: Build & Test**
```bash
cargo build --workspace
cargo test --workspace
cargo clippy --workspace -- -D warnings
cargo fmt --check
```

**Gate 2: Code Review** (run as background agent)
- Launch `developer:code-reviewer` agent
- Review all code written in this phase
- Check: bugs, logic errors, code quality, project convention adherence
- Check: 3-layer architecture, Amount for money, no serde derives on domain types
- Check: test coverage adequacy

**Gate 3: Security Review** (run as background agent)
- Launch `developer:code-security` agent
- Check: SQL injection, path traversal, input validation
- Check: file upload safety, sensitive data exposure
- Check: error message leakage

**Gate 4: Coverage Check**
```bash
# Install if needed: cargo install cargo-llvm-cov
cargo llvm-cov --workspace

# Phase-specific thresholds:
# Phase 1: calc/ = 100%, amount.rs = 100%
# Phase 2: db/ = 90%+
# Phase 3: gen/ = 100%
# Phase 4: api/ = 90%+
# Phase 5: service/ = 90%+
# Phase 6: app/ = reasonable (GPUI views hard to test)
# Phase 7: overall 90%+
```

**Gate 5: Phase-Specific Checks**
- Phase 1: Run proptest with 1000 cases for Amount and calc modules
- Phase 2: Verify migration bridge with mock goose_db_version table
- Phase 3: All golden files committed, XSD validation passes
- Phase 4: wiremock tests for all external APIs
- Phase 5: All services are Send + Sync (compile check)
- Phase 6: App launches and renders dashboard (visual verification)
- Phase 7: Real database import test, CLI commands work

### Go/No-Go Decision

After all gates pass:
- **GO:** Update phase task to `completed`, proceed to next phase
- **NO-GO:** Fix all issues, re-run gates, repeat until pass

If stuck on a blocker:
1. Document the blocker clearly in the task
2. Propose workaround or alternative approach
3. Ask user for decision before proceeding

## Phase Execution Order

```
Phase 0: POC (GPUI + cage + grim)       → DONE
Phase 1: Foundation                       → DONE (221 tests)
Phase 2: Persistence                      → DONE (61 tests)
Phase 3: Generation                       → DONE (73 tests)
Phase 4: External APIs                    → DONE (50 tests)
Phase 5: Services                         → DONE (32 tests)
Phase 6: GPUI App                         → PARTIAL (17 read-only views, 8 stubs, 19 StubView routes)
Phase 7: Polish                           → DONE (CLI + docs)
Phase 8: PDF Generation                   → DONE (15 tests, typst-bake)
Phase 9: View Completion                  → PARTIAL (views exist but all read-only, no forms/actions)
Phase 10: Quality Gates                   → PARTIAL (clippy done, reviews done, UI not usable)

Phase 11: UI Infrastructure               → DONE (9 components, 4 routes, 18 services, NavigateEvent)
Phase 12: Invoice CRUD                    → NOT STARTED
Phase 13: Expense+Contact CRUD           → NOT STARTED
Phase 14: Settings CRUD                   → NOT STARTED
Phase 15: Recurring Templates             → NOT STARTED
Phase 16: VAT Management                  → NOT STARTED
Phase 17: Tax Filing                      → NOT STARTED
Phase 18: Reports+Dashboard               → NOT STARTED
Phase 19: Import+UX Polish               → NOT STARTED
```

**Current state:** 444 tests passing, 8 crates, ~25k lines Rust code.
**Backend is production-ready.** The gap is entirely in the GPUI UI layer.
**See:** `rust/implementation/STATUS.md` for detailed gap analysis.
**See:** `rust/implementation/phase-11-ui-completion.md` for the UI completion RFC.

Phases 0-10 established the full backend + read-only views. Phases 11-19 make the UI fully functional.

## Team Structure Per Phase

### Phase 0: POC (no teammates -- lead only)

Phase 0 is small enough for the lead to do alone. No TeamCreate needed.

**Tasks:**
1. Create `rust/flake.nix` with nix devshell (rust, vulkan, wayland, libxkbcommon, fontconfig, cage, grim, cargo-llvm-cov)
2. Create minimal GPUI app (`rust/crates/zfaktury-app/`) with:
   - Dark background (#1e1d21), "ZFaktury" title text, one button
   - `--route` and `--exit-after` CLI args (clap)
3. Create `rust/scripts/headless-screenshot.sh` (cage + WLR_BACKENDS=headless + grim)
4. Verify: `nix develop` → `cargo build` → app opens on Hyprland → headless screenshot works

**Acceptance:** Screenshot PNG shows dark bg + text, no window on user's desktop.

**If fails:** STOP. Report to user. Do NOT proceed to Phase 1.

### Phase 1: Foundation (4 teammates)

```
TeamCreate:
  - name: "p1-domain"
    files: rust/crates/zfaktury-domain/src/**
    prompt: "Implement all domain types (contact, invoice, expense, tax, investment, recurring, audit, ocr, import, errors, settings, amount). Read Go internal/domain/*.go for exact field definitions. No serde derives on domain structs. Amount(i64) with Add/Sub/Mul(f64)/Display/Ord/Eq/Copy. Run cargo build && cargo test."

  - name: "p1-config"
    files: rust/crates/zfaktury-config/src/**
    prompt: "Implement config crate: TOML loading, ZFAKTURY_DATA_DIR env override, fail-fast on missing required config. Read Go internal/config/config.go. Use serde::Deserialize for TOML. Run cargo build && cargo test."

  - name: "p1-calc-tax"
    files: rust/crates/zfaktury-core/src/calc/{vat,income_tax,insurance,constants}.rs
    prompt: "Implement calc modules: constants.rs (2024-2026 tax constants), vat.rs (VAT return calculation), income_tax.rs (progressive 15%/23%), insurance.rs (social 29.2%, health 13.5%). Read Go internal/calc/*.go. 100% test coverage with rstest tables + proptest properties. Run cargo build && cargo test."

  - name: "p1-calc-credits"
    files: rust/crates/zfaktury-core/src/calc/{credits,deductions,fifo,annual_base,recurring}.rs
    prompt: "Implement calc modules: credits.rs (spouse, personal, child benefit), deductions.rs (mortgage, pension, etc.), fifo.rs (FIFO cost basis algorithm from Go investment_income_svc.go:193-313), annual_base.rs (year revenue/expense filtering), recurring.rs (next occurrence). Read Go sources. 100% test coverage. Run cargo build && cargo test."
```

Lead tasks after teammates finish:
- Create workspace Cargo.toml
- Implement zfaktury-testutil (needs domain types from p1-domain)
- Merge all worktrees
- Integrate lib.rs re-exports
- Run full workspace build + tests

### Phase 2: Persistence (3 teammates)

```
TeamCreate:
  - name: "p2-entity-repos"
    files: rust/crates/zfaktury-db/src/{contact,invoice,expense,sequence,category,document,invoice_document,recurring_invoice,recurring_expense,status_history,reminder}_repo.rs
    prompt: "Implement core entity repositories with rusqlite. Read Go internal/repository/{contact,invoice,expense,sequence,category,document,...}_repo.go. Include scan helpers, soft deletes, pagination. Integration tests with in-memory SQLite."

  - name: "p2-tax-repos"
    files: rust/crates/zfaktury-db/src/{vat_return,vat_control,vies,income_tax_return,social_insurance,health_insurance,tax_year_settings,tax_prepayment,tax_spouse_credit,tax_child_credit,tax_personal_credits,tax_deduction,tax_deduction_document}_repo.rs
    prompt: "Implement tax/VAT repositories with rusqlite. Read Go internal/repository/{vat_return,vat_control,vies,...}_repo.go. Include link tables (vat_return_invoices, etc.). Integration tests."

  - name: "p2-other-repos"
    files: rust/crates/zfaktury-db/src/{audit_log,backup,dashboard,report,investment_document,capital_income,security_transaction,fakturoid_import,settings}_repo.rs
    prompt: "Implement remaining repositories. Read Go internal/repository/{audit_log,backup,dashboard,report,...}_repo.go. Include FIFO-specific queries (list_buys_for_fifo, update_fifo_results). Integration tests."
```

Lead tasks: Repository trait definitions in zfaktury-core, migrations (port from Go), goose→refinery bridge, helpers.rs

### Phase 3: Generation (3 teammates)

```
TeamCreate:
  - name: "p3-pdf-qr"
    files: rust/crates/zfaktury-gen/src/{pdf,qr}/**
    prompt: "Implement invoice PDF via typst + QR/SPAYD generation. Read Go internal/pdf/*.go. Golden file tests for PDF metadata. SPAYD string verification tests."

  - name: "p3-vat-xml"
    files: rust/crates/zfaktury-gen/src/{xml/{vat_return,control_statement,vies},isdoc}/**
    prompt: "Implement VAT XML (DPHDP3, DPHKH1, DPHSHV) + ISDOC 6.0.2. Read Go internal/vatxml/*.go and internal/isdoc/*.go. XSD validation + golden file tests."

  - name: "p3-tax-xml-csv"
    files: rust/crates/zfaktury-gen/src/{xml/{income_tax,social_insurance,health_insurance},csv}/**
    prompt: "Implement annual tax XML (DPFDP5, CSSZ, ZP) + CSV export. Read Go internal/annualtaxxml/*.go. XSD validation + golden file tests."
```

### Phase 4: External APIs (3 teammates)

```
TeamCreate:
  - name: "p4-ares-cnb"
    files: rust/crates/zfaktury-api/src/{ares,cnb}/**
    prompt: "Implement ARES + CNB clients with reqwest. Read Go internal/ares/ and internal/service/cnb/. wiremock tests."

  - name: "p4-email-ocr"
    files: rust/crates/zfaktury-api/src/{email,ocr}/**
    prompt: "Implement SMTP sender (lettre) + OCR providers (Anthropic, OpenAI). Read Go internal/service/email/ and internal/service/ocr/. Tests with lettre test transport + wiremock."

  - name: "p4-fakturoid-fio"
    files: rust/crates/zfaktury-api/src/{fakturoid,fio}/**
    prompt: "Implement Fakturoid (OAuth2, pagination) + FIO Bank clients. Read Go internal/fakturoid/. wiremock tests."
```

### Phase 5: Services (4 teammates)

```
TeamCreate:
  - name: "p5-core-services"
    files: rust/crates/zfaktury-core/src/service/{contact,invoice,expense,sequence,category,document,invoice_document}_svc.rs
    prompt: "Implement core entity services. Read Go internal/service/{contact,invoice,expense,...}_svc.go. Mock repos with mockall. Test CRUD, validation, state machines, error wrapping, audit logging."

  - name: "p5-tax-services"
    files: rust/crates/zfaktury-core/src/service/{vat_return,vat_control,vies,income_tax_return,social_insurance,health_insurance}_svc.rs
    prompt: "Implement tax filing services. Read Go internal/service/{vat_return,income_tax_return,...}_svc.go. Test recalculate flows, filing type validation, status transitions."

  - name: "p5-tax-support"
    files: rust/crates/zfaktury-core/src/service/{tax_credits,tax_deduction_document,tax_year_settings,tax_calendar,investment_income,investment_document,investment_extraction,tax_document_extraction}_svc.rs
    prompt: "Implement tax support services. Read Go internal/service/{tax_credits,investment_income,...}_svc.go. Test credit/deduction computation, FIFO recalculation trigger, extraction flows."

  - name: "p5-utility-services"
    files: rust/crates/zfaktury-core/src/service/{recurring_invoice,recurring_expense,import,ocr,overdue,reminder,report,dashboard,backup,backup_storage_local,backup_storage_s3,fakturoid_import}_svc.rs
    prompt: "Implement utility services. Read Go internal/service/{recurring,import,overdue,reminder,report,dashboard,backup,fakturoid_import}_svc.go. Test recurring generation, overdue detection, backup/restore."
```

Lead tasks: AuditService (dependency for all), service wiring (AppServices container), dependency integration

### Phase 6: GPUI App (5 teammates)

```
TeamCreate:
  - name: "p6-skeleton"
    files: rust/crates/zfaktury-app/src/{app,theme,navigation,root,sidebar,title_bar}.rs
    prompt: "Build GPUI app skeleton: window, custom themes (Dark/Light), sidebar navigation, routing. Read frontend/src/lib/components/Layout.svelte for navigation structure, frontend/src/app.css for colors."

  - name: "p6-invoices-contacts"
    files: rust/crates/zfaktury-app/src/views/{invoices,contacts}/**
    prompt: "Build invoice views (list with virtual scroll, detail, create/edit form, items editor) + contact views. Read frontend/src/routes/invoices/ and contacts/."

  - name: "p6-expenses-recurring"
    files: rust/crates/zfaktury-app/src/views/{expenses,recurring}/**
    prompt: "Build expense views (list, detail, form, import, review) + recurring invoice/expense views. Read frontend/src/routes/expenses/ and recurring/."

  - name: "p6-tax-views"
    files: rust/crates/zfaktury-app/src/views/{vat,tax}/**
    prompt: "Build all tax views: VAT overview/returns/control/VIES, income tax, social/health insurance, credits, prepayments, investments. Read frontend/src/routes/vat/ and tax/."

  - name: "p6-settings-reports"
    files: rust/crates/zfaktury-app/src/views/{settings,reports,dashboard}.rs
    prompt: "Build settings pages (firma, email, sequences, categories, PDF, audit log, backup, fakturoid import) + reports (5 tabs with charts) + dashboard (stat cards, charts, recent tables). Read frontend/src/routes/settings/ and reports/."
```

Lead tasks: Command palette, split-view, shared components (dialogs, pickers, status badge, charts), keyboard shortcuts, toast system

### Phase 7: Polish (3 teammates)

```
TeamCreate:
  - name: "p7-cli-animations"
    files: rust/src/main.rs (CLI), rust/crates/zfaktury-app/src/animations.rs
    prompt: "Implement CLI subcommands (backup, restore, migrate via clap), drag & drop, all animations (sidebar 200ms, page 150ms, dialog 200ms, chart 400ms). Numbers NEVER animate."

  - name: "p7-accessibility-cleanup"
    files: (audit existing, modify as needed)
    prompt: "Accessibility audit: focus management, keyboard navigation, contrast ratios. Final cleanup: remove TODOs, verify error messages, cross-platform paths. Performance verification."

  - name: "p7-documentation"
    files: rust/docs/ARCHITECTURE.md, rust/docs/GUI-DEVELOPMENT.md, rust/CLAUDE.md
    prompt: "Write documentation based on the implemented codebase:
    1. ARCHITECTURE.md: workspace structure, crate graph, 3-layer architecture, data flow (View→Service→Repo), Amount money system, error handling (thiserror/anyhow), threading model (main vs background_executor), config system, migration strategy.
    2. GUI-DEVELOPMENT.md: GPUI concepts (Views, Context, Elements, Actions, KeyBindings), theme system, navigation/routing, data loading pattern, component inventory, sidebar, split-view, command palette, virtual scrolling, animations, headless testing (cage+grim), how to add a new screen, how to add a new form.
    3. CLAUDE.md: build commands (nix develop, cargo), coding standards, test conventions, file naming, dependencies, adding new features checklist.
    Read the actual implemented code to write accurate documentation -- do NOT copy from RFC files."
```

Lead tasks: Final test pass, coverage enforcement, real DB import test, commit

## Nix Devshell

ALL builds and tests run inside `nix develop` in the `rust/` directory. The `rust/flake.nix` provides:
- Rust toolchain (stable + rust-src + rust-analyzer)
- GPUI build deps: pkg-config, vulkan-loader, wayland, libxkbcommon, fontconfig, freetype, openssl
- Headless GUI testing: cage, grim
- Dev tools: cargo-llvm-cov
- LD_LIBRARY_PATH and VK_ICD_FILENAMES configured automatically

**Every `cargo` command must be run from within `nix develop`.** Teammates should be instructed to `cd rust && nix develop` before building.

## GUI Verification Protocol

GUI verification uses 3 tiers. Agents use Tier 1 and Tier 2. User uses Tier 3.

### Tier 1: Programmatic (`#[gpui::test]`)
- Headless, no display needed
- Tests navigation, data binding, component state, keyboard shortcuts
- Part of `cargo test` -- runs automatically

### Tier 2: Headless Screenshots (cage + grim)
- Uses `rust/scripts/headless-screenshot.sh` wrapper
- Runs app in isolated Wayland session (cage + WLR_BACKENDS=headless)
- Captures PNG screenshot via grim
- Zero impact on user's Hyprland desktop (separate XDG_RUNTIME_DIR)
- Real GPU rendering via RADV Vulkan

**Usage:**
```bash
./rust/scripts/headless-screenshot.sh ./target/debug/zfaktury-app /tmp/screenshot.png 4
./rust/scripts/headless-screenshot.sh "./target/debug/zfaktury-app --route /invoices" /tmp/invoices.png 4
```

**Per-screen verification (Phase 6 quality gate):**
1. Build app with test fixture database
2. For each of the 43 routes: launch with `--route <path>` + `--exit-after 5`
3. Take screenshot via headless-screenshot.sh
4. View screenshot (Claude can read images) and verify:
   - Layout correct (sidebar, content, panels)
   - Czech labels present and correct
   - Data displayed from fixture DB
   - No rendering glitches
   - Theme colors applied
5. Save screenshots to `rust/tests/screenshots/<route>.png`

### Tier 3: Manual (user only)
- User launches app on their desktop at the end of Phase 7
- Not automated -- happens once for final polish

### CLI Arguments for Testing
The app binary MUST support:
- `--route <path>` -- navigate to specific screen on launch
- `--exit-after <seconds>` -- auto-close after N seconds
- `--db <path>` -- use specific database file (for test fixtures)

## Critical Rules

1. **NEVER skip tests.** Every function needs tests. 100% for calc/gen, 90%+ for rest.
2. **NEVER use float for money.** Always Amount(i64). The only exception is display formatting.
3. **NEVER hardcode test data in production code.** Use builders from zfaktury-testutil.
4. **ALWAYS wrap errors with context.** `anyhow::Context` or explicit `map_err`.
5. **ALWAYS run on background thread** for DB/network/PDF operations in GPUI.
6. **ALWAYS read the Go source** before implementing. Don't guess business logic.
7. **Domain types have NO serde derives.** Only DTOs/API types get serde.
8. **Czech text must be accurate.** Verify against Go frontend source.
9. **Commit after each phase passes all gates.** Clean, descriptive message.
10. **Ask the user if blocked.** Don't silently skip or mock.
11. **ALWAYS update tasks.** Mark `in_progress` when starting, `completed` when done. Never lose track.
12. **Use TeamCreate for parallelism.** Don't spawn individual Agent calls -- use team-based orchestration.

## Reference Quick Links

| What | Where |
|------|-------|
| Domain types | `internal/domain/*.go` |
| Amount type | `internal/domain/money.go` |
| Calc modules | `internal/calc/*.go` |
| Repository interfaces | `internal/repository/interfaces.go` |
| Repository implementations | `internal/repository/*.go` |
| Services | `internal/service/*.go` |
| Service wiring | `internal/server/server.go` (`wireRouter`) |
| Migrations | `internal/database/migrations/*.sql` |
| PDF generation | `internal/pdf/*.go` |
| VAT XML | `internal/vatxml/*.go` |
| Annual tax XML | `internal/annualtaxxml/*.go` |
| ISDOC | `internal/isdoc/*.go` |
| API clients | `internal/ares/`, `internal/cnb/`, `internal/fakturoid/` |
| Email | `internal/service/email/` |
| OCR | `internal/service/ocr/` |
| Config | `internal/config/config.go` |
| Frontend types | `frontend/src/lib/api/client.ts` |
| Theme colors | `frontend/src/app.css` |
| Navigation | `frontend/src/lib/components/Layout.svelte` |
| Status labels | `frontend/src/lib/utils/invoice.ts` |
| All routes | `frontend/src/routes/**/+page.svelte` |
| Nix devshell | `rust/flake.nix` |
| Headless screenshot | `rust/scripts/headless-screenshot.sh` |
| Architecture docs | `rust/docs/ARCHITECTURE.md` (created in Phase 7) |
| GUI dev guide | `rust/docs/GUI-DEVELOPMENT.md` (created in Phase 7) |

## Output Format

After each phase completion, provide a status report and update tasks:

```
## Phase N: [Name] - COMPLETE

### Tasks Completed
- [x] Task 1 description
- [x] Task 2 description
- ...

### Files Created/Modified
- list of all files with line counts

### Test Results
- cargo test: X passed, 0 failed
- cargo clippy: 0 warnings
- Coverage: X% (threshold: Y%)

### Review Findings
- Code review: [summary]
- Security review: [summary]
- Issues found and fixed: [list]

### Ready for Phase N+1: YES/NO
```

Then:
- TaskUpdate: phase task → completed
- TaskUpdate: next phase task → in_progress

---

## Phases 11-19: UI Completion

These phases make the GPUI desktop UI fully functional -- 100% feature parity with the SvelteKit frontend. The backend is already complete; all work is in `zfaktury-app`.

**Detailed RFC:** `rust/implementation/phase-11-ui-completion.md`

### Phase 11: UI Infrastructure (lead only, BLOCKS ALL OTHER UI PHASES)

**Goal:** Build reusable form components, navigation system, and wire missing services.

**Tasks:**
```
TaskCreate: "P11: Add gpui-component dependency or build minimal form widgets" (status: pending)
TaskCreate: "P11: TextInput component (editable text field)" (status: pending)
TaskCreate: "P11: NumberInput component (Amount-aware numeric input)" (status: pending)
TaskCreate: "P11: Select/Dropdown component" (status: pending)
TaskCreate: "P11: DateInput component (dd.mm.yyyy Czech format)" (status: pending)
TaskCreate: "P11: Checkbox/Toggle component" (status: pending)
TaskCreate: "P11: Button component (loading/disabled states)" (status: pending)
TaskCreate: "P11: ConfirmDialog component (modal overlay)" (status: pending)
TaskCreate: "P11: Toast notification system" (status: pending)
TaskCreate: "P11: View-level NavigateEvent emission (row clicks, buttons)" (status: pending)
TaskCreate: "P11: Add missing routes (ContactNew, InvoiceEdit, ExpenseEdit, ContactEdit)" (status: pending)
TaskCreate: "P11: Wire 18 missing services into AppServices" (status: pending)
TaskCreate: "P11: Build + test gates" (status: pending)
```

**Shared files (lead only):**
- `rust/crates/zfaktury-app/src/app.rs` -- AppServices expansion
- `rust/crates/zfaktury-app/src/root.rs` -- content view event subscriptions
- `rust/crates/zfaktury-app/src/navigation.rs` -- new Route variants
- `rust/crates/zfaktury-app/src/components/mod.rs` -- register all new components
- `rust/crates/zfaktury-app/Cargo.toml` -- add gpui-component if used

**Acceptance:** All components render in isolation. Navigation works from any view. `cargo build --workspace` passes.

---

### Phase 12: Invoice CRUD (2 teammates)

**Goal:** Make invoice management fully functional -- create, edit, actions, exports.

**Tasks:**
```
TaskCreate: "P12: Invoice items editor component (add/remove/edit items, live totals)" (status: pending)
TaskCreate: "P12: Invoice form -- create mode (customer select, dates, items, save)" (status: pending)
TaskCreate: "P12: Invoice form -- edit mode (load existing, pre-fill, update)" (status: pending)
TaskCreate: "P12: Invoice detail -- action buttons (edit, delete, send, mark paid)" (status: pending)
TaskCreate: "P12: Invoice detail -- duplicate, credit note, settle proforma" (status: pending)
TaskCreate: "P12: Invoice detail -- PDF download, ISDOC download, QR download" (status: pending)
TaskCreate: "P12: Invoice detail -- send email dialog (recipient, subject, body, attachments)" (status: pending)
TaskCreate: "P12: Invoice list -- row click, search, status filter, type filter, pagination" (status: pending)
TaskCreate: "P12: Invoice list -- 'New' button, 'Check overdue' button" (status: pending)
TaskCreate: "P12: Build + test gates" (status: pending)
```

**TeamCreate:**
```
  - name: "p12-invoice-form"
    files: rust/crates/zfaktury-app/src/views/invoice_form.rs, rust/crates/zfaktury-app/src/components/invoice_items_editor.rs
    prompt: "Rewrite invoice_form.rs with real form inputs using the Phase 11 components. Implement invoice_items_editor.rs component. Form must: load contacts for customer select, support create (InvoiceNew) and edit (InvoiceEdit) modes, calculate totals live, validate (customer required, min 1 item), call invoices.create() or invoices.update(), navigate to InvoiceDetail on success. Read frontend/src/routes/invoices/new/+page.svelte and frontend/src/lib/components/InvoiceItemsEditor.svelte for exact UX. Run cargo build."

  - name: "p12-invoice-actions"
    files: rust/crates/zfaktury-app/src/views/invoice_detail.rs, rust/crates/zfaktury-app/src/views/invoice_list.rs
    prompt: "Add action buttons to invoice_detail.rs: Edit (navigate to InvoiceEdit), Delete (confirm dialog + invoices.delete), Mark Sent (invoices.mark_as_sent), Mark Paid (dialog with amount+date + invoices.mark_as_paid), Duplicate (invoices.duplicate + navigate), Credit Note (dialog + invoices.create_credit_note), Settle Proforma, PDF download (PdfGenerator), ISDOC download (IsdocGenerator), QR download (QrGenerator), Send Email (dialog with EmailSender). Enhance invoice_list.rs: row click navigation, search input, status/type filter dropdowns, pagination controls, 'New' button. Read frontend/src/routes/invoices/[id]/+page.svelte for UX. Run cargo build."
```

**Acceptance:** Full invoice lifecycle: create with items → view detail → send → mark paid → download PDF.

---

### Phase 13: Expense+Contact CRUD (2 teammates)

**Goal:** Make expense and contact management fully functional.

**Tasks:**
```
TaskCreate: "P13: Expense form -- create/edit with category/vendor select, items, save" (status: pending)
TaskCreate: "P13: Expense detail -- edit, delete, tax review toggle, document upload" (status: pending)
TaskCreate: "P13: Expense list -- row click, search, filters, pagination, 'New' button" (status: pending)
TaskCreate: "P13: Contact form (new file) -- create/edit with ARES lookup button" (status: pending)
TaskCreate: "P13: Contact detail -- edit, delete, favorite toggle" (status: pending)
TaskCreate: "P13: Contact list -- row click, search, 'New' button" (status: pending)
TaskCreate: "P13: Build + test gates" (status: pending)
```

**TeamCreate:**
```
  - name: "p13-expenses"
    files: rust/crates/zfaktury-app/src/views/{expense_form,expense_detail,expense_list}.rs
    prompt: "Rewrite expense_form.rs: category select (CategoryService), vendor select (ContactService), date/amount/VAT/business% inputs, optional expense items, save via expenses.create()/update(). Add actions to expense_detail.rs: Edit, Delete, Mark/Unmark Tax Reviewed, document upload (file picker + DocumentService). Enhance expense_list.rs: row click, search, date range filter, pagination, 'New' button. Read frontend/src/routes/expenses/. Run cargo build."

  - name: "p13-contacts"
    files: rust/crates/zfaktury-app/src/views/{contact_form,contact_detail,contact_list}.rs
    prompt: "Create contact_form.rs: name, type (company/individual), ICO with ARES lookup button (AresClient), DIC, address, contact info, bank details, payment terms, notes. Save via contacts.create()/update(). Add to contact_detail.rs: Edit, Delete (confirm), Favorite toggle. Enhance contact_list.rs: row click, search by name/ICO/email, 'New' button. Read frontend/src/routes/contacts/. Run cargo build."
```

**New file:** `rust/crates/zfaktury-app/src/views/contact_form.rs`

**Acceptance:** Create contact → create expense for that vendor → view detail → tax review.

---

### Phase 14: Settings CRUD (2 teammates)

**Goal:** Make all settings pages editable.

**Tasks:**
```
TaskCreate: "P14: Settings Firma -- editable form with save/cancel" (status: pending)
TaskCreate: "P14: Settings Email -- editable SMTP, test email button" (status: pending)
TaskCreate: "P14: Settings Sequences -- CRUD (create, edit, delete)" (status: pending)
TaskCreate: "P14: Settings Categories -- CRUD with color input" (status: pending)
TaskCreate: "P14: Settings PDF (new view) -- logo upload, accent color, footer, toggles" (status: pending)
TaskCreate: "P14: Settings Backup -- wire BackupService, create/list/delete" (status: pending)
TaskCreate: "P14: Settings Audit -- add filters (entity type, action, date)" (status: pending)
TaskCreate: "P14: Build + test gates" (status: pending)
```

**TeamCreate:**
```
  - name: "p14-settings-forms"
    files: rust/crates/zfaktury-app/src/views/{settings_firma,settings_email,settings_sequences,settings_categories}.rs
    prompt: "Make settings_firma.rs editable: toggle edit mode, text inputs for all fields, save via settings.set_bulk(), cancel reverts. Make settings_email.rs editable: SMTP fields, template fields, attachment toggles, 'Test email' button via EmailSender. Make settings_sequences.rs CRUD: 'New' button, inline edit/delete per row. Make settings_categories.rs CRUD: 'New' button, color hex input, edit/delete. Read frontend/src/routes/settings/. Run cargo build."

  - name: "p14-settings-misc"
    files: rust/crates/zfaktury-app/src/views/{settings_pdf,settings_backup,settings_audit}.rs
    prompt: "Create settings_pdf.rs (new file): logo upload (file picker), accent color input, footer text, QR toggle, bank details toggle, preview button. Make settings_backup.rs functional: wire BackupService, 'Create backup' button, load history list, download/delete buttons. Make settings_audit.rs filterable: entity type dropdown, action dropdown, date range inputs. Read frontend/src/routes/settings/. Run cargo build."
```

**New file:** `rust/crates/zfaktury-app/src/views/settings_pdf.rs`

**Acceptance:** Edit company info → save → verify persisted. Create backup → see in list.

---

### Phase 15: Recurring Templates (1 teammate)

**Goal:** Full CRUD for recurring invoice and expense templates.

**Tasks:**
```
TaskCreate: "P15: Recurring invoice detail + form (new files)" (status: pending)
TaskCreate: "P15: Recurring expense detail + form (new files)" (status: pending)
TaskCreate: "P15: Recurring lists -- row click, 'New', generate, activate/deactivate" (status: pending)
TaskCreate: "P15: Build + test gates" (status: pending)
```

**TeamCreate:**
```
  - name: "p15-recurring"
    files: rust/crates/zfaktury-app/src/views/{recurring_invoice_detail,recurring_invoice_form,recurring_expense_detail,recurring_expense_form,recurring_invoice_list,recurring_expense_list}.rs
    prompt: "Create 4 new view files. recurring_invoice_detail.rs: display all fields + items, action buttons (Edit, Delete, Generate Next, Activate/Deactivate). recurring_invoice_form.rs: customer select, items editor (reuse from P12), frequency select, dates, save. recurring_expense_detail.rs + form.rs: similar pattern. Enhance both list views: row click, 'New' button, status indicators, generate button. Read frontend/src/routes/recurring/ and expenses/recurring/. Run cargo build."
```

**New files:** `recurring_invoice_detail.rs`, `recurring_invoice_form.rs`, `recurring_expense_detail.rs`, `recurring_expense_form.rs`

**Acceptance:** Create recurring invoice → generate instance → verify created.

---

### Phase 16: VAT Management (1 teammate)

**Goal:** Full VAT returns, control statements, and VIES summaries.

**Tasks:**
```
TaskCreate: "P16: VAT overview -- wire services, load real data, navigation" (status: pending)
TaskCreate: "P16: VAT return detail -- actions (recalculate, XML, mark filed, delete)" (status: pending)
TaskCreate: "P16: VAT return form (new file)" (status: pending)
TaskCreate: "P16: VAT control statement detail + form (new files)" (status: pending)
TaskCreate: "P16: VIES summary detail + form (new files)" (status: pending)
TaskCreate: "P16: Build + test gates" (status: pending)
```

**TeamCreate:**
```
  - name: "p16-vat"
    files: rust/crates/zfaktury-app/src/views/{vat_overview,vat_return_detail,vat_return_form,vat_control_detail,vat_control_form,vies_detail,vies_form}.rs
    prompt: "Wire VATReturnService, VATControlStatementService, VIESSummaryService into views. Enhance vat_overview.rs: load real data per quarter, status badges, click navigation, 'New' buttons. Enhance vat_return_detail.rs: add Recalculate/Generate XML/Mark Filed/Delete buttons. Create vat_return_form.rs: period selector (year, month/quarter), filing type, auto-calculate button. Create vat_control_detail.rs + form.rs: display lines (A.4, A.5, B.2, B.3), same actions. Create vies_detail.rs + form.rs: EU trade lines, same actions. Read frontend/src/routes/vat/. Run cargo build."
```

**New files:** `vat_return_form.rs`, `vat_control_detail.rs`, `vat_control_form.rs`, `vies_detail.rs`, `vies_form.rs`

**Acceptance:** Create VAT return → recalculate → generate XML → mark filed.

---

### Phase 17: Tax Filing (2 teammates)

**Goal:** Full income tax, social/health insurance, credits, prepayments, investments.

**Tasks:**
```
TaskCreate: "P17: Tax overview -- wire services, load data, create buttons" (status: pending)
TaskCreate: "P17: Income tax detail + form (new files)" (status: pending)
TaskCreate: "P17: Social insurance detail + form (new files)" (status: pending)
TaskCreate: "P17: Health insurance detail + form (new files)" (status: pending)
TaskCreate: "P17: Tax credits -- spouse/children/personal CRUD, deductions CRUD" (status: pending)
TaskCreate: "P17: Tax prepayments -- monthly data, edit per month" (status: pending)
TaskCreate: "P17: Tax investments -- capital income CRUD, securities CRUD, FIFO" (status: pending)
TaskCreate: "P17: Build + test gates" (status: pending)
```

**TeamCreate:**
```
  - name: "p17-tax-filings"
    files: rust/crates/zfaktury-app/src/views/{tax_overview,tax_income_detail,tax_income_form,tax_social_detail,tax_social_form,tax_health_detail,tax_health_form}.rs
    prompt: "Wire IncomeTaxReturnService, SocialInsuranceService, HealthInsuranceService. Enhance tax_overview.rs: real data, status badges, 'Create' buttons. Create tax_income_detail.rs: display revenue/expenses/base/tax/credits/bonuses, actions (recalculate, XML DPFDP5, mark filed, delete). Create tax_income_form.rs: year, filing type, auto-calculate. Create tax_social_detail.rs + form.rs: assessment base, insurance rate, prepayments, difference. Create tax_health_detail.rs + form.rs: same pattern. Read frontend/src/routes/tax/. Run cargo build."

  - name: "p17-tax-support"
    files: rust/crates/zfaktury-app/src/views/{tax_credits,tax_prepayments,tax_investments}.rs
    prompt: "Wire TaxCreditsService, TaxDeductionService, InvestmentIncomeService, TaxYearSettingsService. Rewrite tax_credits.rs: spouse credit form (months, ZTP, income), children list with add/edit/delete, personal credits (student/disability toggles), deductions list with CRUD and document upload. Rewrite tax_prepayments.rs: load monthly data, editable amounts per month, totals. Rewrite tax_investments.rs: capital income entries CRUD, security transactions CRUD, FIFO recalculate button, document upload, year summary card. Read frontend/src/routes/tax/. Run cargo build."
```

**New files:** `tax_income_detail.rs`, `tax_income_form.rs`, `tax_social_detail.rs`, `tax_social_form.rs`, `tax_health_detail.rs`, `tax_health_form.rs`

**Acceptance:** Create income tax return → recalculate → verify credits applied → generate XML.

---

### Phase 18: Reports+Dashboard (1 teammate)

**Goal:** Full reports with all tabs, enhanced dashboard.

**Tasks:**
```
TaskCreate: "P18: Reports -- revenue tab (monthly/quarterly bars)" (status: pending)
TaskCreate: "P18: Reports -- expenses tab (monthly + category breakdown)" (status: pending)
TaskCreate: "P18: Reports -- profit & loss tab (combined)" (status: pending)
TaskCreate: "P18: Reports -- top customers tab (ranked list)" (status: pending)
TaskCreate: "P18: Reports -- tax calendar tab (deadlines)" (status: pending)
TaskCreate: "P18: Reports -- year selector, CSV export buttons" (status: pending)
TaskCreate: "P18: Dashboard -- quick action buttons, chart bars, clickable stats" (status: pending)
TaskCreate: "P18: Build + test gates" (status: pending)
```

**TeamCreate:**
```
  - name: "p18-reports-dashboard"
    files: rust/crates/zfaktury-app/src/views/{reports,dashboard}.rs
    prompt: "Expand reports.rs to 5 tabs: Revenue (ReportService.revenue_report, bar chart using colored divs), Expenses (expense_report + category breakdown), Profit & Loss (profit_loss_report, combined bars), Top Customers (top_customers, ranked table), Tax Calendar (TaxCalendarService, deadline list). Add year selector affecting all tabs. Add CSV export buttons (save file dialog). Enhance dashboard.rs: quick action buttons (New Invoice, New Expense) with click navigation, monthly bar chart (colored div widths), clickable stat cards → navigate to lists. Read frontend/src/routes/reports/ and dashboard. Run cargo build."
```

**Note:** GPUI has no chart library. Use simple colored `div().w(px(value))` bars proportional to values. Functional parity, not pixel-perfect charts.

**Acceptance:** Select year → see revenue/expense data → CSV export works.

---

### Phase 19: Import+UX Polish (2 teammates)

**Goal:** Fakturoid import, expense OCR import, UX polish.

**Tasks:**
```
TaskCreate: "P19: Fakturoid import -- input fields, preview, import with progress" (status: pending)
TaskCreate: "P19: Expense import (new file) -- file picker, OCR, review dialog" (status: pending)
TaskCreate: "P19: Expense review (new file) -- bulk tax review" (status: pending)
TaskCreate: "P19: Toast notifications after save/delete/send operations" (status: pending)
TaskCreate: "P19: Confirm dialogs before all destructive actions" (status: pending)
TaskCreate: "P19: Loading/error/empty states for all views" (status: pending)
TaskCreate: "P19: Keyboard shortcuts (Ctrl+S save, Escape back, Enter on row)" (status: pending)
TaskCreate: "P19: Back navigation button on all detail/form views" (status: pending)
TaskCreate: "P19: Final headless screenshot verification of all routes" (status: pending)
TaskCreate: "P19: Build + clippy + test gates" (status: pending)
```

**TeamCreate:**
```
  - name: "p19-import"
    files: rust/crates/zfaktury-app/src/views/{import_fakturoid,expense_import,expense_review}.rs
    prompt: "Rewrite import_fakturoid.rs: input fields (slug, email, client_id, secret), 'Preview' button (ImportService preview), 'Import' button with progress indicator, result summary (contacts/invoices/expenses/attachments counts). Create expense_import.rs: file picker dialog (cx.open_file_dialog()), upload document, OCR processing (OCRService), review dialog showing extracted data with editable fields, confirm creates expense. Create expense_review.rs: list unreviewed expenses, checkboxes per row, bulk 'Mark reviewed'/'Unmark' buttons. Read frontend/src/routes/import/ and expenses/import/. Run cargo build."

  - name: "p19-ux-polish"
    files: (audit all views, modify as needed)
    prompt: "UX polish across all views: 1) Add toast notifications (success/error) after every save/create/update/delete/send operation. 2) Add confirm dialogs before every delete action and filing operations. 3) Add loading spinner while data loads, error alert on failure, empty state with action button when no data. 4) Add keyboard shortcuts: Ctrl+S in forms = save, Escape = go back, Enter on focused list row = navigate to detail, Tab between form fields. 5) Add 'Back' button (NavigationState::go_back()) to all detail and form views. Run cargo build."
```

**New files:** `expense_import.rs`, `expense_review.rs`

**Acceptance:** Full UX test: every route has loading/error/empty states, every destructive action has confirmation, every save shows toast.

---

### Phase-Specific Quality Gates (Phases 11-19)

After each phase:

```bash
cargo build --workspace
cargo test --workspace
cargo clippy --workspace -- -D warnings
```

After Phase 19 (final):
- Headless screenshot all 47 routes (43 original + 4 new)
- Code review agent on all new/modified view files
- Security review on any file upload / email / external API usage
- Verify every Route in navigation.rs renders a functional view (not StubView)
