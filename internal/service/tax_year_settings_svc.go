package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// TaxYearSettingsService provides business logic for per-year tax settings and prepayments.
type TaxYearSettingsService struct {
	settingsRepo   repository.TaxYearSettingsRepo
	prepaymentRepo repository.TaxPrepaymentRepo
	audit          *AuditService
}

// NewTaxYearSettingsService creates a new TaxYearSettingsService.
func NewTaxYearSettingsService(
	settingsRepo repository.TaxYearSettingsRepo,
	prepaymentRepo repository.TaxPrepaymentRepo,
	audit *AuditService,
) *TaxYearSettingsService {
	return &TaxYearSettingsService{
		settingsRepo:   settingsRepo,
		prepaymentRepo: prepaymentRepo,
		audit:          audit,
	}
}

// GetByYear returns tax year settings for a given year within the given company,
// defaulting to zero values if not found.
func (s *TaxYearSettingsService) GetByYear(ctx context.Context, companyID int64, year int) (*domain.TaxYearSettings, error) {
	tys, err := s.settingsRepo.GetByYear(ctx, companyID, year)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return &domain.TaxYearSettings{Year: year}, nil
		}
		return nil, fmt.Errorf("fetching tax_year_settings for year %d: %w", year, err)
	}
	return tys, nil
}

// GetPrepayments returns 12 months of prepayments for a given year within the given company,
// filling missing months with zeros.
func (s *TaxYearSettingsService) GetPrepayments(ctx context.Context, companyID int64, year int) ([]domain.TaxPrepayment, error) {
	existing, err := s.prepaymentRepo.ListByYear(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("listing prepayments for year %d: %w", year, err)
	}

	byMonth := make(map[int]domain.TaxPrepayment, len(existing))
	for _, tp := range existing {
		byMonth[tp.Month] = tp
	}

	result := make([]domain.TaxPrepayment, 12)
	for m := 1; m <= 12; m++ {
		if tp, ok := byMonth[m]; ok {
			result[m-1] = tp
		} else {
			result[m-1] = domain.TaxPrepayment{Year: year, Month: m}
		}
	}
	return result, nil
}

// Save upserts both tax year settings and all 12 months of prepayments within the given company.
func (s *TaxYearSettingsService) Save(ctx context.Context, companyID int64, year int, flatRatePercent int, prepayments []domain.TaxPrepayment) error {
	// Fetch existing settings before upsert for audit logging.
	existing, _ := s.settingsRepo.GetByYear(ctx, companyID, year)

	tys := &domain.TaxYearSettings{
		Year:            year,
		FlatRatePercent: flatRatePercent,
	}
	if err := s.settingsRepo.Upsert(ctx, companyID, tys); err != nil {
		return fmt.Errorf("saving tax_year_settings: %w", err)
	}

	if err := s.prepaymentRepo.UpsertAll(ctx, companyID, year, prepayments); err != nil {
		return fmt.Errorf("saving prepayments: %w", err)
	}

	if s.audit != nil {
		action := "create"
		if existing != nil {
			action = "update"
		}
		s.audit.Log(ctx, "tax_year_settings", int64(year), action, existing, map[string]any{
			"year":              year,
			"flat_rate_percent": flatRatePercent,
		})
	}

	return nil
}

// GetPrepaymentSums returns the annual sum of prepayments for a given year within the given company.
func (s *TaxYearSettingsService) GetPrepaymentSums(ctx context.Context, companyID int64, year int) (taxTotal, socialTotal, healthTotal domain.Amount, err error) {
	return s.prepaymentRepo.SumByYear(ctx, companyID, year)
}
