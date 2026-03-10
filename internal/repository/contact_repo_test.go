package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestContactRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	c := &domain.Contact{
		Type:    domain.ContactTypeCompany,
		Name:    "Acme s.r.o.",
		ICO:     "12345678",
		DIC:     "CZ12345678",
		Street:  "Hlavni 1",
		City:    "Praha",
		ZIP:     "11000",
		Country: "CZ",
		Email:   "info@acme.cz",
	}

	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if c.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if c.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestContactRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	seeded := testutil.SeedContact(t, db, &domain.Contact{
		Name: "GetByID Test",
		ICO:  "11111111",
	})

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Name != "GetByID Test" {
		t.Errorf("Name = %q, want %q", got.Name, "GetByID Test")
	}
	if got.ICO != "11111111" {
		t.Errorf("ICO = %q, want %q", got.ICO, "11111111")
	}
}

func TestContactRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent contact")
	}
	if !strings.Contains(err.Error(), sql.ErrNoRows.Error()) {
		t.Errorf("expected sql.ErrNoRows in error, got: %v", err)
	}
}

func TestContactRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	seeded := testutil.SeedContact(t, db, &domain.Contact{Name: "Before Update"})

	seeded.Name = "After Update"
	seeded.Email = "updated@test.cz"
	if err := repo.Update(ctx, seeded); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Name != "After Update" {
		t.Errorf("Name = %q, want %q", got.Name, "After Update")
	}
	if got.Email != "updated@test.cz" {
		t.Errorf("Email = %q, want %q", got.Email, "updated@test.cz")
	}
}

func TestContactRepository_Delete_SoftDelete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	seeded := testutil.SeedContact(t, db, nil)

	if err := repo.Delete(ctx, seeded.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Should not be found via GetByID (filters deleted_at IS NULL).
	_, err := repo.GetByID(ctx, seeded.ID)
	if err == nil {
		t.Error("expected error when getting soft-deleted contact")
	}

	// Should still exist in DB.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM contacts WHERE id = ?", seeded.ID).Scan(&count); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if count != 1 {
		t.Errorf("expected contact to still exist in DB, count = %d", count)
	}
}

func TestContactRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent contact")
	}
}

func TestContactRepository_Delete_AlreadyDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	seeded := testutil.SeedContact(t, db, nil)
	if err := repo.Delete(ctx, seeded.ID); err != nil {
		t.Fatalf("first Delete() error: %v", err)
	}

	err := repo.Delete(ctx, seeded.ID)
	if err == nil {
		t.Error("expected error when deleting already-deleted contact")
	}
}

func TestContactRepository_FindByICO(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	testutil.SeedContact(t, db, &domain.Contact{Name: "ICO Test", ICO: "99887766"})

	got, err := repo.FindByICO(ctx, "99887766")
	if err != nil {
		t.Fatalf("FindByICO() error: %v", err)
	}
	if got.Name != "ICO Test" {
		t.Errorf("Name = %q, want %q", got.Name, "ICO Test")
	}
}

func TestContactRepository_FindByICO_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	_, err := repo.FindByICO(ctx, "00000000")
	if err == nil {
		t.Error("expected error for non-existent ICO")
	}
}

func TestContactRepository_List_All(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	testutil.SeedContact(t, db, &domain.Contact{Name: "Alpha"})
	testutil.SeedContact(t, db, &domain.Contact{Name: "Beta"})
	testutil.SeedContact(t, db, &domain.Contact{Name: "Gamma"})

	contacts, total, err := repo.List(ctx, domain.ContactFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(contacts) != 3 {
		t.Errorf("len(contacts) = %d, want 3", len(contacts))
	}
}

func TestContactRepository_List_Search(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	testutil.SeedContact(t, db, &domain.Contact{Name: "Acme Corp"})
	testutil.SeedContact(t, db, &domain.Contact{Name: "Beta Inc"})

	contacts, total, err := repo.List(ctx, domain.ContactFilter{Search: "Acme"})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(contacts) != 1 {
		t.Errorf("len(contacts) = %d, want 1", len(contacts))
	}
	if contacts[0].Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", contacts[0].Name, "Acme Corp")
	}
}

func TestContactRepository_List_TypeFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	testutil.SeedContact(t, db, &domain.Contact{Name: "Company", Type: domain.ContactTypeCompany})
	testutil.SeedContact(t, db, &domain.Contact{Name: "Individual", Type: domain.ContactTypeIndividual})

	contacts, total, err := repo.List(ctx, domain.ContactFilter{Type: domain.ContactTypeIndividual})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if contacts[0].Name != "Individual" {
		t.Errorf("Name = %q, want %q", contacts[0].Name, "Individual")
	}
}

func TestContactRepository_List_Pagination(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	// Create 5 contacts.
	for i := 0; i < 5; i++ {
		testutil.SeedContact(t, db, &domain.Contact{Name: string(rune('A'+i)) + " Contact"})
	}

	// Get first page of 2.
	contacts, total, err := repo.List(ctx, domain.ContactFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(contacts) != 2 {
		t.Errorf("len(contacts) = %d, want 2", len(contacts))
	}

	// Get second page.
	contacts2, _, err := repo.List(ctx, domain.ContactFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("List() page 2 error: %v", err)
	}
	if len(contacts2) != 2 {
		t.Errorf("len(contacts2) = %d, want 2", len(contacts2))
	}
}

func TestContactRepository_List_ExcludesSoftDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	c := testutil.SeedContact(t, db, &domain.Contact{Name: "To Delete"})
	testutil.SeedContact(t, db, &domain.Contact{Name: "Keep"})

	if err := repo.Delete(ctx, c.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	contacts, total, err := repo.List(ctx, domain.ContactFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if contacts[0].Name != "Keep" {
		t.Errorf("Name = %q, want %q", contacts[0].Name, "Keep")
	}
}
