package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)


// TaxCreditsService provides business logic for tax credits and deductions.
type TaxCreditsService struct {
	spouseRepo    repository.TaxSpouseCreditRepo
	childRepo     repository.TaxChildCreditRepo
	personalRepo  repository.TaxPersonalCreditsRepo
	deductionRepo repository.TaxDeductionRepo
}

// NewTaxCreditsService creates a new TaxCreditsService.
func NewTaxCreditsService(
	spouseRepo repository.TaxSpouseCreditRepo,
	childRepo repository.TaxChildCreditRepo,
	personalRepo repository.TaxPersonalCreditsRepo,
	deductionRepo repository.TaxDeductionRepo,
) *TaxCreditsService {
	return &TaxCreditsService{
		spouseRepo:    spouseRepo,
		childRepo:     childRepo,
		personalRepo:  personalRepo,
		deductionRepo: deductionRepo,
	}
}

// --- Spouse credit CRUD ---

// UpsertSpouse validates and persists a spouse credit for the given year.
func (s *TaxCreditsService) UpsertSpouse(ctx context.Context, credit *domain.TaxSpouseCredit) error {
	if credit.Year < 2000 || credit.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if credit.SpouseName == "" {
		return fmt.Errorf("spouse name is required: %w", domain.ErrInvalidInput)
	}
	if credit.SpouseIncome < 0 || credit.SpouseIncome > domain.NewAmount(100_000_000, 0) {
		return fmt.Errorf("spouse income out of valid range: %w", domain.ErrInvalidInput)
	}
	if credit.MonthsClaimed < 1 || credit.MonthsClaimed > 12 {
		return fmt.Errorf("months claimed must be 1-12: %w", domain.ErrInvalidInput)
	}
	if err := s.spouseRepo.Upsert(ctx, credit); err != nil {
		return fmt.Errorf("upserting spouse credit: %w", err)
	}
	return nil
}

// GetSpouse retrieves the spouse credit for a given year.
func (s *TaxCreditsService) GetSpouse(ctx context.Context, year int) (*domain.TaxSpouseCredit, error) {
	credit, err := s.spouseRepo.GetByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching spouse credit: %w", err)
	}
	return credit, nil
}

// DeleteSpouse removes the spouse credit for a given year.
func (s *TaxCreditsService) DeleteSpouse(ctx context.Context, year int) error {
	if err := s.spouseRepo.DeleteByYear(ctx, year); err != nil {
		return fmt.Errorf("deleting spouse credit: %w", err)
	}
	return nil
}

// --- Child credit CRUD ---

// CreateChild validates and persists a new child credit entry.
func (s *TaxCreditsService) CreateChild(ctx context.Context, credit *domain.TaxChildCredit) error {
	if credit.Year < 2000 || credit.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if credit.ChildOrder < 1 || credit.ChildOrder > 3 {
		return fmt.Errorf("child order must be 1-3: %w", domain.ErrInvalidInput)
	}
	if credit.MonthsClaimed < 1 || credit.MonthsClaimed > 12 {
		return fmt.Errorf("months claimed must be 1-12: %w", domain.ErrInvalidInput)
	}
	if err := s.childRepo.Create(ctx, credit); err != nil {
		return fmt.Errorf("creating child credit: %w", err)
	}
	return nil
}

// UpdateChild validates and updates an existing child credit entry.
func (s *TaxCreditsService) UpdateChild(ctx context.Context, credit *domain.TaxChildCredit) error {
	if credit.Year < 2000 || credit.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if credit.ChildOrder < 1 || credit.ChildOrder > 3 {
		return fmt.Errorf("child order must be 1-3: %w", domain.ErrInvalidInput)
	}
	if credit.MonthsClaimed < 1 || credit.MonthsClaimed > 12 {
		return fmt.Errorf("months claimed must be 1-12: %w", domain.ErrInvalidInput)
	}
	if err := s.childRepo.Update(ctx, credit); err != nil {
		return fmt.Errorf("updating child credit: %w", err)
	}
	return nil
}

// DeleteChild removes a child credit entry by ID.
func (s *TaxCreditsService) DeleteChild(ctx context.Context, id int64) error {
	if err := s.childRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting child credit: %w", err)
	}
	return nil
}

// ListChildren retrieves all child credit entries for a given year.
func (s *TaxCreditsService) ListChildren(ctx context.Context, year int) ([]domain.TaxChildCredit, error) {
	children, err := s.childRepo.ListByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing child credits: %w", err)
	}
	return children, nil
}

// --- Personal credits CRUD ---

// UpsertPersonal validates and persists personal credits for the given year.
func (s *TaxCreditsService) UpsertPersonal(ctx context.Context, credits *domain.TaxPersonalCredits) error {
	if credits.Year < 2000 || credits.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if credits.StudentMonths < 0 || credits.StudentMonths > 12 {
		return fmt.Errorf("student months must be 0-12: %w", domain.ErrInvalidInput)
	}
	if credits.DisabilityLevel < 0 || credits.DisabilityLevel > 3 {
		return fmt.Errorf("disability level must be 0-3: %w", domain.ErrInvalidInput)
	}
	if err := s.personalRepo.Upsert(ctx, credits); err != nil {
		return fmt.Errorf("upserting personal credits: %w", err)
	}
	return nil
}

// GetPersonal retrieves personal credits for a given year.
func (s *TaxCreditsService) GetPersonal(ctx context.Context, year int) (*domain.TaxPersonalCredits, error) {
	credits, err := s.personalRepo.GetByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching personal credits: %w", err)
	}
	return credits, nil
}

// --- Deduction CRUD ---

// validDeductionCategories contains the allowed deduction category values.
var validDeductionCategories = map[string]bool{
	domain.DeductionMortgage:      true,
	domain.DeductionLifeInsurance: true,
	domain.DeductionPension:       true,
	domain.DeductionDonation:      true,
	domain.DeductionUnionDues:     true,
}

// CreateDeduction validates and persists a new deduction entry.
func (s *TaxCreditsService) CreateDeduction(ctx context.Context, ded *domain.TaxDeduction) error {
	if ded.Year < 2000 || ded.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if !validDeductionCategories[ded.Category] {
		return fmt.Errorf("invalid deduction category %q: %w", ded.Category, domain.ErrInvalidInput)
	}
	if ded.ClaimedAmount < 0 || ded.ClaimedAmount > domain.NewAmount(100_000_000, 0) {
		return fmt.Errorf("claimed amount out of valid range: %w", domain.ErrInvalidInput)
	}
	if err := s.deductionRepo.Create(ctx, ded); err != nil {
		return fmt.Errorf("creating deduction: %w", err)
	}
	return nil
}

// UpdateDeduction validates and updates an existing deduction entry.
func (s *TaxCreditsService) UpdateDeduction(ctx context.Context, ded *domain.TaxDeduction) error {
	if ded.Year < 2000 || ded.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if !validDeductionCategories[ded.Category] {
		return fmt.Errorf("invalid deduction category %q: %w", ded.Category, domain.ErrInvalidInput)
	}
	if ded.ClaimedAmount < 0 || ded.ClaimedAmount > domain.NewAmount(100_000_000, 0) {
		return fmt.Errorf("claimed amount out of valid range: %w", domain.ErrInvalidInput)
	}
	if err := s.deductionRepo.Update(ctx, ded); err != nil {
		return fmt.Errorf("updating deduction: %w", err)
	}
	return nil
}

// DeleteDeduction removes a deduction entry by ID.
func (s *TaxCreditsService) DeleteDeduction(ctx context.Context, id int64) error {
	if err := s.deductionRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting deduction: %w", err)
	}
	return nil
}

// GetDeduction retrieves a deduction entry by ID.
func (s *TaxCreditsService) GetDeduction(ctx context.Context, id int64) (*domain.TaxDeduction, error) {
	ded, err := s.deductionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching deduction: %w", err)
	}
	return ded, nil
}

// ListDeductions retrieves all deduction entries for a given year.
func (s *TaxCreditsService) ListDeductions(ctx context.Context, year int) ([]domain.TaxDeduction, error) {
	deductions, err := s.deductionRepo.ListByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing deductions: %w", err)
	}
	return deductions, nil
}

// --- Computation methods ---

// ComputeCredits computes the spouse, disability, and student credit amounts for the given year.
// Returns zero amounts for credits that have no data (ErrNotFound).
func (s *TaxCreditsService) ComputeCredits(ctx context.Context, year int) (spouseCredit, disabilityCredit, studentCredit domain.Amount, err error) {
	constants, err := GetTaxConstants(year)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("getting tax constants for credits: %w", err)
	}

	// Spouse credit.
	spouse, err := s.spouseRepo.GetByYear(ctx, year)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return 0, 0, 0, fmt.Errorf("fetching spouse credit for compute: %w", err)
	}
	if spouse != nil && spouse.SpouseIncome < constants.SpouseIncomeLimit {
		spouseCredit = constants.SpouseCredit.Multiply(float64(spouse.MonthsClaimed) / 12.0)
		if spouse.SpouseZTP {
			spouseCredit *= 2
		}
	}

	// Personal credits (student + disability).
	personal, err := s.personalRepo.GetByYear(ctx, year)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return 0, 0, 0, fmt.Errorf("fetching personal credits for compute: %w", err)
	}
	if personal != nil {
		// Student credit: proportional by months.
		if personal.IsStudent && personal.StudentMonths > 0 {
			studentCredit = constants.StudentCredit.Multiply(float64(personal.StudentMonths) / 12.0)
		}

		// Disability credit by level.
		switch personal.DisabilityLevel {
		case 1:
			disabilityCredit = constants.DisabilityCredit1
		case 2:
			disabilityCredit = constants.DisabilityCredit3
		case 3:
			disabilityCredit = constants.DisabilityZTPP
		}
	}

	return spouseCredit, disabilityCredit, studentCredit, nil
}

// ComputeChildBenefit computes the total child benefit amount for the given year.
func (s *TaxCreditsService) ComputeChildBenefit(ctx context.Context, year int) (domain.Amount, error) {
	constants, err := GetTaxConstants(year)
	if err != nil {
		return 0, fmt.Errorf("getting tax constants for child benefit: %w", err)
	}

	children, err := s.childRepo.ListByYear(ctx, year)
	if err != nil {
		return 0, fmt.Errorf("listing children for benefit compute: %w", err)
	}

	var total domain.Amount
	for _, child := range children {
		var base domain.Amount
		switch child.ChildOrder {
		case 1:
			base = constants.ChildBenefit1
		case 2:
			base = constants.ChildBenefit2
		default:
			base = constants.ChildBenefit3Plus
		}

		amount := base.Multiply(float64(child.MonthsClaimed) / 12.0)
		if child.ZTP {
			amount *= 2
		}
		total += amount
	}

	return total, nil
}

// ComputeDeductions computes allowed deduction amounts for the given year,
// applying statutory caps per category. Updates each deduction's MaxAmount and
// AllowedAmount via the repository.
func (s *TaxCreditsService) ComputeDeductions(ctx context.Context, year int, taxBase domain.Amount) (domain.Amount, error) {
	constants, err := GetTaxConstants(year)
	if err != nil {
		return 0, fmt.Errorf("getting tax constants for deductions: %w", err)
	}

	deductions, err := s.deductionRepo.ListByYear(ctx, year)
	if err != nil {
		return 0, fmt.Errorf("listing deductions for compute: %w", err)
	}

	// Calculate category caps.
	categoryCaps := map[string]domain.Amount{
		domain.DeductionMortgage:      constants.DeductionCapMortgage,
		domain.DeductionLifeInsurance: constants.DeductionCapLifeInsurance,
		domain.DeductionPension:       constants.DeductionCapPension,
		domain.DeductionUnionDues:     constants.DeductionCapUnionDues,
		domain.DeductionDonation:      taxBase.Multiply(0.15),
	}

	// Track remaining cap per category.
	remainingCap := make(map[string]domain.Amount)
	for cat, cap := range categoryCaps {
		remainingCap[cat] = cap
	}

	var totalAllowed domain.Amount
	for i := range deductions {
		ded := &deductions[i]
		cap := categoryCaps[ded.Category]
		remaining := remainingCap[ded.Category]

		allowed := ded.ClaimedAmount
		if allowed > remaining {
			allowed = remaining
		}
		if allowed < 0 {
			allowed = 0
		}

		ded.MaxAmount = cap
		ded.AllowedAmount = allowed
		remainingCap[ded.Category] = remaining - allowed
		totalAllowed += allowed

		if err := s.deductionRepo.Update(ctx, ded); err != nil {
			return 0, fmt.Errorf("updating deduction %d after compute: %w", ded.ID, err)
		}
	}

	return totalAllowed, nil
}

// CopyFromYear copies credits and deductions from sourceYear to targetYear.
// Skips if the target year already has data. Copied entries have zero computed amounts.
func (s *TaxCreditsService) CopyFromYear(ctx context.Context, sourceYear, targetYear int) error {
	if sourceYear == targetYear {
		return fmt.Errorf("source and target year must differ: %w", domain.ErrInvalidInput)
	}

	// Check if target already has data -- skip each entity type independently.
	spouseCopied := false
	existingSpouse, err := s.spouseRepo.GetByYear(ctx, targetYear)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("checking existing spouse credit for copy: %w", err)
	}
	if existingSpouse != nil {
		spouseCopied = true
	}

	childrenCopied := false
	existingChildren, err := s.childRepo.ListByYear(ctx, targetYear)
	if err != nil {
		return fmt.Errorf("checking existing child credits for copy: %w", err)
	}
	if len(existingChildren) > 0 {
		childrenCopied = true
	}

	personalCopied := false
	existingPersonal, err := s.personalRepo.GetByYear(ctx, targetYear)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("checking existing personal credits for copy: %w", err)
	}
	if existingPersonal != nil {
		personalCopied = true
	}

	deductionsCopied := false
	existingDeductions, err := s.deductionRepo.ListByYear(ctx, targetYear)
	if err != nil {
		return fmt.Errorf("checking existing deductions for copy: %w", err)
	}
	if len(existingDeductions) > 0 {
		deductionsCopied = true
	}

	// Copy spouse credit.
	if !spouseCopied {
		srcSpouse, err := s.spouseRepo.GetByYear(ctx, sourceYear)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("fetching source spouse credit for copy: %w", err)
		}
		if srcSpouse != nil {
			newSpouse := *srcSpouse
			newSpouse.ID = 0
			newSpouse.Year = targetYear
			newSpouse.MonthsClaimed = 12
			newSpouse.CreditAmount = 0
			if err := s.spouseRepo.Upsert(ctx, &newSpouse); err != nil {
				return fmt.Errorf("copying spouse credit: %w", err)
			}
		}
	}

	// Copy children.
	if !childrenCopied {
		srcChildren, err := s.childRepo.ListByYear(ctx, sourceYear)
		if err != nil {
			return fmt.Errorf("fetching source child credits for copy: %w", err)
		}
		for _, child := range srcChildren {
			newChild := child
			newChild.ID = 0
			newChild.Year = targetYear
			newChild.MonthsClaimed = 12
			newChild.CreditAmount = 0
			if err := s.childRepo.Create(ctx, &newChild); err != nil {
				return fmt.Errorf("copying child credit: %w", err)
			}
		}
	}

	// Copy personal credits.
	if !personalCopied {
		srcPersonal, err := s.personalRepo.GetByYear(ctx, sourceYear)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("fetching source personal credits for copy: %w", err)
		}
		if srcPersonal != nil {
			newPersonal := *srcPersonal
			newPersonal.Year = targetYear
			newPersonal.CreditStudent = 0
			newPersonal.CreditDisability = 0
			if err := s.personalRepo.Upsert(ctx, &newPersonal); err != nil {
				return fmt.Errorf("copying personal credits: %w", err)
			}
		}
	}

	// Copy deductions (empty amounts, same categories).
	if !deductionsCopied {
		srcDeductions, err := s.deductionRepo.ListByYear(ctx, sourceYear)
		if err != nil {
			return fmt.Errorf("fetching source deductions for copy: %w", err)
		}
		for _, ded := range srcDeductions {
			newDed := ded
			newDed.ID = 0
			newDed.Year = targetYear
			newDed.ClaimedAmount = 0
			newDed.MaxAmount = 0
			newDed.AllowedAmount = 0
			if err := s.deductionRepo.Create(ctx, &newDed); err != nil {
				return fmt.Errorf("copying deduction: %w", err)
			}
		}
	}

	return nil
}
