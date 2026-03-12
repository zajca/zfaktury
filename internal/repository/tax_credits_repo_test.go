package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

// --- TaxSpouseCreditRepository ---

func TestTaxSpouseCreditRepository_Upsert(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxSpouseCreditRepository(db)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:             2025,
		SpouseName:       "Jana Novakova",
		SpouseBirthNumber: "8555012345",
		SpouseIncome:     domain.NewAmount(50000, 0),
		SpouseZTP:        false,
		MonthsClaimed:    12,
		CreditAmount:     domain.NewAmount(24840, 0),
	}

	if err := repo.Upsert(ctx, credit); err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	if credit.ID == 0 {
		t.Error("expected non-zero ID after Upsert")
	}
	if credit.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if credit.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestTaxSpouseCreditRepository_Upsert_Replace(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxSpouseCreditRepository(db)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:             2025,
		SpouseName:       "Jana Novakova",
		SpouseBirthNumber: "8555012345",
		SpouseIncome:     domain.NewAmount(50000, 0),
		MonthsClaimed:    12,
		CreditAmount:     domain.NewAmount(24840, 0),
	}
	if err := repo.Upsert(ctx, credit); err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	// Upsert again with different values for the same year.
	credit2 := &domain.TaxSpouseCredit{
		Year:             2025,
		SpouseName:       "Jana Novakova",
		SpouseBirthNumber: "8555012345",
		SpouseIncome:     domain.NewAmount(60000, 0),
		SpouseZTP:        true,
		MonthsClaimed:    6,
		CreditAmount:     domain.NewAmount(12420, 0),
	}
	if err := repo.Upsert(ctx, credit2); err != nil {
		t.Fatalf("Upsert() replace error: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("GetByYear() error: %v", err)
	}
	if got.MonthsClaimed != 6 {
		t.Errorf("MonthsClaimed = %d, want 6", got.MonthsClaimed)
	}
	if !got.SpouseZTP {
		t.Error("SpouseZTP = false, want true")
	}
	if got.CreditAmount != domain.NewAmount(12420, 0) {
		t.Errorf("CreditAmount = %d, want %d", got.CreditAmount, domain.NewAmount(12420, 0))
	}
}

func TestTaxSpouseCreditRepository_GetByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxSpouseCreditRepository(db)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:             2025,
		SpouseName:       "Jana Novakova",
		SpouseBirthNumber: "8555012345",
		SpouseIncome:     domain.NewAmount(50000, 0),
		SpouseZTP:        true,
		MonthsClaimed:    8,
		CreditAmount:     domain.NewAmount(24840, 0),
	}
	if err := repo.Upsert(ctx, credit); err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("GetByYear() error: %v", err)
	}

	if got.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Year)
	}
	if got.SpouseName != "Jana Novakova" {
		t.Errorf("SpouseName = %q, want %q", got.SpouseName, "Jana Novakova")
	}
	if got.SpouseBirthNumber != "8555012345" {
		t.Errorf("SpouseBirthNumber = %q, want %q", got.SpouseBirthNumber, "8555012345")
	}
	if got.SpouseIncome != domain.NewAmount(50000, 0) {
		t.Errorf("SpouseIncome = %d, want %d", got.SpouseIncome, domain.NewAmount(50000, 0))
	}
	if !got.SpouseZTP {
		t.Error("SpouseZTP = false, want true")
	}
	if got.MonthsClaimed != 8 {
		t.Errorf("MonthsClaimed = %d, want 8", got.MonthsClaimed)
	}
	if got.CreditAmount != domain.NewAmount(24840, 0) {
		t.Errorf("CreditAmount = %d, want %d", got.CreditAmount, domain.NewAmount(24840, 0))
	}
}

func TestTaxSpouseCreditRepository_GetByYear_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxSpouseCreditRepository(db)
	ctx := context.Background()

	_, err := repo.GetByYear(ctx, 2099)
	if err == nil {
		t.Error("expected error for non-existent year")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestTaxSpouseCreditRepository_DeleteByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxSpouseCreditRepository(db)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:             2025,
		SpouseName:       "Jana Novakova",
		SpouseBirthNumber: "8555012345",
		SpouseIncome:     domain.NewAmount(50000, 0),
		MonthsClaimed:    12,
		CreditAmount:     domain.NewAmount(24840, 0),
	}
	if err := repo.Upsert(ctx, credit); err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	if err := repo.DeleteByYear(ctx, 2025); err != nil {
		t.Fatalf("DeleteByYear() error: %v", err)
	}

	_, err := repo.GetByYear(ctx, 2025)
	if err == nil {
		t.Error("expected error after delete")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestTaxSpouseCreditRepository_DeleteByYear_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxSpouseCreditRepository(db)
	ctx := context.Background()

	err := repo.DeleteByYear(ctx, 2099)
	if err == nil {
		t.Error("expected error for non-existent year")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// --- TaxChildCreditRepository ---

func TestTaxChildCreditRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxChildCreditRepository(db)
	ctx := context.Background()

	credit := &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Petr Novak",
		BirthNumber:   "1510151234",
		ChildOrder:    1,
		MonthsClaimed: 12,
		ZTP:           false,
		CreditAmount:  domain.NewAmount(15204, 0),
	}

	if err := repo.Create(ctx, credit); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if credit.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if credit.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestTaxChildCreditRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxChildCreditRepository(db)
	ctx := context.Background()

	credit := &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Petr Novak",
		BirthNumber:   "1510151234",
		ChildOrder:    1,
		MonthsClaimed: 12,
		ZTP:           false,
		CreditAmount:  domain.NewAmount(15204, 0),
	}
	if err := repo.Create(ctx, credit); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	credit.MonthsClaimed = 6
	credit.ZTP = true
	credit.CreditAmount = domain.NewAmount(30408, 0)

	if err := repo.Update(ctx, credit); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	// Verify via ListByYear.
	children, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("ListByYear() returned %d items, want 1", len(children))
	}
	if children[0].MonthsClaimed != 6 {
		t.Errorf("MonthsClaimed = %d, want 6", children[0].MonthsClaimed)
	}
	if !children[0].ZTP {
		t.Error("ZTP = false, want true")
	}
	if children[0].CreditAmount != domain.NewAmount(30408, 0) {
		t.Errorf("CreditAmount = %d, want %d", children[0].CreditAmount, domain.NewAmount(30408, 0))
	}
}

func TestTaxChildCreditRepository_Update_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxChildCreditRepository(db)
	ctx := context.Background()

	credit := &domain.TaxChildCredit{
		ID:            99999,
		Year:          2025,
		ChildName:     "Nonexistent",
		BirthNumber:   "0000000000",
		ChildOrder:    1,
		MonthsClaimed: 12,
		CreditAmount:  domain.NewAmount(15204, 0),
	}

	err := repo.Update(ctx, credit)
	if err == nil {
		t.Error("expected error for non-existent update")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestTaxChildCreditRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxChildCreditRepository(db)
	ctx := context.Background()

	credit := &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Petr Novak",
		BirthNumber:   "1510151234",
		ChildOrder:    1,
		MonthsClaimed: 12,
		CreditAmount:  domain.NewAmount(15204, 0),
	}
	if err := repo.Create(ctx, credit); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, credit.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	children, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(children) != 0 {
		t.Errorf("expected empty list after delete, got %d items", len(children))
	}
}

func TestTaxChildCreditRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxChildCreditRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent delete")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestTaxChildCreditRepository_ListByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxChildCreditRepository(db)
	ctx := context.Background()

	// Create multiple children for 2025.
	children := []domain.TaxChildCredit{
		{Year: 2025, ChildName: "Child 1", BirthNumber: "1111111111", ChildOrder: 1, MonthsClaimed: 12, CreditAmount: domain.NewAmount(15204, 0)},
		{Year: 2025, ChildName: "Child 2", BirthNumber: "2222222222", ChildOrder: 2, MonthsClaimed: 12, CreditAmount: domain.NewAmount(22320, 0)},
		{Year: 2025, ChildName: "Child 3", BirthNumber: "3333333333", ChildOrder: 3, MonthsClaimed: 6, ZTP: true, CreditAmount: domain.NewAmount(27840, 0)},
	}
	for i := range children {
		if err := repo.Create(ctx, &children[i]); err != nil {
			t.Fatalf("Create() child %d error: %v", i+1, err)
		}
	}

	// Create a child for a different year.
	otherYear := &domain.TaxChildCredit{
		Year: 2024, ChildName: "Child Other", BirthNumber: "4444444444", ChildOrder: 1, MonthsClaimed: 12, CreditAmount: domain.NewAmount(15204, 0),
	}
	if err := repo.Create(ctx, otherYear); err != nil {
		t.Fatalf("Create() other year error: %v", err)
	}

	// List for 2025 should return 3 children ordered by child_order.
	result, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear(2025) error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("ListByYear(2025) returned %d items, want 3", len(result))
	}
	for i, c := range result {
		if c.ChildOrder != i+1 {
			t.Errorf("result[%d].ChildOrder = %d, want %d", i, c.ChildOrder, i+1)
		}
	}
	if result[2].ZTP != true {
		t.Error("result[2].ZTP = false, want true")
	}

	// List for 2024 should return 1.
	result2024, err := repo.ListByYear(ctx, 2024)
	if err != nil {
		t.Fatalf("ListByYear(2024) error: %v", err)
	}
	if len(result2024) != 1 {
		t.Errorf("ListByYear(2024) returned %d items, want 1", len(result2024))
	}

	// List for non-existent year should return empty slice.
	resultEmpty, err := repo.ListByYear(ctx, 2099)
	if err != nil {
		t.Fatalf("ListByYear(2099) error: %v", err)
	}
	if len(resultEmpty) != 0 {
		t.Errorf("ListByYear(2099) returned %d items, want 0", len(resultEmpty))
	}
}

// --- TaxPersonalCreditsRepository ---

func TestTaxPersonalCreditsRepository_Upsert(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxPersonalCreditsRepository(db)
	ctx := context.Background()

	credits := &domain.TaxPersonalCredits{
		Year:             2025,
		IsStudent:        true,
		StudentMonths:    10,
		DisabilityLevel:  1,
		CreditStudent:    domain.NewAmount(4020, 0),
		CreditDisability: domain.NewAmount(2520, 0),
	}

	if err := repo.Upsert(ctx, credits); err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	if credits.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if credits.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestTaxPersonalCreditsRepository_Upsert_Replace(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxPersonalCreditsRepository(db)
	ctx := context.Background()

	credits := &domain.TaxPersonalCredits{
		Year:             2025,
		IsStudent:        true,
		StudentMonths:    10,
		DisabilityLevel:  0,
		CreditStudent:    domain.NewAmount(4020, 0),
		CreditDisability: 0,
	}
	if err := repo.Upsert(ctx, credits); err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	// Upsert again with different values.
	credits2 := &domain.TaxPersonalCredits{
		Year:             2025,
		IsStudent:        false,
		StudentMonths:    0,
		DisabilityLevel:  2,
		CreditStudent:    0,
		CreditDisability: domain.NewAmount(5040, 0),
	}
	if err := repo.Upsert(ctx, credits2); err != nil {
		t.Fatalf("Upsert() replace error: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("GetByYear() error: %v", err)
	}
	if got.IsStudent {
		t.Error("IsStudent = true, want false after replace")
	}
	if got.DisabilityLevel != 2 {
		t.Errorf("DisabilityLevel = %d, want 2", got.DisabilityLevel)
	}
	if got.CreditDisability != domain.NewAmount(5040, 0) {
		t.Errorf("CreditDisability = %d, want %d", got.CreditDisability, domain.NewAmount(5040, 0))
	}
}

func TestTaxPersonalCreditsRepository_GetByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxPersonalCreditsRepository(db)
	ctx := context.Background()

	credits := &domain.TaxPersonalCredits{
		Year:             2025,
		IsStudent:        true,
		StudentMonths:    10,
		DisabilityLevel:  1,
		CreditStudent:    domain.NewAmount(4020, 0),
		CreditDisability: domain.NewAmount(2520, 0),
	}
	if err := repo.Upsert(ctx, credits); err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("GetByYear() error: %v", err)
	}

	if got.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Year)
	}
	if !got.IsStudent {
		t.Error("IsStudent = false, want true")
	}
	if got.StudentMonths != 10 {
		t.Errorf("StudentMonths = %d, want 10", got.StudentMonths)
	}
	if got.DisabilityLevel != 1 {
		t.Errorf("DisabilityLevel = %d, want 1", got.DisabilityLevel)
	}
	if got.CreditStudent != domain.NewAmount(4020, 0) {
		t.Errorf("CreditStudent = %d, want %d", got.CreditStudent, domain.NewAmount(4020, 0))
	}
	if got.CreditDisability != domain.NewAmount(2520, 0) {
		t.Errorf("CreditDisability = %d, want %d", got.CreditDisability, domain.NewAmount(2520, 0))
	}
}

func TestTaxPersonalCreditsRepository_GetByYear_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxPersonalCreditsRepository(db)
	ctx := context.Background()

	_, err := repo.GetByYear(ctx, 2099)
	if err == nil {
		t.Error("expected error for non-existent year")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
