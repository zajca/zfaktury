# ZFaktury Rust Rewrite - Agent Orchestrator Prompt

## Mission

You are implementing a complete rewrite of ZFaktury (Czech invoicing app) from Go to Rust. The rewrite preserves all existing features with a new GPUI desktop UI and significantly improved test coverage.

**Working directory:** `rust/` folder in the ZFaktury repository.

**Reference code:** Go source in `internal/`, frontend in `frontend/src/`.

**Implementation RFCs:** `rust/implementation/phase-{1-7}-*.md` - These are your detailed specifications. Follow them meticulously.

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
Phase 0: POC (GPUI + cage + grim) → Verify →
Phase 1: Foundation → Gates →
Phase 2: Persistence → Gates →
Phase 3: Generation → Gates →
Phase 4: External APIs → Gates →
Phase 5: Services → Gates →
Phase 6: GPUI App → Gates →
Phase 7: Polish → Final Gates → Done
```

Phases MUST be sequential (each depends on previous). Within a phase, maximize parallelism via TeamCreate.

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
