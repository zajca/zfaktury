package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestSequenceRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{
		Prefix:        "FV",
		NextNumber:    1,
		Year:          2026,
		FormatPattern: "{prefix}{year}{number:04d}",
	}

	if err := repo.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if seq.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
}

func TestSequenceRepository_Create_DuplicatePrefixYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	seq1 := &domain.InvoiceSequence{
		Prefix: "FV", NextNumber: 1, Year: 2026,
		FormatPattern: "{prefix}{year}{number:04d}",
	}
	if err := repo.Create(ctx, seq1); err != nil {
		t.Fatalf("Create() first error: %v", err)
	}

	seq2 := &domain.InvoiceSequence{
		Prefix: "FV", NextNumber: 1, Year: 2026,
		FormatPattern: "{prefix}{year}{number:04d}",
	}
	err := repo.Create(ctx, seq2)
	if err == nil {
		t.Error("expected error for duplicate prefix+year")
	}
}

func TestSequenceRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	got, err := repo.GetByID(ctx, seqID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Prefix != "FV" {
		t.Errorf("Prefix = %q, want %q", got.Prefix, "FV")
	}
	if got.Year != 2026 {
		t.Errorf("Year = %d, want %d", got.Year, 2026)
	}
	if got.NextNumber != 1 {
		t.Errorf("NextNumber = %d, want %d", got.NextNumber, 1)
	}
}

func TestSequenceRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent sequence")
	}
	if !strings.Contains(err.Error(), sql.ErrNoRows.Error()) {
		t.Errorf("expected sql.ErrNoRows in error, got: %v", err)
	}
}

func TestSequenceRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	seq, err := repo.GetByID(ctx, seqID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	seq.NextNumber = 10
	if err := repo.Update(ctx, seq); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, seqID)
	if err != nil {
		t.Fatalf("GetByID() after update error: %v", err)
	}
	if got.NextNumber != 10 {
		t.Errorf("NextNumber = %d, want %d", got.NextNumber, 10)
	}
}

func TestSequenceRepository_Delete_SoftDelete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	if err := repo.Delete(ctx, seqID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Should not be found via GetByID (filters deleted_at IS NULL).
	_, err := repo.GetByID(ctx, seqID)
	if err == nil {
		t.Error("expected error when getting soft-deleted sequence")
	}

	// Should still exist in DB.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM invoice_sequences WHERE id = ?", seqID).Scan(&count); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if count != 1 {
		t.Errorf("expected sequence to still exist in DB, count = %d", count)
	}
}

func TestSequenceRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent sequence")
	}
}

func TestSequenceRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	testutil.SeedInvoiceSequence(t, db, "FV", 2026)
	testutil.SeedInvoiceSequence(t, db, "ZF", 2026)
	testutil.SeedInvoiceSequence(t, db, "FV", 2025)

	sequences, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sequences) != 3 {
		t.Errorf("len(sequences) = %d, want 3", len(sequences))
	}
}

func TestSequenceRepository_List_ExcludesSoftDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	id1 := testutil.SeedInvoiceSequence(t, db, "FV", 2026)
	testutil.SeedInvoiceSequence(t, db, "ZF", 2026)

	if err := repo.Delete(ctx, id1); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	sequences, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sequences) != 1 {
		t.Errorf("len(sequences) = %d, want 1", len(sequences))
	}
}

func TestSequenceRepository_GetByPrefixAndYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	testutil.SeedInvoiceSequence(t, db, "FV", 2026)
	testutil.SeedInvoiceSequence(t, db, "ZF", 2026)

	got, err := repo.GetByPrefixAndYear(ctx, "ZF", 2026)
	if err != nil {
		t.Fatalf("GetByPrefixAndYear() error: %v", err)
	}
	if got.Prefix != "ZF" {
		t.Errorf("Prefix = %q, want %q", got.Prefix, "ZF")
	}
	if got.Year != 2026 {
		t.Errorf("Year = %d, want %d", got.Year, 2026)
	}
}

func TestSequenceRepository_GetByPrefixAndYear_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	_, err := repo.GetByPrefixAndYear(ctx, "XX", 2099)
	if err == nil {
		t.Error("expected error for non-existent prefix+year")
	}
	if !strings.Contains(err.Error(), sql.ErrNoRows.Error()) {
		t.Errorf("expected sql.ErrNoRows in error, got: %v", err)
	}
}

func TestSequenceRepository_CountInvoicesBySequenceID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	// No invoices yet.
	count, err := repo.CountInvoicesBySequenceID(ctx, seqID)
	if err != nil {
		t.Fatalf("CountInvoicesBySequenceID() error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	// Seed an invoice referencing this sequence.
	customer := testutil.SeedContact(t, db, nil)
	testutil.SeedInvoice(t, db, customer.ID, []domain.InvoiceItem{
		{Description: "Test", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
	})
	// Link it to the sequence.
	db.ExecContext(ctx, "UPDATE invoices SET sequence_id = ? WHERE id = 1", seqID)

	count, err = repo.CountInvoicesBySequenceID(ctx, seqID)
	if err != nil {
		t.Fatalf("CountInvoicesBySequenceID() after seeding error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}
