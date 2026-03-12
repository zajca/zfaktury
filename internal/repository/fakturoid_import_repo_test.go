package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestFakturoidImportLogRepository_CreateAndFind(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewFakturoidImportLogRepository(db)
	ctx := context.Background()

	entry := &domain.FakturoidImportLog{
		FakturoidEntityType: "contact",
		FakturoidID:         42,
		LocalEntityType:     "contact",
		LocalID:             7,
	}

	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if entry.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}

	// Find by Fakturoid ID.
	found, err := repo.FindByFakturoidID(ctx, "contact", 42)
	if err != nil {
		t.Fatalf("FindByFakturoidID() error: %v", err)
	}
	if found == nil {
		t.Fatal("FindByFakturoidID() returned nil, expected entry")
	}
	if found.ID != entry.ID {
		t.Errorf("ID = %d, want %d", found.ID, entry.ID)
	}
	if found.FakturoidEntityType != "contact" {
		t.Errorf("FakturoidEntityType = %q, want %q", found.FakturoidEntityType, "contact")
	}
	if found.FakturoidID != 42 {
		t.Errorf("FakturoidID = %d, want %d", found.FakturoidID, 42)
	}
	if found.LocalEntityType != "contact" {
		t.Errorf("LocalEntityType = %q, want %q", found.LocalEntityType, "contact")
	}
	if found.LocalID != 7 {
		t.Errorf("LocalID = %d, want %d", found.LocalID, 7)
	}
	if found.ImportedAt.IsZero() {
		t.Error("expected ImportedAt to be set")
	}
}

func TestFakturoidImportLogRepository_FindByFakturoidID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewFakturoidImportLogRepository(db)
	ctx := context.Background()

	found, err := repo.FindByFakturoidID(ctx, "contact", 99999)
	if err != nil {
		t.Fatalf("FindByFakturoidID() error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for non-existent entry, got %+v", found)
	}
}

func TestFakturoidImportLogRepository_ListByEntityType(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewFakturoidImportLogRepository(db)
	ctx := context.Background()

	// Create entries of different types.
	entries := []domain.FakturoidImportLog{
		{FakturoidEntityType: "contact", FakturoidID: 1, LocalEntityType: "contact", LocalID: 10},
		{FakturoidEntityType: "contact", FakturoidID: 2, LocalEntityType: "contact", LocalID: 11},
		{FakturoidEntityType: "invoice", FakturoidID: 3, LocalEntityType: "invoice", LocalID: 20},
	}
	for i := range entries {
		if err := repo.Create(ctx, &entries[i]); err != nil {
			t.Fatalf("Create() entry %d error: %v", i, err)
		}
	}

	// List contacts only.
	contacts, err := repo.ListByEntityType(ctx, "contact")
	if err != nil {
		t.Fatalf("ListByEntityType() error: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("len(contacts) = %d, want 2", len(contacts))
	}

	// List invoices only.
	invoices, err := repo.ListByEntityType(ctx, "invoice")
	if err != nil {
		t.Fatalf("ListByEntityType() error: %v", err)
	}
	if len(invoices) != 1 {
		t.Errorf("len(invoices) = %d, want 1", len(invoices))
	}

	// List non-existent type.
	empty, err := repo.ListByEntityType(ctx, "expense")
	if err != nil {
		t.Fatalf("ListByEntityType() error: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("len(empty) = %d, want 0", len(empty))
	}
}

func TestFakturoidImportLogRepository_UniqueConstraint(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewFakturoidImportLogRepository(db)
	ctx := context.Background()

	entry1 := &domain.FakturoidImportLog{
		FakturoidEntityType: "contact",
		FakturoidID:         42,
		LocalEntityType:     "contact",
		LocalID:             7,
	}
	if err := repo.Create(ctx, entry1); err != nil {
		t.Fatalf("Create() first entry error: %v", err)
	}

	// Attempt to insert duplicate (same fakturoid_entity_type + fakturoid_id).
	entry2 := &domain.FakturoidImportLog{
		FakturoidEntityType: "contact",
		FakturoidID:         42,
		LocalEntityType:     "contact",
		LocalID:             99,
	}
	err := repo.Create(ctx, entry2)
	if err == nil {
		t.Fatal("expected error for duplicate (fakturoid_entity_type, fakturoid_id)")
	}
	if !strings.Contains(err.Error(), "UNIQUE constraint") {
		t.Errorf("expected UNIQUE constraint error, got: %v", err)
	}

	// Different entity type with same fakturoid_id should succeed.
	entry3 := &domain.FakturoidImportLog{
		FakturoidEntityType: "invoice",
		FakturoidID:         42,
		LocalEntityType:     "invoice",
		LocalID:             15,
	}
	if err := repo.Create(ctx, entry3); err != nil {
		t.Fatalf("Create() different entity type error: %v", err)
	}
}
