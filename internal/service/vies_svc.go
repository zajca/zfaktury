package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/vatxml"
)

// VIESSummaryService provides business logic for VIES summary management.
type VIESSummaryService struct {
	repo     repository.VIESSummaryRepo
	invoices repository.InvoiceRepo
	contacts repository.ContactRepo
	audit    *AuditService
}

// NewVIESSummaryService creates a new VIESSummaryService.
func NewVIESSummaryService(
	repo repository.VIESSummaryRepo,
	invoices repository.InvoiceRepo,
	contacts repository.ContactRepo,
	audit *AuditService,
) *VIESSummaryService {
	return &VIESSummaryService{
		repo:     repo,
		invoices: invoices,
		contacts: contacts,
		audit:    audit,
	}
}

// Create validates and persists a new VIES summary.
func (s *VIESSummaryService) Create(ctx context.Context, vs *domain.VIESSummary) error {
	if vs.Period.Year < 2000 || vs.Period.Year > 2100 {
		return fmt.Errorf("creating VIES summary: %w: year out of valid range", domain.ErrInvalidInput)
	}
	if vs.Period.Quarter < 1 || vs.Period.Quarter > 4 {
		return fmt.Errorf("creating VIES summary: %w: quarter must be 1-4", domain.ErrInvalidInput)
	}
	if vs.FilingType == "" {
		vs.FilingType = domain.FilingTypeRegular
	}
	switch vs.FilingType {
	case domain.FilingTypeRegular, domain.FilingTypeCorrective, domain.FilingTypeSupplementary:
		// ok
	default:
		return fmt.Errorf("creating VIES summary: %w: invalid filing_type", domain.ErrInvalidInput)
	}

	// Check for duplicate period.
	existing, err := s.repo.GetByPeriod(ctx, vs.Period.Year, vs.Period.Quarter, vs.FilingType)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("checking existing VIES summary: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("creating VIES summary: %w: summary already exists for this period", domain.ErrDuplicateNumber)
	}

	vs.Status = domain.FilingStatusDraft
	if err := s.repo.Create(ctx, vs); err != nil {
		return fmt.Errorf("creating VIES summary: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "vies_summary", vs.ID, "create", nil, vs)
	}
	return nil
}

// GetByID retrieves a VIES summary by ID.
func (s *VIESSummaryService) GetByID(ctx context.Context, id int64) (*domain.VIESSummary, error) {
	vs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching VIES summary: %w", err)
	}
	return vs, nil
}

// GetLines retrieves lines for a VIES summary.
func (s *VIESSummaryService) GetLines(ctx context.Context, viesSummaryID int64) ([]domain.VIESSummaryLine, error) {
	lines, err := s.repo.GetLines(ctx, viesSummaryID)
	if err != nil {
		return nil, fmt.Errorf("fetching VIES summary lines: %w", err)
	}
	return lines, nil
}

// List retrieves all VIES summaries for a given year.
func (s *VIESSummaryService) List(ctx context.Context, year int) ([]domain.VIESSummary, error) {
	summaries, err := s.repo.List(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing VIES summaries: %w", err)
	}
	return summaries, nil
}

// Delete removes a VIES summary and its lines.
func (s *VIESSummaryService) Delete(ctx context.Context, id int64) error {
	vs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching VIES summary for deletion: %w", err)
	}
	if vs.Status == domain.FilingStatusFiled {
		return fmt.Errorf("deleting VIES summary: %w: cannot delete a filed summary", domain.ErrInvalidInput)
	}

	if err := s.repo.DeleteLines(ctx, id); err != nil {
		return fmt.Errorf("deleting VIES summary lines: %w", err)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting VIES summary: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "vies_summary", id, "delete", nil, nil)
	}
	return nil
}

// quarterDateRange returns the start and end dates for a given year and quarter.
func quarterDateRange(year, quarter int) (time.Time, time.Time) {
	startMonth := time.Month((quarter-1)*3 + 1)
	start := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 3, 0).Add(-time.Nanosecond)
	return start, end
}

// Recalculate recomputes lines for a VIES summary from invoice data.
// It finds all sent/paid/overdue invoices in the quarter, identifies EU partners,
// groups by partner DIC, and sums base amounts (SubtotalAmount, no VAT for intra-EU).
// Credit notes reduce amounts.
func (s *VIESSummaryService) Recalculate(ctx context.Context, id int64) error {
	vs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching VIES summary for recalculation: %w", err)
	}
	if vs.Status == domain.FilingStatusFiled {
		return fmt.Errorf("recalculating VIES summary: %w: cannot recalculate a filed summary", domain.ErrInvalidInput)
	}

	startDate, endDate := quarterDateRange(vs.Period.Year, vs.Period.Quarter)

	// Fetch invoices in the quarter period by delivery_date.
	// We need to fetch all invoices and filter by delivery_date and status.
	// Use a broad date filter on issue_date and then filter by delivery_date in code.
	filter := domain.InvoiceFilter{
		DateFrom: &startDate,
		DateTo:   &endDate,
		Limit:    10000,
		Offset:   0,
	}

	invoices, _, err := s.invoices.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("listing invoices for VIES recalculation: %w", err)
	}

	// Group amounts by partner DIC.
	type partnerData struct {
		dic         string
		countryCode string
		amount      domain.Amount
	}
	partnerMap := make(map[string]*partnerData)

	for i := range invoices {
		inv := &invoices[i]

		// Only consider sent, paid, or overdue invoices.
		if inv.Status != domain.InvoiceStatusSent &&
			inv.Status != domain.InvoiceStatusPaid &&
			inv.Status != domain.InvoiceStatusOverdue {
			continue
		}

		// Filter by delivery_date within the quarter.
		if inv.DeliveryDate.Before(startDate) || inv.DeliveryDate.After(endDate) {
			continue
		}

		// Load customer to check if EU partner.
		customer, err := s.contacts.GetByID(ctx, inv.CustomerID)
		if err != nil {
			return fmt.Errorf("fetching customer %d: %w", inv.CustomerID, err)
		}

		if !customer.IsEUPartner() {
			continue
		}

		dic := customer.DIC
		pd, ok := partnerMap[dic]
		if !ok {
			pd = &partnerData{
				dic:         dic,
				countryCode: customer.DICCountryCode(),
			}
			partnerMap[dic] = pd
		}

		// Use SubtotalAmount (base, no VAT) for intra-EU services.
		// Credit notes have negative SubtotalAmount, so they naturally reduce the total.
		if inv.Type == domain.InvoiceTypeCreditNote {
			pd.amount = pd.amount.Sub(inv.SubtotalAmount)
		} else {
			pd.amount = pd.amount.Add(inv.SubtotalAmount)
		}
	}

	// Build new lines.
	var lines []domain.VIESSummaryLine
	for _, pd := range partnerMap {
		if pd.amount.IsZero() {
			continue
		}
		lines = append(lines, domain.VIESSummaryLine{
			VIESSummaryID: id,
			PartnerDIC:    pd.dic,
			CountryCode:   pd.countryCode,
			TotalAmount:   pd.amount,
			ServiceCode:   "3",
		})
	}

	// Replace old lines with new ones.
	if err := s.repo.DeleteLines(ctx, id); err != nil {
		return fmt.Errorf("deleting old VIES summary lines: %w", err)
	}
	if len(lines) > 0 {
		if err := s.repo.CreateLines(ctx, lines); err != nil {
			return fmt.Errorf("creating new VIES summary lines: %w", err)
		}
	}

	if s.audit != nil {
		s.audit.Log(ctx, "vies_summary", id, "recalculate", nil, nil)
	}
	return nil
}

// GenerateXML generates the EPO XML for a VIES summary and stores it.
func (s *VIESSummaryService) GenerateXML(ctx context.Context, id int64, dic string) error {
	vs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching VIES summary for XML generation: %w", err)
	}

	lines, err := s.repo.GetLines(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching VIES summary lines for XML generation: %w", err)
	}

	gen := vatxml.VIESSummaryGenerator{}
	xmlData, err := gen.Generate(vs, lines, dic)
	if err != nil {
		return fmt.Errorf("generating VIES XML: %w", err)
	}

	vs.XMLData = xmlData
	if err := s.repo.Update(ctx, vs); err != nil {
		return fmt.Errorf("storing VIES XML data: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "vies_summary", id, "generate_xml", nil, nil)
	}
	return nil
}

// MarkFiled marks a VIES summary as filed.
func (s *VIESSummaryService) MarkFiled(ctx context.Context, id int64) error {
	vs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching VIES summary for filing: %w", err)
	}

	if vs.Status == domain.FilingStatusFiled {
		return fmt.Errorf("marking VIES summary as filed: %w: already filed", domain.ErrInvalidInput)
	}

	now := time.Now()
	vs.Status = domain.FilingStatusFiled
	vs.FiledAt = &now

	if err := s.repo.Update(ctx, vs); err != nil {
		return fmt.Errorf("updating VIES summary status: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "vies_summary", id, "mark_filed", nil, nil)
	}
	return nil
}
