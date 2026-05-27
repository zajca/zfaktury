# Flexible Invoice Sequence Format

Date: 2026-05-26
Status: Draft (awaiting user review)

## Problem

The "Číselné řady" (invoice number series) feature currently locks every
sequence into a single hardcoded layout:

```
{prefix}{year}{number:04d}     ->   FV20260001
```

A `format_pattern` column already exists in `invoice_sequences` and is exposed
through the API and a UI text field, but the value is **stored and ignored** --
both `service.FormatPreview` and `repository.InvoiceRepository.GetNextNumber`
build the formatted number with `fmt.Sprintf("%s%d%04d", prefix, year, number)`.
Both call sites carry an explicit `NOTE: format_pattern is not yet implemented`
comment.

This blocks two concrete use cases:

- Continue an existing numbering scheme from a system being migrated -- format
  `77-26-012` (prefix `77`, dash, 2-digit year, dash, 3-digit zero-padded
  counter).
- Use a similar style for a second company -- `42-26-005`.

The user already has invoices with those numbers and wants the next ones to
follow naturally.

## Goals

1. Honour `format_pattern` end-to-end: preview, persistence, and live
   `GetNextNumber` all render through the same engine.
2. Support short year (`YY`), arbitrary separators, and counter width 1-6.
3. UI for non-technical users: a small builder constructs the pattern from
   three controls; an "Advanced" toggle still allows the raw template.
4. 100% backward compatibility for existing sequences (pattern
   `{prefix}{year}{number:04d}` must render to the same string as today).
5. Validate patterns at creation/update so bad templates can never reach the
   number generator.

## Non-goals

- Per-company global uniqueness on `invoices.invoice_number`. The table-level
  `UNIQUE` is global today; that is pre-existing behaviour and out of scope
  here.
- Schema migration. The `format_pattern` column already exists.
- Pattern macros beyond the tokens listed below (no months, no random
  components, no per-locale year formats).
- Bulk renumbering of existing invoices.

## Design

### Token grammar

A single render function evaluates a template against
`(prefix string, year int, number int)`.

| Token                  | Meaning                                                          |
|------------------------|------------------------------------------------------------------|
| `{prefix}`             | prefix verbatim                                                  |
| `{yyyy}`               | 4-digit year (e.g. `2026`). `{year}` is accepted as a silent alias for backward compatibility with the existing schema default. |
| `{yy}`                 | 2-digit year, zero-padded (`year mod 100`)                       |
| `{number}`             | counter, no padding                                              |
| `{number:Nd}`          | counter, zero-padded to width N (N in 1..6; `N` is a placeholder, e.g. `{number:03d}` or `{number:04d}`). Width is a **minimum**: numbers exceeding the width render at full length without truncation (matches Go `fmt.Sprintf("%03d", 1000) == "1000"`). |
| any other characters   | literal (including `-`, `/`, `.`, space, letters)                |

**Not supported:** literal `{` or `}` characters in the template. The parser
treats every `{` as the start of a token. Czech invoice numbers don't use
brace literals in practice; revisit if real demand appears.

**User-facing cheat-sheet** shows only the canonical forms
(`{prefix} {yyyy} {yy} {number} {number:03d} {number:04d}`). The `{year}`
alias is accepted by the parser but kept out of the docs to avoid two ways of
spelling the same thing in new templates.

Examples:

| Pattern                                 | Inputs                       | Output       |
|-----------------------------------------|------------------------------|--------------|
| `{prefix}-{yy}-{number:03d}`            | prefix=`77`, year=2026, n=12 | `77-26-012`  |
| `{prefix}-{yy}-{number:03d}`            | prefix=`42`, year=2026, n=5  | `42-26-005`  |
| `{prefix}{year}{number:04d}` (legacy)   | prefix=`FV`, year=2026, n=1  | `FV20260001` |
| `FV/{year}/{number:03d}`                | prefix=`X`, year=2026, n=7   | `FV/2026/007`|

Note: `{prefix}` is optional in the template -- the literal `FV/...` example
above shows the prefix can be baked into the literal text if the user prefers.
The DB still requires the `prefix` column for `UNIQUE(company_id, prefix, year)`
and for the existing auto-assignment logic in `InvoiceService.Create`, so the
prefix is still stored.

### Validation rules

Enforced in `internal/format.ValidatePattern` and called from
`SequenceService.Create` and `SequenceService.Update`:

1. Trimmed pattern must be non-empty. `ValidatePattern("")` returns an error
   so an empty pattern can never reach `Render`. This is defence-in-depth:
   the `format_pattern` column is `NOT NULL DEFAULT '{prefix}{year}{number:04d}'`
   (see `001_initial_schema.sql:42` and `025_multi_company.sql:147`), and
   `SequenceService.Create` and `GetOrCreateForYear` both fill the default
   when the request omits it (`sequence_svc.go:35-37`, `:173`). An empty
   pattern reaching the renderer would be a code bug, and the validator
   makes that bug loud rather than silent.
2. Pattern must contain exactly one `{number...}` token. Zero would make all
   invoices in the sequence collide; more than one is ambiguous.
3. The width N in `{number:Nd}` must be 1..6. (Six is plenty: 999 999 invoices
   per sequence per year.)
4. Any unrecognised `{...}` token is a validation error.

Errors wrap `domain.ErrInvalidInput`.

### Architecture

```
internal/format/sequence.go        (new)
  Render(pattern, prefix string, year, number int) string
  ValidatePattern(pattern string) error

internal/service/sequence_svc.go   (modified)
  - Create / Update call format.ValidatePattern
  - FormatPreview calls format.Render
  - Drop "not yet implemented" comment

internal/repository/invoice_repo.go (modified)
  - GetNextNumber calls format.Render
  - Drop "not yet implemented" comment
```

The renderer lives in a new package so the repository can use it without
importing the service layer (which would create a cycle).

### Frontend

Single page touched: `frontend/src/routes/settings/sequences/+page.svelte`.

Keep the existing layout (Prefix, Rok, Počáteční číslo, Formát, Náhled) and
make the **Formát** text input the single source of truth for the template.
The user types the whole pattern -- including any separators -- directly into
that field, e.g. `{prefix}-{yy}-{number:03d}`. Any text outside `{...}` is a
literal separator, so dashes, slashes, spaces, or letters all just work.

**Live preview.** A `Náhled:` line below the form re-renders on every
keystroke using the TS port of the renderer, fed by the current `prefix`,
`year`, and `next_number` form fields. Example: typing
`{prefix}-{yy}-{number:03d}` with prefix `77`, year 2026, next 13 shows
`Náhled: 77-26-013`. Invalid templates show `Náhled: --` plus an inline
validation message ("Neplatná šablona: ...").

**Token cheat-sheet below the field** (small muted text, one line):

```
Tokeny: {prefix} {yyyy} {yy} {number} {number:03d} {number:04d}
        ...vše ostatní je oddělovač.
```

`HelpTip topic="prefix-format"` is updated with full examples and the user
can click it for the long form.

**Token insertion buttons (optional polish, low priority).** Small chip
buttons next to the field for `{prefix}`, `{yyyy}`, `{yy}`, `{number:03d}`,
`{number:04d}`. Clicking inserts at the caret. Implemented only if it fits
into the implementation budget; not load-bearing -- the field is fully usable
without them.

**Defaults.** The Create form keeps the current default
`{prefix}{year}{number:04d}` to avoid changing behaviour for users who do
not customise. The user simply edits the field to
`{prefix}-{yy}-{number:03d}` when creating their migration sequence.

**Editing existing sequences** still allows full template edits; the inline
warning "Změna formátu se projeví u nově generovaných čísel; již vystavené
faktury zůstanou beze změny." appears when the pattern field is dirtied.

A new utility `frontend/src/lib/utils/sequence-format.ts` ports the renderer
to TypeScript. The page imports it instead of doing inline string-padding.

**Parity with the Go renderer is enforced through a shared fixture file**:
`internal/format/testdata/render_cases.json` ships a list of
`{pattern, prefix, year, number, want}` objects. Both `internal/format/sequence_test.go`
and `frontend/src/lib/utils/sequence-format.test.ts` load and assert against
this single source of truth. Adding a new edge case means adding one JSON
entry, never two parallel test bodies. The fixture must cover at minimum:
the legacy default, the user's migration target, all four padding variants,
the `{year}` alias, and the overflow case below.

### Backward compatibility

- Default value of the column (`{prefix}{year}{number:04d}`) is left
  untouched in the schema; the renderer produces the same byte-for-byte output
  for that pattern, so any existing sequence keeps its numbering.
- Existing invoices are not touched.
- API shape (`prefix`, `next_number`, `year`, `format_pattern`, `preview`)
  is unchanged.
- A Go test fixture replays the current hardcoded format against the new
  renderer to lock the parity guarantee.

### Help content

`frontend/src/lib/data/help-content.ts -> 'prefix-format'` is updated to
describe the new tokens in Czech, including the `{yy}` and `{number:03d}`
variants and a few worked examples.

### Tests

Backend:

- `internal/format/sequence_test.go` -- table-driven coverage of every token,
  width edge cases, all validation error paths. Must include the overflow
  case: pattern `{number:03d}`, number `1000`, expected output `"1000"`
  (width is a minimum, no truncation).
- `internal/service/sequence_svc_test.go` -- new cases for invalid patterns
  on `Create` / `Update`, plus a parity test asserting `FormatPreview` of the
  legacy default equals `"%s%d%04d"`.
- `internal/repository/invoice_repo_test.go` -- `GetNextNumber` with a
  non-default pattern (`{prefix}-{yy}-{number:03d}`).

Frontend:

- `frontend/src/lib/utils/sequence-format.test.ts` -- mirror of the Go test
  fixtures.
- `frontend/src/routes/settings/sequences/page.test.ts` -- new cases:
  preview reflects the template field on each keystroke, invalid template
  shows fallback preview and inline error, validation error from the API
  surfaces in the form.

## Risks & mitigations

- **Pattern collisions across sequences within a company.** Two different
  patterns could theoretically produce the same string (e.g. prefix `77-26`
  no-separator vs prefix `77` with `-26-` literal). The pre-existing
  `invoices.invoice_number UNIQUE` constraint will reject the second insert
  with a friendly error in the service layer; no new corruption risk.
- **User edits an existing pattern mid-year.** Allowed today; staying allowed.
  Add an inline warning in the edit row: "Změna formátu se projeví u nově
  generovaných čísel; již vystavené faktury zůstanou beze změny."
- **TS / Go renderer drift.** Mitigated by shared fixtures and tests on both
  sides.

## Out of scope follow-ups (separate work items)

- Add `UNIQUE(company_id, invoice_number)` and drop the global UNIQUE on
  `invoices.invoice_number` once multi-company users start needing
  overlapping numbers across companies.
- Optional `{mm}` (zero-padded month) token. Implementing it well requires
  deciding what month is rendered (issue-date month vs. render-time month)
  and most likely a follow-up where sequences are keyed by
  `(company_id, prefix, year, month)` with monthly counter reset --
  significantly more scope than just "add a token". Defer until a real user
  asks.
- Escaping for literal `{` / `}` (`{{` / `}}` style). Not in real Czech
  invoice numbers; revisit only on demand.

## Feedback log

Review at `docs/superpowers/plans/2026-05-26-flexible-sequence-format-feedback.md`
(dated 2026-05-26) was processed:

| # | Item                              | Disposition  | Where addressed                                       |
|---|-----------------------------------|--------------|-------------------------------------------------------|
| 1 | Standardise on `{yyyy}` / `{yy}`  | Adopted with constraint -- `{year}` kept as silent alias to avoid touching the schema `DEFAULT` and existing service strings. `{year2}` dropped. | Token grammar table |
| 2 | Escape `{` / `}`                  | Documented as not supported; YAGNI for Czech invoice numbers. | Token grammar "Not supported" note + Out of scope follow-ups |
| 3 | `{mm}` month token now            | Pushback. Reviewer flagged it as nice-to-have, not a blocker; user did not ask for it; doing it well requires monthly-keyed sequences (counter reset) which is a separate feature. | Out of scope follow-ups |
| 4 | `{number:Nd}` vs `{number:04d}`   | Adopted; clarified that `N` is a placeholder. | Token grammar table |
| 5 | Front/back parity                 | Adopted; shared JSON fixture file specified. | Frontend section |
| 6 | Number overflows padding          | Adopted; width is a minimum, no truncation; explicit test required. | Token grammar table + Tests |
| 7 | Empty-pattern defence             | Adopted; documented the three layers that prevent it and the validator that turns any future bug into a loud error. | Validation rules |
