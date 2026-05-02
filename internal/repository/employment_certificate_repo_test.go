package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func mustParseDay(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatalf("parse day %q: %v", s, err)
	}
	return tt
}

func seedEmploymentCertificate(t *testing.T, repo *EmploymentCertificateRepository, cert *domain.EmploymentCertificate) *domain.EmploymentCertificate {
	t.Helper()
	if cert == nil {
		cert = &domain.EmploymentCertificate{}
	}
	if cert.Year == 0 {
		cert.Year = 2025
	}
	if cert.CertificateType == "" {
		cert.CertificateType = domain.CertificateAdvance
	}
	if cert.ContractType == "" {
		cert.ContractType = domain.ContractDPC
	}
	if cert.EmployerName == "" {
		cert.EmployerName = "Acme s.r.o."
	}
	if cert.EmployerICO == "" {
		cert.EmployerICO = "12345678"
	}
	if cert.PeriodFrom.IsZero() {
		cert.PeriodFrom = mustParseDay(t, "2025-01-01")
	}
	if cert.PeriodTo.IsZero() {
		cert.PeriodTo = mustParseDay(t, "2025-12-31")
	}
	if cert.Status == "" {
		cert.Status = "draft"
	}

	ctx := context.Background()
	if err := repo.Create(ctx, cert); err != nil {
		t.Fatalf("seedEmploymentCertificate: %v", err)
	}
	return cert
}

func TestEmploymentCertificateRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	cert := &domain.EmploymentCertificate{
		Year:                   2025,
		CertificateType:        domain.CertificateAdvance,
		EmployerName:           "Acme s.r.o.",
		EmployerICO:            "12345678",
		EmployerAddress:        "Praha 1",
		ContractType:           domain.ContractDPC,
		PeriodFrom:             mustParseDay(t, "2025-01-01"),
		PeriodTo:               mustParseDay(t, "2025-06-30"),
		GrossIncome:            120_000_00,
		AdvanceTaxWithheld:     18_000_00,
		AnnualSettlementRefund: 1_000_00,
		MonthlyBonusPaid:       7_650_00,
		Status:                 "draft",
	}
	if err := repo.Create(ctx, cert); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if cert.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByID(ctx, cert.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.EmployerName != "Acme s.r.o." {
		t.Errorf("EmployerName = %q", got.EmployerName)
	}
	if got.EmployerICO != "12345678" {
		t.Errorf("EmployerICO = %q", got.EmployerICO)
	}
	if got.GrossIncome != 120_000_00 {
		t.Errorf("GrossIncome = %d", got.GrossIncome)
	}
	if got.MonthlyBonusPaid != 7_650_00 {
		t.Errorf("MonthlyBonusPaid = %d", got.MonthlyBonusPaid)
	}
	if !got.PeriodFrom.Equal(mustParseDay(t, "2025-01-01")) {
		t.Errorf("PeriodFrom = %v", got.PeriodFrom)
	}
	if !got.PeriodTo.Equal(mustParseDay(t, "2025-06-30")) {
		t.Errorf("PeriodTo = %v", got.PeriodTo)
	}
	if got.IncludeWithholdingInDAP {
		t.Error("expected IncludeWithholdingInDAP=false by default")
	}
}

func TestEmploymentCertificateRepository_Create_WithDocumentID(t *testing.T) {
	db := testutil.NewTestDB(t)
	docRepo := NewEmploymentDocumentRepository(db)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	doc := seedEmploymentDocument(t, docRepo, nil)

	cert := &domain.EmploymentCertificate{
		Year:            2025,
		DocumentID:      &doc.ID,
		CertificateType: domain.CertificateAdvance,
		EmployerName:    "Acme",
		EmployerICO:     "12345678",
		PeriodFrom:      mustParseDay(t, "2025-01-01"),
		PeriodTo:        mustParseDay(t, "2025-12-31"),
		Status:          "draft",
	}
	if err := repo.Create(ctx, cert); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, cert.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.DocumentID == nil || *got.DocumentID != doc.ID {
		t.Errorf("DocumentID = %v, want %d", got.DocumentID, doc.ID)
	}
}

func TestEmploymentCertificateRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEmploymentCertificateRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	cert := seedEmploymentCertificate(t, repo, nil)
	cert.GrossIncome = 200_000_00
	cert.IncludeWithholdingInDAP = true
	cert.Status = "confirmed"

	if err := repo.Update(ctx, cert); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, cert.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.GrossIncome != 200_000_00 {
		t.Errorf("GrossIncome = %d", got.GrossIncome)
	}
	if !got.IncludeWithholdingInDAP {
		t.Error("expected IncludeWithholdingInDAP=true after update")
	}
	if got.Status != "confirmed" {
		t.Errorf("Status = %q, want confirmed", got.Status)
	}
}

func TestEmploymentCertificateRepository_Update_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	cert := &domain.EmploymentCertificate{
		ID:              99999,
		Year:            2025,
		CertificateType: domain.CertificateAdvance,
		EmployerICO:     "12345678",
		ContractType:    domain.ContractDPC,
		PeriodFrom:      mustParseDay(t, "2025-01-01"),
		PeriodTo:        mustParseDay(t, "2025-12-31"),
		Status:          "draft",
	}
	err := repo.Update(ctx, cert)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEmploymentCertificateRepository_Delete_SoftDelete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	cert := seedEmploymentCertificate(t, repo, nil)

	if err := repo.Delete(ctx, cert.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// GetByID excludes soft-deleted.
	if _, err := repo.GetByID(ctx, cert.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after soft delete, got %v", err)
	}

	// ListByYear should not return it.
	list, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list after soft delete, got %d", len(list))
	}
}

func TestEmploymentCertificateRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	if err := repo.Delete(ctx, 99999); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEmploymentCertificateRepository_ListByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	seedEmploymentCertificate(t, repo, &domain.EmploymentCertificate{
		Year: 2025, EmployerName: "B", EmployerICO: "11111111",
		PeriodFrom: mustParseDay(t, "2025-01-01"), PeriodTo: mustParseDay(t, "2025-06-30"),
	})
	seedEmploymentCertificate(t, repo, &domain.EmploymentCertificate{
		Year: 2025, EmployerName: "A", EmployerICO: "22222222",
		PeriodFrom: mustParseDay(t, "2025-01-01"), PeriodTo: mustParseDay(t, "2025-06-30"),
	})
	seedEmploymentCertificate(t, repo, &domain.EmploymentCertificate{
		Year: 2024, EmployerName: "Old", EmployerICO: "33333333",
		PeriodFrom: mustParseDay(t, "2024-01-01"), PeriodTo: mustParseDay(t, "2024-12-31"),
	})

	list, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	// Ordered by employer_name.
	if list[0].EmployerName != "A" {
		t.Errorf("ordering: list[0].EmployerName = %q, want A", list[0].EmployerName)
	}
}

func TestEmploymentCertificateRepository_ListConfirmedByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	confirmed := seedEmploymentCertificate(t, repo, &domain.EmploymentCertificate{
		Year: 2025, EmployerName: "A", EmployerICO: "11111111",
		PeriodFrom: mustParseDay(t, "2025-01-01"), PeriodTo: mustParseDay(t, "2025-06-30"),
		Status: "draft",
	})
	confirmed.Status = "confirmed"
	if err := repo.Update(ctx, confirmed); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Draft cert should be excluded.
	seedEmploymentCertificate(t, repo, &domain.EmploymentCertificate{
		Year: 2025, EmployerName: "B", EmployerICO: "22222222",
		PeriodFrom: mustParseDay(t, "2025-01-01"), PeriodTo: mustParseDay(t, "2025-06-30"),
		Status: "draft",
	})

	list, err := repo.ListConfirmedByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListConfirmedByYear: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].ID != confirmed.ID {
		t.Errorf("expected cert %d, got %d", confirmed.ID, list[0].ID)
	}
}

func TestEmploymentCertificateRepository_ListConfirmedByYear_ExcludesDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	cert := seedEmploymentCertificate(t, repo, &domain.EmploymentCertificate{
		Year: 2025, EmployerName: "A", EmployerICO: "11111111",
		PeriodFrom: mustParseDay(t, "2025-01-01"), PeriodTo: mustParseDay(t, "2025-06-30"),
		Status: "confirmed",
	})

	if err := repo.Delete(ctx, cert.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	list, err := repo.ListConfirmedByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListConfirmedByYear: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

// TestEmploymentCertificateRepository_UniqueReplace verifies that the UNIQUE
// (year, employer_ico, certificate_type, period_from, period_to) ON CONFLICT
// REPLACE clause overwrites the original row when the same key is reused. The
// new row gets a fresh primary key, but only one row remains for the unique
// tuple.
func TestEmploymentCertificateRepository_UniqueReplace(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentCertificateRepository(db)
	ctx := context.Background()

	first := seedEmploymentCertificate(t, repo, &domain.EmploymentCertificate{
		Year:            2025,
		CertificateType: domain.CertificateAdvance,
		EmployerName:    "Old name",
		EmployerICO:     "12345678",
		PeriodFrom:      mustParseDay(t, "2025-01-01"),
		PeriodTo:        mustParseDay(t, "2025-12-31"),
		GrossIncome:     50_000_00,
		Status:          "confirmed",
	})

	// Insert again with same UNIQUE tuple but different values.
	second := &domain.EmploymentCertificate{
		Year:            2025,
		CertificateType: domain.CertificateAdvance,
		EmployerName:    "Corrected name",
		EmployerICO:     "12345678",
		PeriodFrom:      mustParseDay(t, "2025-01-01"),
		PeriodTo:        mustParseDay(t, "2025-12-31"),
		GrossIncome:     75_000_00,
		Status:          "draft",
	}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create (second): %v", err)
	}

	// Only one row remains — the original is gone.
	list, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1 (UNIQUE REPLACE should keep only one row)", len(list))
	}
	if list[0].EmployerName != "Corrected name" {
		t.Errorf("EmployerName = %q, want %q (latest values should win)", list[0].EmployerName, "Corrected name")
	}
	if list[0].GrossIncome != 75_000_00 {
		t.Errorf("GrossIncome = %d, want 7500000", list[0].GrossIncome)
	}
	// Sanity: original ID is gone.
	if _, err := repo.GetByID(ctx, first.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for replaced original (id=%d), got %v", first.ID, err)
	}
}
