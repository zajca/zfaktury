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
