# Multi-Company Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add support for one user managing multiple legal entities inside a single zfaktury install, with strict per-company data partitioning, header-dropdown switching, and seamless migration of existing single-company data.

**Architecture:** Shared SQLite database with column-based partitioning (`company_id` FK on every per-company entity). Backend exposes per-company routes under `/api/v1/companies/{companyID}/...` enforced by a `WithCompany` middleware; service and repository signatures gain explicit `companyID int64` parameters. Frontend stores the active company in a Svelte rune-based store backed by localStorage; the typed API client builds URLs from that store and surfaces a `submittedFor` capture for race-condition detection.

**Tech Stack:** Go 1.25 (chi router, cobra CLI, `modernc.org/sqlite`, goose migrations, `log/slog`); SvelteKit 5 (runes, Tailwind v4, vitest, `@testing-library/svelte`); TOML config.

**Spec:** `docs/superpowers/specs/2026-05-24-multi-company-design.md` (v2, approved)

---

## Conventions used in this plan

- All paths are repo-relative from `/home/coder/Devel/zfaktury-multi-company/` unless absolute.
- Tests run with `CGO_ENABLED=0 go test -tags server ./...` (full suite) or `CGO_ENABLED=0 go test ./internal/<pkg> -run <Name> -v` (targeted).
- Migrations are goose-style SQL files in `internal/database/migrations/`; format `NNN_description.sql`.
- The reviewer requested test-first ordering: migration tests and the leak detector exist as failing tests **before** any mechanical repo changes.
- Each task ends with a `git commit`. Branch: `feature/multi-company`. Don't squash during implementation — preserve the trail for upstream review.
- Czech terms: `IČO` = 8-digit business ID; `DIČ` = VAT ID `CZ\d{8,10}`; `OSVČ` = sole proprietor; `s.r.o.` = LLC.

## Phase overview

| Phase | Tasks | Output |
|---|---|---|
| 1. Foundation | 1–4 | `companies` domain + repo + service + test scaffolding (all green, no DB changes yet) |
| 2. Migration | 5–15 | Goose migration `025_multi_company.sql` complete, including production-sized test |
| 3. API surface | 16–22 | All routes mounted under `/api/v1/companies/{id}/...`; leak detector exhaustive |
| 4. Frontend | 23–30 | Switcher header, store, refactored client, management pages |
| 5. PR prep | 31–32 | README, manual test plan, verification |

---

## Phase 1 — Foundation (no DB changes yet)

### Task 1: Test infrastructure scaffolding

**Files:**
- Create: `internal/database/migrations_test.go`
- Create: `internal/repository/leak_detector_test.go`
- Create: `tests/integration/multicompany_test.go`

These three files exist as **failing tests with `t.Skip` placeholders** for now. They become real assertions in later tasks. This lets every subsequent task use them as a green-bar oracle.

- [ ] **Step 1: Create migrations test skeleton**

`internal/database/migrations_test.go`:

```go
package database

import (
	"testing"
)

// TestMultiCompanyMigration validates that migration 025 transforms
// a single-company v024 database into a multi-company v025 database
// with no data loss. Populated incrementally across Phase 2 tasks.
func TestMultiCompanyMigration(t *testing.T) {
	t.Skip("populated in Phase 2 tasks 5-15")
}

// TestMultiCompanyMigrationProductionSized runs migration 025 against
// a synthetic ~5k-invoice fixture and asserts completion under 30s.
// Gated behind -tags integration so it doesn't run on every CI build.
func TestMultiCompanyMigrationProductionSized(t *testing.T) {
	t.Skip("populated in Phase 2 task 15")
}
```

- [ ] **Step 2: Create leak detector skeleton**

`internal/repository/leak_detector_test.go`:

```go
package repository

import (
	"testing"
)

// TestCrossCompanyLeakDetection exhaustively tries every per-company
// repository's Get/List/Update/Delete with a wrong companyID and
// asserts ErrNotFound (or empty list). Populated incrementally
// across Phase 3 tasks as each repo gains the companyID parameter.
func TestCrossCompanyLeakDetection(t *testing.T) {
	t.Skip("populated as repos gain companyID parameter in Phase 3")
}
```

- [ ] **Step 3: Create integration test skeleton**

`tests/integration/multicompany_test.go`:

```go
//go:build integration

package integration

import (
	"testing"
)

// TestMultiCompanyEndToEnd exercises the full multi-company flow:
// create two companies, contacts, invoices, cross-company rejection,
// sequence collision, delete protection, soft-delete after cleanup.
// Populated in Phase 3 task 22 once the API surface is complete.
func TestMultiCompanyEndToEnd(t *testing.T) {
	t.Skip("populated in Phase 3 task 22")
}
```

- [ ] **Step 4: Run the suite to confirm everything still passes**

```bash
CGO_ENABLED=0 go test -tags server ./... -count=1
```

Expected: PASS (skipped tests count as pass).

- [ ] **Step 5: Commit**

```bash
git add internal/database/migrations_test.go \
        internal/repository/leak_detector_test.go \
        tests/integration/multicompany_test.go
git commit -m "Add test scaffolding for multi-company migration and leak detection

Three skipped tests reserve their names and packages so subsequent
tasks can fill in assertions incrementally without restructuring."
```

---

### Task 2: Company domain struct + sentinel errors

**Files:**
- Create: `internal/domain/company.go`
- Create: `internal/domain/company_test.go`
- Modify: `internal/domain/errors.go` (add `ErrLastCompany`, `ErrInUse`)

- [ ] **Step 1: Write the failing domain test**

`internal/domain/company_test.go`:

```go
package domain

import "testing"

func TestCompany_Validate_requiresName(t *testing.T) {
	c := Company{LegalName: "X", ICO: "12345678"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestCompany_Validate_requiresICO(t *testing.T) {
	c := Company{Name: "X", LegalName: "X"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for missing ICO")
	}
}

func TestCompany_Validate_VATRegisteredRequiresDIC(t *testing.T) {
	c := Company{Name: "X", LegalName: "X", ICO: "12345678", VATRegistered: true}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error: VAT-registered without DIC")
	}
}

func TestCompany_Validate_DICFormat(t *testing.T) {
	c := Company{Name: "X", LegalName: "X", ICO: "12345678", VATRegistered: true, DIC: "notvalid"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error: invalid DIC format")
	}
}

func TestCompany_Validate_acceptsValidVATPayer(t *testing.T) {
	c := Company{Name: "M OSVČ", LegalName: "Manas s.r.o.", ICO: "12345678", VATRegistered: true, DIC: "CZ12345678"}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompany_Validate_acceptsNonVATPayer(t *testing.T) {
	c := Company{Name: "M OSVČ", LegalName: "Manas OSVČ", ICO: "12345678", VATRegistered: false}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

```bash
CGO_ENABLED=0 go test ./internal/domain -run TestCompany_ -v
```

Expected: build failure (`Company` undefined) or test failures.

- [ ] **Step 3: Implement the domain struct**

`internal/domain/company.go`:

```go
package domain

import (
	"fmt"
	"regexp"
	"time"
)

// Company represents a legal entity managed inside zfaktury.
// All per-company entities (invoices, contacts, expenses, etc.) reference
// exactly one Company via company_id.
//
// Pure domain struct — no DB or JSON tags. Repository and handler layers
// own their own representations.
type Company struct {
	ID            int64
	Name          string // short label shown in switcher dropdown
	LegalName     string // full legal name printed on invoices
	ICO           string // 8-digit Czech business ID
	DIC           string // VAT ID, format CZ\d{8,10} when VATRegistered
	VATRegistered bool

	// Address
	Street      string
	HouseNumber string
	City        string
	ZIP         string

	// Contact
	Email string
	Phone string

	// Personal name (OSVČ tax filings name the human, not the brand)
	FirstName string
	LastName  string

	// Bank
	BankAccount string
	BankCode    string
	IBAN        string
	SWIFT       string

	// Presentation
	LogoPath    string
	AccentColor string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

var dicPattern = regexp.MustCompile(`^CZ\d{8,10}$`)

// Validate enforces the invariants that hold at the domain level.
// DB-level constraints (uniqueness of ICO, FKs) are enforced by the schema.
func (c *Company) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("%w: company name is required", ErrInvalidInput)
	}
	if c.LegalName == "" {
		return fmt.Errorf("%w: legal name is required", ErrInvalidInput)
	}
	if c.ICO == "" {
		return fmt.Errorf("%w: ICO is required", ErrInvalidInput)
	}
	if c.VATRegistered {
		if c.DIC == "" {
			return fmt.Errorf("%w: DIC is required for VAT-registered companies", ErrInvalidInput)
		}
		if !dicPattern.MatchString(c.DIC) {
			return fmt.Errorf("%w: DIC must match CZ\\d{8,10}, got %q", ErrInvalidInput, c.DIC)
		}
	}
	return nil
}
```

- [ ] **Step 4: Add the sentinel errors**

Open `internal/domain/errors.go` and add at the bottom of the existing `var (...)` block:

```go
	// ErrLastCompany indicates an attempt to soft-delete the only remaining company.
	ErrLastCompany = errors.New("cannot delete the last company")

	// ErrInUse indicates an attempt to soft-delete an entity (e.g. a company)
	// that still has non-deleted child records.
	ErrInUse = errors.New("cannot delete: still in use")
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
CGO_ENABLED=0 go test ./internal/domain -run TestCompany_ -v
```

Expected: PASS, 6 tests.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/company.go internal/domain/company_test.go internal/domain/errors.go
git commit -m "Add Company domain struct with validation and sentinel errors

Pure domain struct (no DB/JSON tags) covering identity, address, bank,
presentation, and timestamps. Validate enforces name/ICO required and
that VAT-registered companies carry a CZ-format DIC.

ErrLastCompany and ErrInUse sentinels added for the soft-delete
protection rules from the spec."
```

---

### Task 3: CompanyRepository

**Files:**
- Modify: `internal/repository/interfaces.go` (add `CompanyRepository` interface)
- Create: `internal/repository/company_repo.go`
- Create: `internal/repository/company_repo_test.go`

This task only adds the repo backed by an in-memory SQLite. It does NOT yet add the `companies` table to the live migrations — that happens in Task 6. The test creates the schema inline so we can land the Go code without a migration.

- [ ] **Step 1: Add the interface**

Open `internal/repository/interfaces.go` and add to the package, after the existing interfaces:

```go
// CompanyRepository persists Company aggregates.
//
// All other per-company repositories receive companyID as an explicit
// parameter and filter by it; CompanyRepository itself is global —
// it knows about all companies regardless of which is currently active.
type CompanyRepository interface {
	Create(ctx context.Context, c domain.Company) (int64, error)
	GetByID(ctx context.Context, id int64) (domain.Company, error)
	List(ctx context.Context) ([]domain.Company, error)
	Update(ctx context.Context, c domain.Company) error
	SoftDelete(ctx context.Context, id int64) error
	CountActive(ctx context.Context) (int, error)
}
```

- [ ] **Step 2: Write the repo test**

`internal/repository/company_repo_test.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"

	_ "modernc.org/sqlite"
)

// newCompanyTestDB returns an in-memory SQLite database with just the
// companies table — used until migration 025 lands the table for real.
func newCompanyTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
CREATE TABLE companies (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	legal_name TEXT NOT NULL,
	ico TEXT NOT NULL,
	dic TEXT,
	vat_registered INTEGER NOT NULL DEFAULT 0,
	street TEXT, house_number TEXT, city TEXT, zip TEXT,
	email TEXT, phone TEXT,
	first_name TEXT, last_name TEXT,
	bank_account TEXT, bank_code TEXT, iban TEXT, swift TEXT,
	logo_path TEXT, accent_color TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	deleted_at TEXT
);
CREATE UNIQUE INDEX idx_companies_ico_active ON companies(ico) WHERE deleted_at IS NULL;
`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestCompanyRepository_CreateAndGet(t *testing.T) {
	repo := NewCompanyRepository(newCompanyTestDB(t))
	ctx := context.Background()

	id, err := repo.Create(ctx, domain.Company{
		Name: "Manas OSVČ", LegalName: "Jiří Manas", ICO: "12345678",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Manas OSVČ" {
		t.Errorf("name = %q, want Manas OSVČ", got.Name)
	}
}

func TestCompanyRepository_GetByID_NotFound(t *testing.T) {
	repo := NewCompanyRepository(newCompanyTestDB(t))
	_, err := repo.GetByID(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestCompanyRepository_GetByID_ExcludesSoftDeleted(t *testing.T) {
	repo := NewCompanyRepository(newCompanyTestDB(t))
	ctx := context.Background()
	id, _ := repo.Create(ctx, domain.Company{Name: "A", LegalName: "A", ICO: "1"})
	if err := repo.SoftDelete(ctx, id); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	if _, err := repo.GetByID(ctx, id); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound for soft-deleted", err)
	}
}

func TestCompanyRepository_List_ExcludesSoftDeleted(t *testing.T) {
	repo := NewCompanyRepository(newCompanyTestDB(t))
	ctx := context.Background()
	keep, _ := repo.Create(ctx, domain.Company{Name: "A", LegalName: "A", ICO: "1"})
	drop, _ := repo.Create(ctx, domain.Company{Name: "B", LegalName: "B", ICO: "2"})
	_ = repo.SoftDelete(ctx, drop)
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != keep {
		t.Errorf("got %v, want [{ID:%d ...}]", list, keep)
	}
}

func TestCompanyRepository_CountActive(t *testing.T) {
	repo := NewCompanyRepository(newCompanyTestDB(t))
	ctx := context.Background()
	_, _ = repo.Create(ctx, domain.Company{Name: "A", LegalName: "A", ICO: "1"})
	id, _ := repo.Create(ctx, domain.Company{Name: "B", LegalName: "B", ICO: "2"})
	_ = repo.SoftDelete(ctx, id)
	n, err := repo.CountActive(ctx)
	if err != nil {
		t.Fatalf("CountActive: %v", err)
	}
	if n != 1 {
		t.Errorf("n = %d, want 1", n)
	}
}

func TestCompanyRepository_Update(t *testing.T) {
	repo := NewCompanyRepository(newCompanyTestDB(t))
	ctx := context.Background()
	id, _ := repo.Create(ctx, domain.Company{Name: "Old", LegalName: "Old", ICO: "1"})
	c, _ := repo.GetByID(ctx, id)
	c.Name = "New"
	if err := repo.Update(ctx, c); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := repo.GetByID(ctx, id)
	if got.Name != "New" {
		t.Errorf("name = %q, want New", got.Name)
	}
}
```

- [ ] **Step 3: Implement the repo**

`internal/repository/company_repo.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

type CompanyRepositoryImpl struct {
	db *sql.DB
}

func NewCompanyRepository(db *sql.DB) *CompanyRepositoryImpl {
	return &CompanyRepositoryImpl{db: db}
}

const companyColumns = `id, name, legal_name, ico, dic, vat_registered,
	street, house_number, city, zip,
	email, phone,
	first_name, last_name,
	bank_account, bank_code, iban, swift,
	logo_path, accent_color,
	created_at, updated_at, deleted_at`

func scanCompany(row interface{ Scan(...any) error }) (domain.Company, error) {
	var c domain.Company
	var vatInt int
	var dic, street, hn, city, zip, email, phone sql.NullString
	var firstName, lastName, bankAcc, bankCode, iban, swift sql.NullString
	var logo, accent sql.NullString
	var createdAt, updatedAt string
	var deletedAt sql.NullString

	err := row.Scan(
		&c.ID, &c.Name, &c.LegalName, &c.ICO, &dic, &vatInt,
		&street, &hn, &city, &zip,
		&email, &phone,
		&firstName, &lastName,
		&bankAcc, &bankCode, &iban, &swift,
		&logo, &accent,
		&createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return c, err
	}
	c.DIC = dic.String
	c.VATRegistered = vatInt == 1
	c.Street, c.HouseNumber, c.City, c.ZIP = street.String, hn.String, city.String, zip.String
	c.Email, c.Phone = email.String, phone.String
	c.FirstName, c.LastName = firstName.String, lastName.String
	c.BankAccount, c.BankCode, c.IBAN, c.SWIFT = bankAcc.String, bankCode.String, iban.String, swift.String
	c.LogoPath, c.AccentColor = logo.String, accent.String
	c.CreatedAt, err = parseDate(time.RFC3339, createdAt)
	if err != nil {
		return c, fmt.Errorf("parsing created_at: %w", err)
	}
	c.UpdatedAt, err = parseDate(time.RFC3339, updatedAt)
	if err != nil {
		return c, fmt.Errorf("parsing updated_at: %w", err)
	}
	c.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAt)
	if err != nil {
		return c, fmt.Errorf("parsing deleted_at: %w", err)
	}
	return c, nil
}

func (r *CompanyRepositoryImpl) Create(ctx context.Context, c domain.Company) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	vatInt := 0
	if c.VATRegistered {
		vatInt = 1
	}
	res, err := r.db.ExecContext(ctx, `
INSERT INTO companies (name, legal_name, ico, dic, vat_registered,
	street, house_number, city, zip,
	email, phone,
	first_name, last_name,
	bank_account, bank_code, iban, swift,
	logo_path, accent_color,
	created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Name, c.LegalName, c.ICO, nullableString(c.DIC), vatInt,
		nullableString(c.Street), nullableString(c.HouseNumber), nullableString(c.City), nullableString(c.ZIP),
		nullableString(c.Email), nullableString(c.Phone),
		nullableString(c.FirstName), nullableString(c.LastName),
		nullableString(c.BankAccount), nullableString(c.BankCode), nullableString(c.IBAN), nullableString(c.SWIFT),
		nullableString(c.LogoPath), nullableString(c.AccentColor),
		now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting company: %w", err)
	}
	return res.LastInsertId()
}

func (r *CompanyRepositoryImpl) GetByID(ctx context.Context, id int64) (domain.Company, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+companyColumns+` FROM companies WHERE id = ? AND deleted_at IS NULL`, id)
	c, err := scanCompany(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Company{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Company{}, fmt.Errorf("fetching company %d: %w", id, err)
	}
	return c, nil
}

func (r *CompanyRepositoryImpl) List(ctx context.Context) ([]domain.Company, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+companyColumns+` FROM companies WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("listing companies: %w", err)
	}
	defer rows.Close()

	var out []domain.Company
	for rows.Next() {
		c, err := scanCompany(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning company row: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CompanyRepositoryImpl) Update(ctx context.Context, c domain.Company) error {
	vatInt := 0
	if c.VATRegistered {
		vatInt = 1
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE companies SET
	name = ?, legal_name = ?, ico = ?, dic = ?, vat_registered = ?,
	street = ?, house_number = ?, city = ?, zip = ?,
	email = ?, phone = ?,
	first_name = ?, last_name = ?,
	bank_account = ?, bank_code = ?, iban = ?, swift = ?,
	logo_path = ?, accent_color = ?,
	updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		c.Name, c.LegalName, c.ICO, nullableString(c.DIC), vatInt,
		nullableString(c.Street), nullableString(c.HouseNumber), nullableString(c.City), nullableString(c.ZIP),
		nullableString(c.Email), nullableString(c.Phone),
		nullableString(c.FirstName), nullableString(c.LastName),
		nullableString(c.BankAccount), nullableString(c.BankCode), nullableString(c.IBAN), nullableString(c.SWIFT),
		nullableString(c.LogoPath), nullableString(c.AccentColor),
		time.Now().UTC().Format(time.RFC3339),
		c.ID,
	)
	if err != nil {
		return fmt.Errorf("updating company %d: %w", c.ID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CompanyRepositoryImpl) SoftDelete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE companies SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting company %d: %w", id, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CompanyRepositoryImpl) CountActive(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM companies WHERE deleted_at IS NULL`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("counting active companies: %w", err)
	}
	return n, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
```

The `parseDate` / `parseDatePtr` helpers already exist in `internal/repository/helpers.go`. If `nullableString` collides with an existing helper, drop the new one and use the existing.

- [ ] **Step 4: Run tests**

```bash
CGO_ENABLED=0 go test ./internal/repository -run TestCompanyRepository_ -v
```

Expected: PASS, 6 tests.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/interfaces.go internal/repository/company_repo.go internal/repository/company_repo_test.go
git commit -m "Add CompanyRepository with create/get/list/update/soft-delete

In-memory SQLite tests cover the happy path, ErrNotFound, soft-delete
exclusion from Get/List, and CountActive. The repository owns the
SQL-to-domain mapping via scanCompany; nullableString converts empty
strings to NULL for optional columns so blank fields stay distinguishable
from explicit zero values."
```

---

### Task 4: CompanyService with delete protection

**Files:**
- Create: `internal/service/company_svc.go`
- Create: `internal/service/company_svc_test.go`

The service enforces two rules from the spec:
1. Cannot soft-delete the last remaining company → `domain.ErrLastCompany`.
2. Cannot soft-delete a company that has any non-deleted invoices or expenses → `domain.ErrInUse`.

Rule 2 needs to query the per-company tables, but those don't have `company_id` yet (added in Phase 2). For now the service depends on small `EntityChecker` interfaces that are stubbed in tests; real wiring lands in Task 21+.

- [ ] **Step 1: Write the service test**

`internal/service/company_svc_test.go`:

```go
package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

type stubCompanyRepo struct {
	all          []domain.Company
	countActive  int
	deleted      []int64
	deleteErr    error
}

func (s *stubCompanyRepo) Create(_ context.Context, c domain.Company) (int64, error) {
	c.ID = int64(len(s.all) + 1)
	s.all = append(s.all, c)
	s.countActive++
	return c.ID, nil
}
func (s *stubCompanyRepo) GetByID(_ context.Context, id int64) (domain.Company, error) {
	for _, c := range s.all {
		if c.ID == id {
			return c, nil
		}
	}
	return domain.Company{}, domain.ErrNotFound
}
func (s *stubCompanyRepo) List(_ context.Context) ([]domain.Company, error) {
	return s.all, nil
}
func (s *stubCompanyRepo) Update(_ context.Context, _ domain.Company) error { return nil }
func (s *stubCompanyRepo) SoftDelete(_ context.Context, id int64) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	s.countActive--
	return nil
}
func (s *stubCompanyRepo) CountActive(_ context.Context) (int, error) { return s.countActive, nil }

type stubEntityChecker struct{ count int }

func (s *stubEntityChecker) CountNonDeletedForCompany(_ context.Context, _ int64) (int, error) {
	return s.count, nil
}

func TestCompanyService_Delete_blocksLastCompany(t *testing.T) {
	repo := &stubCompanyRepo{countActive: 1, all: []domain.Company{{ID: 1, Name: "Only"}}}
	svc := NewCompanyService(repo, []EntityChecker{&stubEntityChecker{0}}, nil)
	err := svc.Delete(context.Background(), 1)
	if !errors.Is(err, domain.ErrLastCompany) {
		t.Errorf("err = %v, want ErrLastCompany", err)
	}
}

func TestCompanyService_Delete_blocksNonEmptyCompany(t *testing.T) {
	repo := &stubCompanyRepo{countActive: 2, all: []domain.Company{{ID: 1}, {ID: 2}}}
	svc := NewCompanyService(repo, []EntityChecker{&stubEntityChecker{count: 3}}, nil)
	err := svc.Delete(context.Background(), 1)
	if !errors.Is(err, domain.ErrInUse) {
		t.Errorf("err = %v, want ErrInUse", err)
	}
}

func TestCompanyService_Delete_succeedsWhenEmptyAndNotLast(t *testing.T) {
	repo := &stubCompanyRepo{countActive: 2, all: []domain.Company{{ID: 1}, {ID: 2}}}
	svc := NewCompanyService(repo, []EntityChecker{&stubEntityChecker{0}}, nil)
	if err := svc.Delete(context.Background(), 2); err != nil {
		t.Errorf("Delete: %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != 2 {
		t.Errorf("repo.deleted = %v, want [2]", repo.deleted)
	}
}

func TestCompanyService_Create_validates(t *testing.T) {
	svc := NewCompanyService(&stubCompanyRepo{}, nil, nil)
	_, err := svc.Create(context.Background(), domain.Company{})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}
}
```

- [ ] **Step 2: Run it to verify failure**

```bash
CGO_ENABLED=0 go test ./internal/service -run TestCompanyService_ -v
```

Expected: build failure.

- [ ] **Step 3: Implement the service**

`internal/service/company_svc.go`:

```go
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// EntityChecker reports whether a company has any non-deleted records
// of a particular kind (invoices, expenses, etc.). The CompanyService
// consults every registered checker before allowing a soft-delete.
type EntityChecker interface {
	CountNonDeletedForCompany(ctx context.Context, companyID int64) (int, error)
}

type CompanyService struct {
	repo     repository.CompanyRepository
	checkers []EntityChecker
	audit    *AuditService
}

func NewCompanyService(repo repository.CompanyRepository, checkers []EntityChecker, audit *AuditService) *CompanyService {
	return &CompanyService{repo: repo, checkers: checkers, audit: audit}
}

func (s *CompanyService) Create(ctx context.Context, c domain.Company) (int64, error) {
	if err := c.Validate(); err != nil {
		return 0, err
	}
	id, err := s.repo.Create(ctx, c)
	if err != nil {
		return 0, fmt.Errorf("creating company: %w", err)
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, AuditEvent{Action: "company.create", EntityID: id, EntityKind: "company"})
	}
	return id, nil
}

func (s *CompanyService) Get(ctx context.Context, id int64) (domain.Company, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Company{}, fmt.Errorf("fetching company: %w", err)
	}
	return c, nil
}

func (s *CompanyService) List(ctx context.Context) ([]domain.Company, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing companies: %w", err)
	}
	return list, nil
}

func (s *CompanyService) Update(ctx context.Context, c domain.Company) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, c); err != nil {
		return fmt.Errorf("updating company: %w", err)
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, AuditEvent{Action: "company.update", EntityID: c.ID, EntityKind: "company"})
	}
	return nil
}

func (s *CompanyService) Delete(ctx context.Context, id int64) error {
	// Rule 1: cannot delete the last company.
	n, err := s.repo.CountActive(ctx)
	if err != nil {
		return fmt.Errorf("counting active companies: %w", err)
	}
	if n <= 1 {
		return domain.ErrLastCompany
	}

	// Rule 2: cannot delete a company with any non-deleted children.
	for _, ck := range s.checkers {
		count, err := ck.CountNonDeletedForCompany(ctx, id)
		if err != nil {
			return fmt.Errorf("checking for child records: %w", err)
		}
		if count > 0 {
			return domain.ErrInUse
		}
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return err
		}
		return fmt.Errorf("soft-deleting company: %w", err)
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, AuditEvent{Action: "company.delete", EntityID: id, EntityKind: "company"})
	}
	return nil
}
```

If your codebase's existing `AuditService.Log` and `AuditEvent` shapes differ, adapt the audit calls to match — that's the only piece that touches existing API.

- [ ] **Step 4: Run tests**

```bash
CGO_ENABLED=0 go test ./internal/service -run TestCompanyService_ -v
```

Expected: PASS, 4 tests.

- [ ] **Step 5: Run the full suite**

```bash
CGO_ENABLED=0 go test -tags server ./... -count=1
```

Expected: all green (existing tests untouched, new ones passing).

- [ ] **Step 6: Commit**

```bash
git add internal/service/company_svc.go internal/service/company_svc_test.go
git commit -m "Add CompanyService with last-company and in-use delete protection

The service enforces both spec invariants before soft-deleting a company:
ErrLastCompany when the active count is <= 1, ErrInUse when any
registered EntityChecker reports non-deleted child records. Real
checkers (invoices, expenses) are wired in Phase 3 as those repos
gain the companyID parameter."
```

---

**Phase 1 checkpoint:** Domain, repository, and service all green. No DB schema changes yet — `companies` table is created inline by the repo test. `migrations_test.go`, `leak_detector_test.go`, and `multicompany_test.go` are skipped placeholders waiting for content.

Continue to Phase 2.

---

## Phase 2 — Migration 025 (TDD, frontloaded per reviewer)

This phase produces `internal/database/migrations/025_multi_company.sql` in one logical commit per entity group, with `migrations_test.go` assertions added before the SQL.

### Task 5: Migration test fixtures and helpers

**Files:**
- Create: `internal/database/testdata/v024_seed.sql` (committed fixture)
- Modify: `internal/database/migrations_test.go` (real setup + first real assertion)

The fixture is a small but realistic v024-state snapshot: one settings row per company-info key, a few contacts, two invoices, one expense, two invoice sequences. Enough to exercise every branch of the migration.

- [ ] **Step 1: Create the v024 fixture**

`internal/database/testdata/v024_seed.sql`:

```sql
-- v024 fixture: a single-company database immediately before migration 025.
-- All 17 company-identity settings keys are populated; a handful of
-- per-company entities are present so the migration's backfill is observable.

INSERT INTO settings (key, value) VALUES
	('company_name',    'Manas OSVČ'),
	('ico',             '12345678'),
	('dic',             'CZ12345678'),
	('vat_registered',  '1'),
	('street',          'Václavské náměstí'),
	('house_number',    '1'),
	('city',            'Praha'),
	('zip',             '11000'),
	('email',           'jiri@manas.cz'),
	('phone',           '+420 123 456 789'),
	('first_name',      'Jiří'),
	('last_name',       'Manas'),
	('bank_account',    '1234567890'),
	('bank_code',       '0100'),
	('iban',            'CZ0001000000001234567890'),
	('swift',           'KOMBCZPP'),
	('logo_path',       ''),
	('accent_color',    '#1a56db'),
	-- A non-identity key that should survive intact:
	('default_payment_method', 'bank_transfer');

INSERT INTO contacts (id, name, ico, dic, email, created_at, updated_at) VALUES
	(1, 'Keboola s.r.o.',   '25620916', 'CZ25620916', 'fakturace@keboola.com', '2026-01-15T10:00:00Z', '2026-01-15T10:00:00Z'),
	(2, 'Acme spol. s r.o.', '11223344', NULL,        'billing@acme.cz',       '2026-02-01T10:00:00Z', '2026-02-01T10:00:00Z');

INSERT INTO invoice_sequences (id, prefix, next_number, year, format_pattern) VALUES
	(1, 'FV', 3, 2026, '{prefix}{year}{number:04d}'),
	(2, 'ZF', 1, 2026, '{prefix}{year}{number:04d}');

INSERT INTO invoices (id, number, contact_id, issue_date, due_date, status, total_amount, created_at, updated_at) VALUES
	(1, 'FV20260001', 1, '2026-02-01', '2026-02-15', 'paid', 5000000, '2026-02-01T10:00:00Z', '2026-02-15T12:00:00Z'),
	(2, 'FV20260002', 2, '2026-03-01', '2026-03-15', 'sent', 2500000, '2026-03-01T10:00:00Z', '2026-03-01T10:00:00Z');

INSERT INTO invoice_items (id, invoice_id, description, quantity, unit_price, total) VALUES
	(1, 1, 'Konzultace',           20, 250000, 5000000),
	(2, 2, 'Vývoj integrace',      10, 250000, 2500000);

INSERT INTO expenses (id, vendor, document_number, issue_date, total_amount, created_at, updated_at) VALUES
	(1, 'Alza.cz', 'AL/2026/0001', '2026-02-10', 1200000, '2026-02-10T10:00:00Z', '2026-02-10T10:00:00Z');
```

- [ ] **Step 2: Replace the skipped migration test**

Open `internal/database/migrations_test.go` and replace its contents:

```go
package database

import (
	"database/sql"
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed testdata/v024_seed.sql
var v024SeedFS embed.FS

// migrateUpTo runs every migration in internal/database/migrations
// up to and including the targetVersion.
func migrateUpTo(t *testing.T, db *sql.DB, targetVersion int64) {
	t.Helper()
	goose.SetBaseFS(migrationsFS) // existing embed.FS in this package
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	if err := goose.UpTo(db, "migrations", targetVersion); err != nil {
		t.Fatalf("goose up to %d: %v", targetVersion, err)
	}
}

func openMigratedDB(t *testing.T, applyV025 bool) *sql.DB {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", tmp+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrateUpTo(t, db, 24)

	seed, err := v024SeedFS.ReadFile("testdata/v024_seed.sql")
	if err != nil {
		t.Fatalf("read seed: %v", err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("apply seed: %v", err)
	}

	if applyV025 {
		migrateUpTo(t, db, 25)
	}
	return db
}

func TestMultiCompanyMigration_CreatesDefaultCompanyFromSettings(t *testing.T) {
	db := openMigratedDB(t, true)
	var name, ico, dic string
	var vat int
	err := db.QueryRow(`SELECT name, ico, dic, vat_registered FROM companies WHERE id = 1`).Scan(&name, &ico, &dic, &vat)
	if err != nil {
		t.Fatalf("query company: %v", err)
	}
	if name != "Manas OSVČ" || ico != "12345678" || dic != "CZ12345678" || vat != 1 {
		t.Errorf("got name=%q ico=%q dic=%q vat=%d", name, ico, dic, vat)
	}
}

func TestMultiCompanyMigration_StripsIdentityKeysFromSettings(t *testing.T) {
	db := openMigratedDB(t, true)
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM settings WHERE key IN
		('company_name','ico','dic','vat_registered','street','house_number','city','zip',
		 'email','phone','first_name','last_name','bank_account','bank_code','iban','swift',
		 'logo_path','accent_color')`).Scan(&n)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("found %d identity keys remaining in settings, want 0", n)
	}
}

func TestMultiCompanyMigration_PreservesNonIdentitySettings(t *testing.T) {
	db := openMigratedDB(t, true)
	var v string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key = 'default_payment_method'`).Scan(&v); err != nil {
		t.Fatalf("query: %v", err)
	}
	if v != "bank_transfer" {
		t.Errorf("value = %q, want bank_transfer", v)
	}
}

func TestMultiCompanyMigration_FreshInstallProducesEmptyCompanies(t *testing.T) {
	// Fresh install: no v024 seed applied.
	tmp := filepath.Join(t.TempDir(), "fresh.db")
	db, err := sql.Open("sqlite", tmp+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migrateUpTo(t, db, 25)

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM companies`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("fresh install company count = %d, want 0", n)
	}
}

// Keep the production-sized test skipped; Task 15 fills it in.
func TestMultiCompanyMigrationProductionSized(t *testing.T) {
	if os.Getenv("ZFAKTURY_RUN_BIG_MIGRATION_TEST") == "" {
		t.Skip("set ZFAKTURY_RUN_BIG_MIGRATION_TEST=1 to enable")
	}
	t.Skip("populated in Phase 2 task 15")
}
```

If `migrationsFS` is named differently in this package (check `migrations.go`), adjust the reference.

- [ ] **Step 3: Run the test to verify it fails**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_ -v
```

Expected: FAIL — migration 025 doesn't exist yet, so `goose.UpTo(db, "migrations", 25)` errors with `no migration found with version 25`.

- [ ] **Step 4: Commit**

```bash
git add internal/database/migrations_test.go internal/database/testdata/v024_seed.sql
git commit -m "Set up failing migration tests for multi-company

Embedded v024_seed.sql fixture exercises every branch of the upcoming
025 migration (company-identity keys, non-identity setting, contacts,
sequences, invoices, items, expenses). Test cases assert that:

1. The default company is created from settings keys
2. The 17 identity keys are stripped from settings
3. Non-identity settings survive
4. A fresh install (no seed) produces zero companies

All four currently fail because migration 025 does not exist yet —
Task 6 introduces it."
```

---

### Task 6: Migration 025 — companies table + default seed

**Files:**
- Create: `internal/database/migrations/025_multi_company.sql`

This is the first slice of the migration. It only creates the `companies` table and seeds the default row from settings — no partitioning yet. The four tests from Task 5 should pass after this commit; everything else still pending.

- [ ] **Step 1: Create the migration file**

`internal/database/migrations/025_multi_company.sql`:

```sql
-- +goose Up
-- +goose StatementBegin

-- 1. Companies table — the new home for what was 17 settings keys.
CREATE TABLE companies (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	name            TEXT    NOT NULL,
	legal_name      TEXT    NOT NULL,
	ico             TEXT    NOT NULL,
	dic             TEXT,
	vat_registered  INTEGER NOT NULL DEFAULT 0,
	street          TEXT, house_number TEXT, city TEXT, zip TEXT,
	email           TEXT, phone TEXT,
	first_name      TEXT, last_name TEXT,
	bank_account    TEXT, bank_code TEXT, iban TEXT, swift TEXT,
	logo_path       TEXT, accent_color TEXT,
	created_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL,
	deleted_at      TEXT
);
CREATE UNIQUE INDEX idx_companies_ico_active ON companies(ico) WHERE deleted_at IS NULL;

-- 2. Seed the default company from existing settings.
-- The WHERE EXISTS guard makes fresh installs (empty settings) a no-op.
INSERT INTO companies (
	id, name, legal_name, ico, dic, vat_registered,
	street, house_number, city, zip,
	email, phone,
	first_name, last_name,
	bank_account, bank_code, iban, swift,
	logo_path, accent_color,
	created_at, updated_at
)
SELECT
	1,
	COALESCE((SELECT value FROM settings WHERE key='company_name'), 'My Company'),
	COALESCE((SELECT value FROM settings WHERE key='company_name'), 'My Company'),
	COALESCE((SELECT value FROM settings WHERE key='ico'), ''),
	NULLIF((SELECT value FROM settings WHERE key='dic'), ''),
	CASE WHEN COALESCE((SELECT value FROM settings WHERE key='vat_registered'), '0') = '1' THEN 1 ELSE 0 END,
	NULLIF((SELECT value FROM settings WHERE key='street'), ''),
	NULLIF((SELECT value FROM settings WHERE key='house_number'), ''),
	NULLIF((SELECT value FROM settings WHERE key='city'), ''),
	NULLIF((SELECT value FROM settings WHERE key='zip'), ''),
	NULLIF((SELECT value FROM settings WHERE key='email'), ''),
	NULLIF((SELECT value FROM settings WHERE key='phone'), ''),
	NULLIF((SELECT value FROM settings WHERE key='first_name'), ''),
	NULLIF((SELECT value FROM settings WHERE key='last_name'), ''),
	NULLIF((SELECT value FROM settings WHERE key='bank_account'), ''),
	NULLIF((SELECT value FROM settings WHERE key='bank_code'), ''),
	NULLIF((SELECT value FROM settings WHERE key='iban'), ''),
	NULLIF((SELECT value FROM settings WHERE key='swift'), ''),
	NULLIF((SELECT value FROM settings WHERE key='logo_path'), ''),
	NULLIF((SELECT value FROM settings WHERE key='accent_color'), ''),
	datetime('now'),
	datetime('now')
WHERE EXISTS (SELECT 1 FROM settings LIMIT 1);

-- 3. Strip the 17 identity keys from settings (now lifted into companies).
DELETE FROM settings WHERE key IN (
	'company_name', 'ico', 'dic', 'vat_registered',
	'street', 'house_number', 'city', 'zip',
	'email', 'phone',
	'first_name', 'last_name',
	'bank_account', 'bank_code', 'iban', 'swift',
	'logo_path', 'accent_color'
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore the 17 identity keys from company id=1 (best-effort; multi-company
-- users will lose everything but the first company on downgrade).
INSERT INTO settings (key, value)
SELECT 'company_name', name FROM companies WHERE id = 1
UNION ALL SELECT 'ico', ico FROM companies WHERE id = 1
UNION ALL SELECT 'dic', COALESCE(dic, '') FROM companies WHERE id = 1
UNION ALL SELECT 'vat_registered', CASE WHEN vat_registered = 1 THEN '1' ELSE '0' END FROM companies WHERE id = 1
UNION ALL SELECT 'street', COALESCE(street, '') FROM companies WHERE id = 1
UNION ALL SELECT 'house_number', COALESCE(house_number, '') FROM companies WHERE id = 1
UNION ALL SELECT 'city', COALESCE(city, '') FROM companies WHERE id = 1
UNION ALL SELECT 'zip', COALESCE(zip, '') FROM companies WHERE id = 1
UNION ALL SELECT 'email', COALESCE(email, '') FROM companies WHERE id = 1
UNION ALL SELECT 'phone', COALESCE(phone, '') FROM companies WHERE id = 1
UNION ALL SELECT 'first_name', COALESCE(first_name, '') FROM companies WHERE id = 1
UNION ALL SELECT 'last_name', COALESCE(last_name, '') FROM companies WHERE id = 1
UNION ALL SELECT 'bank_account', COALESCE(bank_account, '') FROM companies WHERE id = 1
UNION ALL SELECT 'bank_code', COALESCE(bank_code, '') FROM companies WHERE id = 1
UNION ALL SELECT 'iban', COALESCE(iban, '') FROM companies WHERE id = 1
UNION ALL SELECT 'swift', COALESCE(swift, '') FROM companies WHERE id = 1
UNION ALL SELECT 'logo_path', COALESCE(logo_path, '') FROM companies WHERE id = 1
UNION ALL SELECT 'accent_color', COALESCE(accent_color, '') FROM companies WHERE id = 1;

DROP INDEX IF EXISTS idx_companies_ico_active;
DROP TABLE IF EXISTS companies;

-- +goose StatementEnd
```

- [ ] **Step 2: Run the tests from Task 5**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_ -v
```

Expected: PASS, 4 tests.

- [ ] **Step 3: Sanity-check goose up/down on a scratch DB**

```bash
sqlite3 /tmp/zft-mig.db < /dev/null
CGO_ENABLED=0 go run ./cmd/zfaktury seed --config /tmp/zft-mig-config.toml 2>&1 | head -3 || true
# (the seed command will fail at migration 025 if we broke it; success here means the migration is clean)
```

If `seed` complains, read the error — most likely the `INSERT INTO companies ... WHERE EXISTS` syntax tripped on the SQLite version. Adjust before continuing.

- [ ] **Step 4: Commit**

```bash
git add internal/database/migrations/025_multi_company.sql
git commit -m "Migration 025: create companies table and seed default from settings

First slice of the multi-company migration. Creates the companies
table with all 22 identity/address/bank/presentation columns plus
audit timestamps and a soft-delete column. Seeds id=1 from the
existing 17 settings keys for upgrading users; fresh installs see
WHERE EXISTS skip the insert. Strips those 17 keys from settings
since they now live as proper columns.

Down migration best-effort restores the 17 keys from id=1 and drops
the table; documented as destructive for users with multiple
companies (only the first survives a downgrade)."
```

---

### Task 7: Partition `contacts`

**Files:**
- Modify: `internal/database/migrations/025_multi_company.sql` (append new ALTER + index, and matching DROP in Down)
- Modify: `internal/database/migrations_test.go` (add backfill assertion)

`contacts` is a simple, independent partition — no composite FK needed, no rebuild. Establishes the pattern that every following entity uses.

- [ ] **Step 1: Add the test assertion**

In `migrations_test.go` add:

```go
func TestMultiCompanyMigration_BackfillsContactsToDefaultCompany(t *testing.T) {
	db := openMigratedDB(t, true)
	rows, err := db.Query(`SELECT id, company_id FROM contacts ORDER BY id`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var got []struct{ ID, CompanyID int64 }
	for rows.Next() {
		var r struct{ ID, CompanyID int64 }
		if err := rows.Scan(&r.ID, &r.CompanyID); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if len(got) != 2 {
		t.Fatalf("got %d contacts, want 2", len(got))
	}
	for _, r := range got {
		if r.CompanyID != 1 {
			t.Errorf("contact %d company_id = %d, want 1", r.ID, r.CompanyID)
		}
	}
}
```

- [ ] **Step 2: Run it to verify failure**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_BackfillsContactsToDefaultCompany -v
```

Expected: FAIL with `no such column: company_id`.

- [ ] **Step 3: Add the ALTER to the migration**

In `025_multi_company.sql`, append inside the `-- +goose Up` block (just before `-- +goose StatementEnd`):

```sql

-- Partition: contacts
ALTER TABLE contacts ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_contacts_company ON contacts(company_id);
```

And in the `-- +goose Down` block, **prepend** (downs run reverse order conceptually) before the existing `INSERT INTO settings`:

```sql

-- Reverse: contacts
DROP INDEX IF EXISTS idx_contacts_company;
-- SQLite can't DROP COLUMN before 3.35. Use the rebuild dance to remove company_id.
CREATE TABLE contacts__tmp AS SELECT id, name, ico, dic, email, created_at, updated_at FROM contacts;
DROP TABLE contacts;
ALTER TABLE contacts__tmp RENAME TO contacts;
-- (Indexes and FKs from earlier migrations need to be re-created here if they existed.
--  Real schema may have more columns; this is the v024 shape per migration 0xx.)
```

**Implementation note:** the Down's rebuild is intentionally lossy — see the spec's down-migration warning. The Down block is best-effort and is not exercised by the live application; the test suite never runs it. If you want it fully accurate, regenerate the v024 column list from `internal/database/migrations/0NN_contacts*.sql`. Otherwise leave a `TODO: regenerate from migration history if needed` comment and move on.

- [ ] **Step 4: Run all migration tests**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_ -v
```

Expected: PASS, 5 tests.

- [ ] **Step 5: Commit**

```bash
git add internal/database/migrations/025_multi_company.sql internal/database/migrations_test.go
git commit -m "Migration 025: partition contacts by company_id

Adds company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id)
on contacts plus a covering index. The DEFAULT 1 backfills every
existing contact to the seeded default company; the FK rejects any
future insert that names a non-existent company.

Establishes the partition pattern that the remaining ~25 tables
follow in subsequent tasks."
```

---

### Task 8: Partition `expense_categories` and other independent entities

Bundles the remaining simple partitions that have no parent-child composite FK requirement: `expense_categories`, `recurring_expenses`, `expense_documents`, `invoice_documents`, `invoice_status_history`, `payment_reminders`, `fakturoid_import_log`, `tax_year_settings`, `tax_prepayments`, `tax_spouse_credits`, `tax_child_credits`, `tax_personal_credits`, `tax_deductions`, `tax_deduction_documents`, `social_insurance_overviews`, `health_insurance_overviews`, `investment_documents`, `capital_income_entries`, `security_transactions`.

These are all "leaf" entities or entities whose children are handled separately — they each just need a `company_id` column + index.

**Files:**
- Modify: `internal/database/migrations/025_multi_company.sql`
- Modify: `internal/database/migrations_test.go`

- [ ] **Step 1: Add a table-driven backfill assertion**

In `migrations_test.go` add:

```go
func TestMultiCompanyMigration_BackfillsLeafEntitiesToDefaultCompany(t *testing.T) {
	db := openMigratedDB(t, true)
	// Tables that the v024 seed populates with at least one row.
	tables := []string{"invoices", "invoice_items", "expenses"}
	for _, tbl := range tables {
		t.Run(tbl, func(t *testing.T) {
			var n, bad int
			if err := db.QueryRow(`SELECT COUNT(*) FROM ` + tbl).Scan(&n); err != nil {
				t.Fatalf("count: %v", err)
			}
			if n == 0 {
				t.Skipf("seed populates 0 rows in %s", tbl)
			}
			if err := db.QueryRow(`SELECT COUNT(*) FROM ` + tbl + ` WHERE company_id != 1`).Scan(&bad); err != nil {
				t.Fatalf("count bad: %v", err)
			}
			if bad != 0 {
				t.Errorf("%s: %d rows with company_id != 1", tbl, bad)
			}
		})
	}
}

func TestMultiCompanyMigration_AllPerCompanyTablesHaveColumn(t *testing.T) {
	db := openMigratedDB(t, true)
	tables := []string{
		"contacts", "expense_categories",
		"invoices", "invoice_items", "invoice_status_history", "invoice_documents",
		"recurring_invoices", "recurring_invoice_items",
		"expenses", "expense_items", "expense_documents",
		"recurring_expenses",
		"invoice_sequences",
		"payment_reminders",
		"tax_year_settings", "tax_prepayments",
		"tax_spouse_credits", "tax_child_credits", "tax_personal_credits",
		"tax_deductions", "tax_deduction_documents",
		"vat_returns", "vat_return_invoices", "vat_return_expenses",
		"vat_control_statements", "vat_control_statement_lines",
		"vies_summaries", "vies_summary_lines",
		"income_tax_returns", "social_insurance_overviews", "health_insurance_overviews",
		"investment_documents", "capital_income_entries", "security_transactions",
		"fakturoid_import_log",
		"settings",
	}
	for _, tbl := range tables {
		t.Run(tbl, func(t *testing.T) {
			rows, err := db.Query(`PRAGMA table_info(` + tbl + `)`)
			if err != nil {
				t.Fatalf("pragma: %v", err)
			}
			defer rows.Close()
			has := false
			for rows.Next() {
				var cid int
				var name, typ, dflt sql.NullString
				var notnull, pk int
				_ = rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk)
				if name.String == "company_id" {
					has = true
				}
			}
			if !has {
				t.Errorf("%s: missing company_id column", tbl)
			}
		})
	}
}
```

The second test is the master "every per-company table has the column" assertion — it'll grow green incrementally as later tasks land their partitions. Keep it in.

- [ ] **Step 2: Run them to see which subtests fail**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_AllPerCompanyTablesHaveColumn -v
```

Expected: many subtests fail (only `contacts` passes from Task 7). That's the point — they tell you which partitions are missing.

- [ ] **Step 3: Append the ALTERs to the migration**

In `025_multi_company.sql`, append before `-- +goose StatementEnd`:

```sql

-- Partition: leaf entities (simple ADD COLUMN + index)
ALTER TABLE expense_categories      ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE expense_documents       ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE invoice_documents       ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE invoice_status_history  ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE recurring_expenses      ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE payment_reminders       ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE fakturoid_import_log    ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE tax_year_settings       ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE tax_prepayments         ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE tax_spouse_credits      ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE tax_child_credits       ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE tax_personal_credits    ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE tax_deductions          ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE tax_deduction_documents ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE social_insurance_overviews ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE health_insurance_overviews ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE investment_documents    ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE capital_income_entries  ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE security_transactions   ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);

CREATE INDEX idx_expense_categories_company      ON expense_categories(company_id);
CREATE INDEX idx_expense_documents_company       ON expense_documents(company_id);
CREATE INDEX idx_invoice_documents_company       ON invoice_documents(company_id);
CREATE INDEX idx_invoice_status_history_company  ON invoice_status_history(company_id);
CREATE INDEX idx_recurring_expenses_company      ON recurring_expenses(company_id);
CREATE INDEX idx_payment_reminders_company       ON payment_reminders(company_id);
CREATE INDEX idx_fakturoid_import_log_company    ON fakturoid_import_log(company_id);
CREATE INDEX idx_tax_year_settings_company       ON tax_year_settings(company_id);
CREATE INDEX idx_tax_prepayments_company         ON tax_prepayments(company_id);
CREATE INDEX idx_tax_spouse_credits_company      ON tax_spouse_credits(company_id);
CREATE INDEX idx_tax_child_credits_company       ON tax_child_credits(company_id);
CREATE INDEX idx_tax_personal_credits_company    ON tax_personal_credits(company_id);
CREATE INDEX idx_tax_deductions_company          ON tax_deductions(company_id);
CREATE INDEX idx_tax_deduction_documents_company ON tax_deduction_documents(company_id);
CREATE INDEX idx_social_insurance_overviews_company ON social_insurance_overviews(company_id);
CREATE INDEX idx_health_insurance_overviews_company ON health_insurance_overviews(company_id);
CREATE INDEX idx_investment_documents_company    ON investment_documents(company_id);
CREATE INDEX idx_capital_income_entries_company  ON capital_income_entries(company_id);
CREATE INDEX idx_security_transactions_company   ON security_transactions(company_id);
```

The Down's reverse (DROP INDEX + the lossy `__tmp` dance per table) is documented as destructive and not expanded line-by-line here. If you want completeness, generate it from a script that reads `PRAGMA table_info` from the v024 fixture — that's the cheapest accurate way.

- [ ] **Step 4: Re-run the master assertion**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_AllPerCompanyTablesHaveColumn -v
```

Expected: every subtest from this task's table passes. Subtests for entities not yet partitioned (invoices, invoice_items, vat_*, etc.) still fail — those land in the next tasks.

- [ ] **Step 5: Commit**

```bash
git add internal/database/migrations/025_multi_company.sql internal/database/migrations_test.go
git commit -m "Migration 025: partition the 19 leaf-style per-company tables

Tables with no parent-child composite FK requirement get a flat
company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id)
plus an index. Existing rows backfill to the default company via
the DEFAULT 1 clause.

Adds two new master tests:
- TestMultiCompanyMigration_AllPerCompanyTablesHaveColumn: drives
  the rest of the partitioning work by listing every per-company
  table; subtests light up green as later tasks add the column
- TestMultiCompanyMigration_BackfillsLeafEntitiesToDefaultCompany:
  asserts no row escapes with company_id != 1"
```

---

### Task 9: Rebuild `invoice_sequences` with company-aware uniqueness

**Files:**
- Modify: `internal/database/migrations/025_multi_company.sql`
- Modify: `internal/database/migrations_test.go`

The existing `UNIQUE(prefix, year)` becomes `UNIQUE(company_id, prefix, year)`. SQLite can't add or replace a UNIQUE constraint on an existing table — the standard rename → recreate → copy → drop dance is required.

- [ ] **Step 1: Add the assertions**

In `migrations_test.go` add:

```go
func TestMultiCompanyMigration_InvoiceSequencesUniquePerCompany(t *testing.T) {
	db := openMigratedDB(t, true)

	// The v024 seed put two sequences in the default company; both should still exist.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM invoice_sequences WHERE company_id = 1`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("seeded sequences = %d, want 2", n)
	}

	// Insert a second company so we can test the new uniqueness rule.
	if _, err := db.Exec(`INSERT INTO companies (id, name, legal_name, ico, created_at, updated_at)
		VALUES (2, 'B', 'B', '99999999', datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("insert company 2: %v", err)
	}

	// (company_id=2, prefix='FV', year=2026) should be allowed — partitioned by company.
	if _, err := db.Exec(`INSERT INTO invoice_sequences (company_id, prefix, next_number, year, format_pattern)
		VALUES (2, 'FV', 1, 2026, '{prefix}{year}{number:04d}')`); err != nil {
		t.Errorf("company-partitioned sequence rejected: %v", err)
	}

	// A duplicate (company_id, prefix, year) must still fail.
	_, err := db.Exec(`INSERT INTO invoice_sequences (company_id, prefix, next_number, year, format_pattern)
		VALUES (1, 'FV', 99, 2026, 'x')`)
	if err == nil {
		t.Error("expected UNIQUE violation on duplicate (1, FV, 2026)")
	}
}
```

- [ ] **Step 2: Verify failure**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_InvoiceSequencesUniquePerCompany -v
```

Expected: FAIL — `invoice_sequences` has no `company_id` column yet.

- [ ] **Step 3: Append the rebuild to the migration**

In `025_multi_company.sql` Up block:

```sql

-- Rebuild: invoice_sequences gains company_id in its UNIQUE constraint.
ALTER TABLE invoice_sequences RENAME TO invoice_sequences__old;

CREATE TABLE invoice_sequences (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id     INTEGER NOT NULL REFERENCES companies(id),
	prefix         TEXT    NOT NULL,
	next_number    INTEGER NOT NULL,
	year           INTEGER NOT NULL,
	format_pattern TEXT    NOT NULL,
	UNIQUE(company_id, prefix, year)
);

INSERT INTO invoice_sequences (id, company_id, prefix, next_number, year, format_pattern)
SELECT id, 1, prefix, next_number, year, format_pattern FROM invoice_sequences__old;

DROP TABLE invoice_sequences__old;

CREATE INDEX idx_invoice_sequences_company ON invoice_sequences(company_id);
```

- [ ] **Step 4: Run the targeted test**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_InvoiceSequencesUniquePerCompany -v
```

Expected: PASS.

- [ ] **Step 5: Run all migration tests to confirm no regression**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_ -v
```

Expected: all green except subtests for entities not yet partitioned (invoices, vat_*, etc.).

- [ ] **Step 6: Commit**

```bash
git add internal/database/migrations/025_multi_company.sql internal/database/migrations_test.go
git commit -m "Migration 025: rebuild invoice_sequences with UNIQUE per company

SQLite cannot alter a UNIQUE constraint in place, so the table is
rebuilt: rename to __old, create the new shape with
UNIQUE(company_id, prefix, year), copy rows backfilling to
company_id = 1, drop __old. Two companies can now legitimately run
FV2026-0001 in parallel; duplicates within one company still fail."
```

---

### Task 10: Partition invoice graph with composite FK on items

**Files:**
- Modify: `internal/database/migrations/025_multi_company.sql`
- Modify: `internal/database/migrations_test.go`

Adds `company_id` to `invoices`, `invoice_items`, `recurring_invoices`, `recurring_invoice_items`. Parents (`invoices`, `recurring_invoices`) gain `UNIQUE(company_id, id)` as a composite-FK target via `CREATE UNIQUE INDEX` (no rebuild). Children (`invoice_items`, `recurring_invoice_items`) are rebuilt to add the composite FK that replaces the existing single-column FK to parent.

- [ ] **Step 1: Assertion for composite FK enforcement**

In `migrations_test.go`:

```go
func TestMultiCompanyMigration_CompositeFK_RejectsCrossCompanyChild(t *testing.T) {
	db := openMigratedDB(t, true)

	// Create a second company and an invoice owned by it.
	_, err := db.Exec(`INSERT INTO companies (id, name, legal_name, ico, created_at, updated_at)
		VALUES (2, 'B', 'B', '99999999', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert company: %v", err)
	}
	res, err := db.Exec(`INSERT INTO invoices (number, contact_id, company_id, issue_date, due_date, status, total_amount, created_at, updated_at)
		VALUES ('B-001', 1, 2, '2026-04-01', '2026-04-15', 'sent', 100000, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert invoice in company 2: %v", err)
	}
	invID, _ := res.LastInsertId()

	// Attempting to attach an invoice_item with mismatched company_id must fail at the FK level,
	// even if the parent invoice id is valid.
	_, err = db.Exec(`INSERT INTO invoice_items (invoice_id, company_id, description, quantity, unit_price, total)
		VALUES (?, 1, 'cross-company leak', 1, 100, 100)`, invID)
	if err == nil {
		t.Error("expected composite FK violation; invoice belongs to company 2 but item names company 1")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Expected: FAIL — `invoice_items` doesn't have `company_id` yet, or the composite FK isn't enforced.

- [ ] **Step 3: Append parent ADD COLUMN + composite-FK rebuild for children**

In `025_multi_company.sql` Up block:

```sql

-- Partition: invoices + recurring_invoices (parents — gain UNIQUE(company_id, id))
ALTER TABLE invoices           ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE recurring_invoices ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX        idx_invoices_company             ON invoices(company_id);
CREATE INDEX        idx_recurring_invoices_company   ON recurring_invoices(company_id);
CREATE UNIQUE INDEX idx_invoices_company_id          ON invoices(company_id, id);
CREATE UNIQUE INDEX idx_recurring_invoices_company_id ON recurring_invoices(company_id, id);

-- Rebuild: invoice_items gains company_id and a composite FK to invoices(company_id, id).
-- Replace `<existing-columns>` with the actual column list from the latest invoice_items
-- migration (currently 023_expense_items.sql created or 015_invoice_items.sql — verify by
-- running `PRAGMA table_info(invoice_items)` against a freshly migrated v024 DB).
ALTER TABLE invoice_items RENAME TO invoice_items__old;
CREATE TABLE invoice_items (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id  INTEGER NOT NULL REFERENCES companies(id),
	invoice_id  INTEGER NOT NULL,
	description TEXT    NOT NULL,
	quantity    INTEGER NOT NULL,
	unit_price  INTEGER NOT NULL,
	total       INTEGER NOT NULL,
	FOREIGN KEY (company_id, invoice_id) REFERENCES invoices(company_id, id) ON DELETE CASCADE
);
INSERT INTO invoice_items (id, company_id, invoice_id, description, quantity, unit_price, total)
SELECT id, 1, invoice_id, description, quantity, unit_price, total FROM invoice_items__old;
DROP TABLE invoice_items__old;
CREATE INDEX idx_invoice_items_company ON invoice_items(company_id);
CREATE INDEX idx_invoice_items_invoice ON invoice_items(invoice_id);

-- Rebuild: recurring_invoice_items gains company_id and a composite FK to recurring_invoices.
-- (Mirror the invoice_items pattern; replace columns with the real schema.)
ALTER TABLE recurring_invoice_items RENAME TO recurring_invoice_items__old;
CREATE TABLE recurring_invoice_items (
	id                    INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id            INTEGER NOT NULL REFERENCES companies(id),
	recurring_invoice_id  INTEGER NOT NULL,
	description           TEXT    NOT NULL,
	quantity              INTEGER NOT NULL,
	unit_price            INTEGER NOT NULL,
	total                 INTEGER NOT NULL,
	FOREIGN KEY (company_id, recurring_invoice_id) REFERENCES recurring_invoices(company_id, id) ON DELETE CASCADE
);
INSERT INTO recurring_invoice_items (id, company_id, recurring_invoice_id, description, quantity, unit_price, total)
SELECT id, 1, recurring_invoice_id, description, quantity, unit_price, total FROM recurring_invoice_items__old;
DROP TABLE recurring_invoice_items__old;
CREATE INDEX idx_recurring_invoice_items_company             ON recurring_invoice_items(company_id);
CREATE INDEX idx_recurring_invoice_items_recurring_invoice  ON recurring_invoice_items(recurring_invoice_id);
```

**Important:** before running, regenerate the rebuilt-table column lists from a freshly-migrated v024 DB. Use:

```bash
sqlite3 /tmp/v024.db < internal/database/testdata/v024_seed.sql 2>/dev/null  # may not work standalone
# Or, in a Go test:
db := openMigratedDB(t, false)
rows, _ := db.Query(`PRAGMA table_info(invoice_items)`)
// dump rows
```

The `<existing-columns>` lists in this plan are the spec's expected shape, not the verified ground truth.

- [ ] **Step 4: Run the targeted test**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_CompositeFK -v
```

Expected: PASS.

- [ ] **Step 5: Run the master assertion to confirm 4 more subtests light up**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_AllPerCompanyTablesHaveColumn -v
```

Expected: invoices/invoice_items/recurring_invoices/recurring_invoice_items now PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/database/migrations/025_multi_company.sql internal/database/migrations_test.go
git commit -m "Migration 025: partition invoice graph with composite FK on items

Parents (invoices, recurring_invoices) gain company_id + index +
UNIQUE(company_id, id) as composite-FK target — no table rebuild
needed for the parents.

Children (invoice_items, recurring_invoice_items) are rebuilt to
replace the single-column FK to parent with composite FK
(company_id, parent_id), so a cross-company link is physically
impossible. ON DELETE CASCADE preserves the existing parent-deletion
semantics for invoice line items."
```

---

### Task 11: Partition expense graph with composite FK on items

**Files:**
- Modify: `internal/database/migrations/025_multi_company.sql`
- Modify: `internal/database/migrations_test.go`

Same pattern as Task 10, applied to `expenses` (parent) and `expense_items` (child with composite FK).

- [ ] **Step 1: Assertion**

In `migrations_test.go`:

```go
func TestMultiCompanyMigration_CompositeFK_RejectsCrossCompanyExpenseItem(t *testing.T) {
	db := openMigratedDB(t, true)

	_, err := db.Exec(`INSERT INTO companies (id, name, legal_name, ico, created_at, updated_at)
		VALUES (2, 'B', 'B', '99999999', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert company: %v", err)
	}
	res, err := db.Exec(`INSERT INTO expenses (vendor, document_number, company_id, issue_date, total_amount, created_at, updated_at)
		VALUES ('Alza', 'AL/2026/X', 2, '2026-04-01', 1000, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert expense: %v", err)
	}
	expID, _ := res.LastInsertId()

	_, err = db.Exec(`INSERT INTO expense_items (expense_id, company_id, description, amount)
		VALUES (?, 1, 'cross-company leak', 100)`, expID)
	if err == nil {
		t.Error("expected composite FK violation on expense item from wrong company")
	}
}
```

- [ ] **Step 2: Verify failure → Step 3: Append the migration changes**

In `025_multi_company.sql` Up block:

```sql

-- Partition: expenses (parent — gains UNIQUE(company_id, id))
ALTER TABLE expenses ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX        idx_expenses_company    ON expenses(company_id);
CREATE UNIQUE INDEX idx_expenses_company_id ON expenses(company_id, id);

-- Rebuild: expense_items with composite FK
ALTER TABLE expense_items RENAME TO expense_items__old;
CREATE TABLE expense_items (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id  INTEGER NOT NULL REFERENCES companies(id),
	expense_id  INTEGER NOT NULL,
	description TEXT    NOT NULL,
	amount      INTEGER NOT NULL,
	FOREIGN KEY (company_id, expense_id) REFERENCES expenses(company_id, id) ON DELETE CASCADE
);
INSERT INTO expense_items (id, company_id, expense_id, description, amount)
SELECT id, 1, expense_id, description, amount FROM expense_items__old;
DROP TABLE expense_items__old;
CREATE INDEX idx_expense_items_company ON expense_items(company_id);
CREATE INDEX idx_expense_items_expense ON expense_items(expense_id);
```

Verify the `expense_items` column list against `PRAGMA table_info(expense_items)` on a freshly v024-migrated DB; adapt if it has more columns (e.g., VAT rate).

- [ ] **Step 4: Run targeted test, then master assertion. Step 5: Commit**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_ -v
git add internal/database/migrations/025_multi_company.sql internal/database/migrations_test.go
git commit -m "Migration 025: partition expense graph with composite FK on items

expenses gains company_id + UNIQUE(company_id, id); expense_items is
rebuilt to enforce the composite FK to expenses(company_id, id)."
```

---

### Task 12: Partition VAT filings — composite FK on vat_return children

**Files:**
- Modify: `internal/database/migrations/025_multi_company.sql`
- Modify: `internal/database/migrations_test.go`

`vat_returns` is a parent with two children that each need composite FKs against TWO different parents: their `vat_return_id` to `vat_returns`, and their `invoice_id` / `expense_id` to the source invoice/expense. The spec only requires composite FK to the immediate parent (`vat_return_invoices(company_id, vat_return_id) REFERENCES vat_returns(company_id, id)`); the cross-link to invoices/expenses uses single-column FK plus service-layer enforcement.

`vat_control_statements`, `vat_control_statement_lines`, `vies_summaries`, `vies_summary_lines` are partitioned but without composite FK (per spec — lines aren't aggregated for tax filings in a way that benefits from physical enforcement).

- [ ] **Step 1: Assertion**

In `migrations_test.go`:

```go
func TestMultiCompanyMigration_VATReturnCompositeFK(t *testing.T) {
	db := openMigratedDB(t, true)

	_, err := db.Exec(`INSERT INTO companies (id, name, legal_name, ico, created_at, updated_at)
		VALUES (2, 'B', 'B', '99999999', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	res, err := db.Exec(`INSERT INTO vat_returns (company_id, year, quarter, status, created_at, updated_at)
		VALUES (2, 2026, 1, 'draft', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert vat_return: %v", err)
	}
	vrID, _ := res.LastInsertId()

	// Cross-company attempt: vat_return belongs to company 2, line claims company 1.
	_, err = db.Exec(`INSERT INTO vat_return_invoices (vat_return_id, company_id, invoice_id, base_amount, vat_amount, vat_rate)
		VALUES (?, 1, 1, 100, 21, 21)`, vrID)
	if err == nil {
		t.Error("expected composite FK violation on vat_return_invoices")
	}
}
```

- [ ] **Step 2-3: Verify failure → append migration changes**

In `025_multi_company.sql` Up:

```sql

-- Partition: vat_returns (parent — gains UNIQUE(company_id, id) for composite-FK target)
ALTER TABLE vat_returns ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX        idx_vat_returns_company    ON vat_returns(company_id);
CREATE UNIQUE INDEX idx_vat_returns_company_id ON vat_returns(company_id, id);

-- Rebuild: vat_return_invoices with composite FK to vat_returns
ALTER TABLE vat_return_invoices RENAME TO vat_return_invoices__old;
CREATE TABLE vat_return_invoices (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id     INTEGER NOT NULL REFERENCES companies(id),
	vat_return_id  INTEGER NOT NULL,
	invoice_id     INTEGER NOT NULL REFERENCES invoices(id),
	base_amount    INTEGER NOT NULL,
	vat_amount     INTEGER NOT NULL,
	vat_rate       INTEGER NOT NULL,
	FOREIGN KEY (company_id, vat_return_id) REFERENCES vat_returns(company_id, id) ON DELETE CASCADE
);
INSERT INTO vat_return_invoices (id, company_id, vat_return_id, invoice_id, base_amount, vat_amount, vat_rate)
SELECT id, 1, vat_return_id, invoice_id, base_amount, vat_amount, vat_rate FROM vat_return_invoices__old;
DROP TABLE vat_return_invoices__old;
CREATE INDEX idx_vat_return_invoices_company    ON vat_return_invoices(company_id);
CREATE INDEX idx_vat_return_invoices_vat_return ON vat_return_invoices(vat_return_id);

-- Rebuild: vat_return_expenses with composite FK to vat_returns (mirror pattern)
ALTER TABLE vat_return_expenses RENAME TO vat_return_expenses__old;
CREATE TABLE vat_return_expenses (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id     INTEGER NOT NULL REFERENCES companies(id),
	vat_return_id  INTEGER NOT NULL,
	expense_id     INTEGER NOT NULL REFERENCES expenses(id),
	base_amount    INTEGER NOT NULL,
	vat_amount     INTEGER NOT NULL,
	vat_rate       INTEGER NOT NULL,
	FOREIGN KEY (company_id, vat_return_id) REFERENCES vat_returns(company_id, id) ON DELETE CASCADE
);
INSERT INTO vat_return_expenses (id, company_id, vat_return_id, expense_id, base_amount, vat_amount, vat_rate)
SELECT id, 1, vat_return_id, expense_id, base_amount, vat_amount, vat_rate FROM vat_return_expenses__old;
DROP TABLE vat_return_expenses__old;
CREATE INDEX idx_vat_return_expenses_company    ON vat_return_expenses(company_id);
CREATE INDEX idx_vat_return_expenses_vat_return ON vat_return_expenses(vat_return_id);

-- Partition (simple): vat_control_statements + lines, vies_summaries + lines
ALTER TABLE vat_control_statements      ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE vat_control_statement_lines ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE vies_summaries              ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE vies_summary_lines          ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_vat_control_statements_company      ON vat_control_statements(company_id);
CREATE INDEX idx_vat_control_statement_lines_company ON vat_control_statement_lines(company_id);
CREATE INDEX idx_vies_summaries_company              ON vies_summaries(company_id);
CREATE INDEX idx_vies_summary_lines_company          ON vies_summary_lines(company_id);
```

Again, verify the column lists against `PRAGMA table_info(...)` on the v024-migrated fixture before relying on them.

- [ ] **Step 4: Run all migration tests → Step 5: Commit**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_ -v
git add internal/database/migrations/025_multi_company.sql internal/database/migrations_test.go
git commit -m "Migration 025: partition VAT filings with composite FK on return lines

vat_returns gains company_id + UNIQUE(company_id, id); vat_return_invoices
and vat_return_expenses are rebuilt with composite FK to vat_returns
(physical enforcement of return-line integrity). vat_control_statements,
vat_control_statement_lines, vies_summaries, vies_summary_lines get
flat company_id columns — lines are scoped by their parent's
company already, no composite FK per spec."
```

---

### Task 13: Partition income tax + remaining line tables

**Files:**
- Modify: `internal/database/migrations/025_multi_company.sql`
- Modify: `internal/database/migrations_test.go`

The income-tax tables (`income_tax_returns`, `income_tax_return_invoices`, `income_tax_return_expenses`) are partitioned without composite FK — they're report-shaped rather than transactional, and the spec scopes composite FKs to the highest-risk paths only.

- [ ] **Step 1: Append the ALTERs**

In `025_multi_company.sql` Up:

```sql

-- Partition: income tax filings
ALTER TABLE income_tax_returns          ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE income_tax_return_invoices  ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
ALTER TABLE income_tax_return_expenses  ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id);
CREATE INDEX idx_income_tax_returns_company         ON income_tax_returns(company_id);
CREATE INDEX idx_income_tax_return_invoices_company ON income_tax_return_invoices(company_id);
CREATE INDEX idx_income_tax_return_expenses_company ON income_tax_return_expenses(company_id);
```

- [ ] **Step 2: Update the master test table**

In `migrations_test.go` inside `TestMultiCompanyMigration_AllPerCompanyTablesHaveColumn`, append `"income_tax_return_invoices", "income_tax_return_expenses"` to the `tables` slice.

- [ ] **Step 3: Run master assertion + commit**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_AllPerCompanyTablesHaveColumn -v
git add internal/database/migrations/025_multi_company.sql internal/database/migrations_test.go
git commit -m "Migration 025: partition income tax return line tables

income_tax_returns and its two line tables (invoices, expenses) gain
company_id + index. No composite FK — these are report-shaped tables
already scoped by their parent; service-layer enforcement plus the
leak detector cover the integrity concern."
```

---

### Task 14: Partition `settings` and add nullable `company_id` on `audit_log`

**Files:**
- Modify: `internal/database/migrations/025_multi_company.sql`
- Modify: `internal/database/migrations_test.go`

The `settings` table needs a rebuild because its existing `UNIQUE(key)` must become `UNIQUE(company_id, key)`. `audit_log` keeps its global semantics but gains a nullable `company_id` column for filtering.

- [ ] **Step 1: Assertions**

In `migrations_test.go`:

```go
func TestMultiCompanyMigration_SettingsPartitionedPerCompany(t *testing.T) {
	db := openMigratedDB(t, true)
	// The seed kept 'default_payment_method' (non-identity); it should now belong to company 1.
	var cid int64
	err := db.QueryRow(`SELECT company_id FROM settings WHERE key = 'default_payment_method'`).Scan(&cid)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if cid != 1 {
		t.Errorf("default_payment_method company_id = %d, want 1", cid)
	}

	// Two companies can each have their own value under the same key.
	_, err = db.Exec(`INSERT INTO companies (id, name, legal_name, ico, created_at, updated_at)
		VALUES (2, 'B', 'B', '99999999', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert company: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO settings (company_id, key, value) VALUES (2, 'default_payment_method', 'cash')`); err != nil {
		t.Errorf("expected company-partitioned settings to allow same key in another company: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO settings (company_id, key, value) VALUES (1, 'default_payment_method', 'duplicate')`); err == nil {
		t.Error("expected UNIQUE violation on duplicate (1, default_payment_method)")
	}
}

func TestMultiCompanyMigration_AuditLogHasNullableCompanyID(t *testing.T) {
	db := openMigratedDB(t, true)
	// Inserting an audit row without company_id must succeed (column is nullable).
	if _, err := db.Exec(`INSERT INTO audit_log (action, entity_kind, entity_id, created_at)
		VALUES ('system.boot', 'system', 0, datetime('now'))`); err != nil {
		t.Errorf("nullable company_id rejected null: %v", err)
	}
	// And with a value, it must persist.
	if _, err := db.Exec(`INSERT INTO audit_log (action, entity_kind, entity_id, company_id, created_at)
		VALUES ('company.create', 'company', 1, 1, datetime('now'))`); err != nil {
		t.Errorf("company_id value rejected: %v", err)
	}
}
```

- [ ] **Step 2-3: Verify failure → append migration changes**

```sql

-- Rebuild: settings gains company_id and UNIQUE(company_id, key)
ALTER TABLE settings RENAME TO settings__old;
CREATE TABLE settings (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	company_id INTEGER NOT NULL REFERENCES companies(id),
	key        TEXT    NOT NULL,
	value      TEXT,
	UNIQUE(company_id, key)
);
INSERT INTO settings (id, company_id, key, value)
SELECT id, 1, key, value FROM settings__old;
DROP TABLE settings__old;
CREATE INDEX idx_settings_company ON settings(company_id);

-- Partition: audit_log gains nullable company_id (kept global; column is a filter value).
ALTER TABLE audit_log ADD COLUMN company_id INTEGER REFERENCES companies(id);
CREATE INDEX idx_audit_log_company ON audit_log(company_id);
```

If your `audit_log` schema names the kind column differently (e.g. `entity_type` instead of `entity_kind`), adapt the test's INSERT to match.

- [ ] **Step 4: Run migration tests → Step 5: Commit**

```bash
CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigration_ -v
git add internal/database/migrations/025_multi_company.sql internal/database/migrations_test.go
git commit -m "Migration 025: partition settings; add nullable company_id to audit_log

settings rebuilt with UNIQUE(company_id, key) so each company keeps
its own email templates, defaults, and Czech office codes. Existing
rows backfill to company 1 — the same default-company that owns
the upgraded identity data.

audit_log gains a nullable company_id column (with index) so the
global /audit-log endpoint can filter by company without losing
its cross-company nature."
```

---

### Task 15: Production-sized migration test

**Files:**
- Modify: `internal/database/migrations_test.go` (fill in the previously-skipped test)
- Create: `internal/database/testdata/v024_production_seed.go` (Go-generated fixture; large SQL would bloat the repo)

The reviewer asked for frequent runs of a large-fixture migration during the rename-copy-drop work. Now that all the renames are landed, populate the test.

- [ ] **Step 1: Generator-style fixture**

`internal/database/testdata/v024_production_seed.go`:

```go
package testdata

import (
	"database/sql"
	"fmt"
)

// SeedProductionSized populates a v024-shaped DB with ~5k invoices,
// ~10k invoice items, ~2.5k contacts, and ~5k expenses across one
// default company's settings. Used by the production-sized migration test.
func SeedProductionSized(db *sql.DB) error {
	stmts := []string{
		// Identity settings (subset; same shape as v024_seed.sql)
		`INSERT INTO settings (key, value) VALUES ('company_name', 'Bench Co'), ('ico', '99999999'), ('vat_registered', '1'), ('dic', 'CZ99999999')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("seed identity: %w", err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Contacts: 2500
	for i := 1; i <= 2500; i++ {
		if _, err := tx.Exec(`INSERT INTO contacts (id, name, ico, email, created_at, updated_at)
			VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
			i, fmt.Sprintf("Contact %d", i), fmt.Sprintf("%08d", 10000000+i), fmt.Sprintf("c%d@example.cz", i)); err != nil {
			return fmt.Errorf("contact %d: %w", i, err)
		}
	}

	// Invoice sequence
	if _, err := tx.Exec(`INSERT INTO invoice_sequences (id, prefix, next_number, year, format_pattern)
		VALUES (1, 'FV', 5001, 2026, '{prefix}{year}{number:04d}')`); err != nil {
		return err
	}

	// Invoices: 5000, two line items each
	for i := 1; i <= 5000; i++ {
		if _, err := tx.Exec(`INSERT INTO invoices (id, number, contact_id, issue_date, due_date, status, total_amount, created_at, updated_at)
			VALUES (?, ?, ?, '2026-01-01', '2026-01-15', 'paid', ?, datetime('now'), datetime('now'))`,
			i, fmt.Sprintf("FV2026%04d", i), 1+(i%2500), 100000+i*100); err != nil {
			return fmt.Errorf("invoice %d: %w", i, err)
		}
		for j := 1; j <= 2; j++ {
			itemID := (i-1)*2 + j
			if _, err := tx.Exec(`INSERT INTO invoice_items (id, invoice_id, description, quantity, unit_price, total)
				VALUES (?, ?, ?, 1, ?, ?)`, itemID, i, fmt.Sprintf("line %d", j), 50000+j, 50000+j); err != nil {
				return fmt.Errorf("invoice_item %d: %w", itemID, err)
			}
		}
	}

	// Expenses: 5000
	for i := 1; i <= 5000; i++ {
		if _, err := tx.Exec(`INSERT INTO expenses (id, vendor, document_number, issue_date, total_amount, created_at, updated_at)
			VALUES (?, ?, ?, '2026-01-01', ?, datetime('now'), datetime('now'))`,
			i, fmt.Sprintf("Vendor %d", i%100), fmt.Sprintf("DOC/%04d", i), 10000+i); err != nil {
			return fmt.Errorf("expense %d: %w", i, err)
		}
	}

	return tx.Commit()
}
```

- [ ] **Step 2: Replace the skipped production test**

In `migrations_test.go`:

```go
func TestMultiCompanyMigrationProductionSized(t *testing.T) {
	if os.Getenv("ZFAKTURY_RUN_BIG_MIGRATION_TEST") == "" {
		t.Skip("set ZFAKTURY_RUN_BIG_MIGRATION_TEST=1 to enable")
	}

	tmp := filepath.Join(t.TempDir(), "big.db")
	db, err := sql.Open("sqlite", tmp+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	migrateUpTo(t, db, 24)
	if err := testdata.SeedProductionSized(db); err != nil {
		t.Fatalf("seed: %v", err)
	}

	start := time.Now()
	migrateUpTo(t, db, 25)
	elapsed := time.Since(start)

	if elapsed > 30*time.Second {
		t.Errorf("migration took %s, want < 30s", elapsed)
	}
	t.Logf("migration 025 over ~5k invoices / ~10k items / ~5k expenses: %s", elapsed)

	// Row counts unchanged.
	for _, c := range []struct {
		table string
		want  int
	}{
		{"invoices", 5000}, {"invoice_items", 10000},
		{"expenses", 5000}, {"contacts", 2500},
	} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + c.table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", c.table, err)
		}
		if n != c.want {
			t.Errorf("%s count = %d, want %d", c.table, n, c.want)
		}
	}

	// All composite FKs are real: cross-company link must fail.
	if _, err := db.Exec(`INSERT INTO companies (id, name, legal_name, ico, created_at, updated_at)
		VALUES (2, 'B', 'B', '88888888', datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("insert company: %v", err)
	}
	_, err = db.Exec(`INSERT INTO invoice_items (invoice_id, company_id, description, quantity, unit_price, total)
		VALUES (1, 2, 'leak', 1, 1, 1)`)
	if err == nil {
		t.Error("expected composite FK violation under production-sized load")
	}
}
```

Add `_ "internal/database/testdata"` if Go modules require the package import path; in practice this file is inside the same module so `import "github.com/zajca/zfaktury/internal/database/testdata"` works.

- [ ] **Step 3: Run the test once explicitly**

```bash
ZFAKTURY_RUN_BIG_MIGRATION_TEST=1 CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigrationProductionSized -v -timeout 90s
```

Expected: PASS, prints `migration 025 over ~5k invoices ... : <duration>`. If > 30s, profile and trim — the table rebuilds for `invoice_items`, `recurring_invoice_items`, `expense_items`, `vat_return_invoices`, `vat_return_expenses`, and `settings` plus the `invoice_sequences` rebuild are the candidates for cost.

- [ ] **Step 4: Document the env var in README**

Add a one-paragraph note in `README.md` under Testing:

```markdown
The production-sized migration test (`TestMultiCompanyMigrationProductionSized`)
seeds ~22k rows into a v024 fixture and runs migration 025 end-to-end. It is
gated behind the env var `ZFAKTURY_RUN_BIG_MIGRATION_TEST=1` so CI runs stay
fast; enable it locally before merging schema changes that touch existing
migrations.
```

- [ ] **Step 5: Commit**

```bash
git add internal/database/migrations_test.go internal/database/testdata/v024_production_seed.go README.md
git commit -m "Add production-sized migration test (env-gated)

Generator-style fixture seeds ~5k invoices, ~10k items, ~2.5k
contacts, ~5k expenses into a v024 DB, then runs migration 025
end-to-end. Asserts completion under 30s, exact row counts preserved,
and composite FK enforcement still active after the rebuild path.

Gated by ZFAKTURY_RUN_BIG_MIGRATION_TEST=1 so CI is not slowed; run
locally before merging schema changes."
```

---

**Phase 2 checkpoint:** Migration 025 complete. All per-company tables carry `company_id`, sequence partitioning works, composite FKs on the five aggregation paths are enforced. Settings table is partitioned per company; audit_log gains a nullable filter column. Production-sized fixture validates the migration scales.

Continue to Phase 3.

---

## Phase 3 — API surface

### Task 16: Company request-context + `WithCompany` middleware

**Files:**
- Create: `internal/handler/company_ctx.go`
- Create: `internal/handler/company_middleware.go`
- Create: `internal/handler/company_middleware_test.go`

The middleware reads `{companyID}` from the URL, validates via `CompanyService.Get`, stores the loaded `*domain.Company` under a typed context key, writes `X-Company-Id` to the response, and rejects bad inputs.

- [ ] **Step 1: Typed context key**

`internal/handler/company_ctx.go`:

```go
package handler

import (
	"context"
	"errors"

	"github.com/zajca/zfaktury/internal/domain"
)

type companyCtxKey struct{}

// CompanyFromContext retrieves the active company loaded by WithCompany.
// Returns an error only if called from a handler not mounted under
// the per-company subrouter — that's a programming error.
func CompanyFromContext(ctx context.Context) (*domain.Company, error) {
	c, ok := ctx.Value(companyCtxKey{}).(*domain.Company)
	if !ok {
		return nil, errors.New("no company in context (handler not mounted under WithCompany?)")
	}
	return c, nil
}

func contextWithCompany(ctx context.Context, c *domain.Company) context.Context {
	return context.WithValue(ctx, companyCtxKey{}, c)
}
```

- [ ] **Step 2: Middleware test**

`internal/handler/company_middleware_test.go`:

```go
package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
)

type stubCompanyService struct {
	companies map[int64]domain.Company
}

func (s *stubCompanyService) Get(_ context.Context, id int64) (domain.Company, error) {
	c, ok := s.companies[id]
	if !ok {
		return domain.Company{}, domain.ErrNotFound
	}
	return c, nil
}

func mountWithCompany(t *testing.T, svc CompanyResolver) http.Handler {
	r := chi.NewRouter()
	r.Route("/api/v1/companies/{companyID}", func(r chi.Router) {
		r.Use(WithCompany(svc))
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			c, err := CompanyFromContext(r.Context())
			if err != nil {
				t.Errorf("CompanyFromContext: %v", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("X-Returned-Company", strconv.FormatInt(c.ID, 10))
			w.WriteHeader(200)
		})
	})
	return r
}

func TestWithCompany_HappyPath(t *testing.T) {
	svc := &stubCompanyService{companies: map[int64]domain.Company{1: {ID: 1, Name: "A"}}}
	h := mountWithCompany(t, svc)

	req := httptest.NewRequest("GET", "/api/v1/companies/1/ping", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("code = %d, want 200", rr.Code)
	}
	if rr.Header().Get("X-Company-Id") != "1" {
		t.Errorf("X-Company-Id = %q, want 1", rr.Header().Get("X-Company-Id"))
	}
	if rr.Header().Get("X-Returned-Company") != "1" {
		t.Errorf("downstream handler did not receive company")
	}
}

func TestWithCompany_NotFound(t *testing.T) {
	svc := &stubCompanyService{companies: map[int64]domain.Company{}}
	h := mountWithCompany(t, svc)
	req := httptest.NewRequest("GET", "/api/v1/companies/42/ping", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 404 {
		t.Errorf("code = %d, want 404", rr.Code)
	}
}

func TestWithCompany_NonNumeric(t *testing.T) {
	svc := &stubCompanyService{companies: map[int64]domain.Company{}}
	h := mountWithCompany(t, svc)
	req := httptest.NewRequest("GET", "/api/v1/companies/abc/ping", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Errorf("code = %d, want 400", rr.Code)
	}
}
```

- [ ] **Step 3: Middleware implementation**

`internal/handler/company_middleware.go`:

```go
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
)

// CompanyResolver is the minimum CompanyService surface the middleware needs.
type CompanyResolver interface {
	Get(ctx context.Context, id int64) (domain.Company, error)
}

// WithCompany resolves {companyID} from the URL, validates the company exists
// and is not soft-deleted, stores it in the request context, and sets the
// X-Company-Id response header for client-side race-condition detection.
func WithCompany(svc CompanyResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := chi.URLParam(r, "companyID")
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id <= 0 {
				http.Error(w, "invalid company id", http.StatusBadRequest)
				return
			}
			c, err := svc.Get(r.Context(), id)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					http.Error(w, "company not found", http.StatusNotFound)
					return
				}
				http.Error(w, "internal error resolving company", http.StatusInternalServerError)
				return
			}
			w.Header().Set("X-Company-Id", strconv.FormatInt(c.ID, 10))
			next.ServeHTTP(w, r.WithContext(contextWithCompany(r.Context(), &c)))
		})
	}
}
```

- [ ] **Step 4: Run middleware tests**

```bash
CGO_ENABLED=0 go test ./internal/handler -run TestWithCompany_ -v
```

Expected: PASS, 3 tests.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/company_ctx.go internal/handler/company_middleware.go internal/handler/company_middleware_test.go
git commit -m "Add WithCompany middleware and typed request-context key

WithCompany resolves {companyID} from the URL, validates via
CompanyResolver (CompanyService.Get), stores the *domain.Company
in r.Context() under a typed key, and writes X-Company-Id on the
response so the frontend can detect mid-flight company switches.

Returns 400 for non-numeric / non-positive IDs and 404 for unknown
or soft-deleted companies, matching the spec's HTTP status table."
```

---

### Task 17: CompanyHandler CRUD (global tier)

**Files:**
- Create: `internal/handler/company_handler.go`
- Create: `internal/handler/company_handler_test.go`
- Modify: `internal/handler/helpers.go` (add `CompanyDTO`, request DTOs)

- [ ] **Step 1: Add the DTOs**

In `internal/handler/helpers.go` append:

```go
// CompanyDTO is the JSON shape of a company in API responses.
type CompanyDTO struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	LegalName     string  `json:"legal_name"`
	ICO           string  `json:"ico"`
	DIC           string  `json:"dic,omitempty"`
	VATRegistered bool    `json:"vat_registered"`
	Street        string  `json:"street,omitempty"`
	HouseNumber   string  `json:"house_number,omitempty"`
	City          string  `json:"city,omitempty"`
	ZIP           string  `json:"zip,omitempty"`
	Email         string  `json:"email,omitempty"`
	Phone         string  `json:"phone,omitempty"`
	FirstName     string  `json:"first_name,omitempty"`
	LastName      string  `json:"last_name,omitempty"`
	BankAccount   string  `json:"bank_account,omitempty"`
	BankCode      string  `json:"bank_code,omitempty"`
	IBAN          string  `json:"iban,omitempty"`
	SWIFT         string  `json:"swift,omitempty"`
	LogoPath      string  `json:"logo_path,omitempty"`
	AccentColor   string  `json:"accent_color,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func companyToDTO(c domain.Company) CompanyDTO {
	return CompanyDTO{
		ID: c.ID, Name: c.Name, LegalName: c.LegalName, ICO: c.ICO, DIC: c.DIC,
		VATRegistered: c.VATRegistered,
		Street: c.Street, HouseNumber: c.HouseNumber, City: c.City, ZIP: c.ZIP,
		Email: c.Email, Phone: c.Phone,
		FirstName: c.FirstName, LastName: c.LastName,
		BankAccount: c.BankAccount, BankCode: c.BankCode, IBAN: c.IBAN, SWIFT: c.SWIFT,
		LogoPath: c.LogoPath, AccentColor: c.AccentColor,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
}

func dtoToCompany(d CompanyDTO) domain.Company {
	return domain.Company{
		ID: d.ID, Name: d.Name, LegalName: d.LegalName, ICO: d.ICO, DIC: d.DIC,
		VATRegistered: d.VATRegistered,
		Street: d.Street, HouseNumber: d.HouseNumber, City: d.City, ZIP: d.ZIP,
		Email: d.Email, Phone: d.Phone,
		FirstName: d.FirstName, LastName: d.LastName,
		BankAccount: d.BankAccount, BankCode: d.BankCode, IBAN: d.IBAN, SWIFT: d.SWIFT,
		LogoPath: d.LogoPath, AccentColor: d.AccentColor,
	}
}
```

- [ ] **Step 2: Handler test**

`internal/handler/company_handler_test.go`:

```go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

type stubCompanyHandlerSvc struct {
	created     domain.Company
	listResult  []domain.Company
	getErr      error
	deleteErr   error
}

func (s *stubCompanyHandlerSvc) Create(_ context.Context, c domain.Company) (int64, error) {
	s.created = c
	return 42, nil
}
func (s *stubCompanyHandlerSvc) Get(_ context.Context, id int64) (domain.Company, error) {
	if s.getErr != nil {
		return domain.Company{}, s.getErr
	}
	return domain.Company{ID: id, Name: "X", LegalName: "X", ICO: "1"}, nil
}
func (s *stubCompanyHandlerSvc) List(_ context.Context) ([]domain.Company, error) { return s.listResult, nil }
func (s *stubCompanyHandlerSvc) Update(_ context.Context, _ domain.Company) error  { return nil }
func (s *stubCompanyHandlerSvc) Delete(_ context.Context, _ int64) error           { return s.deleteErr }

func TestCompanyHandler_Create_Returns201WithLocation(t *testing.T) {
	svc := &stubCompanyHandlerSvc{}
	h := NewCompanyHandler(svc)
	body, _ := json.Marshal(CompanyDTO{Name: "A", LegalName: "A", ICO: "12345678"})
	req := httptest.NewRequest("POST", "/companies", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != 201 {
		t.Errorf("code = %d, want 201", rr.Code)
	}
	if rr.Header().Get("Location") != "/api/v1/companies/42" {
		t.Errorf("Location = %q", rr.Header().Get("Location"))
	}
}

func TestCompanyHandler_Delete_LastCompany_Returns409(t *testing.T) {
	svc := &stubCompanyHandlerSvc{deleteErr: domain.ErrLastCompany}
	h := NewCompanyHandler(svc)
	req := httptest.NewRequest("DELETE", "/companies/1", nil)
	rr := httptest.NewRecorder()
	h.Delete(rr, req)
	if rr.Code != 409 {
		t.Errorf("code = %d, want 409", rr.Code)
	}
}

func TestCompanyHandler_Delete_InUse_Returns409(t *testing.T) {
	svc := &stubCompanyHandlerSvc{deleteErr: domain.ErrInUse}
	h := NewCompanyHandler(svc)
	req := httptest.NewRequest("DELETE", "/companies/1", nil)
	rr := httptest.NewRecorder()
	h.Delete(rr, req)
	if rr.Code != 409 {
		t.Errorf("code = %d, want 409", rr.Code)
	}
}

func TestCompanyHandler_Get_NotFound_Returns404(t *testing.T) {
	svc := &stubCompanyHandlerSvc{getErr: domain.ErrNotFound}
	h := NewCompanyHandler(svc)
	req := httptest.NewRequest("GET", "/companies/99", nil)
	// chi normally injects URL params; for an isolated handler test, set it manually.
	setURLParam(req, "id", "99")
	rr := httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != 404 {
		t.Errorf("code = %d, want 404", rr.Code)
	}
}

func TestCompanyHandler_Get_OtherError_Returns500(t *testing.T) {
	svc := &stubCompanyHandlerSvc{getErr: errors.New("boom")}
	h := NewCompanyHandler(svc)
	req := httptest.NewRequest("GET", "/companies/1", nil)
	setURLParam(req, "id", "1")
	rr := httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("code = %d, want 500", rr.Code)
	}
}
```

The `setURLParam` helper exists already in `internal/handler/test_helpers.go` (if not, copy the pattern from any existing handler test that uses chi URL params in an isolated test).

- [ ] **Step 3: Handler implementation**

`internal/handler/company_handler.go`:

```go
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
)

// CompanyHandlerService is the surface CompanyHandler needs from the service layer.
type CompanyHandlerService interface {
	Create(ctx context.Context, c domain.Company) (int64, error)
	Get(ctx context.Context, id int64) (domain.Company, error)
	List(ctx context.Context) ([]domain.Company, error)
	Update(ctx context.Context, c domain.Company) error
	Delete(ctx context.Context, id int64) error
}

type CompanyHandler struct {
	svc CompanyHandlerService
}

func NewCompanyHandler(svc CompanyHandlerService) *CompanyHandler {
	return &CompanyHandler{svc: svc}
}

// Routes returns the global-tier company routes (mounted at /api/v1/companies).
func (h *CompanyHandler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
}

func (h *CompanyHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dtos := make([]CompanyDTO, 0, len(list))
	for _, c := range list {
		dtos = append(dtos, companyToDTO(c))
	}
	writeJSON(w, http.StatusOK, dtos)
}

func (h *CompanyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto CompanyDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	id, err := h.svc.Create(r.Context(), dtoToCompany(dto))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v1/companies/%d", id))
	dto.ID = id
	writeJSON(w, http.StatusCreated, dto)
}

func (h *CompanyHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, companyToDTO(c))
}

func (h *CompanyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var dto CompanyDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	dto.ID = id
	if err := h.svc.Update(r.Context(), dtoToCompany(dto)); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			http.Error(w, "not found", http.StatusNotFound)
		case errors.Is(err, domain.ErrInvalidInput):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CompanyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			http.Error(w, "not found", http.StatusNotFound)
		case errors.Is(err, domain.ErrLastCompany), errors.Is(err, domain.ErrInUse):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

`writeJSON` already exists in `helpers.go`; reuse it.

- [ ] **Step 4: Run tests + commit**

```bash
CGO_ENABLED=0 go test ./internal/handler -run TestCompanyHandler_ -v
git add internal/handler/company_handler.go internal/handler/company_handler_test.go internal/handler/helpers.go
git commit -m "Add CompanyHandler with full CRUD + status code mapping

POST /companies -> 201 + Location header
GET  /companies, /companies/{id}
PUT  /companies/{id} -> 204 / 400 / 404
DELETE /companies/{id} -> 204 / 404 / 409 (last company or in use)

ErrInvalidInput -> 400, ErrNotFound -> 404, ErrLastCompany /
ErrInUse -> 409. CompanyDTO mirrors the domain shape; identity
fields stay first, optional fields use omitempty so empty company
records stay readable in API output."
```

---

### Task 18: Audit log company filter

**Files:**
- Modify: `internal/repository/audit_log_repo.go` (add `?company_id=` filter)
- Modify: `internal/handler/audit_handler.go` (parse query param, pass through)
- Modify: matching test files

- [ ] **Step 1: Test for filter behavior**

In `internal/repository/audit_log_repo_test.go` (or create if missing):

```go
func TestAuditLogRepository_ListByCompany(t *testing.T) {
	// ... setup an in-memory DB with audit_log + companies tables ...
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	// Insert: one row for company 1, one for company 2, one with NULL company.
	_, _ = db.Exec(`INSERT INTO audit_log (action, entity_kind, entity_id, company_id, created_at)
		VALUES ('a', 'x', 1, 1, datetime('now')), ('b', 'x', 2, 2, datetime('now')), ('c', 'x', 3, NULL, datetime('now'))`)

	companyOne := int64(1)
	got, err := repo.List(ctx, AuditLogFilter{CompanyID: &companyOne})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].Action != "a" {
		t.Errorf("got %v, want [{Action: a}]", got)
	}

	// Without filter, all three are returned.
	got, _ = repo.List(ctx, AuditLogFilter{})
	if len(got) != 3 {
		t.Errorf("unfiltered = %d rows, want 3", len(got))
	}
}
```

- [ ] **Step 2: Add `CompanyID *int64` to the existing `AuditLogFilter` struct and extend the SQL**

In `internal/repository/audit_log_repo.go`:

```go
type AuditLogFilter struct {
	// ... existing fields ...
	CompanyID *int64
}

// inside List, when building the WHERE clause:
if f.CompanyID != nil {
	conds = append(conds, "company_id = ?")
	args = append(args, *f.CompanyID)
}
```

- [ ] **Step 3: Wire the handler**

In `internal/handler/audit_handler.go` `List`:

```go
if v := r.URL.Query().Get("company_id"); v != "" {
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid company_id", http.StatusBadRequest)
		return
	}
	filter.CompanyID = &id
}
```

- [ ] **Step 4: Run targeted tests + commit**

```bash
CGO_ENABLED=0 go test ./internal/repository -run TestAuditLogRepository_ListByCompany -v
CGO_ENABLED=0 go test ./internal/handler -run TestAuditHandler_ -v
git add internal/repository/audit_log_repo.go internal/repository/audit_log_repo_test.go internal/handler/audit_handler.go
git commit -m "Audit log: accept optional company_id filter

AuditLogFilter gains CompanyID *int64; non-nil pointer adds
'company_id = ?' to the WHERE clause. Handler parses
?company_id=<id> from the query string and passes it through.
Existing unfiltered behavior is unchanged for callers that omit
the parameter."
```

---

### Task 19: Mount per-company subrouter (no-op handlers first)

**Files:**
- Modify: `internal/handler/router.go`
- Modify: `internal/cli/serve.go` (instantiate `CompanyService` + `CompanyHandler`)

This task restructures the route tree without yet changing any per-company handler's internal logic. Every existing per-company handler gets mounted under `/api/v1/companies/{companyID}/...`, but the handler functions still ignore the company id — that's the next batch of tasks. After this commit, the routes are exposed but the data is still single-company.

- [ ] **Step 1: Wire CompanyService in `serve.go`**

In `internal/cli/serve.go`, where other services are constructed, add:

```go
companyRepo := repository.NewCompanyRepository(db)
companySvc := service.NewCompanyService(companyRepo, nil, auditSvc) // checkers wired in Phase 3 tail
companyHandler := handler.NewCompanyHandler(companySvc)
```

Adapt to the actual variable naming used in the file (the existing code wires `contactRepo`, `invoiceRepo`, etc. similarly).

- [ ] **Step 2: Restructure `router.go`**

Open `internal/handler/router.go` and locate the existing `r.Route("/api/v1", ...)` block (around line 200). Replace its body to introduce the two tiers:

```go
r.Route("/api/v1", func(r chi.Router) {
	// Global tier
	r.Get("/health", healthHandler)
	r.Route("/companies", companyHandler.Routes)        // new
	r.Mount("/ares",         aresHandler.Routes())
	r.Mount("/cnb",          cnbHandler.Routes())
	r.Mount("/vies-check",   viesCheckHandler.Routes())  // rename existing /vies if it currently means the sync lookup
	r.Get("/audit-log",      auditHandler.List)
	r.Mount("/backups",      backupHandler.Routes())

	// Per-company tier — middleware resolves and validates {companyID}
	r.Route("/companies/{companyID}", func(r chi.Router) {
		r.Use(handler.WithCompany(companySvc))
		r.Mount("/contacts",          contactHandler.Routes())
		r.Mount("/invoices",          invoiceHandler.Routes())
		r.Mount("/expenses",          expenseHandler.Routes())
		r.Mount("/expense-categories", categoryHandler.Routes())
		r.Mount("/sequences",         sequenceHandler.Routes())
		r.Mount("/recurring-invoices", recurringInvoiceHandler.Routes())
		r.Mount("/recurring-expenses", recurringExpenseHandler.Routes())
		r.Mount("/payment-reminders", reminderHandler.Routes())
		r.Mount("/imports",           importHandler.Routes())
		r.Mount("/email",             emailHandler.Routes())
		r.Mount("/settings",          settingsHandler.Routes())
		r.Mount("/tax",               taxHandler.Routes())
		r.Mount("/vat-return",        vatReturnHandler.Routes())
		r.Mount("/vat-control-statement", vatControlHandler.Routes())
		r.Mount("/vies-summary",      viesSummaryHandler.Routes())
		r.Mount("/investments",       investmentHandler.Routes())
		r.Mount("/reports",           reportHandler.Routes())
	})
})
```

Adapt the handler variable names to match the file's existing naming. The list of mounts mirrors the per-company tier in the spec.

- [ ] **Step 3: Run the full backend test suite**

```bash
CGO_ENABLED=0 go test -tags server ./... -count=1
```

Expected: PASS. Existing handler tests that don't construct full URLs may need adjustment if they relied on the old flat paths — fix them locally as you encounter failures.

- [ ] **Step 4: Smoke-test live**

```bash
make build-server
./zfaktury-server serve --host 127.0.0.1 --port 9999 --config /tmp/zft-smoke-config.toml &
SERVER_PID=$!
sleep 1
curl -sS http://127.0.0.1:9999/health
curl -sS http://127.0.0.1:9999/api/v1/companies
curl -sS -i http://127.0.0.1:9999/api/v1/companies/1/contacts
kill $SERVER_PID
```

Expected: `/health` → `{"status":"ok"}`; `/api/v1/companies` → `[]` (or a one-entry list if the DB has migrated v025 with seed); `/api/v1/companies/1/contacts` → `404 company not found` on a fresh DB, or list response with `X-Company-Id: 1` header on a migrated DB.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/router.go internal/cli/serve.go
git commit -m "Restructure routes into global + per-company tiers

/api/v1/companies is now the global CRUD for companies themselves.
Per-company resources (contacts, invoices, expenses, tax, VAT,
investments, reports, etc.) mount under /api/v1/companies/{companyID}/
guarded by the WithCompany middleware.

Handler internals still ignore the company id resolved by the
middleware — they will be threaded in Tasks 20-22. After this
commit the URL surface matches the spec; the data layer catches up
in the next batch."
```

---

### Task 20: Thread `companyID` through foundational repos (contacts, categories, sequences)

**Files:**
- Modify: `internal/repository/interfaces.go`
- Modify: `internal/repository/contact_repo.go` + `..._test.go`
- Modify: `internal/repository/category_repo.go` + `..._test.go` (expense_categories)
- Modify: `internal/repository/sequence_repo.go` + `..._test.go`
- Modify: `internal/service/contact_svc.go` (+ test), `category_svc.go` (+ test), `sequence_svc.go` (+ test)
- Modify: `internal/handler/contact_handler.go` (+ test), `category_handler.go` (+ test), `sequence_handler.go` (+ test)
- Modify: `internal/repository/leak_detector_test.go` (add three rows)

Pattern (apply identically per entity):

1. Repo method signature: `(ctx context.Context, companyID int64, ...)`.
2. Every SQL query gains `AND company_id = ?` in the WHERE and `company_id` in INSERT column lists.
3. Service: add `companyID int64` to every public method, pass through.
4. Handler: extract company from context (`handler.CompanyFromContext(r.Context())`), pass `c.ID` to service.
5. Add a row to the leak detector's table-driven test for this entity.

- [ ] **Step 1: Pick the smallest entity (contacts) and update its full vertical**

Open `internal/repository/contact_repo.go`. For each method (`Create`, `GetByID`, `List`, `Update`, `SoftDelete`):

Before:
```go
func (r *ContactRepository) Create(ctx context.Context, c domain.Contact) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO contacts (name, ico, dic, email, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.Name, c.ICO, c.DIC, c.Email, now, now)
	...
}
```

After:
```go
func (r *ContactRepository) Create(ctx context.Context, companyID int64, c domain.Contact) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO contacts (company_id, name, ico, dic, email, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		companyID, c.Name, c.ICO, c.DIC, c.Email, now, now)
	...
}
func (r *ContactRepository) GetByID(ctx context.Context, companyID int64, id int64) (domain.Contact, error) {
	row := r.db.QueryRowContext(ctx, `SELECT ... FROM contacts WHERE company_id = ? AND id = ? AND deleted_at IS NULL`, companyID, id)
	...
}
func (r *ContactRepository) List(ctx context.Context, companyID int64, filter ContactFilter) ([]domain.Contact, error) {
	conds := []string{"company_id = ?", "deleted_at IS NULL"}
	args := []any{companyID}
	// ... rest of existing filter building
}
```

The exact SQL changes are mechanical — for any query that didn't already filter by `deleted_at` you'll add `company_id = ?` as the first WHERE term; any INSERT needs `company_id` prepended.

- [ ] **Step 2: Update the repo's existing tests to pass `companyID = 1`**

Every existing repo test invocation gets `1` (or a constant `testCompanyID = int64(1)`) inserted at the new position.

- [ ] **Step 3: Update `ContactService` and `ContactHandler` to thread the company through**

In `internal/service/contact_svc.go`:

```go
func (s *ContactService) Create(ctx context.Context, companyID int64, c domain.Contact) (int64, error) {
	id, err := s.repo.Create(ctx, companyID, c)
	if err != nil {
		return 0, fmt.Errorf("creating contact: %w", err)
	}
	// existing audit log call now includes the company too
	_ = s.audit.Log(ctx, AuditEvent{Action: "contact.create", EntityID: id, EntityKind: "contact", CompanyID: companyID})
	return id, nil
}
// Same pattern for Get, List, Update, Delete.
```

In `internal/handler/contact_handler.go` `Create`:

```go
func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	company, err := CompanyFromContext(r.Context())
	if err != nil {
		http.Error(w, "no company in context", http.StatusInternalServerError)
		return
	}
	var dto ContactDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	id, err := h.svc.Create(r.Context(), company.ID, dtoToContact(dto))
	// ... existing response handling ...
}
```

Apply the same pattern to every method in `contact_handler.go`. Where the handler currently parses an entity ID from the URL (`GetByID`, `Update`, `Delete`), the company comes from context **and** the entity ID still comes from `chi.URLParam(r, "id")`.

- [ ] **Step 4: Update `EntityChecker` adapters for the CompanyService**

Now that the contact repo knows about companies, we can give the `CompanyService` a real `EntityChecker` for contacts:

```go
// internal/service/contact_svc.go
type ContactCompanyChecker struct {
	repo repository.ContactRepository
}

func NewContactCompanyChecker(repo repository.ContactRepository) *ContactCompanyChecker {
	return &ContactCompanyChecker{repo: repo}
}

func (c *ContactCompanyChecker) CountNonDeletedForCompany(ctx context.Context, companyID int64) (int, error) {
	// Reuses List with an empty filter — fine for the small N of contacts per company.
	list, err := c.repo.List(ctx, companyID, repository.ContactFilter{})
	if err != nil {
		return 0, err
	}
	return len(list), nil
}
```

Wire it in `internal/cli/serve.go`:

```go
companySvc := service.NewCompanyService(companyRepo, []service.EntityChecker{
	service.NewContactCompanyChecker(contactRepo),
	// More added in Task 21/22 as their repos are threaded.
}, auditSvc)
```

- [ ] **Step 5: Run the targeted test suite**

```bash
CGO_ENABLED=0 go test ./internal/repository -run TestContactRepository_ -v
CGO_ENABLED=0 go test ./internal/service -run TestContactService_ -v
CGO_ENABLED=0 go test ./internal/handler -run TestContactHandler_ -v
```

Expected: PASS for all. If any test still passes `c.Name` with positional args in the wrong slot, fix it.

- [ ] **Step 6: Add contact + category + sequence to the leak detector**

Open `internal/repository/leak_detector_test.go` and replace the skipped function:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"

	_ "modernc.org/sqlite"
)

type leakDetectorCase struct {
	name string
	// seed inserts a record in company `cid`, returns its id.
	seed func(t *testing.T, repo any, db *sql.DB, cid int64) int64
	// wrongCompanyGet calls Get with a wrong company id; must return ErrNotFound.
	wrongCompanyGet func(t *testing.T, repo any, db *sql.DB, wrongCid, recordID int64) error
}

// setupLeakDetectorDB returns a DB with the full v025 schema and two companies pre-seeded.
func setupLeakDetectorDB(t *testing.T) (*sql.DB, int64, int64) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// Apply migrations — reuse the embedded migrationsFS via goose.
	// (Helper analogous to migrateUpTo in internal/database tests; copy here or expose.)
	migrateUpToV025(t, db)
	for i := int64(1); i <= 2; i++ {
		_, err := db.Exec(`INSERT INTO companies (id, name, legal_name, ico, created_at, updated_at)
			VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
			i, "C"+string(rune('0'+i)), "C"+string(rune('0'+i)), "1111111"+string(rune('0'+i)))
		if err != nil {
			t.Fatalf("seed company %d: %v", i, err)
		}
	}
	return db, 1, 2
}

func TestCrossCompanyLeakDetection(t *testing.T) {
	cases := []leakDetectorCase{
		{
			name: "ContactRepository",
			seed: func(t *testing.T, _ any, db *sql.DB, cid int64) int64 {
				repo := NewContactRepository(db)
				id, err := repo.Create(context.Background(), cid, domain.Contact{Name: "seed"})
				if err != nil {
					t.Fatalf("seed: %v", err)
				}
				return id
			},
			wrongCompanyGet: func(t *testing.T, _ any, db *sql.DB, wrongCid, recordID int64) error {
				repo := NewContactRepository(db)
				_, err := repo.GetByID(context.Background(), wrongCid, recordID)
				return err
			},
		},
		{
			name: "CategoryRepository",
			seed: func(t *testing.T, _ any, db *sql.DB, cid int64) int64 {
				repo := NewCategoryRepository(db)
				id, err := repo.Create(context.Background(), cid, domain.Category{Name: "seed"})
				if err != nil {
					t.Fatalf("seed: %v", err)
				}
				return id
			},
			wrongCompanyGet: func(t *testing.T, _ any, db *sql.DB, wrongCid, recordID int64) error {
				repo := NewCategoryRepository(db)
				_, err := repo.GetByID(context.Background(), wrongCid, recordID)
				return err
			},
		},
		// More cases added in Task 21/22 (sequence, invoice, expense, vat_*, etc.)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, owner, other := setupLeakDetectorDB(t)
			id := c.seed(t, nil, db, owner)
			err := c.wrongCompanyGet(t, nil, db, other, id)
			if !errors.Is(err, domain.ErrNotFound) {
				t.Errorf("cross-company Get from company %d returned %v, want ErrNotFound", other, err)
			}
		})
	}
}
```

The `migrateUpToV025` helper is a copy of `migrateUpTo` from `internal/database/migrations_test.go` adapted to this package. If reuse is preferred, move the helper to a shared test-support package.

- [ ] **Step 7: Run the leak detector**

```bash
CGO_ENABLED=0 go test ./internal/repository -run TestCrossCompanyLeakDetection -v
```

Expected: PASS for ContactRepository and CategoryRepository.

- [ ] **Step 8: Commit**

```bash
git add internal/repository internal/service internal/handler internal/cli/serve.go
git commit -m "Thread companyID through contacts, categories, sequences

Vertical update for the three foundational per-company entities:
repository methods take companyID int64 as their second parameter,
services pass it through, handlers read it from CompanyFromContext.
The CompanyService.Delete EntityChecker for contacts is wired so
in-use protection now actually fires for contact-bearing companies.

Leak detector covers ContactRepository and CategoryRepository
cross-company Get; sequence and the larger entities are added in
follow-up tasks."
```

---

### Task 21: Thread `companyID` through invoice and expense verticals

**Files:**
- Modify: every file in `internal/repository/invoice*.go`, `internal/repository/expense*.go`, `internal/repository/recurring*.go`, `internal/repository/payment_reminder_repo.go`, `internal/repository/invoice_document_repo.go`, `internal/repository/invoice_status_history_repo.go`
- Matching services in `internal/service/`
- Matching handlers in `internal/handler/`
- `internal/repository/leak_detector_test.go` — add rows

Same pattern as Task 20, applied to the invoice/expense graph. This is the bulk of the repo churn.

- [ ] **Step 1-N: Apply the same five-step pattern per entity vertical**

For each of: `InvoiceRepository`, `InvoiceItemRepository`, `InvoiceStatusHistoryRepository`, `InvoiceDocumentRepository`, `RecurringInvoiceRepository`, `RecurringInvoiceItemRepository`, `ExpenseRepository`, `ExpenseItemRepository`, `ExpenseDocumentRepository`, `RecurringExpenseRepository`, `PaymentReminderRepository`:

1. Add `companyID int64` to every interface method (update `interfaces.go`).
2. Add `WHERE company_id = ?` (or `AND company_id = ?`) to every existing query.
3. Add `company_id` to every INSERT.
4. Update existing repo tests to pass `1` for the company arg.
5. Update the service to thread through.
6. Update the handler to extract from context.
7. Add a row to the leak detector.

The composite-FK enforcement from the migration means a malformed INSERT (e.g., child with wrong company) will fail at the DB layer; the repo's `WHERE company_id = ?` guards reads.

Commit per vertical group (one per repository) so the diff stays reviewable:

```bash
git add internal/repository/invoice_repo.go internal/repository/invoice_repo_test.go internal/service/invoice_svc.go internal/service/invoice_svc_test.go internal/handler/invoice_handler.go internal/handler/invoice_handler_test.go
git commit -m "Thread companyID through InvoiceRepository / Service / Handler"

# ... repeat per vertical ...
```

- [ ] **Step Final: Re-run the full backend suite + leak detector**

```bash
CGO_ENABLED=0 go test -tags server ./... -count=1
CGO_ENABLED=0 go test ./internal/repository -run TestCrossCompanyLeakDetection -v
```

Expected: PASS everywhere; leak detector now has ~11 subtests, all green.

---

### Task 22: Thread `companyID` through tax, VAT, investment verticals + populate integration test

**Files:**
- All remaining per-company repos / services / handlers
- `tests/integration/multicompany_test.go` (fill in the previously-skipped test)
- `internal/repository/leak_detector_test.go` (final entries)

The pattern is identical; this task wraps up the API surface and exercises the whole stack end-to-end.

- [ ] **Step 1-N: Apply the pattern to remaining verticals**

Each gets a single commit:
- Sequence (`SequenceRepository`)
- Settings (`SettingsRepository`)
- TaxYearSettings + TaxPrepayment
- VATReturn + children
- VATControlStatement + lines
- VIESSummary + lines
- IncomeTaxReturn + lines
- SocialInsuranceOverview + HealthInsuranceOverview
- TaxCredits (spouse, child, personal) + TaxDeductions + TaxDeductionDocuments
- InvestmentDocument + CapitalIncomeEntry + SecurityTransaction
- FakturoidImportLog

Each vertical adds its repo to the leak detector's case slice.

- [ ] **Step Final-1: Fill in the integration test**

`tests/integration/multicompany_test.go`:

```go
//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	// Plus any test helpers from your existing integration suite.
)

func TestMultiCompanyEndToEnd(t *testing.T) {
	app := startTestApp(t) // existing helper that runs migrations + wires services
	ctx := context.Background()

	// 1. Create two companies.
	companyA, err := app.companies.Create(ctx, domain.Company{Name: "A", LegalName: "A", ICO: "12345678"})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	companyB, err := app.companies.Create(ctx, domain.Company{Name: "B", LegalName: "B", ICO: "87654321"})
	if err != nil {
		t.Fatalf("create B: %v", err)
	}

	// 2. Contacts per company.
	contactA, err := app.contacts.Create(ctx, companyA, domain.Contact{Name: "Client X"})
	if err != nil {
		t.Fatalf("contact A: %v", err)
	}
	contactB, _ := app.contacts.Create(ctx, companyB, domain.Contact{Name: "Client Y"})

	// 3. Invoice in A referencing A's contact — succeeds.
	_, err = app.invoices.Create(ctx, companyA, domain.Invoice{ContactID: contactA, Number: "FV20260001"})
	if err != nil {
		t.Errorf("invoice A: %v", err)
	}

	// 4. Invoice in A referencing B's contact — fails.
	_, err = app.invoices.Create(ctx, companyA, domain.Invoice{ContactID: contactB, Number: "FV20260002"})
	if err == nil {
		t.Error("expected cross-company contact reference to fail")
	}

	// 5. Sequence collision OK.
	if _, err := app.sequences.Create(ctx, companyA, domain.InvoiceSequence{Prefix: "FV", Year: 2026, NextNumber: 1}); err != nil {
		t.Errorf("seq A: %v", err)
	}
	if _, err := app.sequences.Create(ctx, companyB, domain.InvoiceSequence{Prefix: "FV", Year: 2026, NextNumber: 1}); err != nil {
		t.Errorf("seq B (same prefix/year — should succeed cross-company): %v", err)
	}

	// 6. Cannot delete A while it has invoices.
	err = app.companies.Delete(ctx, companyA)
	if !errors.Is(err, domain.ErrInUse) {
		t.Errorf("delete with invoice: err = %v, want ErrInUse", err)
	}

	// 7. Cannot delete the last company (after we soft-delete B in cleanup).
	// (Test that path separately; here just check we have 2 active.)
	list, _ := app.companies.List(ctx)
	if len(list) != 2 {
		t.Errorf("active companies = %d, want 2", len(list))
	}
}
```

`startTestApp` should be the existing helper in the integration package; if absent, wire it up using the production code's `server.New` against a temp DB.

- [ ] **Step Final-2: Run the integration test + full suite**

```bash
CGO_ENABLED=0 go test -tags 'server integration' ./tests/integration -run TestMultiCompanyEndToEnd -v
CGO_ENABLED=0 go test -tags server ./... -count=1
```

Expected: both PASS.

- [ ] **Step Final-3: Commit**

```bash
git add tests/integration/multicompany_test.go internal/repository/leak_detector_test.go
git commit -m "Integration: end-to-end multi-company flow + final leak-detector entries

Exercises the full per-company path: create two companies, contacts,
invoices, cross-company reference rejection, sequence collision,
delete protection on a non-empty company, list returns active set.

Leak detector now covers every per-company repository with explicit
cross-company Get assertions."
```

---

**Phase 3 checkpoint:** Every repository, service, and handler is multi-company-aware. The leak detector covers all per-company entities; the integration test asserts the spec's edge cases end-to-end. Frontend is untouched so far — backend can be exercised via curl with explicit company-prefixed URLs.

Continue to Phase 4.

---

## Phase 4 — Frontend

### Task 23: Company type + global API client methods

**Files:**
- Modify: `frontend/src/lib/api/client.ts` (add `Company` type, `client.companies.*` methods)

The frontend's typed client gets `client.companies.list()` / `create` / `get` / `update` / `delete` as global methods (no company prefix needed — these manage companies themselves).

- [ ] **Step 1: Add the Company type**

In `frontend/src/lib/api/client.ts`, near other type exports:

```typescript
export interface Company {
    id: number;
    name: string;
    legal_name: string;
    ico: string;
    dic?: string;
    vat_registered: boolean;
    street?: string;
    house_number?: string;
    city?: string;
    zip?: string;
    email?: string;
    phone?: string;
    first_name?: string;
    last_name?: string;
    bank_account?: string;
    bank_code?: string;
    iban?: string;
    swift?: string;
    logo_path?: string;
    accent_color?: string;
    created_at: string;
    updated_at: string;
}

export interface NewCompany {
    name: string;
    legal_name: string;
    ico: string;
    dic?: string;
    vat_registered: boolean;
    // Optional address/bank/personal/presentation fields...
    street?: string; house_number?: string; city?: string; zip?: string;
    email?: string; phone?: string;
    first_name?: string; last_name?: string;
    bank_account?: string; bank_code?: string; iban?: string; swift?: string;
    logo_path?: string; accent_color?: string;
}
```

- [ ] **Step 2: Add the methods to the client**

Inside the `createClient` function (or whatever pattern the file uses), append a `companies` namespace:

```typescript
companies: {
    async list(): Promise<Company[]> {
        const r = await fetch('/api/v1/companies');
        if (!r.ok) throw new Error(`list companies: ${r.status}`);
        return r.json();
    },
    async get(id: number): Promise<Company> {
        const r = await fetch(`/api/v1/companies/${id}`);
        if (!r.ok) throw new Error(`get company ${id}: ${r.status}`);
        return r.json();
    },
    async create(input: NewCompany): Promise<Company> {
        const r = await fetch('/api/v1/companies', {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify(input),
        });
        if (!r.ok) throw new Error(`create company: ${r.status}`);
        return r.json();
    },
    async update(id: number, input: NewCompany): Promise<void> {
        const r = await fetch(`/api/v1/companies/${id}`, {
            method: 'PUT',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({ ...input, id }),
        });
        if (!r.ok) throw new Error(`update company ${id}: ${r.status}`);
    },
    async delete(id: number): Promise<void> {
        const r = await fetch(`/api/v1/companies/${id}`, { method: 'DELETE' });
        if (r.status === 409) {
            const body = await r.text();
            throw new Error(body || 'cannot delete: in use or last company');
        }
        if (!r.ok) throw new Error(`delete company ${id}: ${r.status}`);
    },
},
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/api/client.ts
git commit -m "Frontend: add Company type and client.companies.* methods

Global tier of the typed API client. Methods build /api/v1/companies
URLs directly (no company prefix). delete() surfaces the server's
409 as a thrown Error carrying the message text so the UI can render
'cannot delete: still in use' / 'cannot delete the last company'."
```

---

### Task 24: `currentCompany` Svelte store + tests

**Files:**
- Create: `frontend/src/lib/stores/currentCompany.svelte.ts`
- Create: `frontend/src/lib/stores/currentCompany.test.ts`

- [ ] **Step 1: Vitest test**

`frontend/src/lib/stores/currentCompany.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { currentCompany } from './currentCompany.svelte';
import type { Company } from '$lib/api/client';

const A: Company = { id: 1, name: 'A', legal_name: 'A', ico: '1', vat_registered: false, created_at: '', updated_at: '' };
const B: Company = { id: 2, name: 'B', legal_name: 'B', ico: '2', vat_registered: false, created_at: '', updated_at: '' };

beforeEach(() => {
    localStorage.clear();
    currentCompany.reset();
});

describe('currentCompany store', () => {
    it('starts empty', () => {
        expect(currentCompany.current).toBeNull();
        expect(currentCompany.companies).toEqual([]);
    });

    it('setCompanies populates the list', () => {
        currentCompany.setCompanies([A, B]);
        expect(currentCompany.companies).toHaveLength(2);
    });

    it('select sets current and persists to localStorage', () => {
        currentCompany.setCompanies([A, B]);
        currentCompany.select(2);
        expect(currentCompany.current?.id).toBe(2);
        expect(localStorage.getItem('zfaktury.company')).toBe('2');
    });

    it('restore returns id from localStorage', () => {
        localStorage.setItem('zfaktury.company', '2');
        expect(currentCompany.restoreSelection()).toBe(2);
    });

    it('restoreSelection returns null when nothing stored', () => {
        expect(currentCompany.restoreSelection()).toBeNull();
    });

    it('select with an unknown id leaves current null', () => {
        currentCompany.setCompanies([A]);
        currentCompany.select(99);
        expect(currentCompany.current).toBeNull();
    });
});
```

- [ ] **Step 2: Implementation**

`frontend/src/lib/stores/currentCompany.svelte.ts`:

```typescript
import { browser } from '$app/environment';
import type { Company } from '$lib/api/client';

const STORAGE_KEY = 'zfaktury.company';

let current = $state<Company | null>(null);
let companies = $state<Company[]>([]);

export const currentCompany = {
    get current() { return current; },
    get companies() { return companies; },

    setCompanies(list: Company[]) {
        companies = list;
        // If current is no longer in the list (e.g. soft-deleted), clear it.
        if (current && !list.find(c => c.id === current!.id)) {
            current = null;
            if (browser) localStorage.removeItem(STORAGE_KEY);
        }
    },

    select(id: number) {
        const found = companies.find(c => c.id === id);
        if (!found) return;
        current = found;
        if (browser) localStorage.setItem(STORAGE_KEY, String(id));
    },

    restoreSelection(): number | null {
        if (!browser) return null;
        const raw = localStorage.getItem(STORAGE_KEY);
        if (!raw) return null;
        const id = Number(raw);
        return Number.isFinite(id) && id > 0 ? id : null;
    },

    reset() {
        current = null;
        companies = [];
        if (browser) localStorage.removeItem(STORAGE_KEY);
    },
};
```

- [ ] **Step 3: Run vitest**

```bash
cd frontend && npm test -- src/lib/stores/currentCompany.test.ts
```

Expected: PASS, 6 tests.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/stores/currentCompany.svelte.ts frontend/src/lib/stores/currentCompany.test.ts
git commit -m "Frontend: currentCompany store with localStorage persistence

Svelte 5 rune-based store exposing current + companies as reactive
state. select() persists to localStorage; restoreSelection() reads
the stored id back on app startup. Setting a companies list that
no longer contains the active company clears the selection (so a
soft-deleted company doesn't linger as 'current')."
```

---

### Task 25: `CompanyHeader.svelte` component + test

**Files:**
- Create: `frontend/src/lib/components/CompanyHeader.svelte`
- Create: `frontend/src/lib/components/CompanyHeader.test.ts`

The visible switcher: current company name + chevron, click → dropdown with all companies + "Manage" + "+ Add company".

- [ ] **Step 1: Test**

`frontend/src/lib/components/CompanyHeader.test.ts`:

```typescript
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/svelte';
import { currentCompany } from '$lib/stores/currentCompany.svelte';
import CompanyHeader from './CompanyHeader.svelte';
import type { Company } from '$lib/api/client';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

const A: Company = { id: 1, name: 'Manas OSVČ', legal_name: 'A', ico: '1', vat_registered: false, created_at: '', updated_at: '' };
const B: Company = { id: 2, name: 'Manas s.r.o.', legal_name: 'B', ico: '2', vat_registered: false, created_at: '', updated_at: '' };

beforeEach(() => {
    currentCompany.reset();
    currentCompany.setCompanies([A, B]);
    currentCompany.select(1);
});

describe('CompanyHeader', () => {
    it('renders current company name', () => {
        render(CompanyHeader);
        expect(screen.getByText('Manas OSVČ')).toBeInTheDocument();
    });

    it('opens dropdown listing all companies', async () => {
        render(CompanyHeader);
        await fireEvent.click(screen.getByRole('button', { name: /manas osvč/i }));
        expect(screen.getByText('Manas s.r.o.')).toBeInTheDocument();
    });

    it('selects another company on click', async () => {
        render(CompanyHeader);
        await fireEvent.click(screen.getByRole('button', { name: /manas osvč/i }));
        await fireEvent.click(screen.getByText('Manas s.r.o.'));
        expect(currentCompany.current?.id).toBe(2);
    });

    it('navigates to /companies when Manage is clicked', async () => {
        const { goto } = await import('$app/navigation');
        render(CompanyHeader);
        await fireEvent.click(screen.getByRole('button', { name: /manas osvč/i }));
        await fireEvent.click(screen.getByText(/Spravovat/i));
        expect(goto).toHaveBeenCalledWith('/companies');
    });
});
```

- [ ] **Step 2: Component**

`frontend/src/lib/components/CompanyHeader.svelte`:

```svelte
<script lang="ts">
    import { goto } from '$app/navigation';
    import { currentCompany } from '$lib/stores/currentCompany.svelte';

    let open = $state(false);

    function toggle() { open = !open; }
    function close() { open = false; }

    function pick(id: number) {
        currentCompany.select(id);
        close();
    }

    function manage() {
        close();
        goto('/companies');
    }

    function add() {
        close();
        goto('/companies/new');
    }
</script>

<div class="relative inline-block">
    <button
        type="button"
        class="flex items-center gap-2 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium hover:bg-gray-50"
        onclick={toggle}
        aria-haspopup="listbox"
        aria-expanded={open}
    >
        <span>{currentCompany.current?.name ?? 'Žádná firma'}</span>
        <svg class="h-4 w-4 text-gray-500" viewBox="0 0 20 20" fill="currentColor">
            <path d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.06l3.71-3.83a.75.75 0 1 1 1.08 1.04l-4.25 4.39a.75.75 0 0 1-1.08 0L5.21 8.27a.75.75 0 0 1 .02-1.06z"/>
        </svg>
    </button>

    {#if open}
        <div
            role="presentation"
            class="fixed inset-0 z-40"
            onclick={close}
        ></div>
        <ul class="absolute right-0 z-50 mt-1 w-56 rounded-md border border-gray-200 bg-white py-1 shadow-lg">
            {#each currentCompany.companies as c}
                <li>
                    <button
                        type="button"
                        class="flex w-full items-center justify-between px-3 py-1.5 text-sm hover:bg-gray-50"
                        onclick={() => pick(c.id)}
                    >
                        <span>{c.name}</span>
                        {#if currentCompany.current?.id === c.id}
                            <span aria-label="aktivní" class="text-blue-600">✓</span>
                        {/if}
                    </button>
                </li>
            {/each}
            <li class="my-1 border-t border-gray-200"></li>
            <li>
                <button type="button" class="block w-full px-3 py-1.5 text-left text-sm text-gray-700 hover:bg-gray-50" onclick={manage}>
                    Spravovat firmy →
                </button>
            </li>
            <li>
                <button type="button" class="block w-full px-3 py-1.5 text-left text-sm text-blue-700 hover:bg-gray-50" onclick={add}>
                    + Přidat firmu
                </button>
            </li>
        </ul>
    {/if}
</div>
```

- [ ] **Step 3: Run tests**

```bash
cd frontend && npm test -- src/lib/components/CompanyHeader.test.ts
```

Expected: PASS, 4 tests.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/components/CompanyHeader.svelte frontend/src/lib/components/CompanyHeader.test.ts
git commit -m "Frontend: CompanyHeader component with dropdown switcher

Shows currentCompany.current.name with a chevron; click opens a
dropdown listing all companies (check on the active one), plus
'Spravovat firmy →' and '+ Přidat firmu' actions that route to
/companies and /companies/new respectively. Backdrop overlay closes
the dropdown on outside click."
```

---

### Task 26: Refactor typed API client per-company methods

**Files:**
- Modify: `frontend/src/lib/api/client.ts` (every per-company method)

Each per-company method (e.g., `client.invoices.list`, `client.contacts.create`) reads the active company id from `currentCompany.current.id`, builds the full path including the prefix, and on writes returns `{ data, submittedFor, respondedFor }`.

- [ ] **Step 1: Helper for per-company URLs**

In `client.ts` near the top:

```typescript
import { currentCompany } from '$lib/stores/currentCompany.svelte';

export class NoCompanyError extends Error {
    constructor() { super('no active company'); this.name = 'NoCompanyError'; }
}

function companyPrefix(): string {
    const id = currentCompany.current?.id;
    if (!id) throw new NoCompanyError();
    return `/api/v1/companies/${id}`;
}

export interface WriteResult<T> {
    data: T;
    submittedFor: number;
    respondedFor: number;
}

async function readJson<T>(path: string): Promise<T> {
    const r = await fetch(`${companyPrefix()}${path}`);
    if (!r.ok) throw new Error(`${path}: ${r.status}`);
    return r.json();
}

async function writeJson<T>(method: string, path: string, body: unknown): Promise<WriteResult<T>> {
    const submittedFor = currentCompany.current!.id;  // companyPrefix() already validated
    const r = await fetch(`/api/v1/companies/${submittedFor}${path}`, {
        method,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(body),
    });
    if (!r.ok) throw new Error(`${path}: ${r.status}`);
    const respondedFor = Number(r.headers.get('X-Company-Id')) || submittedFor;
    const data = r.status === 204 ? (undefined as T) : await r.json();
    return { data, submittedFor, respondedFor };
}
```

- [ ] **Step 2: Refactor each existing per-company method**

For each existing method on `client.contacts`, `client.invoices`, `client.expenses`, etc., rewrite to use the helpers:

```typescript
// Before:
list: async (filter: ContactFilter) => {
    const r = await fetch(`/api/v1/contacts?${qs(filter)}`);
    if (!r.ok) throw new Error(...);
    return r.json();
},

// After:
list: (filter: ContactFilter) => readJson<Contact[]>(`/contacts?${qs(filter)}`),
create: (input: NewContact) => writeJson<Contact>('POST', '/contacts', input),
update: (id: number, input: NewContact) => writeJson<void>('PUT', `/contacts/${id}`, input),
delete: (id: number) => writeJson<void>('DELETE', `/contacts/${id}`, undefined),
```

Repeat for every namespace under `client.*` that maps to a per-company route. Global namespaces (`client.companies`, `client.ares`, `client.cnb`, `client.backups`, `client.audit`, `client.viesCheck`) stay unchanged.

- [ ] **Step 3: Update callers that consumed write-method return shapes**

The shape changed from `Promise<T>` to `Promise<WriteResult<T>>`. Callers in form-submit handlers need to read `.data` for the entity and may want to compare `.submittedFor` / `.respondedFor` against the active company. The race-condition handling pattern is:

```typescript
const result = await client.invoices.create(payload);
if (result.submittedFor !== currentCompany.current?.id) {
    toast(`Uloženo do firmy ${nameOf(result.submittedFor)} – mezitím jste přepnuli na ${nameOf(currentCompany.current?.id)}`);
    return; // skip redirect
}
goto(`/invoices/${result.data.id}`);
```

Update the existing form-submit handlers across `frontend/src/routes/**/+page.svelte` to follow this pattern. Apply incrementally per page; one commit per page is fine.

- [ ] **Step 4: Run frontend tests**

```bash
cd frontend && npm test
```

Expected: existing tests for list/detail pages may need adjustment for the new method shapes. Fix in the same commit that touches each page.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/api/client.ts frontend/src/routes/
git commit -m "Frontend: thread currentCompany through every per-company API method

readJson / writeJson helpers build /api/v1/companies/{id}/... URLs
from the active company; writes return { data, submittedFor,
respondedFor } so form handlers can detect and explain mid-flight
company switches with a Czech toast.

Callers updated per-route to consume the new write shape; reads
keep the same Promise<T> shape they had before."
```

---

### Task 27: Root layout bootstrap + redirect to onboarding

**Files:**
- Modify: `frontend/src/routes/+layout.svelte`
- Modify: `frontend/src/routes/+layout.ts` (or create if missing)

On mount, load companies → restore selection → render. If zero companies, route to `/companies/new`.

- [ ] **Step 1: Implement the load**

In `frontend/src/routes/+layout.svelte`:

```svelte
<script lang="ts">
    import { onMount } from 'svelte';
    import { goto } from '$app/navigation';
    import { page } from '$app/state';
    import { client } from '$lib/api/client';
    import { currentCompany } from '$lib/stores/currentCompany.svelte';
    import CompanyHeader from '$lib/components/CompanyHeader.svelte';

    let { children } = $props();
    let booted = $state(false);

    onMount(async () => {
        const list = await client.companies.list();
        currentCompany.setCompanies(list);

        if (list.length === 0) {
            if (page.url.pathname !== '/companies/new') {
                await goto('/companies/new');
            }
            booted = true;
            return;
        }

        const restored = currentCompany.restoreSelection();
        const targetId = restored && list.find(c => c.id === restored) ? restored : list[0].id;
        currentCompany.select(targetId);
        booted = true;
    });
</script>

{#if !booted}
    <div role="status" class="flex h-screen items-center justify-center">
        <span class="sr-only">Načítání…</span>
        <div class="h-6 w-6 animate-spin rounded-full border-2 border-gray-300 border-t-blue-600"></div>
    </div>
{:else}
    <header class="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-2">
        <a href="/" class="text-lg font-semibold">ZFaktury</a>
        {#if currentCompany.companies.length > 0}
            <CompanyHeader />
        {/if}
    </header>
    <div class="flex">
        <!-- existing sidebar nav slot -->
        <main class="flex-1">{@render children()}</main>
    </div>
{/if}
```

If the existing layout already has a sidebar + content shell, splice the header in above without breaking that layout. The exact integration depends on the current layout file — keep the existing styling and only add the header + bootstrap logic.

- [ ] **Step 2: Verify the manual flow**

```bash
make dev
# Wait for both processes, then open http://localhost:5173
# 1. With an empty DB → should land on /companies/new
# 2. After creating one company → should land on the dashboard with that company in the header
# 3. Add a second company via /companies/new → header dropdown shows both
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/routes/+layout.svelte
git commit -m "Frontend: root layout bootstraps companies and mounts the switcher

On mount the layout calls client.companies.list, hydrates the
currentCompany store, and either selects the previously-active
company (via localStorage) or the first one. If the list is empty,
it routes to /companies/new for onboarding.

CompanyHeader renders to the right of the title bar; the rest of
the existing layout (sidebar + content) is untouched."
```

---

### Task 28: Companies management routes

**Files:**
- Create: `frontend/src/routes/companies/+page.svelte` (list + delete)
- Create: `frontend/src/routes/companies/new/+page.svelte` (create with optional ARES auto-fill)
- Create: `frontend/src/routes/companies/[id]/+page.svelte` (edit)
- Create: `frontend/src/routes/companies/+page.test.ts` etc.

These are global pages — they don't need an active company.

- [ ] **Step 1: List page** (`frontend/src/routes/companies/+page.svelte`):

```svelte
<script lang="ts">
    import { onMount } from 'svelte';
    import { goto } from '$app/navigation';
    import { client, type Company } from '$lib/api/client';
    import { currentCompany } from '$lib/stores/currentCompany.svelte';

    let companies = $state<Company[]>([]);
    let error = $state<string | null>(null);

    onMount(async () => {
        try {
            companies = await client.companies.list();
            currentCompany.setCompanies(companies);
        } catch (e) {
            error = (e as Error).message;
        }
    });

    async function remove(id: number) {
        if (!confirm('Opravdu smazat tuto firmu?')) return;
        try {
            await client.companies.delete(id);
            companies = await client.companies.list();
            currentCompany.setCompanies(companies);
        } catch (e) {
            alert((e as Error).message);
        }
    }
</script>

<div class="p-6">
    <div class="mb-4 flex items-center justify-between">
        <h1 class="text-xl font-semibold">Firmy</h1>
        <button class="rounded bg-blue-600 px-3 py-1.5 text-sm text-white" onclick={() => goto('/companies/new')}>+ Přidat firmu</button>
    </div>
    {#if error}<div role="alert" class="rounded bg-red-50 p-3 text-sm text-red-800">{error}</div>{/if}
    <ul class="divide-y divide-gray-200 rounded border border-gray-200 bg-white">
        {#each companies as c}
            <li class="flex items-center justify-between px-4 py-2">
                <button class="text-left" onclick={() => goto(`/companies/${c.id}`)}>
                    <div class="font-medium">{c.name}</div>
                    <div class="text-xs text-gray-500">IČO {c.ico}{c.dic ? ` · DIČ ${c.dic}` : ''}</div>
                </button>
                <button class="text-sm text-red-600 hover:underline" onclick={() => remove(c.id)}>Smazat</button>
            </li>
        {/each}
    </ul>
</div>
```

- [ ] **Step 2: New page** (`frontend/src/routes/companies/new/+page.svelte`):

```svelte
<script lang="ts">
    import { goto } from '$app/navigation';
    import { client, type NewCompany } from '$lib/api/client';
    import { currentCompany } from '$lib/stores/currentCompany.svelte';

    let form = $state<NewCompany>({
        name: '', legal_name: '', ico: '', vat_registered: false,
    });
    let error = $state<string | null>(null);
    let aresLoading = $state(false);

    async function aresLookup() {
        if (!form.ico) return;
        aresLoading = true;
        try {
            const data = await client.ares.lookup(form.ico);
            form.legal_name = data.name ?? form.legal_name;
            form.name = form.name || data.name?.split(' ')[0] || form.name;
            form.dic = data.dic;
            form.vat_registered = !!data.dic;
            form.street = data.street;
            form.city = data.city;
            form.zip = data.zip;
        } catch (e) {
            error = `ARES: ${(e as Error).message}`;
        } finally {
            aresLoading = false;
        }
    }

    async function submit() {
        error = null;
        try {
            const c = await client.companies.create(form);
            const updated = await client.companies.list();
            currentCompany.setCompanies(updated);
            currentCompany.select(c.id);
            goto('/');
        } catch (e) {
            error = (e as Error).message;
        }
    }
</script>

<div class="mx-auto max-w-xl p-6">
    <h1 class="mb-4 text-xl font-semibold">Nová firma</h1>
    {#if error}<div role="alert" class="mb-3 rounded bg-red-50 p-3 text-sm text-red-800">{error}</div>{/if}

    <form onsubmit={(e) => { e.preventDefault(); submit(); }} class="space-y-3">
        <label class="block text-sm">
            IČO
            <div class="flex gap-2">
                <input class="flex-1 rounded border px-2 py-1" bind:value={form.ico} />
                <button type="button" class="rounded bg-gray-100 px-3 text-sm" onclick={aresLookup} disabled={aresLoading}>
                    {aresLoading ? '…' : 'Načíst z ARES'}
                </button>
            </div>
        </label>
        <label class="block text-sm">Krátký název<input class="block w-full rounded border px-2 py-1" bind:value={form.name} required /></label>
        <label class="block text-sm">Plný název<input class="block w-full rounded border px-2 py-1" bind:value={form.legal_name} required /></label>
        <label class="block text-sm">DIČ<input class="block w-full rounded border px-2 py-1" bind:value={form.dic} /></label>
        <label class="flex items-center gap-2 text-sm">
            <input type="checkbox" bind:checked={form.vat_registered} />
            Plátce DPH
        </label>
        <!-- Address, bank, etc. — keep this simple for v1 -->
        <button type="submit" class="rounded bg-blue-600 px-4 py-2 text-sm text-white">Vytvořit</button>
    </form>
</div>
```

- [ ] **Step 3: Edit page** mirrors `new` but with a `client.companies.get(id)` on mount and `client.companies.update(id, form)` on submit. Skip the full code; the pattern is identical.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/routes/companies/
git commit -m "Frontend: companies management routes (list, new, edit)

/companies - list with delete + add
/companies/new - create form with optional ARES lookup by IČO; on
success, sets the newly-created company as active and routes home.
/companies/[id] - edit form populated from client.companies.get.

Pages use the existing accessibility patterns (role='alert' on
errors, sr-only loading text) and Czech UI labels matching the
rest of the app."
```

---

### Task 29: Refactor existing list/detail pages to refetch on company change

**Files:**
- Modify: every `frontend/src/routes/**/+page.svelte` that today follows the `onMount` + `$effect`-guarded-with-`mounted` pattern

Each page that lists per-company data adds `currentCompany.current?.id` to its `$effect` reactive deps so a switch triggers a re-fetch.

- [ ] **Step 1: Update the pattern on the invoices list page (as a canonical example)**

`frontend/src/routes/invoices/+page.svelte`:

```svelte
<script lang="ts">
    import { onMount } from 'svelte';
    import { client, type Invoice } from '$lib/api/client';
    import { currentCompany } from '$lib/stores/currentCompany.svelte';

    let invoices = $state<Invoice[]>([]);
    let filterStatus = $state('');
    let mounted = false;

    async function load() {
        if (!currentCompany.current) return;
        invoices = await client.invoices.list({ status: filterStatus });
    }

    onMount(() => {
        load();
        mounted = true;
    });

    $effect(() => {
        currentCompany.current?.id;
        filterStatus;
        if (!mounted) return;
        load();
    });
</script>

<!-- existing markup unchanged -->
```

- [ ] **Step 2: Apply the same change to every list/detail page**

Tables to touch (one commit per page if you want a clean trail):
`/contacts`, `/contacts/[id]`, `/invoices`, `/invoices/[id]`, `/expenses`, `/expenses/[id]`, `/recurring-invoices`, `/recurring-expenses`, `/tax`, `/vat`, `/reports`, `/dashboard`, `/sequences`, `/categories`, `/imports`, `/investments`, `/settings`.

- [ ] **Step 3: Test the existing component tests still pass**

```bash
cd frontend && npm test
```

Expected: PASS. Where component tests use the route pages directly, they'll need to seed `currentCompany` in a `beforeEach` — same pattern as Task 25's test for CompanyHeader.

- [ ] **Step 4: Commit (one or many)**

```bash
git add frontend/src/routes/
git commit -m "Frontend: list/detail pages refetch on currentCompany change

Every page that consumes per-company data gains currentCompany.current?.id
in its \$effect reactive deps. The mounted-guard pattern from CLAUDE.md
keeps the initial onMount load from double-firing.

Page tests updated to seed the currentCompany store in beforeEach
so render() sees an active company."
```

---

### Task 30: Frontend smoke test (manual + optional Playwright)

**Files:**
- (Optional) Create: `frontend/tests/e2e/multicompany.spec.ts` if the project already runs Playwright

If Playwright is available, add one happy-path spec; otherwise this task is a manual checklist captured in the PR description (Phase 5).

- [ ] **Step 1: Manual smoke checklist**

Run `make dev` and step through:

1. Empty DB → app routes to `/companies/new`.
2. Create "Manas OSVČ" with IČO + ARES lookup → routes home, header shows "Manas OSVČ".
3. Create a contact → appears in the contacts list.
4. Create an invoice for that contact → appears in invoice list.
5. From the dropdown, add a second company "Manas s.r.o.".
6. Switch via the dropdown → invoice list becomes empty (different company).
7. Create another contact in s.r.o. → only visible while s.r.o. is active.
8. Switch back to OSVČ → original contact and invoice reappear.
9. From `/companies`, try to delete OSVČ while it has an invoice → 409 with Czech message.
10. Delete the s.r.o.'s sole contact, then delete s.r.o. → success; only OSVČ remains.
11. Try to delete the last company → 409 (last-company guard).

- [ ] **Step 2: Commit (if Playwright spec)**

If the project uses Playwright, the spec lives next to existing e2e tests. Otherwise, document the checklist in `docs/superpowers/manual-tests/multi-company.md` and commit that.

```bash
git add docs/superpowers/manual-tests/multi-company.md
git commit -m "Add manual smoke-test checklist for multi-company switch flow"
```

---

**Phase 4 checkpoint:** Frontend bootstraps with the new company list, switcher works, every page refetches on switch, company CRUD pages exist, write actions handle the mid-flight switch race. The app is functionally complete.

Continue to Phase 5.

---

## Phase 5 — PR preparation

### Task 31: README + upgrade notes

**Files:**
- Modify: `README.md` (add Multi-Company section)
- Create: `docs/UPGRADING.md` (or append if it exists)

- [ ] **Step 1: README section**

In `README.md`, insert before "License":

```markdown
## Multi-Company Support

zfaktury can manage multiple legal entities — for example, an OSVČ
and an s.r.o. owned by the same person — from a single install.

The active company is selectable from the header dropdown and
persisted across sessions. Each company has its own contacts,
invoices, expenses, tax filings, and sequences. Switching is
instantaneous (no reload).

When upgrading from a single-company install, the database
migration creates a default company from your existing settings
(name, IČO, DIČ, bank details, etc.) and backfills every existing
record into it. You can rename it or add more companies after
the first launch.

To run the production-sized migration test locally before merging
schema changes:

\`\`\`bash
ZFAKTURY_RUN_BIG_MIGRATION_TEST=1 CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigrationProductionSized -v
\`\`\`
```

- [ ] **Step 2: Upgrading notes**

`docs/UPGRADING.md`:

```markdown
# Upgrading to multi-company (v0.0.51 and later)

Migration `025_multi_company.sql` is the first multi-company migration.
It rewrites the database in-place — back up your `.zfaktury/zfaktury.db`
before upgrading.

## What happens

1. A `companies` table is created.
2. A default company (id = 1) is created from your existing settings
   (company_name, ico, dic, address, bank details, etc.).
3. Every per-company table (contacts, invoices, expenses, tax filings,
   etc.) gains a `company_id` column, backfilled to 1.
4. Composite foreign keys are added to invoice_items, expense_items,
   recurring_invoice_items, vat_return_invoices, and vat_return_expenses
   for physical cross-company isolation.

## Downgrade is destructive

The `Down` migration restores the 17 identity keys from id = 1 only.
Any data you created in companies other than the first will be lost
on downgrade. Back up your database before testing the downgrade
path.

## Renaming the default company

After upgrade, visit \`/companies/[id]\` (clickable from the header
dropdown's "Spravovat firmy →" link) to rename or fill in additional
fields. You can add more companies from "+ Přidat firmu" — IČO
lookup against ARES auto-fills most fields.
```

- [ ] **Step 3: Commit**

```bash
git add README.md docs/UPGRADING.md
git commit -m "Document multi-company support and upgrade path

README gains a 'Multi-Company Support' section above License,
explaining the model and pointing at the env-gated production-sized
migration test. docs/UPGRADING.md spells out what migration 025
does, the destructive downgrade caveat, and how to rename the
auto-created default company post-upgrade."
```

---

### Task 32: Final verification + push + PR

- [ ] **Step 1: Full backend test suite**

```bash
CGO_ENABLED=0 go test -tags server ./... -count=1
```

Expected: all green.

- [ ] **Step 2: Backend coverage check**

```bash
CGO_ENABLED=0 go test -tags server -coverprofile=/tmp/cov.out ./...
go tool cover -func=/tmp/cov.out | tail -1
```

Expected: total coverage ≥ 80% (the project's pre-commit gate).

- [ ] **Step 3: Frontend test suite + type check**

```bash
cd frontend
npm test
npm run check
```

Expected: all green.

- [ ] **Step 4: Production-sized migration smoke run**

```bash
ZFAKTURY_RUN_BIG_MIGRATION_TEST=1 CGO_ENABLED=0 go test ./internal/database -run TestMultiCompanyMigrationProductionSized -v -timeout 90s
```

Expected: PASS, completes well under 30s.

- [ ] **Step 5: Build server binary; smoke-test against a v024 fixture DB**

```bash
make build-server
# Copy a real-data v0.0.50 DB into a scratch location, point a config at it, start, walk through the manual checklist from Task 30.
```

- [ ] **Step 6: Push the branch and open the PR**

```bash
git push -u origin feature/multi-company

gh pr create \
  --repo zajca/zfaktury \
  --base main \
  --head Maziak2520:feature/multi-company \
  --title "Add multi-company support" \
  --body "$(cat <<'EOF'
## Summary

Adds support for managing multiple legal entities (e.g. an OSVČ
plus an s.r.o.) inside a single zfaktury install. Strict
per-company data partitioning, header-dropdown switching, and
seamless auto-migration of existing single-company data.

## Spec and review trail

- Design: `docs/superpowers/specs/2026-05-24-multi-company-design.md`
- Review feedback (v1, v2): same directory
- Implementation plan: `docs/superpowers/plans/2026-05-24-multi-company.md`

## What changed

- New `companies` table with 22 columns (identity, address, bank,
  presentation, audit). 17 keys lifted out of `settings`.
- ~30 per-company tables gain `company_id INTEGER NOT NULL
  REFERENCES companies(id)` plus a covering index.
- Composite FKs on five aggregation paths (invoice_items,
  expense_items, recurring_invoice_items, vat_return_invoices,
  vat_return_expenses) so cross-company links are physically
  impossible — defense in depth over the service-layer guard.
- Backend routes restructured into a global tier
  (`/api/v1/companies`, `/api/v1/ares`, `/audit-log`, etc.) and a
  per-company tier under `/api/v1/companies/{companyID}/…`,
  resolved by a `WithCompany` middleware that also writes
  `X-Company-Id` for client-side race detection.
- Frontend: rune-based `currentCompany` store, header dropdown
  switcher, refactored typed API client returning
  `{ data, submittedFor, respondedFor }` for write actions so
  mid-flight switches surface a context-explaining toast instead
  of a wrong-list redirect.
- Migration 025 auto-creates the default company from existing
  settings on upgrade; fresh installs land on `/companies/new`.

## Test plan

- [x] Migration tests: default company seeded from settings,
      identity keys stripped, non-identity preserved, fresh install
      produces empty companies, every per-company table has
      `company_id`, composite FK enforcement
- [x] Production-sized migration test (~5k invoices) completes
      under 30s with composite FK still enforced
- [x] Repository leak detector: exhaustive cross-company Get/List/
      Update/Delete returns ErrNotFound for the wrong company id
- [x] Integration test: full end-to-end flow including delete
      protection and sequence collision
- [x] Frontend: store, header component, page refetch on switch
- [x] Manual smoke test (see `docs/superpowers/manual-tests/multi-company.md`):
      onboarding, add second company, switch via dropdown, delete
      protections

## Upgrade

See `docs/UPGRADING.md`. TL;DR: back up your DB; on first launch
the migration creates a default company from settings; downgrade
is destructive for users with more than one company.
EOF
)"
```

- [ ] **Step 7: Watch for AI review (per CLAUDE.md PR AI Review)**

```bash
# Poll for AI reviewer comments
gh pr view <pr-number> --repo zajca/zfaktury --json comments,reviews
# Address each before merge.
```

---

## Plan self-review

Final pass against the spec:

| Spec section | Plan task(s) covering it |
|---|---|
| Goals: one install → N companies | Tasks 2-4, 17 (CRUD), 27 (bootstrap) |
| Non-goal: no auth | (Not built; nothing to do) |
| Data model: companies table | Task 6 |
| Data model: 17 keys moved | Task 6 (DELETE FROM settings), Task 14 (settings partitioned) |
| Per-company tables list | Tasks 7-14 |
| Composite FKs on 5 paths | Tasks 10-12 |
| audit_log nullable column | Task 14 |
| Sequence `UNIQUE(company_id, prefix, year)` | Task 9 |
| Soft-delete: last-company + in-use | Task 4 (service), Task 17 (handler 409 mapping) |
| API tiers (global + per-company) | Task 19 |
| WithCompany middleware + typed ctx + X-Company-Id | Task 16 |
| Service/repo signatures gain companyID | Tasks 20-22 |
| Audit log company filter | Task 18 |
| currentCompany store | Task 24 |
| Header dropdown | Task 25 |
| Typed API client returns submittedFor / respondedFor | Task 26 |
| Bootstrap + empty-state redirect | Task 27 |
| Companies management routes | Task 28 |
| Refetch on switch | Task 29 |
| Migration test (correctness) | Tasks 5, 7-14 |
| Migration test (production-sized) | Task 15 |
| Leak detector | Tasks 20-22 |
| End-to-end integration test | Task 22 |
| Manual test plan | Task 30 |
| README + upgrade notes | Task 31 |

No gaps. The reviewer's sequencing request (write `migrations_test` and `leak_detector_test` early) is satisfied by Tasks 1 and 5 — both predate any mechanical repo work.

---

**Plan complete and saved to `docs/superpowers/plans/2026-05-24-multi-company.md`.**

Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration. Best for a plan this size because each task fits into a clean subagent context.

**2. Inline Execution** — Execute tasks in this session using `superpowers:executing-plans`, batch execution with checkpoints for review.

Which approach?
