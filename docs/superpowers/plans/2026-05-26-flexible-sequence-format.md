# Flexible Invoice Sequence Format Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the `format_pattern` column on `invoice_sequences` actually drive how invoice numbers render, so users can use formats like `77-26-012` (`{prefix}-{yy}-{number:03d}`) instead of the hardcoded `FV20260001`.

**Architecture:** Add a small `internal/format` package owning template parsing, validation, and rendering. The service layer (`FormatPreview`) and the repository (`GetNextNumber`) both call it -- no duplicated logic. A TypeScript port lives at `frontend/src/lib/utils/sequence-format.ts` and is exercised by the same JSON test fixtures (`internal/format/testdata/render_cases.json`) as the Go tests, so the two implementations cannot drift.

**Tech Stack:** Go 1.25 (stdlib only — no new deps), SQLite via `modernc.org/sqlite`, Svelte 5 + SvelteKit + TypeScript, Vitest + @testing-library/svelte for frontend tests.

**Spec:** `docs/superpowers/specs/2026-05-26-flexible-sequence-format-design.md`

---

## File Structure

**Created:**
- `internal/format/sequence.go` — `Render` + `ValidatePattern` + token parser
- `internal/format/sequence_test.go` — table-driven tests, JSON fixture loader, all error paths
- `internal/format/testdata/render_cases.json` — shared fixture (Go + TS tests both load it)
- `frontend/src/lib/utils/sequence-format.ts` — TS port of `Render` + `validatePattern`
- `frontend/src/lib/utils/sequence-format.test.ts` — mirrors Go tests via the shared JSON

**Modified:**
- `internal/service/sequence_svc.go` — call `format.ValidatePattern` in Create/Update; rewrite `FormatPreview` to call `format.Render`; drop the "not yet implemented" comment
- `internal/repository/invoice_repo.go` — rewrite `GetNextNumber` to call `format.Render`; drop the "not yet implemented" comment
- `internal/repository/invoice_repo_test.go` — add `TestInvoiceRepository_GetNextNumber_CustomPattern`
- `internal/service/sequence_svc_test.go` — add invalid-pattern cases; assert legacy default still renders byte-identical
- `internal/testutil/testutil.go` — extend `SeedInvoiceSequence` to accept an optional custom pattern (keep current callers compiling)
- `frontend/src/routes/settings/sequences/+page.svelte` — preview now uses the TS renderer; invalid templates show inline error; mid-year-edit warning text; small Czech token cheat-sheet under the Formát field
- `frontend/src/routes/settings/sequences/page.test.ts` — new cases for live preview, invalid pattern UI, edit warning
- `frontend/src/lib/data/help-content.ts` — refresh the `prefix-format` topic with the new tokens and examples

**Not touched:** schema (the `format_pattern` column already exists), API DTOs, all other handlers, routes, dependency wiring.

---

## Task 1: Add the Go renderer package with JSON fixture

**Files:**
- Create: `internal/format/testdata/render_cases.json`
- Create: `internal/format/sequence_test.go`
- Create: `internal/format/sequence.go`

The renderer is plain stdlib. We TDD it against the shared fixture so the same data file later drives the TS tests.

- [ ] **Step 1: Create the shared fixture file**

Create `internal/format/testdata/render_cases.json` with this exact content:

```json
{
  "render_cases": [
    {
      "name": "legacy default",
      "pattern": "{prefix}{year}{number:04d}",
      "prefix": "FV",
      "year": 2026,
      "number": 1,
      "want": "FV20260001"
    },
    {
      "name": "legacy default high number",
      "pattern": "{prefix}{year}{number:04d}",
      "prefix": "ZF",
      "year": 2025,
      "number": 42,
      "want": "ZF20250042"
    },
    {
      "name": "user migration target 77",
      "pattern": "{prefix}-{yy}-{number:03d}",
      "prefix": "77",
      "year": 2026,
      "number": 12,
      "want": "77-26-012"
    },
    {
      "name": "user migration target 42",
      "pattern": "{prefix}-{yy}-{number:03d}",
      "prefix": "42",
      "year": 2026,
      "number": 5,
      "want": "42-26-005"
    },
    {
      "name": "yyyy alias",
      "pattern": "{prefix}-{yyyy}-{number:04d}",
      "prefix": "FV",
      "year": 2026,
      "number": 7,
      "want": "FV-2026-0007"
    },
    {
      "name": "year alias still works",
      "pattern": "FV/{year}/{number:03d}",
      "prefix": "X",
      "year": 2026,
      "number": 7,
      "want": "FV/2026/007"
    },
    {
      "name": "no padding",
      "pattern": "{prefix}-{number}",
      "prefix": "INV",
      "year": 2026,
      "number": 42,
      "want": "INV-42"
    },
    {
      "name": "width 1",
      "pattern": "{number:1d}",
      "prefix": "X",
      "year": 2026,
      "number": 5,
      "want": "5"
    },
    {
      "name": "width 6",
      "pattern": "{number:6d}",
      "prefix": "X",
      "year": 2026,
      "number": 1,
      "want": "000001"
    },
    {
      "name": "overflow does not truncate",
      "pattern": "{number:03d}",
      "prefix": "X",
      "year": 2026,
      "number": 1000,
      "want": "1000"
    },
    {
      "name": "yy below 2000 (year=1999)",
      "pattern": "{yy}",
      "prefix": "X",
      "year": 1999,
      "number": 1,
      "want": "99"
    },
    {
      "name": "yy zero-padded for year=2005",
      "pattern": "{yy}",
      "prefix": "X",
      "year": 2005,
      "number": 1,
      "want": "05"
    },
    {
      "name": "literal slashes and spaces",
      "pattern": "Faktura {prefix} / {yyyy} / {number:04d}",
      "prefix": "FV",
      "year": 2026,
      "number": 12,
      "want": "Faktura FV / 2026 / 0012"
    }
  ],
  "validation_errors": [
    { "name": "empty",                "pattern": "" },
    { "name": "whitespace only",      "pattern": "   " },
    { "name": "missing number token", "pattern": "{prefix}-{yy}" },
    { "name": "two number tokens",    "pattern": "{number:03d}-{number:04d}" },
    { "name": "width zero",           "pattern": "{number:0d}" },
    { "name": "width seven",          "pattern": "{number:7d}" },
    { "name": "unknown token",        "pattern": "{prefix}-{month}-{number:03d}" },
    { "name": "unterminated brace",   "pattern": "{prefix-{number:03d}" },
    { "name": "malformed number arg", "pattern": "{number:xx}" }
  ]
}
```

- [ ] **Step 2: Write the failing Go test**

Create `internal/format/sequence_test.go`:

```go
package format

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

type renderCase struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Prefix  string `json:"prefix"`
	Year    int    `json:"year"`
	Number  int    `json:"number"`
	Want    string `json:"want"`
}

type validationCase struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
}

type fixtureFile struct {
	RenderCases      []renderCase     `json:"render_cases"`
	ValidationErrors []validationCase `json:"validation_errors"`
}

func loadFixture(t *testing.T) fixtureFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "render_cases.json"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	var f fixtureFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	if len(f.RenderCases) == 0 || len(f.ValidationErrors) == 0 {
		t.Fatal("fixture is empty")
	}
	return f
}

func TestRender_Fixtures(t *testing.T) {
	f := loadFixture(t)
	for _, tc := range f.RenderCases {
		t.Run(tc.Name, func(t *testing.T) {
			got := Render(tc.Pattern, tc.Prefix, tc.Year, tc.Number)
			if got != tc.Want {
				t.Errorf("Render(%q, %q, %d, %d) = %q, want %q",
					tc.Pattern, tc.Prefix, tc.Year, tc.Number, got, tc.Want)
			}
		})
	}
}

func TestValidatePattern_ValidFixturesPass(t *testing.T) {
	f := loadFixture(t)
	for _, tc := range f.RenderCases {
		t.Run(tc.Name, func(t *testing.T) {
			if err := ValidatePattern(tc.Pattern); err != nil {
				t.Errorf("ValidatePattern(%q) returned error: %v", tc.Pattern, err)
			}
		})
	}
}

func TestValidatePattern_InvalidFixturesFail(t *testing.T) {
	f := loadFixture(t)
	for _, tc := range f.ValidationErrors {
		t.Run(tc.Name, func(t *testing.T) {
			err := ValidatePattern(tc.Pattern)
			if err == nil {
				t.Fatalf("ValidatePattern(%q) returned nil, want error", tc.Pattern)
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("error does not wrap ErrInvalidInput: %v", err)
			}
		})
	}
}

func TestRender_LegacyParity(t *testing.T) {
	// Locks the byte-for-byte parity with the previous hardcoded format.
	const pattern = "{prefix}{year}{number:04d}"
	cases := []struct {
		prefix string
		year   int
		number int
		want   string
	}{
		{"FV", 2026, 1, "FV20260001"},
		{"ZF", 2025, 42, "ZF20250042"},
		{"DN", 2026, 9999, "DN20269999"},
	}
	for _, c := range cases {
		got := Render(pattern, c.prefix, c.year, c.number)
		if got != c.want {
			t.Errorf("legacy parity broken: Render(%q,%d,%d) = %q, want %q",
				c.prefix, c.year, c.number, got, c.want)
		}
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

```bash
cd /home/coder/Devel/zfaktury
CGO_ENABLED=0 go test ./internal/format/...
```

Expected: compile error — `Render` and `ValidatePattern` are not defined. That's the failing-test state we want before implementation.

- [ ] **Step 4: Implement the renderer**

Create `internal/format/sequence.go`:

```go
// Package format renders and validates invoice-number templates such as
// "{prefix}-{yy}-{number:03d}". The same logic is reused by the service
// layer (for previews) and the repository layer (for GetNextNumber), and a
// TypeScript port at frontend/src/lib/utils/sequence-format.ts is kept in
// lockstep through the shared testdata/render_cases.json fixture.
package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
)

const (
	minNumberWidth = 1
	maxNumberWidth = 6
)

// Render evaluates pattern against (prefix, year, number) and returns the
// formatted invoice number. The caller is responsible for having validated
// the pattern with ValidatePattern; Render on an invalid pattern returns the
// raw pattern unchanged so a programming error is visible rather than silent.
func Render(pattern, prefix string, year, number int) string {
	if ValidatePattern(pattern) != nil {
		return pattern
	}
	var b strings.Builder
	b.Grow(len(pattern) + 8)
	i := 0
	for i < len(pattern) {
		if pattern[i] != '{' {
			b.WriteByte(pattern[i])
			i++
			continue
		}
		end := strings.IndexByte(pattern[i:], '}')
		// ValidatePattern already rejected unterminated braces, so end >= 0.
		token := pattern[i+1 : i+end]
		writeToken(&b, token, prefix, year, number)
		i += end + 1
	}
	return b.String()
}

// ValidatePattern reports whether pattern is a well-formed template that can
// safely drive GetNextNumber. Errors wrap domain.ErrInvalidInput so handlers
// surface them as 422.
func ValidatePattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return fmt.Errorf("format pattern is empty: %w", domain.ErrInvalidInput)
	}
	numberTokens := 0
	i := 0
	for i < len(pattern) {
		if pattern[i] != '{' {
			i++
			continue
		}
		end := strings.IndexByte(pattern[i:], '}')
		if end < 0 {
			return fmt.Errorf("format pattern has unterminated %q: %w", "{", domain.ErrInvalidInput)
		}
		token := pattern[i+1 : i+end]
		kind, err := classifyToken(token)
		if err != nil {
			return err
		}
		if kind == tokenNumber {
			numberTokens++
		}
		i += end + 1
	}
	if numberTokens != 1 {
		return fmt.Errorf("format pattern must contain exactly one {number...} token, found %d: %w", numberTokens, domain.ErrInvalidInput)
	}
	return nil
}

type tokenKind int

const (
	tokenPrefix tokenKind = iota
	tokenYearFull
	tokenYearShort
	tokenNumber
)

func classifyToken(token string) (tokenKind, error) {
	switch token {
	case "prefix":
		return tokenPrefix, nil
	case "yyyy", "year":
		return tokenYearFull, nil
	case "yy":
		return tokenYearShort, nil
	case "number":
		return tokenNumber, nil
	}
	if strings.HasPrefix(token, "number:") {
		width, err := parseNumberWidth(token[len("number:"):])
		if err != nil {
			return 0, err
		}
		_ = width // validated; not needed at classify time
		return tokenNumber, nil
	}
	return 0, fmt.Errorf("unknown format token %q: %w", "{"+token+"}", domain.ErrInvalidInput)
}

// parseNumberWidth accepts strings shaped like "03d" or "4d" and returns the
// numeric width when it falls in [minNumberWidth, maxNumberWidth].
func parseNumberWidth(spec string) (int, error) {
	if !strings.HasSuffix(spec, "d") {
		return 0, fmt.Errorf("number width spec %q must end in 'd': %w", spec, domain.ErrInvalidInput)
	}
	digits := strings.TrimSuffix(spec, "d")
	if digits == "" {
		return 0, fmt.Errorf("number width spec is empty: %w", domain.ErrInvalidInput)
	}
	width, err := strconv.Atoi(digits)
	if err != nil {
		return 0, fmt.Errorf("number width %q is not numeric: %w", digits, domain.ErrInvalidInput)
	}
	if width < minNumberWidth || width > maxNumberWidth {
		return 0, fmt.Errorf("number width %d must be in %d..%d: %w", width, minNumberWidth, maxNumberWidth, domain.ErrInvalidInput)
	}
	return width, nil
}

func writeToken(b *strings.Builder, token, prefix string, year, number int) {
	switch token {
	case "prefix":
		b.WriteString(prefix)
	case "yyyy", "year":
		fmt.Fprintf(b, "%04d", year)
	case "yy":
		fmt.Fprintf(b, "%02d", year%100)
	case "number":
		fmt.Fprintf(b, "%d", number)
	default:
		// Must be "number:Nd" (ValidatePattern guarantees it).
		width, _ := parseNumberWidth(token[len("number:"):])
		fmt.Fprintf(b, "%0*d", width, number)
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd /home/coder/Devel/zfaktury
CGO_ENABLED=0 go test ./internal/format/... -v
```

Expected: all subtests in `TestRender_Fixtures`, `TestValidatePattern_ValidFixturesPass`, `TestValidatePattern_InvalidFixturesFail`, `TestRender_LegacyParity` pass.

- [ ] **Step 6: Commit**

```bash
cd /home/coder/Devel/zfaktury
git add internal/format/
git commit -m "Add format package for invoice number template rendering"
```

---

## Task 2: Wire renderer into SequenceService

**Files:**
- Modify: `internal/service/sequence_svc.go` (drop hardcoded format, call format.Render, add validation)
- Modify: `internal/service/sequence_svc_test.go` (add invalid-pattern tests + legacy parity assertion)

- [ ] **Step 1: Add the failing service tests**

Append to `internal/service/sequence_svc_test.go`:

```go
func TestSequenceService_Create_InvalidPattern(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		pattern string
	}{
		{"missing number token", "{prefix}-{yy}"},
		{"two number tokens", "{number:03d}-{number:04d}"},
		{"width zero", "{number:0d}"},
		{"width seven", "{number:7d}"},
		{"unknown token", "{prefix}-{month}-{number:03d}"},
		{"unterminated brace", "{prefix-{number:03d}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026, FormatPattern: tc.pattern}
			err := svc.Create(ctx, 1, seq)
			if err == nil {
				t.Fatalf("Create() with %q returned nil, want error", tc.pattern)
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("error does not wrap ErrInvalidInput: %v", err)
			}
		})
	}
}

func TestSequenceService_Update_InvalidPattern(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026, NextNumber: 1}
	if err := svc.Create(ctx, 1, seq); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	seq.FormatPattern = "{prefix}-{yy}" // missing number
	if err := svc.Update(ctx, 1, seq); err == nil {
		t.Fatal("Update() with invalid pattern returned nil")
	}
}

func TestFormatPreview_LegacyParity(t *testing.T) {
	// Locks: legacy default pattern must render exactly like the old hardcoded
	// fmt.Sprintf("%s%d%04d", ...).
	cases := []*domain.InvoiceSequence{
		{Prefix: "FV", Year: 2026, NextNumber: 1, FormatPattern: "{prefix}{year}{number:04d}"},
		{Prefix: "ZF", Year: 2025, NextNumber: 42, FormatPattern: "{prefix}{year}{number:04d}"},
		{Prefix: "DN", Year: 2026, NextNumber: 100, FormatPattern: "{prefix}{year}{number:04d}"},
	}
	for _, seq := range cases {
		want := fmt.Sprintf("%s%d%04d", seq.Prefix, seq.Year, seq.NextNumber)
		got := FormatPreview(seq)
		if got != want {
			t.Errorf("FormatPreview(%+v) = %q, want %q", seq, got, want)
		}
	}
}

func TestFormatPreview_CustomPattern(t *testing.T) {
	seq := &domain.InvoiceSequence{
		Prefix: "77", Year: 2026, NextNumber: 12,
		FormatPattern: "{prefix}-{yy}-{number:03d}",
	}
	if got := FormatPreview(seq); got != "77-26-012" {
		t.Errorf("FormatPreview = %q, want 77-26-012", got)
	}
}
```

Add the missing `errors` and `fmt` imports at the top if they are not already present (the existing test file already imports `strings`; check the import block and extend it).

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /home/coder/Devel/zfaktury
CGO_ENABLED=0 go test ./internal/service/... -run 'TestSequenceService_Create_InvalidPattern|TestSequenceService_Update_InvalidPattern|TestFormatPreview' -v
```

Expected: the four new tests fail. `TestSequenceService_Create_InvalidPattern` and `TestSequenceService_Update_InvalidPattern` will pass for the "missing number token" case incidentally if you forget — re-read the failures carefully. `TestFormatPreview_CustomPattern` will fail because `FormatPreview` still hardcodes the legacy format.

- [ ] **Step 3: Edit sequence_svc.go to wire in the renderer**

In `internal/service/sequence_svc.go`:

a) Add the new import inside the existing import block:

```go
"github.com/zajca/zfaktury/internal/format"
```

b) In `Create` (around line 35-37), keep the empty-pattern default but add a validation call right after it. Replace:

```go
	if seq.FormatPattern == "" {
		seq.FormatPattern = "{prefix}{year}{number:04d}"
	}
```

with:

```go
	if seq.FormatPattern == "" {
		seq.FormatPattern = "{prefix}{year}{number:04d}"
	}
	if err := format.ValidatePattern(seq.FormatPattern); err != nil {
		return fmt.Errorf("validating format pattern: %w", err)
	}
```

c) In `Update`, just before the `Fetch existing for audit logging` block, add:

```go
	if err := format.ValidatePattern(seq.FormatPattern); err != nil {
		return fmt.Errorf("validating format pattern: %w", err)
	}
```

d) Replace `FormatPreview` (lines ~187-192) entirely with:

```go
// FormatPreview returns the formatted invoice number the sequence would
// produce for its current next_number. Backed by internal/format.Render so
// preview, persistence, and GetNextNumber stay in lockstep.
func FormatPreview(seq *domain.InvoiceSequence) string {
	return format.Render(seq.FormatPattern, seq.Prefix, seq.Year, seq.NextNumber)
}
```

(Drop the two-line `NOTE: format_pattern is not yet implemented` comment.)

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd /home/coder/Devel/zfaktury
CGO_ENABLED=0 go test ./internal/service/... -v
```

Expected: all sequence service tests pass, including the new invalid-pattern and parity tests, plus the existing ones.

- [ ] **Step 5: Commit**

```bash
cd /home/coder/Devel/zfaktury
git add internal/service/sequence_svc.go internal/service/sequence_svc_test.go
git commit -m "Wire format renderer into SequenceService"
```

---

## Task 3: Wire renderer into InvoiceRepository.GetNextNumber

**Files:**
- Modify: `internal/repository/invoice_repo.go` (rewrite GetNextNumber to use format.Render)
- Modify: `internal/testutil/testutil.go` (extend SeedInvoiceSequence with optional pattern)
- Modify: `internal/repository/invoice_repo_test.go` (add custom-pattern test)

- [ ] **Step 1: Extend the test helper to accept a custom pattern**

In `internal/testutil/testutil.go`, replace the `SeedInvoiceSequence` function (lines 295-312) with:

```go
// SeedInvoiceSequence inserts an invoice sequence into the database for the
// given company. Pass companyID=1 for the default company seeded by NewTestDB.
// The format pattern defaults to the legacy {prefix}{year}{number:04d} format
// when DefaultFormat is empty.
func SeedInvoiceSequence(t *testing.T, db *sql.DB, companyID int64, prefix string, year int) int64 {
	t.Helper()
	return SeedInvoiceSequenceWithPattern(t, db, companyID, prefix, year, "{prefix}{year}{number:04d}")
}

// SeedInvoiceSequenceWithPattern is like SeedInvoiceSequence but lets the
// caller pick the format pattern. Use this to cover non-default templates in
// tests.
func SeedInvoiceSequenceWithPattern(t *testing.T, db *sql.DB, companyID int64, prefix string, year int, pattern string) int64 {
	t.Helper()

	result, err := db.ExecContext(context.Background(), `
		INSERT INTO invoice_sequences (company_id, prefix, next_number, year, format_pattern)
		VALUES (?, ?, 1, ?, ?)`, companyID, prefix, year, pattern)
	if err != nil {
		t.Fatalf("seeding invoice sequence: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("getting sequence id: %v", err)
	}
	return id
}
```

- [ ] **Step 2: Write the failing repo test**

Append to `internal/repository/invoice_repo_test.go`:

```go
func TestInvoiceRepository_GetNextNumber_CustomPattern(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	seqID := testutil.SeedInvoiceSequenceWithPattern(t, db, 1, "77", 2026, "{prefix}-{yy}-{number:03d}")

	num1, err := repo.GetNextNumber(ctx, 1, seqID)
	if err != nil {
		t.Fatalf("GetNextNumber() error: %v", err)
	}
	if num1 != "77-26-001" {
		t.Errorf("first number = %q, want 77-26-001", num1)
	}

	num2, err := repo.GetNextNumber(ctx, 1, seqID)
	if err != nil {
		t.Fatalf("GetNextNumber() second call error: %v", err)
	}
	if num2 != "77-26-002" {
		t.Errorf("second number = %q, want 77-26-002", num2)
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

```bash
cd /home/coder/Devel/zfaktury
CGO_ENABLED=0 go test ./internal/repository/... -run 'TestInvoiceRepository_GetNextNumber_CustomPattern' -v
```

Expected: FAIL. `num1` will be something like `"77202620001"` (hardcoded format) instead of `"77-26-001"`.

- [ ] **Step 4: Rewrite GetNextNumber to use the renderer**

In `internal/repository/invoice_repo.go`:

a) Add the import:

```go
"github.com/zajca/zfaktury/internal/format"
```

b) Replace the body of `GetNextNumber` (lines ~553-586) with:

```go
// GetNextNumber atomically increments the sequence counter and returns the
// formatted invoice number, scoped to the given company. Rendering goes
// through internal/format.Render so the format_pattern column is the single
// source of truth.
func (r *InvoiceRepository) GetNextNumber(ctx context.Context, companyID, sequenceID int64) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("beginning transaction for next number: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var seq domain.InvoiceSequence
	err = tx.QueryRowContext(ctx, `
		SELECT id, prefix, next_number, year, format_pattern
		FROM invoice_sequences WHERE id = ? AND company_id = ?`, sequenceID, companyID,
	).Scan(&seq.ID, &seq.Prefix, &seq.NextNumber, &seq.Year, &seq.FormatPattern)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("invoice sequence %d not found: %w", sequenceID, domain.ErrNotFound)
		}
		return "", fmt.Errorf("querying invoice sequence %d: %w", sequenceID, err)
	}

	number := format.Render(seq.FormatPattern, seq.Prefix, seq.Year, seq.NextNumber)

	_, err = tx.ExecContext(ctx, `
		UPDATE invoice_sequences SET next_number = next_number + 1 WHERE id = ? AND company_id = ?`, sequenceID, companyID)
	if err != nil {
		return "", fmt.Errorf("incrementing sequence %d: %w", sequenceID, err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("committing sequence increment: %w", err)
	}
	return number, nil
}
```

(The only behavioural change is the `format.Render` call replacing the hardcoded `fmt.Sprintf`. The "NOTE: format_pattern is not yet implemented" comment is gone.)

- [ ] **Step 5: Run the full repo+service test suites**

```bash
cd /home/coder/Devel/zfaktury
CGO_ENABLED=0 go test ./internal/repository/... ./internal/service/... -v
```

Expected: all tests pass — both the new custom-pattern test and the existing `TestInvoiceRepository_GetNextNumber` (which uses the legacy default and asserts `FV20260001`, validating parity).

- [ ] **Step 6: Commit**

```bash
cd /home/coder/Devel/zfaktury
git add internal/repository/invoice_repo.go internal/repository/invoice_repo_test.go internal/testutil/testutil.go
git commit -m "Render invoice numbers through format package in GetNextNumber"
```

---

## Task 4: Run the full backend build + test sweep

Catch anything the previous tasks missed (CLI commands, importer/recurring, fakturoid import).

- [ ] **Step 1: Full build**

```bash
cd /home/coder/Devel/zfaktury
CGO_ENABLED=0 go build ./...
```

Expected: no errors.

- [ ] **Step 2: Full test suite**

```bash
cd /home/coder/Devel/zfaktury
CGO_ENABLED=0 go test ./...
```

Expected: all packages pass. If any test that builds invoice numbers manually relied on the old hardcoded format and now sees a custom pattern, fix it by seeding the legacy pattern explicitly or by switching to `SeedInvoiceSequenceWithPattern`.

- [ ] **Step 3: If anything fails, fix and re-run**

Common breakage: a test seeds a sequence and then asserts on the formatted invoice number. The default pattern is still legacy so this should be fine, but if a test was reading `format_pattern` from a row inserted by ad-hoc SQL without the default, the column might be empty. Use `SeedInvoiceSequenceWithPattern` to be explicit.

- [ ] **Step 4: Commit (only if Step 3 needed fixes)**

```bash
cd /home/coder/Devel/zfaktury
git add -A
git commit -m "Fix tests broken by format renderer wiring"
```

---

## Task 5: Add the TypeScript renderer port

**Files:**
- Create: `frontend/src/lib/utils/sequence-format.ts`
- Create: `frontend/src/lib/utils/sequence-format.test.ts`

The TS port mirrors the Go logic exactly and is exercised against the same JSON fixture, so the two implementations cannot drift.

- [ ] **Step 1: Write the failing TS tests**

Create `frontend/src/lib/utils/sequence-format.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { renderSequence, validateSequencePattern } from './sequence-format';

type RenderCase = {
	name: string;
	pattern: string;
	prefix: string;
	year: number;
	number: number;
	want: string;
};

type ValidationCase = {
	name: string;
	pattern: string;
};

type Fixture = {
	render_cases: RenderCase[];
	validation_errors: ValidationCase[];
};

// Repo-root-relative path: vitest runs with cwd = frontend/.
const fixturePath = join(
	__dirname,
	'..',
	'..',
	'..',
	'..',
	'internal',
	'format',
	'testdata',
	'render_cases.json'
);

const fixture: Fixture = JSON.parse(readFileSync(fixturePath, 'utf8'));

describe('renderSequence (parity with Go)', () => {
	for (const tc of fixture.render_cases) {
		it(tc.name, () => {
			expect(renderSequence(tc.pattern, tc.prefix, tc.year, tc.number)).toBe(tc.want);
		});
	}
});

describe('validateSequencePattern (parity with Go)', () => {
	for (const tc of fixture.render_cases) {
		it(`valid: ${tc.name}`, () => {
			expect(validateSequencePattern(tc.pattern)).toBeNull();
		});
	}
	for (const tc of fixture.validation_errors) {
		it(`invalid: ${tc.name}`, () => {
			const err = validateSequencePattern(tc.pattern);
			expect(err).not.toBeNull();
			expect(typeof err).toBe('string');
		});
	}
});
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /home/coder/Devel/zfaktury/frontend
npm test -- src/lib/utils/sequence-format.test.ts
```

Expected: FAIL — module `./sequence-format` does not exist.

- [ ] **Step 3: Implement the TS renderer**

Create `frontend/src/lib/utils/sequence-format.ts`:

```typescript
/**
 * Render and validate invoice number templates such as
 * "{prefix}-{yy}-{number:03d}". This is a TypeScript port of
 * internal/format/sequence.go and is exercised against the same fixture
 * (internal/format/testdata/render_cases.json) so the two implementations
 * cannot drift.
 */

const MIN_NUMBER_WIDTH = 1;
const MAX_NUMBER_WIDTH = 6;

const LEGACY_DEFAULT_PATTERN = '{prefix}{year}{number:04d}';

export { LEGACY_DEFAULT_PATTERN };

/**
 * Render pattern against (prefix, year, number). Returns the formatted
 * invoice number, or the raw pattern unchanged when the template is invalid
 * so a programming error in callers stays visible.
 */
export function renderSequence(pattern: string, prefix: string, year: number, number: number): string {
	if (validateSequencePattern(pattern) !== null) {
		return pattern;
	}
	let out = '';
	let i = 0;
	while (i < pattern.length) {
		if (pattern[i] !== '{') {
			out += pattern[i];
			i++;
			continue;
		}
		const end = pattern.indexOf('}', i);
		// validate() already rejected unterminated braces.
		const token = pattern.substring(i + 1, end);
		out += renderToken(token, prefix, year, number);
		i = end + 1;
	}
	return out;
}

/**
 * Validate pattern. Returns null when valid, or a human-readable error
 * string (English, matching the Go errors closely) when not. The Svelte
 * page translates the error into Czech for display.
 */
export function validateSequencePattern(pattern: string): string | null {
	if (pattern.trim() === '') {
		return 'format pattern is empty';
	}
	let numberTokens = 0;
	let i = 0;
	while (i < pattern.length) {
		if (pattern[i] !== '{') {
			i++;
			continue;
		}
		const end = pattern.indexOf('}', i);
		if (end < 0) {
			return 'format pattern has unterminated "{"';
		}
		const token = pattern.substring(i + 1, end);
		const classification = classifyToken(token);
		if (classification.error !== null) {
			return classification.error;
		}
		if (classification.isNumber) {
			numberTokens++;
		}
		i = end + 1;
	}
	if (numberTokens !== 1) {
		return `format pattern must contain exactly one {number...} token, found ${numberTokens}`;
	}
	return null;
}

type Classification = { isNumber: boolean; error: string | null };

function classifyToken(token: string): Classification {
	switch (token) {
		case 'prefix':
			return { isNumber: false, error: null };
		case 'yyyy':
		case 'year':
			return { isNumber: false, error: null };
		case 'yy':
			return { isNumber: false, error: null };
		case 'number':
			return { isNumber: true, error: null };
	}
	if (token.startsWith('number:')) {
		const widthErr = parseNumberWidth(token.substring('number:'.length));
		if (widthErr.error !== null) {
			return { isNumber: false, error: widthErr.error };
		}
		return { isNumber: true, error: null };
	}
	return { isNumber: false, error: `unknown format token "{${token}}"` };
}

function parseNumberWidth(spec: string): { width: number; error: string | null } {
	if (!spec.endsWith('d')) {
		return { width: 0, error: `number width spec "${spec}" must end in 'd'` };
	}
	const digits = spec.substring(0, spec.length - 1);
	if (digits === '') {
		return { width: 0, error: 'number width spec is empty' };
	}
	if (!/^\d+$/.test(digits)) {
		return { width: 0, error: `number width "${digits}" is not numeric` };
	}
	const width = parseInt(digits, 10);
	if (width < MIN_NUMBER_WIDTH || width > MAX_NUMBER_WIDTH) {
		return { width: 0, error: `number width ${width} must be in ${MIN_NUMBER_WIDTH}..${MAX_NUMBER_WIDTH}` };
	}
	return { width, error: null };
}

function renderToken(token: string, prefix: string, year: number, number: number): string {
	switch (token) {
		case 'prefix':
			return prefix;
		case 'yyyy':
		case 'year':
			return String(year).padStart(4, '0');
		case 'yy':
			return String(year % 100).padStart(2, '0');
		case 'number':
			return String(number);
	}
	// Must be "number:Nd" -- validation already ensured it.
	const { width } = parseNumberWidth(token.substring('number:'.length));
	return String(number).padStart(width, '0');
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd /home/coder/Devel/zfaktury/frontend
npm test -- src/lib/utils/sequence-format.test.ts
```

Expected: every `render_cases` entry and every `validation_errors` entry runs as a subtest and passes.

- [ ] **Step 5: Commit**

```bash
cd /home/coder/Devel/zfaktury
git add frontend/src/lib/utils/sequence-format.ts frontend/src/lib/utils/sequence-format.test.ts
git commit -m "Add TypeScript port of sequence format renderer"
```

---

## Task 6: Update the sequences page to use the new renderer + live preview

**Files:**
- Modify: `frontend/src/routes/settings/sequences/+page.svelte`

The page currently computes the preview inline with `padStart(4, '0')` — replace that with `renderSequence`. Add live-preview-while-typing on the Formát field, an inline validation message for bad templates, a small token cheat-sheet line, and an edit-time warning when the pattern of an existing sequence changes.

- [ ] **Step 1: Open the file**

Re-read `frontend/src/routes/settings/sequences/+page.svelte` so the surgical edits below land on the right lines.

- [ ] **Step 2: Add the import and helpers**

In the `<script lang="ts">` block, just below the existing `import` statements, add:

```typescript
import { renderSequence, validateSequencePattern } from '$lib/utils/sequence-format';
```

- [ ] **Step 3: Replace the inline preview computation**

Find the existing `createPreview` derived value near the bottom of the script block:

```typescript
let createPreview = $derived(
	`${createPrefix}${createYear}${String(createNextNumber).padStart(4, '0')}`
);
```

Replace it with a renderer-backed derived value plus a validation error derived value:

```typescript
let createPatternError = $derived(validateSequencePattern(createFormatPattern));
let createPreview = $derived(
	createPatternError === null
		? renderSequence(createFormatPattern, createPrefix, createYear, createNextNumber)
		: '--'
);
```

- [ ] **Step 4: Show the cheat-sheet and validation error in the create form**

Find the create form block (the `{#if showCreateForm}` section). Inside the `<div>` containing the Formát input (the fourth grid cell), replace the existing `<p class="...">` (the help text under the prefix field, if any) only inside the Formát cell. Specifically, after the Formát `<input>` element, add directly below it:

```svelte
<p class="mt-1 text-xs text-muted">
	Tokeny: <code>{'{prefix}'}</code> <code>{'{yyyy}'}</code> <code>{'{yy}'}</code>
	<code>{'{number}'}</code> <code>{'{number:03d}'}</code> <code>{'{number:04d}'}</code>
	-- vše ostatní je oddělovač.
</p>
{#if createPatternError !== null}
	<p class="mt-1 text-xs text-danger" role="alert">Neplatná šablona: {createPatternError}</p>
{/if}
```

Then in the Náhled line near `Náhled: <span ...>`, leave the `{createPreview}` binding — it's already wired to the new derived value above and will show `--` for invalid templates.

Update the submit button's `disabled` to also block invalid templates:

```svelte
<Button
	type="submit"
	variant="primary"
	disabled={creating || !createPrefix || !createYear || createPatternError !== null}
>
```

- [ ] **Step 5: Add edit-time warning + pattern editing for existing sequences**

The current edit row only lets the user edit `next_number`. The spec allows editing the pattern too, with a warning. Update the edit state at the top of the script block, replacing:

```typescript
let editingId = $state<number | null>(null);
let editNextNumber = $state(1);
```

with:

```typescript
let editingId = $state<number | null>(null);
let editNextNumber = $state(1);
let editFormatPattern = $state('{prefix}{year}{number:04d}');
let editOriginalPattern = $state('{prefix}{year}{number:04d}');
let editPatternError = $derived(validateSequencePattern(editFormatPattern));
let editPatternDirty = $derived(editFormatPattern !== editOriginalPattern);
```

Update `startEdit`:

```typescript
function startEdit(seq: InvoiceSequence) {
	editingId = seq.id;
	editNextNumber = seq.next_number;
	editFormatPattern = seq.format_pattern;
	editOriginalPattern = seq.format_pattern;
}
```

Update `handleUpdate` to pass the edited pattern:

```typescript
async function handleUpdate(seq: InvoiceSequence) {
	saving = true;
	try {
		await sequencesApi.update(seq.id, {
			prefix: seq.prefix,
			year: seq.year,
			next_number: editNextNumber,
			format_pattern: editFormatPattern
		});
		editingId = null;
		await loadSequences();
	} catch (e) {
		toastError(e instanceof Error ? e.message : 'Nepodařilo se uložit změny');
	} finally {
		saving = false;
	}
}
```

In the edit row inside the table (the `{#if editingId === seq.id}` block under the "Další číslo" cell), keep the existing `<input type="number" bind:value={editNextNumber} ...>` and add directly below it (still inside the same `<td>` or a new cell — the cleanest is a small block under the number input):

```svelte
<input
	type="text"
	bind:value={editFormatPattern}
	class="mt-1 w-64 rounded-lg border border-border bg-surface px-2 py-1 font-mono text-xs text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
	aria-label="Formát"
/>
{#if editPatternError !== null}
	<p class="mt-1 text-xs text-danger" role="alert">Neplatná šablona: {editPatternError}</p>
{:else if editPatternDirty}
	<p class="mt-1 text-xs text-warning">
		Změna formátu se projeví u nově generovaných čísel; již vystavené faktury zůstanou beze změny.
	</p>
{/if}
```

Update the edit row's Save button's `disabled` to:

```svelte
disabled={saving || editPatternError !== null}
```

- [ ] **Step 6: Verify the page still type-checks**

```bash
cd /home/coder/Devel/zfaktury/frontend
npm run check
```

Expected: no errors.

- [ ] **Step 7: Manual smoke test in the dev server (only if convenient)**

If `make dev` is already running, open `http://localhost:5173/settings/sequences` and verify that typing `{prefix}-{yy}-{number:03d}` with prefix `77`, year `2026`, počáteční číslo `13` shows `Náhled: 77-26-013`. Typing `{prefix}-{yy}` (no number token) shows `Náhled: --` and the red "Neplatná šablona" line.

If the dev server isn't running, skip — the tests in Task 7 will cover the behaviour.

- [ ] **Step 8: Commit**

```bash
cd /home/coder/Devel/zfaktury
git add frontend/src/routes/settings/sequences/+page.svelte
git commit -m "Live preview + pattern editing on sequences settings page"
```

---

## Task 7: Page tests for live preview + invalid pattern + edit warning

**Files:**
- Modify: `frontend/src/routes/settings/sequences/page.test.ts`

- [ ] **Step 1: Write the failing tests**

Append to `frontend/src/routes/settings/sequences/page.test.ts`:

```typescript
describe('Pattern field behaviour', () => {
	it('live preview updates when the pattern field changes', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('FV')).toBeInTheDocument();
		});

		// Open the create form.
		const novaRada = screen.getByRole('button', { name: /Nová řada/i });
		await fireEvent.click(novaRada);

		const prefixInput = screen.getByLabelText(/Prefix/i) as HTMLInputElement;
		const yearInput = screen.getByLabelText(/^Rok/i) as HTMLInputElement;
		const nextInput = screen.getByLabelText(/Počáteční číslo/i) as HTMLInputElement;
		const formatInput = screen.getByLabelText(/Formát/i) as HTMLInputElement;

		await fireEvent.input(prefixInput, { target: { value: '77' } });
		await fireEvent.input(yearInput, { target: { value: '2026' } });
		await fireEvent.input(nextInput, { target: { value: '13' } });
		await fireEvent.input(formatInput, { target: { value: '{prefix}-{yy}-{number:03d}' } });

		await waitFor(() => {
			expect(screen.getByText('77-26-013')).toBeInTheDocument();
		});
	});

	it('invalid pattern shows an inline error and disables submit', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('FV')).toBeInTheDocument();
		});

		const novaRada = screen.getByRole('button', { name: /Nová řada/i });
		await fireEvent.click(novaRada);

		const formatInput = screen.getByLabelText(/Formát/i) as HTMLInputElement;
		await fireEvent.input(formatInput, { target: { value: '{prefix}-{yy}' } });

		await waitFor(() => {
			expect(screen.getByText(/Neplatná šablona/i)).toBeInTheDocument();
		});

		const submitBtn = screen.getByRole('button', { name: /Vytvořit/i }) as HTMLButtonElement;
		expect(submitBtn.disabled).toBe(true);
	});

	it('editing an existing sequence shows the mid-year change warning', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('FV')).toBeInTheDocument();
		});

		// Click "Upravit" on the first row.
		const upravitButtons = screen.getAllByRole('button', { name: /Upravit/i });
		await fireEvent.click(upravitButtons[0]);

		// The edit row exposes a Formát text input pre-filled with the existing pattern.
		const editFormatInput = screen.getByLabelText('Formát') as HTMLInputElement;
		expect(editFormatInput.value).toBe('{prefix}{year}{number:04d}');

		await fireEvent.input(editFormatInput, {
			target: { value: '{prefix}-{yy}-{number:03d}' }
		});

		await waitFor(() => {
			expect(
				screen.getByText(/Změna formátu se projeví u nově generovaných čísel/i)
			).toBeInTheDocument();
		});
	});
});
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /home/coder/Devel/zfaktury/frontend
npm test -- src/routes/settings/sequences/page.test.ts
```

Expected: at first run the three new cases fail because the edit-row `aria-label="Formát"` and inline error / warning copy did not exist before Task 6 — but Task 6 added them. They should now pass straight away. If they don't, the failure messages indicate which selector or behaviour to align.

- [ ] **Step 3: Fix selector mismatches if any**

If a test fails because two elements share the label `Formát` (create form and edit row), tighten the test by scoping with `within(...)` on the relevant container, or by using a more specific accessible name. Common fix:

```typescript
import { within } from '@testing-library/svelte';
// ...
const editRow = screen.getByRole('row', { name: /FV/i });
const editFormatInput = within(editRow).getByLabelText('Formát');
```

Apply only what's needed.

- [ ] **Step 4: Run all page tests to confirm no regressions**

```bash
cd /home/coder/Devel/zfaktury/frontend
npm test -- src/routes/settings/sequences/page.test.ts
```

Expected: all tests pass, including the existing ones.

- [ ] **Step 5: Commit**

```bash
cd /home/coder/Devel/zfaktury
git add frontend/src/routes/settings/sequences/page.test.ts
git commit -m "Tests: live preview, invalid pattern, mid-year edit warning"
```

---

## Task 8: Refresh the Czech help content for the prefix-format topic

**Files:**
- Modify: `frontend/src/lib/data/help-content.ts`

The existing `prefix-format` topic still mentions `{prefix}{year}-{number:4}` -- a syntax the new parser doesn't accept. Replace the entry's `simple` text with one that uses the real tokens and lists the user's migration target as an example.

- [ ] **Step 1: Update the topic**

In `frontend/src/lib/data/help-content.ts`, replace the existing `'prefix-format'` block (around lines 247-253) with:

```typescript
'prefix-format': {
    title: 'Prefix a formát číselné řady',
    simple:
        'Prefix je text na začátku čísla faktury (např. "FV" pro fakturu vydanou nebo "77" pro číselnou řadu převzatou z jiného systému). Formát určuje, jak vypadá celé číslo. Dostupné tokeny:\n\n' +
        '• {prefix} -- vloží prefix\n' +
        '• {yyyy} -- 4místný rok (např. 2026)\n' +
        '• {yy} -- 2místný rok (např. 26)\n' +
        '• {number} -- pořadové číslo bez doplňování nul\n' +
        '• {number:03d} -- pořadové číslo doplněné nulami na 3 místa (např. 012)\n' +
        '• {number:04d} -- pořadové číslo na 4 místa (např. 0012)\n\n' +
        'Vše ostatní (pomlčky, lomítka, mezery) se zapíše doslovně. Příklady:\n\n' +
        '• {prefix}{year}{number:04d} → FV20260001\n' +
        '• {prefix}-{yy}-{number:03d} → 77-26-012\n' +
        '• FV/{yyyy}/{number:04d} → FV/2026/0001\n\n' +
        'Číslování se resetuje vždy na začátku každého roku, takže první faktura nového roku bude opět 001.',
    legal:
        'Formát číselné řady není zákonem předepsán. Zákon č. 235/2004 Sb. v § 29 vyžaduje pouze to, aby pořadové číslo bylo jednoznačné v rámci číselné řady.\n\nDoporučuje se uvádět rok (např. 2024-001 nebo 24-001) pro snazší orientaci a průkaznost při daňové kontrole. Prefix pomáhá rozlišit typ dokladu (faktury vydané, přijaté, dobropisy atd.).'
},
```

- [ ] **Step 2: Verify type-check still passes**

```bash
cd /home/coder/Devel/zfaktury/frontend
npm run check
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd /home/coder/Devel/zfaktury
git add frontend/src/lib/data/help-content.ts
git commit -m "Refresh Czech help content for invoice number format"
```

---

## Task 9: Final verification

- [ ] **Step 1: Backend full sweep**

```bash
cd /home/coder/Devel/zfaktury
CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
```

Expected: all packages build and all tests pass.

- [ ] **Step 2: Frontend full sweep**

```bash
cd /home/coder/Devel/zfaktury/frontend
npm run check && npm test
```

Expected: type-check clean, all 276+ tests pass.

- [ ] **Step 3: Verify the two `NOTE: not yet implemented` comments are gone**

```bash
cd /home/coder/Devel/zfaktury
grep -rn "format_pattern is not yet implemented" internal/ || echo "clean"
```

Expected: `clean`.

- [ ] **Step 4: Manual end-to-end check (optional, recommended)**

With `make dev` running:

1. Open `http://localhost:5173/settings/sequences`.
2. Click "Nová řada".
3. Set prefix=`77`, rok=2026, počáteční číslo=13, formát=`{prefix}-{yy}-{number:03d}`.
4. Confirm Náhled shows `77-26-013`.
5. Click "Vytvořit".
6. Create a new invoice (via `/invoices/new`) and pick the `77` sequence year=2026 by leaving auto-assignment off, or rely on auto-assignment with a contact that triggers prefix `FV`. The simplest end-to-end check is via API:

```bash
curl -s -X POST http://localhost:8080/api/v1/companies/1/invoices \
  -H 'Content-Type: application/json' \
  -d '{
    "sequence_id": <id of the 77 sequence you just created>,
    "type": "regular",
    "issue_date": "2026-05-26",
    "due_date": "2026-06-09",
    "customer_id": 1,
    "items": [{"description":"Test","quantity":100,"unit":"ks","unit_price":100000,"vat_rate_percent":21}]
  }' | jq .invoice_number
```

Expected output: `"77-26-013"`.

If the manual check is skipped, the repository test from Task 3 covers the same path.

---

## Self-review

**Spec coverage check:**

| Spec section | Task |
|---|---|
| Token grammar table (all tokens, `{year}` alias, overflow) | Task 1 (fixture covers them) |
| Validation rules 1-4 | Task 1 (fixture errors) + Task 2 (service plug-in) |
| `internal/format/sequence.go` | Task 1 |
| Service Create/Update call `ValidatePattern` | Task 2 |
| `FormatPreview` uses `Render` | Task 2 |
| `GetNextNumber` uses `Render` | Task 3 |
| Drop both "not yet implemented" comments | Task 2 + Task 3 (verified in Task 9 Step 3) |
| Frontend single Formát field with live preview | Task 6 |
| Token cheat-sheet under field | Task 6 Step 4 |
| Invalid template -> `Náhled: --` + inline error | Task 6 Step 4 |
| Mid-year edit warning | Task 6 Step 5 |
| Existing sequences: editable pattern | Task 6 Step 5 |
| TS port + shared JSON fixture | Task 5 |
| Backward compatibility (legacy default unchanged) | Task 1 legacy parity test + Task 2 parity test + Task 3 keeps existing GetNextNumber test passing |
| Help content refresh | Task 8 |
| Tests (Go renderer, service, repo, TS renderer, page) | Tasks 1, 2, 3, 5, 7 |

All spec items map to at least one task. No gaps.

**Type consistency check:** `Render`, `ValidatePattern`, `renderSequence`, `validateSequencePattern`, `createPatternError`, `editPatternError`, `editFormatPattern`, `editOriginalPattern`, `editPatternDirty`, `SeedInvoiceSequenceWithPattern` -- each name is defined exactly once and referenced consistently across tasks.

**Placeholder scan:** No "TODO", "TBD", "implement later", or hand-wavy descriptions in any step. Every code block is the actual code to write.

Plan is ready.
