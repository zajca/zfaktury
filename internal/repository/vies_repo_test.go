package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestVIESSummaryRepository_CreateAndGetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVIESSummaryRepository(db)
	ctx := context.Background()

	vs := &domain.VIESSummary{
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 2,
		},
		FilingType: domain.FilingTypeRegular,
		XMLData:    []byte("<xml>test</xml>"),
	}

	if err := repo.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if vs.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if vs.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if vs.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", vs.Status, domain.FilingStatusDraft)
	}

	got, err := repo.GetByID(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Period.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Period.Year)
	}
	if got.Period.Quarter != 2 {
		t.Errorf("Quarter = %d, want 2", got.Period.Quarter)
	}
	if got.FilingType != domain.FilingTypeRegular {
		t.Errorf("FilingType = %q, want %q", got.FilingType, domain.FilingTypeRegular)
	}
	if string(got.XMLData) != "<xml>test</xml>" {
		t.Errorf("XMLData = %q, want %q", got.XMLData, "<xml>test</xml>")
	}
	if got.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", got.Status, domain.FilingStatusDraft)
	}
	if got.FiledAt != nil {
		t.Errorf("FiledAt = %v, want nil", got.FiledAt)
	}
}

func TestVIESSummaryRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVIESSummaryRepository(db)
	ctx := context.Background()

	vs := &domain.VIESSummary{
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 1,
		},
		FilingType: domain.FilingTypeRegular,
		XMLData:    []byte("<xml>original</xml>"),
	}
	if err := repo.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	vs.XMLData = []byte("<xml>updated</xml>")
	vs.Status = domain.FilingStatusReady
	vs.Period.Quarter = 3

	if err := repo.Update(ctx, vs); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if string(got.XMLData) != "<xml>updated</xml>" {
		t.Errorf("XMLData = %q, want %q", got.XMLData, "<xml>updated</xml>")
	}
	if got.Status != domain.FilingStatusReady {
		t.Errorf("Status = %q, want %q", got.Status, domain.FilingStatusReady)
	}
	if got.Period.Quarter != 3 {
		t.Errorf("Quarter = %d, want 3", got.Period.Quarter)
	}
	// UpdatedAt is set by the repo Update method; just verify it's not zero.
	if got.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set after update")
	}
}

func TestVIESSummaryRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVIESSummaryRepository(db)
	ctx := context.Background()

	vs := &domain.VIESSummary{
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 1,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Delete existing should succeed.
	if err := repo.Delete(ctx, vs.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify it is gone.
	_, err := repo.GetByID(ctx, vs.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByID() after delete: got %v, want %v", err, domain.ErrNotFound)
	}

	// Delete non-existent should return ErrNotFound.
	err = repo.Delete(ctx, 99999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete(non-existent): got %v, want %v", err, domain.ErrNotFound)
	}
}

func TestVIESSummaryRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVIESSummaryRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByID(non-existent): got %v, want %v", err, domain.ErrNotFound)
	}
}

func TestVIESSummaryRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVIESSummaryRepository(db)
	ctx := context.Background()

	// Create summaries for 2025.
	for _, q := range []int{1, 2, 3} {
		vs := &domain.VIESSummary{
			Period: domain.TaxPeriod{
				Year:    2025,
				Quarter: q,
			},
			FilingType: domain.FilingTypeRegular,
		}
		if err := repo.Create(ctx, vs); err != nil {
			t.Fatalf("Create() error for Q%d: %v", q, err)
		}
	}

	// Create one for a different year.
	vs2024 := &domain.VIESSummary{
		Period: domain.TaxPeriod{
			Year:    2024,
			Quarter: 4,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vs2024); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// List 2025 should return 3 entries.
	list, err := repo.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List(2025) error: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("List(2025) returned %d items, want 3", len(list))
	}

	// Verify ordering by quarter ASC.
	for i, vs := range list {
		if vs.Period.Quarter != i+1 {
			t.Errorf("list[%d].Quarter = %d, want %d", i, vs.Period.Quarter, i+1)
		}
	}

	// List 2024 should return 1 entry.
	list2024, err := repo.List(ctx, 2024)
	if err != nil {
		t.Fatalf("List(2024) error: %v", err)
	}
	if len(list2024) != 1 {
		t.Errorf("List(2024) returned %d items, want 1", len(list2024))
	}

	// List for year with no entries should return empty.
	listEmpty, err := repo.List(ctx, 2020)
	if err != nil {
		t.Fatalf("List(2020) error: %v", err)
	}
	if len(listEmpty) != 0 {
		t.Errorf("List(2020) returned %d items, want 0", len(listEmpty))
	}
}

func TestVIESSummaryRepository_GetByPeriod(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVIESSummaryRepository(db)
	ctx := context.Background()

	vs := &domain.VIESSummary{
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 2,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Find by matching year, quarter, filing type.
	got, err := repo.GetByPeriod(ctx, 2025, 2, domain.FilingTypeRegular)
	if err != nil {
		t.Fatalf("GetByPeriod() error: %v", err)
	}
	if got.ID != vs.ID {
		t.Errorf("GetByPeriod() ID = %d, want %d", got.ID, vs.ID)
	}

	// Non-existent quarter returns ErrNotFound.
	_, err = repo.GetByPeriod(ctx, 2025, 4, domain.FilingTypeRegular)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByPeriod(wrong quarter): got %v, want %v", err, domain.ErrNotFound)
	}

	// Non-existent filing type returns ErrNotFound.
	_, err = repo.GetByPeriod(ctx, 2025, 2, domain.FilingTypeCorrective)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByPeriod(wrong filingType): got %v, want %v", err, domain.ErrNotFound)
	}

	// Non-existent year returns ErrNotFound.
	_, err = repo.GetByPeriod(ctx, 2020, 2, domain.FilingTypeRegular)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByPeriod(wrong year): got %v, want %v", err, domain.ErrNotFound)
	}
}

func TestVIESSummaryRepository_CreateAndGetLines(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVIESSummaryRepository(db)
	ctx := context.Background()

	// Create a parent summary first.
	vs := &domain.VIESSummary{
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 1,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	lines := []domain.VIESSummaryLine{
		{
			VIESSummaryID: vs.ID,
			PartnerDIC:    "DE123456789",
			CountryCode:   "DE",
			TotalAmount:   domain.NewAmount(50000, 0),
			ServiceCode:   domain.VIESServiceCode3,
		},
		{
			VIESSummaryID: vs.ID,
			PartnerDIC:    "SK2024681012",
			CountryCode:   "SK",
			TotalAmount:   domain.NewAmount(30000, 0),
			ServiceCode:   domain.VIESServiceCode3,
		},
	}

	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatalf("CreateLines() error: %v", err)
	}

	got, err := repo.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetLines() returned %d lines, want 2", len(got))
	}

	// Lines are ordered by country_code, partner_dic -- DE comes before SK.
	if got[0].CountryCode != "DE" {
		t.Errorf("line[0].CountryCode = %q, want %q", got[0].CountryCode, "DE")
	}
	if got[0].PartnerDIC != "DE123456789" {
		t.Errorf("line[0].PartnerDIC = %q, want %q", got[0].PartnerDIC, "DE123456789")
	}
	if got[0].TotalAmount != domain.NewAmount(50000, 0) {
		t.Errorf("line[0].TotalAmount = %d, want %d", got[0].TotalAmount, domain.NewAmount(50000, 0))
	}
	if got[0].ServiceCode != domain.VIESServiceCode3 {
		t.Errorf("line[0].ServiceCode = %q, want %q", got[0].ServiceCode, domain.VIESServiceCode3)
	}
	if got[0].VIESSummaryID != vs.ID {
		t.Errorf("line[0].VIESSummaryID = %d, want %d", got[0].VIESSummaryID, vs.ID)
	}

	if got[1].CountryCode != "SK" {
		t.Errorf("line[1].CountryCode = %q, want %q", got[1].CountryCode, "SK")
	}
	if got[1].PartnerDIC != "SK2024681012" {
		t.Errorf("line[1].PartnerDIC = %q, want %q", got[1].PartnerDIC, "SK2024681012")
	}
	if got[1].TotalAmount != domain.NewAmount(30000, 0) {
		t.Errorf("line[1].TotalAmount = %d, want %d", got[1].TotalAmount, domain.NewAmount(30000, 0))
	}
}

func TestVIESSummaryRepository_DeleteLines(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVIESSummaryRepository(db)
	ctx := context.Background()

	// Create a parent summary.
	vs := &domain.VIESSummary{
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 3,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	lines := []domain.VIESSummaryLine{
		{
			VIESSummaryID: vs.ID,
			PartnerDIC:    "AT111111111",
			CountryCode:   "AT",
			TotalAmount:   domain.NewAmount(10000, 0),
			ServiceCode:   domain.VIESServiceCode3,
		},
		{
			VIESSummaryID: vs.ID,
			PartnerDIC:    "PL2222222222",
			CountryCode:   "PL",
			TotalAmount:   domain.NewAmount(20000, 0),
			ServiceCode:   domain.VIESServiceCode3,
		},
	}
	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatalf("CreateLines() error: %v", err)
	}

	// Verify lines exist.
	got, err := repo.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetLines() returned %d lines, want 2", len(got))
	}

	// Delete all lines.
	if err := repo.DeleteLines(ctx, vs.ID); err != nil {
		t.Fatalf("DeleteLines() error: %v", err)
	}

	// Verify lines are gone.
	got, err = repo.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() after delete error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("GetLines() after delete returned %d lines, want 0", len(got))
	}

	// DeleteLines on a summary with no lines should not error.
	if err := repo.DeleteLines(ctx, vs.ID); err != nil {
		t.Errorf("DeleteLines() on empty: got %v, want nil", err)
	}
}
