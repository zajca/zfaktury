package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupTaxCreditsSvc(t *testing.T) (*TaxCreditsService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	spouseRepo := repository.NewTaxSpouseCreditRepository(db)
	childRepo := repository.NewTaxChildCreditRepository(db)
	personalRepo := repository.NewTaxPersonalCreditsRepository(db)
	deductionRepo := repository.NewTaxDeductionRepository(db)
	svc := NewTaxCreditsService(spouseRepo, childRepo, personalRepo, deductionRepo)
	return svc, db
}

// --- UpsertSpouse tests ---

func TestTaxCreditsService_UpsertSpouse(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Jana Novakova",
		SpouseIncome:  domain.NewAmount(50000, 0),
		MonthsClaimed: 12,
	}

	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	got, err := svc.GetSpouse(ctx, 2025)
	if err != nil {
		t.Fatalf("GetSpouse() error: %v", err)
	}
	if got.SpouseName != "Jana Novakova" {
		t.Errorf("SpouseName = %q, want %q", got.SpouseName, "Jana Novakova")
	}
	if got.MonthsClaimed != 12 {
		t.Errorf("MonthsClaimed = %d, want 12", got.MonthsClaimed)
	}
}

func TestTaxCreditsService_UpsertSpouse_Update(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Jana Novakova",
		SpouseIncome:  domain.NewAmount(50000, 0),
		MonthsClaimed: 12,
	}
	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	// Update with different months.
	credit.MonthsClaimed = 6
	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Fatalf("UpsertSpouse() update error: %v", err)
	}

	got, err := svc.GetSpouse(ctx, 2025)
	if err != nil {
		t.Fatalf("GetSpouse() error: %v", err)
	}
	if got.MonthsClaimed != 6 {
		t.Errorf("MonthsClaimed = %d, want 6", got.MonthsClaimed)
	}
}

func TestTaxCreditsService_UpsertSpouse_InvalidYear(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	for _, year := range []int{1999, 2101} {
		credit := &domain.TaxSpouseCredit{
			Year:          year,
			SpouseName:    "Test",
			SpouseIncome:  0,
			MonthsClaimed: 12,
		}
		err := svc.UpsertSpouse(ctx, credit)
		if err == nil {
			t.Errorf("expected error for year %d", year)
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for year %d, got: %v", year, err)
		}
	}
}

func TestTaxCreditsService_UpsertSpouse_EmptyName(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "",
		MonthsClaimed: 12,
	}
	err := svc.UpsertSpouse(ctx, credit)
	if err == nil {
		t.Error("expected error for empty spouse name")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestTaxCreditsService_UpsertSpouse_InvalidMonths(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	for _, months := range []int{0, 13, -1} {
		credit := &domain.TaxSpouseCredit{
			Year:          2025,
			SpouseName:    "Test",
			MonthsClaimed: months,
		}
		err := svc.UpsertSpouse(ctx, credit)
		if err == nil {
			t.Errorf("expected error for months=%d", months)
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for months=%d, got: %v", months, err)
		}
	}
}

func TestTaxCreditsService_UpsertSpouse_ValidMonthsBoundary(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// MonthsClaimed=1 is valid.
	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Test 1",
		SpouseIncome:  0,
		MonthsClaimed: 1,
	}
	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Errorf("UpsertSpouse(months=1) unexpected error: %v", err)
	}

	// MonthsClaimed=12 is valid (different year to avoid conflict).
	credit2 := &domain.TaxSpouseCredit{
		Year:          2024,
		SpouseName:    "Test 12",
		SpouseIncome:  0,
		MonthsClaimed: 12,
	}
	if err := svc.UpsertSpouse(ctx, credit2); err != nil {
		t.Errorf("UpsertSpouse(months=12) unexpected error: %v", err)
	}
}

func TestTaxCreditsService_UpsertSpouse_NegativeIncome(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Test",
		SpouseIncome:  -1,
		MonthsClaimed: 12,
	}
	err := svc.UpsertSpouse(ctx, credit)
	if err == nil {
		t.Error("expected error for negative spouse income")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestTaxCreditsService_DeleteSpouse(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Jana",
		MonthsClaimed: 12,
	}
	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	if err := svc.DeleteSpouse(ctx, 2025); err != nil {
		t.Fatalf("DeleteSpouse() error: %v", err)
	}

	_, err := svc.GetSpouse(ctx, 2025)
	if err == nil {
		t.Error("expected error after delete")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// --- ComputeCredits tests ---

func TestTaxCreditsService_ComputeCredits_SpouseProportionalMonths(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Create spouse credit with 6 months claimed, income below threshold.
	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Jana",
		SpouseIncome:  domain.NewAmount(50000, 0), // below 68000 CZK
		MonthsClaimed: 6,
	}
	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	spouseCredit, _, _, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	expected := constants.SpouseCredit.Multiply(6.0 / 12.0)
	if spouseCredit != expected {
		t.Errorf("spouseCredit = %d, want %d (half of %d)", spouseCredit, expected, constants.SpouseCredit)
	}
}

func TestTaxCreditsService_ComputeCredits_SpouseFullYear(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Jana",
		SpouseIncome:  domain.NewAmount(60000, 0),
		MonthsClaimed: 12,
	}
	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	spouseCredit, _, _, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	if spouseCredit != constants.SpouseCredit {
		t.Errorf("spouseCredit = %d, want %d", spouseCredit, constants.SpouseCredit)
	}
}

func TestTaxCreditsService_ComputeCredits_SpouseIncomeAboveThreshold(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Spouse income >= 68000 CZK (6_800_000 halere) -> no credit.
	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Jana",
		SpouseIncome:  domain.NewAmount(68000, 0),
		MonthsClaimed: 12,
	}
	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	spouseCredit, _, _, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	if spouseCredit != 0 {
		t.Errorf("spouseCredit = %d, want 0 (income above threshold)", spouseCredit)
	}
}

func TestTaxCreditsService_ComputeCredits_SpouseZTPDoubling(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Jana",
		SpouseIncome:  domain.NewAmount(50000, 0),
		SpouseZTP:     true,
		MonthsClaimed: 12,
	}
	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	spouseCredit, _, _, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	expected := constants.SpouseCredit * 2
	if spouseCredit != expected {
		t.Errorf("spouseCredit = %d, want %d (ZTP doubled)", spouseCredit, expected)
	}
}

func TestTaxCreditsService_ComputeCredits_SpouseZTPPartialMonths(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	credit := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Jana",
		SpouseIncome:  domain.NewAmount(40000, 0),
		SpouseZTP:     true,
		MonthsClaimed: 3,
	}
	if err := svc.UpsertSpouse(ctx, credit); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	spouseCredit, _, _, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	// Proportional then doubled.
	expected := constants.SpouseCredit.Multiply(3.0/12.0) * 2
	if spouseCredit != expected {
		t.Errorf("spouseCredit = %d, want %d (ZTP + 3 months)", spouseCredit, expected)
	}
}

func TestTaxCreditsService_ComputeCredits_StudentProportional(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	personal := &domain.TaxPersonalCredits{
		Year:          2025,
		IsStudent:     true,
		StudentMonths: 8,
	}
	if err := svc.UpsertPersonal(ctx, personal); err != nil {
		t.Fatalf("UpsertPersonal() error: %v", err)
	}

	_, _, studentCredit, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	expected := constants.StudentCredit.Multiply(8.0 / 12.0)
	if studentCredit != expected {
		t.Errorf("studentCredit = %d, want %d", studentCredit, expected)
	}
}

func TestTaxCreditsService_ComputeCredits_StudentFullYear(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	personal := &domain.TaxPersonalCredits{
		Year:          2025,
		IsStudent:     true,
		StudentMonths: 12,
	}
	if err := svc.UpsertPersonal(ctx, personal); err != nil {
		t.Fatalf("UpsertPersonal() error: %v", err)
	}

	_, _, studentCredit, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	if studentCredit != constants.StudentCredit {
		t.Errorf("studentCredit = %d, want %d", studentCredit, constants.StudentCredit)
	}
}

func TestTaxCreditsService_ComputeCredits_StudentNotFlagged(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// IsStudent=false, even with months set -> no credit.
	personal := &domain.TaxPersonalCredits{
		Year:          2025,
		IsStudent:     false,
		StudentMonths: 12,
	}
	if err := svc.UpsertPersonal(ctx, personal); err != nil {
		t.Fatalf("UpsertPersonal() error: %v", err)
	}

	_, _, studentCredit, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	if studentCredit != 0 {
		t.Errorf("studentCredit = %d, want 0 (IsStudent=false)", studentCredit)
	}
}

func TestTaxCreditsService_ComputeCredits_DisabilityLevel1(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	personal := &domain.TaxPersonalCredits{
		Year:            2025,
		DisabilityLevel: 1,
	}
	if err := svc.UpsertPersonal(ctx, personal); err != nil {
		t.Fatalf("UpsertPersonal() error: %v", err)
	}

	_, disabilityCredit, _, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	if disabilityCredit != constants.DisabilityCredit1 {
		t.Errorf("disabilityCredit = %d, want %d (level 1)", disabilityCredit, constants.DisabilityCredit1)
	}
}

func TestTaxCreditsService_ComputeCredits_DisabilityLevel2(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	personal := &domain.TaxPersonalCredits{
		Year:            2025,
		DisabilityLevel: 2,
	}
	if err := svc.UpsertPersonal(ctx, personal); err != nil {
		t.Fatalf("UpsertPersonal() error: %v", err)
	}

	_, disabilityCredit, _, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	if disabilityCredit != constants.DisabilityCredit3 {
		t.Errorf("disabilityCredit = %d, want %d (level 2 -> DisabilityCredit3)", disabilityCredit, constants.DisabilityCredit3)
	}
}

func TestTaxCreditsService_ComputeCredits_DisabilityLevel3_ZTPP(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	personal := &domain.TaxPersonalCredits{
		Year:            2025,
		DisabilityLevel: 3,
	}
	if err := svc.UpsertPersonal(ctx, personal); err != nil {
		t.Fatalf("UpsertPersonal() error: %v", err)
	}

	_, disabilityCredit, _, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	if disabilityCredit != constants.DisabilityZTPP {
		t.Errorf("disabilityCredit = %d, want %d (level 3 -> ZTP/P)", disabilityCredit, constants.DisabilityZTPP)
	}
}

func TestTaxCreditsService_ComputeCredits_NoData(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// No spouse, no personal credits -> all zeros.
	spouse, disability, student, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}
	if spouse != 0 || disability != 0 || student != 0 {
		t.Errorf("expected all zeros, got spouse=%d disability=%d student=%d", spouse, disability, student)
	}
}

func TestTaxCreditsService_ComputeCredits_AllCombined(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Spouse with ZTP, partial months.
	spouseCr := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Jana",
		SpouseIncome:  domain.NewAmount(30000, 0),
		SpouseZTP:     true,
		MonthsClaimed: 6,
	}
	if err := svc.UpsertSpouse(ctx, spouseCr); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	// Student + disability level 1.
	personal := &domain.TaxPersonalCredits{
		Year:            2025,
		IsStudent:       true,
		StudentMonths:   10,
		DisabilityLevel: 1,
	}
	if err := svc.UpsertPersonal(ctx, personal); err != nil {
		t.Fatalf("UpsertPersonal() error: %v", err)
	}

	spouseAmt, disabilityAmt, studentAmt, err := svc.ComputeCredits(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCredits() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)

	expectedSpouse := constants.SpouseCredit.Multiply(6.0/12.0) * 2
	expectedStudent := constants.StudentCredit.Multiply(10.0 / 12.0)
	expectedDisability := constants.DisabilityCredit1

	if spouseAmt != expectedSpouse {
		t.Errorf("spouseAmt = %d, want %d", spouseAmt, expectedSpouse)
	}
	if disabilityAmt != expectedDisability {
		t.Errorf("disabilityAmt = %d, want %d", disabilityAmt, expectedDisability)
	}
	if studentAmt != expectedStudent {
		t.Errorf("studentAmt = %d, want %d", studentAmt, expectedStudent)
	}
}

// --- ComputeChildBenefit tests ---

func TestTaxCreditsService_ComputeChildBenefit_SingleChild(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	child := &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Marek",
		ChildOrder:    1,
		MonthsClaimed: 12,
	}
	if err := svc.CreateChild(ctx, child); err != nil {
		t.Fatalf("CreateChild() error: %v", err)
	}

	total, err := svc.ComputeChildBenefit(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeChildBenefit() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	if total != constants.ChildBenefit1 {
		t.Errorf("total = %d, want %d", total, constants.ChildBenefit1)
	}
}

func TestTaxCreditsService_ComputeChildBenefit_MultipleChildren(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	children := []domain.TaxChildCredit{
		{Year: 2025, ChildName: "Marek", ChildOrder: 1, MonthsClaimed: 12},
		{Year: 2025, ChildName: "Eva", ChildOrder: 2, MonthsClaimed: 12},
		{Year: 2025, ChildName: "Jan", ChildOrder: 3, MonthsClaimed: 12},
	}
	for i := range children {
		if err := svc.CreateChild(ctx, &children[i]); err != nil {
			t.Fatalf("CreateChild(%d) error: %v", i, err)
		}
	}

	total, err := svc.ComputeChildBenefit(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeChildBenefit() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	expected := constants.ChildBenefit1 + constants.ChildBenefit2 + constants.ChildBenefit3Plus
	if total != expected {
		t.Errorf("total = %d, want %d", total, expected)
	}
}

func TestTaxCreditsService_ComputeChildBenefit_PartialMonths(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	child := &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Marek",
		ChildOrder:    1,
		MonthsClaimed: 6,
	}
	if err := svc.CreateChild(ctx, child); err != nil {
		t.Fatalf("CreateChild() error: %v", err)
	}

	total, err := svc.ComputeChildBenefit(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeChildBenefit() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	expected := constants.ChildBenefit1.Multiply(6.0 / 12.0)
	if total != expected {
		t.Errorf("total = %d, want %d", total, expected)
	}
}

func TestTaxCreditsService_ComputeChildBenefit_ZTPDoubling(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	child := &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Marek",
		ChildOrder:    2,
		MonthsClaimed: 12,
		ZTP:           true,
	}
	if err := svc.CreateChild(ctx, child); err != nil {
		t.Fatalf("CreateChild() error: %v", err)
	}

	total, err := svc.ComputeChildBenefit(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeChildBenefit() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	expected := constants.ChildBenefit2 * 2
	if total != expected {
		t.Errorf("total = %d, want %d (ZTP doubled)", total, expected)
	}
}

func TestTaxCreditsService_ComputeChildBenefit_NoChildren(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	total, err := svc.ComputeChildBenefit(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeChildBenefit() error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

// --- Child credit CRUD tests ---

func TestTaxCreditsService_CreateChild_InvalidOrder(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	for _, order := range []int{0, 4, -1} {
		child := &domain.TaxChildCredit{
			Year:          2025,
			ChildOrder:    order,
			MonthsClaimed: 12,
		}
		err := svc.CreateChild(ctx, child)
		if err == nil {
			t.Errorf("expected error for order=%d", order)
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for order=%d, got: %v", order, err)
		}
	}
}

func TestTaxCreditsService_CreateChild_InvalidMonths(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	for _, months := range []int{0, 13} {
		child := &domain.TaxChildCredit{
			Year:          2025,
			ChildOrder:    1,
			MonthsClaimed: months,
		}
		err := svc.CreateChild(ctx, child)
		if err == nil {
			t.Errorf("expected error for months=%d", months)
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for months=%d, got: %v", months, err)
		}
	}
}

func TestTaxCreditsService_ListChildren(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	for _, order := range []int{1, 2} {
		child := &domain.TaxChildCredit{
			Year:          2025,
			ChildName:     "Child",
			ChildOrder:    order,
			MonthsClaimed: 12,
		}
		if err := svc.CreateChild(ctx, child); err != nil {
			t.Fatalf("CreateChild() error: %v", err)
		}
	}

	children, err := svc.ListChildren(ctx, 2025)
	if err != nil {
		t.Fatalf("ListChildren() error: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("ListChildren() returned %d, want 2", len(children))
	}
}

func TestTaxCreditsService_DeleteChild(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	child := &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Marek",
		ChildOrder:    1,
		MonthsClaimed: 12,
	}
	if err := svc.CreateChild(ctx, child); err != nil {
		t.Fatalf("CreateChild() error: %v", err)
	}

	if err := svc.DeleteChild(ctx, child.ID); err != nil {
		t.Fatalf("DeleteChild() error: %v", err)
	}

	children, err := svc.ListChildren(ctx, 2025)
	if err != nil {
		t.Fatalf("ListChildren() error: %v", err)
	}
	if len(children) != 0 {
		t.Errorf("expected 0 children after delete, got %d", len(children))
	}
}

// --- Personal credits tests ---

func TestTaxCreditsService_UpsertPersonal_InvalidStudentMonths(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	for _, months := range []int{-1, 13} {
		personal := &domain.TaxPersonalCredits{
			Year:          2025,
			StudentMonths: months,
		}
		err := svc.UpsertPersonal(ctx, personal)
		if err == nil {
			t.Errorf("expected error for student months=%d", months)
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for months=%d, got: %v", months, err)
		}
	}
}

func TestTaxCreditsService_UpsertPersonal_InvalidDisabilityLevel(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	for _, level := range []int{-1, 4} {
		personal := &domain.TaxPersonalCredits{
			Year:            2025,
			DisabilityLevel: level,
		}
		err := svc.UpsertPersonal(ctx, personal)
		if err == nil {
			t.Errorf("expected error for disability level=%d", level)
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for level=%d, got: %v", level, err)
		}
	}
}

func TestTaxCreditsService_GetPersonal(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	personal := &domain.TaxPersonalCredits{
		Year:            2025,
		IsStudent:       true,
		StudentMonths:   10,
		DisabilityLevel: 2,
	}
	if err := svc.UpsertPersonal(ctx, personal); err != nil {
		t.Fatalf("UpsertPersonal() error: %v", err)
	}

	got, err := svc.GetPersonal(ctx, 2025)
	if err != nil {
		t.Fatalf("GetPersonal() error: %v", err)
	}
	if !got.IsStudent {
		t.Error("expected IsStudent=true")
	}
	if got.StudentMonths != 10 {
		t.Errorf("StudentMonths = %d, want 10", got.StudentMonths)
	}
	if got.DisabilityLevel != 2 {
		t.Errorf("DisabilityLevel = %d, want 2", got.DisabilityLevel)
	}
}

// --- ComputeDeductions tests ---

func TestTaxCreditsService_ComputeDeductions_BelowCap(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionMortgage,
		Description:   "Mortgage interest",
		ClaimedAmount: domain.NewAmount(100000, 0), // 100k CZK, cap is 150k
	}
	if err := svc.CreateDeduction(ctx, ded); err != nil {
		t.Fatalf("CreateDeduction() error: %v", err)
	}

	taxBase := domain.NewAmount(1000000, 0)
	total, err := svc.ComputeDeductions(ctx, 2025, taxBase)
	if err != nil {
		t.Fatalf("ComputeDeductions() error: %v", err)
	}

	if total != domain.NewAmount(100000, 0) {
		t.Errorf("total = %d, want %d (below cap)", total, domain.NewAmount(100000, 0))
	}

	// Verify the deduction was updated with max and allowed amounts.
	deductions, _ := svc.ListDeductions(ctx, 2025)
	if len(deductions) != 1 {
		t.Fatalf("expected 1 deduction, got %d", len(deductions))
	}
	if deductions[0].MaxAmount != deductionCapMortgage {
		t.Errorf("MaxAmount = %d, want %d", deductions[0].MaxAmount, deductionCapMortgage)
	}
	if deductions[0].AllowedAmount != domain.NewAmount(100000, 0) {
		t.Errorf("AllowedAmount = %d, want %d", deductions[0].AllowedAmount, domain.NewAmount(100000, 0))
	}
}

func TestTaxCreditsService_ComputeDeductions_AboveCap(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Life insurance cap is 24000 CZK; claim 30000.
	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionLifeInsurance,
		Description:   "Life insurance",
		ClaimedAmount: domain.NewAmount(30000, 0),
	}
	if err := svc.CreateDeduction(ctx, ded); err != nil {
		t.Fatalf("CreateDeduction() error: %v", err)
	}

	taxBase := domain.NewAmount(1000000, 0)
	total, err := svc.ComputeDeductions(ctx, 2025, taxBase)
	if err != nil {
		t.Fatalf("ComputeDeductions() error: %v", err)
	}

	if total != deductionCapLifeInsurance {
		t.Errorf("total = %d, want %d (capped)", total, deductionCapLifeInsurance)
	}
}

func TestTaxCreditsService_ComputeDeductions_MultipleSameCategory(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Two pension deductions totaling 30000 CZK, cap is 24000.
	ded1 := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionPension,
		Description:   "Pension 1",
		ClaimedAmount: domain.NewAmount(15000, 0),
	}
	ded2 := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionPension,
		Description:   "Pension 2",
		ClaimedAmount: domain.NewAmount(15000, 0),
	}
	if err := svc.CreateDeduction(ctx, ded1); err != nil {
		t.Fatalf("CreateDeduction(1) error: %v", err)
	}
	if err := svc.CreateDeduction(ctx, ded2); err != nil {
		t.Fatalf("CreateDeduction(2) error: %v", err)
	}

	taxBase := domain.NewAmount(1000000, 0)
	total, err := svc.ComputeDeductions(ctx, 2025, taxBase)
	if err != nil {
		t.Fatalf("ComputeDeductions() error: %v", err)
	}

	// First gets 15000, second gets remaining 9000 (24000-15000).
	if total != deductionCapPension {
		t.Errorf("total = %d, want %d (capped across two entries)", total, deductionCapPension)
	}
}

func TestTaxCreditsService_ComputeDeductions_DonationProportionalCap(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Donation cap is 15% of tax base.
	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionDonation,
		Description:   "Charity",
		ClaimedAmount: domain.NewAmount(200000, 0),
	}
	if err := svc.CreateDeduction(ctx, ded); err != nil {
		t.Fatalf("CreateDeduction() error: %v", err)
	}

	taxBase := domain.NewAmount(1000000, 0) // 15% = 150000
	total, err := svc.ComputeDeductions(ctx, 2025, taxBase)
	if err != nil {
		t.Fatalf("ComputeDeductions() error: %v", err)
	}

	expectedCap := taxBase.Multiply(0.15)
	if total != expectedCap {
		t.Errorf("total = %d, want %d (15%% of tax base)", total, expectedCap)
	}
}

func TestTaxCreditsService_ComputeDeductions_UnionDuesCap(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Union dues cap is 3000 CZK.
	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionUnionDues,
		Description:   "Union",
		ClaimedAmount: domain.NewAmount(5000, 0),
	}
	if err := svc.CreateDeduction(ctx, ded); err != nil {
		t.Fatalf("CreateDeduction() error: %v", err)
	}

	taxBase := domain.NewAmount(1000000, 0)
	total, err := svc.ComputeDeductions(ctx, 2025, taxBase)
	if err != nil {
		t.Fatalf("ComputeDeductions() error: %v", err)
	}

	if total != deductionCapUnionDues {
		t.Errorf("total = %d, want %d (union dues cap)", total, deductionCapUnionDues)
	}
}

func TestTaxCreditsService_ComputeDeductions_MixedCategories(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	deductions := []domain.TaxDeduction{
		{Year: 2025, Category: domain.DeductionMortgage, Description: "Mortgage", ClaimedAmount: domain.NewAmount(100000, 0)},
		{Year: 2025, Category: domain.DeductionLifeInsurance, Description: "Life ins", ClaimedAmount: domain.NewAmount(20000, 0)},
		{Year: 2025, Category: domain.DeductionPension, Description: "Pension", ClaimedAmount: domain.NewAmount(10000, 0)},
	}
	for i := range deductions {
		if err := svc.CreateDeduction(ctx, &deductions[i]); err != nil {
			t.Fatalf("CreateDeduction(%d) error: %v", i, err)
		}
	}

	taxBase := domain.NewAmount(1000000, 0)
	total, err := svc.ComputeDeductions(ctx, 2025, taxBase)
	if err != nil {
		t.Fatalf("ComputeDeductions() error: %v", err)
	}

	// All below their caps, so total = sum of claimed.
	expected := domain.NewAmount(100000, 0) + domain.NewAmount(20000, 0) + domain.NewAmount(10000, 0)
	if total != expected {
		t.Errorf("total = %d, want %d", total, expected)
	}
}

func TestTaxCreditsService_ComputeDeductions_NoDeductions(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	total, err := svc.ComputeDeductions(ctx, 2025, domain.NewAmount(1000000, 0))
	if err != nil {
		t.Fatalf("ComputeDeductions() error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

// --- Deduction CRUD tests ---

func TestTaxCreditsService_CreateDeduction_InvalidCategory(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      "invalid_category",
		ClaimedAmount: domain.NewAmount(1000, 0),
	}
	err := svc.CreateDeduction(ctx, ded)
	if err == nil {
		t.Error("expected error for invalid category")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestTaxCreditsService_CreateDeduction_NegativeAmount(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionMortgage,
		ClaimedAmount: -1,
	}
	err := svc.CreateDeduction(ctx, ded)
	if err == nil {
		t.Error("expected error for negative amount")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestTaxCreditsService_UpdateDeduction(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionMortgage,
		Description:   "Original",
		ClaimedAmount: domain.NewAmount(50000, 0),
	}
	if err := svc.CreateDeduction(ctx, ded); err != nil {
		t.Fatalf("CreateDeduction() error: %v", err)
	}

	ded.Description = "Updated"
	ded.ClaimedAmount = domain.NewAmount(75000, 0)
	if err := svc.UpdateDeduction(ctx, ded); err != nil {
		t.Fatalf("UpdateDeduction() error: %v", err)
	}

	got, err := svc.GetDeduction(ctx, ded.ID)
	if err != nil {
		t.Fatalf("GetDeduction() error: %v", err)
	}
	if got.Description != "Updated" {
		t.Errorf("Description = %q, want %q", got.Description, "Updated")
	}
	if got.ClaimedAmount != domain.NewAmount(75000, 0) {
		t.Errorf("ClaimedAmount = %d, want %d", got.ClaimedAmount, domain.NewAmount(75000, 0))
	}
}

func TestTaxCreditsService_DeleteDeduction(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionMortgage,
		ClaimedAmount: domain.NewAmount(50000, 0),
	}
	if err := svc.CreateDeduction(ctx, ded); err != nil {
		t.Fatalf("CreateDeduction() error: %v", err)
	}

	if err := svc.DeleteDeduction(ctx, ded.ID); err != nil {
		t.Fatalf("DeleteDeduction() error: %v", err)
	}

	deductions, err := svc.ListDeductions(ctx, 2025)
	if err != nil {
		t.Fatalf("ListDeductions() error: %v", err)
	}
	if len(deductions) != 0 {
		t.Errorf("expected 0 deductions after delete, got %d", len(deductions))
	}
}

// --- CopyFromYear tests ---

func TestTaxCreditsService_CopyFromYear(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Set up source year 2024 data.
	spouse := &domain.TaxSpouseCredit{
		Year:          2024,
		SpouseName:    "Jana",
		SpouseIncome:  domain.NewAmount(50000, 0),
		SpouseZTP:     true,
		MonthsClaimed: 10,
		CreditAmount:  domain.NewAmount(99999, 0), // should be zeroed
	}
	if err := svc.UpsertSpouse(ctx, spouse); err != nil {
		t.Fatalf("UpsertSpouse() error: %v", err)
	}

	child := &domain.TaxChildCredit{
		Year:          2024,
		ChildName:     "Marek",
		ChildOrder:    1,
		MonthsClaimed: 8,
		ZTP:           true,
		CreditAmount:  domain.NewAmount(88888, 0),
	}
	if err := svc.CreateChild(ctx, child); err != nil {
		t.Fatalf("CreateChild() error: %v", err)
	}

	personal := &domain.TaxPersonalCredits{
		Year:            2024,
		IsStudent:       true,
		StudentMonths:   6,
		DisabilityLevel: 2,
		CreditStudent:   domain.NewAmount(77777, 0),
	}
	if err := svc.UpsertPersonal(ctx, personal); err != nil {
		t.Fatalf("UpsertPersonal() error: %v", err)
	}

	ded := &domain.TaxDeduction{
		Year:          2024,
		Category:      domain.DeductionMortgage,
		Description:   "Mortgage",
		ClaimedAmount: domain.NewAmount(100000, 0),
	}
	if err := svc.CreateDeduction(ctx, ded); err != nil {
		t.Fatalf("CreateDeduction() error: %v", err)
	}

	// Copy to 2025.
	if err := svc.CopyFromYear(ctx, 2024, 2025); err != nil {
		t.Fatalf("CopyFromYear() error: %v", err)
	}

	// Verify spouse was copied.
	copiedSpouse, err := svc.GetSpouse(ctx, 2025)
	if err != nil {
		t.Fatalf("GetSpouse(2025) error: %v", err)
	}
	if copiedSpouse.SpouseName != "Jana" {
		t.Errorf("SpouseName = %q, want %q", copiedSpouse.SpouseName, "Jana")
	}
	if copiedSpouse.MonthsClaimed != 12 {
		t.Errorf("MonthsClaimed = %d, want 12 (reset)", copiedSpouse.MonthsClaimed)
	}
	if copiedSpouse.CreditAmount != 0 {
		t.Errorf("CreditAmount = %d, want 0 (zeroed)", copiedSpouse.CreditAmount)
	}
	if copiedSpouse.SpouseZTP != true {
		t.Error("expected SpouseZTP=true to be preserved")
	}

	// Verify child was copied.
	copiedChildren, err := svc.ListChildren(ctx, 2025)
	if err != nil {
		t.Fatalf("ListChildren(2025) error: %v", err)
	}
	if len(copiedChildren) != 1 {
		t.Fatalf("expected 1 child, got %d", len(copiedChildren))
	}
	if copiedChildren[0].ChildName != "Marek" {
		t.Errorf("ChildName = %q, want %q", copiedChildren[0].ChildName, "Marek")
	}
	if copiedChildren[0].MonthsClaimed != 12 {
		t.Errorf("child MonthsClaimed = %d, want 12", copiedChildren[0].MonthsClaimed)
	}
	if copiedChildren[0].CreditAmount != 0 {
		t.Errorf("child CreditAmount = %d, want 0", copiedChildren[0].CreditAmount)
	}
	if copiedChildren[0].ZTP != true {
		t.Error("expected child ZTP=true preserved")
	}

	// Verify personal was copied.
	copiedPersonal, err := svc.GetPersonal(ctx, 2025)
	if err != nil {
		t.Fatalf("GetPersonal(2025) error: %v", err)
	}
	if !copiedPersonal.IsStudent {
		t.Error("expected IsStudent=true preserved")
	}
	if copiedPersonal.CreditStudent != 0 {
		t.Errorf("CreditStudent = %d, want 0 (zeroed)", copiedPersonal.CreditStudent)
	}
	if copiedPersonal.CreditDisability != 0 {
		t.Errorf("CreditDisability = %d, want 0 (zeroed)", copiedPersonal.CreditDisability)
	}

	// Verify deduction was copied.
	copiedDeds, err := svc.ListDeductions(ctx, 2025)
	if err != nil {
		t.Fatalf("ListDeductions(2025) error: %v", err)
	}
	if len(copiedDeds) != 1 {
		t.Fatalf("expected 1 deduction, got %d", len(copiedDeds))
	}
	if copiedDeds[0].Category != domain.DeductionMortgage {
		t.Errorf("Category = %q, want %q", copiedDeds[0].Category, domain.DeductionMortgage)
	}
	if copiedDeds[0].Description != "Mortgage" {
		t.Errorf("Description = %q, want %q", copiedDeds[0].Description, "Mortgage")
	}
	if copiedDeds[0].ClaimedAmount != 0 {
		t.Errorf("ClaimedAmount = %d, want 0 (zeroed)", copiedDeds[0].ClaimedAmount)
	}
}

func TestTaxCreditsService_CopyFromYear_SameYear(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	err := svc.CopyFromYear(ctx, 2025, 2025)
	if err == nil {
		t.Error("expected error for same source/target year")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestTaxCreditsService_CopyFromYear_SkipsExistingData(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Set up source year.
	srcSpouse := &domain.TaxSpouseCredit{
		Year:          2024,
		SpouseName:    "Source Jana",
		MonthsClaimed: 10,
	}
	if err := svc.UpsertSpouse(ctx, srcSpouse); err != nil {
		t.Fatalf("UpsertSpouse(source) error: %v", err)
	}

	// Set up target year with existing data.
	targetSpouse := &domain.TaxSpouseCredit{
		Year:          2025,
		SpouseName:    "Target Jana",
		MonthsClaimed: 6,
	}
	if err := svc.UpsertSpouse(ctx, targetSpouse); err != nil {
		t.Fatalf("UpsertSpouse(target) error: %v", err)
	}

	// Copy should skip spouse since target already has it.
	if err := svc.CopyFromYear(ctx, 2024, 2025); err != nil {
		t.Fatalf("CopyFromYear() error: %v", err)
	}

	got, err := svc.GetSpouse(ctx, 2025)
	if err != nil {
		t.Fatalf("GetSpouse(2025) error: %v", err)
	}
	// Should still have target data, not source.
	if got.SpouseName != "Target Jana" {
		t.Errorf("SpouseName = %q, want %q (should not be overwritten)", got.SpouseName, "Target Jana")
	}
	if got.MonthsClaimed != 6 {
		t.Errorf("MonthsClaimed = %d, want 6 (should not be overwritten)", got.MonthsClaimed)
	}
}

func TestTaxCreditsService_CopyFromYear_EmptySource(t *testing.T) {
	svc, _ := setupTaxCreditsSvc(t)
	ctx := context.Background()

	// Copy from empty year -- should not fail.
	if err := svc.CopyFromYear(ctx, 2024, 2025); err != nil {
		t.Fatalf("CopyFromYear(empty source) error: %v", err)
	}

	// Target should have no data.
	_, err := svc.GetSpouse(ctx, 2025)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for spouse, got: %v", err)
	}
	children, _ := svc.ListChildren(ctx, 2025)
	if len(children) != 0 {
		t.Errorf("expected 0 children, got %d", len(children))
	}
}
