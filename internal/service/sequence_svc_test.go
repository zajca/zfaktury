package service

import (
	"context"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newSequenceTestStack(t *testing.T) (*SequenceService, *repository.SequenceRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewSequenceRepository(db)
	svc := NewSequenceService(repo)
	return svc, repo
}

func TestSequenceService_Create_Valid(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{
		Prefix: "FV",
		Year:   2026,
	}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if seq.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if seq.NextNumber != 1 {
		t.Errorf("NextNumber = %d, want 1 (default)", seq.NextNumber)
	}
	if seq.FormatPattern == "" {
		t.Error("expected FormatPattern to be set by default")
	}
}

func TestSequenceService_Create_MissingPrefix(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Year: 2026}
	err := svc.Create(ctx, seq)
	if err == nil {
		t.Error("expected error for missing prefix")
	}
}

func TestSequenceService_Create_MissingYear(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV"}
	err := svc.Create(ctx, seq)
	if err == nil {
		t.Error("expected error for missing year")
	}
}

func TestSequenceService_Create_DuplicatePrefixYear(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq1 := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, seq1); err != nil {
		t.Fatalf("Create() first error: %v", err)
	}

	seq2 := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	err := svc.Create(ctx, seq2)
	if err == nil {
		t.Error("expected error for duplicate prefix+year")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestSequenceService_Update_Valid(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	seq.NextNumber = 5
	if err := svc.Update(ctx, seq); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestSequenceService_Update_PreventLoweringNextNumber(t *testing.T) {
	svc, repo := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026, NextNumber: 5}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Simulate that numbers up to 4 have been used (next_number is 5, so max used = 4).
	// The MaxUsedNumber returns next_number - 1 = 4.
	// But we created with NextNumber=5, so after Create the DB value is 5.
	// Let's read back from DB to get current state.
	got, err := repo.GetByID(ctx, seq.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	// Try to set next_number to 3, which is below max used (4).
	got.NextNumber = 3
	err = svc.Update(ctx, got)
	if err == nil {
		t.Error("expected error when lowering next_number below used numbers")
	}
	if !strings.Contains(err.Error(), "already been assigned") {
		t.Errorf("error should mention 'already been assigned', got: %v", err)
	}
}

func TestSequenceService_Delete_NoReferences(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, seq.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify deleted.
	_, err := svc.GetByID(ctx, seq.ID)
	if err == nil {
		t.Error("expected error for soft-deleted sequence")
	}
}

func TestSequenceService_Delete_ZeroID(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestSequenceService_GetOrCreateForYear_ExistingSequence(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	// Create a sequence first.
	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// GetOrCreate should return the existing one.
	got, err := svc.GetOrCreateForYear(ctx, "FV", 2026)
	if err != nil {
		t.Fatalf("GetOrCreateForYear() error: %v", err)
	}
	if got.ID != seq.ID {
		t.Errorf("ID = %d, want %d (should return existing)", got.ID, seq.ID)
	}
}

func TestSequenceService_GetOrCreateForYear_NewSequence(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	got, err := svc.GetOrCreateForYear(ctx, "ZF", 2027)
	if err != nil {
		t.Fatalf("GetOrCreateForYear() error: %v", err)
	}
	if got.ID == 0 {
		t.Error("expected non-zero ID for newly created sequence")
	}
	if got.Prefix != "ZF" {
		t.Errorf("Prefix = %q, want %q", got.Prefix, "ZF")
	}
	if got.Year != 2027 {
		t.Errorf("Year = %d, want %d", got.Year, 2027)
	}
	if got.NextNumber != 1 {
		t.Errorf("NextNumber = %d, want 1", got.NextNumber)
	}
}

func TestSequenceService_List(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	svc.Create(ctx, &domain.InvoiceSequence{Prefix: "FV", Year: 2025})
	svc.Create(ctx, &domain.InvoiceSequence{Prefix: "FV", Year: 2026})

	sequences, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sequences) != 2 {
		t.Errorf("len(sequences) = %d, want 2", len(sequences))
	}
}

func TestFormatPreview(t *testing.T) {
	tests := []struct {
		name   string
		seq    *domain.InvoiceSequence
		expect string
	}{
		{
			name:   "standard format",
			seq:    &domain.InvoiceSequence{Prefix: "FV", Year: 2026, NextNumber: 1},
			expect: "FV20260001",
		},
		{
			name:   "higher number",
			seq:    &domain.InvoiceSequence{Prefix: "ZF", Year: 2025, NextNumber: 42},
			expect: "ZF20250042",
		},
		{
			name:   "credit note prefix",
			seq:    &domain.InvoiceSequence{Prefix: "DN", Year: 2026, NextNumber: 100},
			expect: "DN20260100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPreview(tt.seq)
			if got != tt.expect {
				t.Errorf("FormatPreview() = %q, want %q", got, tt.expect)
			}
		})
	}
}
