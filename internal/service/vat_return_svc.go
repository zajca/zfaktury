package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zajca/zfaktury/internal/calc"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/vatxml"
)

// VATReturnService provides business logic for VAT return management.
type VATReturnService struct {
	repo         repository.VATReturnRepo
	invoiceRepo  repository.InvoiceRepo
	expenseRepo  repository.ExpenseRepo
	settingsRepo repository.SettingsRepo
	audit        *AuditService
}

// NewVATReturnService creates a new VATReturnService.
func NewVATReturnService(
	repo repository.VATReturnRepo,
	invoiceRepo repository.InvoiceRepo,
	expenseRepo repository.ExpenseRepo,
	settingsRepo repository.SettingsRepo,
	audit *AuditService,
) *VATReturnService {
	return &VATReturnService{
		repo:         repo,
		invoiceRepo:  invoiceRepo,
		expenseRepo:  expenseRepo,
		settingsRepo: settingsRepo,
		audit:        audit,
	}
}

// Create validates and persists a new VAT return.
func (s *VATReturnService) Create(ctx context.Context, vr *domain.VATReturn) error {
	if vr.Period.Year < 2000 || vr.Period.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if vr.Period.Month == 0 && vr.Period.Quarter == 0 {
		return fmt.Errorf("month or quarter is required: %w", domain.ErrInvalidInput)
	}
	if vr.Period.Month != 0 && (vr.Period.Month < 1 || vr.Period.Month > 12) {
		return fmt.Errorf("month must be 1-12: %w", domain.ErrInvalidInput)
	}
	if vr.Period.Quarter != 0 && (vr.Period.Quarter < 1 || vr.Period.Quarter > 4) {
		return fmt.Errorf("quarter must be 1-4: %w", domain.ErrInvalidInput)
	}
	if vr.FilingType == "" {
		vr.FilingType = domain.FilingTypeRegular
	}
	switch vr.FilingType {
	case domain.FilingTypeRegular, domain.FilingTypeCorrective, domain.FilingTypeSupplementary:
		// ok
	default:
		return fmt.Errorf("invalid filing_type: %w", domain.ErrInvalidInput)
	}

	// Check for existing filing in same period (for regular filings).
	if vr.FilingType == domain.FilingTypeRegular {
		existing, err := s.repo.GetByPeriod(ctx, vr.Period.Year, vr.Period.Month, vr.Period.Quarter, vr.FilingType)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("checking existing vat_return: %w", err)
		}
		if existing != nil {
			return domain.ErrFilingAlreadyExists
		}
	}

	if vr.Status == "" {
		vr.Status = domain.FilingStatusDraft
	}

	if err := s.repo.Create(ctx, vr); err != nil {
		return fmt.Errorf("creating vat_return: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "vat_return", vr.ID, "create", nil, vr)
	}
	return nil
}

// GetByID retrieves a VAT return by its ID.
func (s *VATReturnService) GetByID(ctx context.Context, id int64) (*domain.VATReturn, error) {
	if id == 0 {
		return nil, fmt.Errorf("vat_return ID is required: %w", domain.ErrInvalidInput)
	}
	vr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching vat_return: %w", err)
	}
	return vr, nil
}

// List retrieves all VAT returns for a given year.
func (s *VATReturnService) List(ctx context.Context, year int) ([]domain.VATReturn, error) {
	if year == 0 {
		year = time.Now().Year()
	}
	returns, err := s.repo.List(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing vat_returns: %w", err)
	}
	return returns, nil
}

// Delete removes a VAT return by ID. Filed returns cannot be deleted.
func (s *VATReturnService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return fmt.Errorf("vat_return ID is required: %w", domain.ErrInvalidInput)
	}

	vr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching vat_return for delete: %w", err)
	}
	if vr.Status == domain.FilingStatusFiled {
		return domain.ErrFilingAlreadyFiled
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting vat_return: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "vat_return", id, "delete", nil, nil)
	}
	return nil
}

// Recalculate recalculates the VAT return amounts from linked invoices and expenses.
func (s *VATReturnService) Recalculate(ctx context.Context, id int64) (*domain.VATReturn, error) {
	if id == 0 {
		return nil, fmt.Errorf("vat_return ID is required: %w", domain.ErrInvalidInput)
	}

	vr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching vat_return for recalculation: %w", err)
	}
	if vr.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}

	// Capture existing state for audit logging.
	existing := *vr

	// Determine date range from period.
	dateFrom, dateTo := periodDateRange(vr.Period)

	// Query invoices in the period: sent, paid, overdue; NOT credit_note type.
	invoices, _, err := s.invoiceRepo.List(ctx, domain.InvoiceFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		Limit:    10000,
		Offset:   0,
	})
	if err != nil {
		return nil, fmt.Errorf("listing invoices for vat_return recalculation: %w", err)
	}

	// Build calc inputs from filtered invoices.
	var calcInvoices []calc.VATInvoiceInput
	var invoiceIDs []int64

	for _, inv := range invoices {
		if inv.DeliveryDate.Before(dateFrom) || inv.DeliveryDate.After(dateTo) {
			continue
		}
		if inv.Status != domain.InvoiceStatusSent && inv.Status != domain.InvoiceStatusPaid && inv.Status != domain.InvoiceStatusOverdue {
			continue
		}

		invoiceIDs = append(invoiceIDs, inv.ID)

		fullInv, err := s.invoiceRepo.GetByID(ctx, inv.ID)
		if err != nil {
			return nil, fmt.Errorf("fetching invoice %d items for vat_return: %w", inv.ID, err)
		}

		if fullInv.Type == domain.InvoiceTypeProforma {
			continue
		}

		ci := calc.VATInvoiceInput{Type: fullInv.Type}
		for _, item := range fullInv.Items {
			ci.Items = append(ci.Items, calc.VATItemInput{
				Base:           domain.Amount(int64(item.Quantity) * int64(item.UnitPrice) / 100),
				VATAmount:      item.VATAmount,
				VATRatePercent: item.VATRatePercent,
			})
		}
		calcInvoices = append(calcInvoices, ci)
	}

	// Query expenses in the period: tax deductible only.
	expenses, _, err := s.expenseRepo.List(ctx, domain.ExpenseFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		Limit:    10000,
		Offset:   0,
	})
	if err != nil {
		return nil, fmt.Errorf("listing expenses for vat_return recalculation: %w", err)
	}

	var calcExpenses []calc.VATExpenseInput
	var expenseIDs []int64

	for _, exp := range expenses {
		if !exp.IsTaxDeductible {
			continue
		}
		if exp.IssueDate.Before(dateFrom) || exp.IssueDate.After(dateTo) {
			continue
		}

		expenseIDs = append(expenseIDs, exp.ID)
		calcExpenses = append(calcExpenses, calc.VATExpenseInput{
			Amount:          exp.Amount,
			VATAmount:       exp.VATAmount,
			VATRatePercent:  exp.VATRatePercent,
			BusinessPercent: exp.BusinessPercent,
		})
	}

	// Pure calculation.
	result := calc.CalculateVATReturn(calcInvoices, calcExpenses)

	vr.OutputVATBase21 = result.OutputVATBase21
	vr.OutputVATAmount21 = result.OutputVATAmount21
	vr.OutputVATBase12 = result.OutputVATBase12
	vr.OutputVATAmount12 = result.OutputVATAmount12
	vr.InputVATBase21 = result.InputVATBase21
	vr.InputVATAmount21 = result.InputVATAmount21
	vr.InputVATBase12 = result.InputVATBase12
	vr.InputVATAmount12 = result.InputVATAmount12
	vr.TotalOutputVAT = result.TotalOutputVAT
	vr.TotalInputVAT = result.TotalInputVAT
	vr.NetVAT = result.NetVAT

	// Persist updated values.
	if err := s.repo.Update(ctx, vr); err != nil {
		return nil, fmt.Errorf("updating vat_return after recalculation: %w", err)
	}

	// Link invoices and expenses.
	if err := s.repo.LinkInvoices(ctx, vr.ID, invoiceIDs); err != nil {
		return nil, fmt.Errorf("linking invoices to vat_return: %w", err)
	}
	if err := s.repo.LinkExpenses(ctx, vr.ID, expenseIDs); err != nil {
		return nil, fmt.Errorf("linking expenses to vat_return: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "vat_return", id, "update", &existing, vr)
	}
	return vr, nil
}

// GenerateXML generates the EPO XML for a VAT return and stores it.
func (s *VATReturnService) GenerateXML(ctx context.Context, id int64) (*domain.VATReturn, error) {
	if id == 0 {
		return nil, fmt.Errorf("vat_return ID is required: %w", domain.ErrInvalidInput)
	}

	vr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching vat_return for XML generation: %w", err)
	}

	info, err := s.buildTaxpayerInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("building taxpayer info for XML generation: %w", err)
	}

	gen := &vatxml.VATReturnGenerator{}
	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		return nil, fmt.Errorf("generating XML for vat_return: %w", err)
	}

	vr.XMLData = xmlData
	if err := s.repo.Update(ctx, vr); err != nil {
		return nil, fmt.Errorf("storing XML data for vat_return: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "vat_return", id, "generate_xml", nil, nil)
	}
	return vr, nil
}

// buildTaxpayerInfo fetches all required settings and builds a TaxpayerInfo.
func (s *VATReturnService) buildTaxpayerInfo(ctx context.Context) (vatxml.TaxpayerInfo, error) {
	getSetting := func(key string) string {
		val, err := s.settingsRepo.Get(ctx, key)
		if err != nil {
			return ""
		}
		return val
	}

	dic := getSetting("dic")
	if dic == "" {
		return vatxml.TaxpayerInfo{}, fmt.Errorf("DIC is required for XML generation, configure it in settings: %w", domain.ErrMissingSetting)
	}

	// Strip CZ prefix for DPHDP3 format.
	dicNum := strings.TrimPrefix(dic, "CZ")

	return vatxml.TaxpayerInfo{
		DIC:       dicNum,
		FirstName: getSetting("first_name"),
		LastName:  getSetting("last_name"),
		Street:    getSetting("street"),
		HouseNum:  getSetting("house_number"),
		ZIP:       getSetting("zip"),
		City:      getSetting("city"),
		Phone:     getSetting("phone"),
		Email:     getSetting("email"),
		UFOCode:   getSetting("c_ufo"),
		PracUFO:   getSetting("c_pracufo"),
		OKEC:      getSetting("c_okec"),
	}, nil
}

// GetXMLData retrieves the stored XML data for a VAT return.
func (s *VATReturnService) GetXMLData(ctx context.Context, id int64) ([]byte, error) {
	if id == 0 {
		return nil, fmt.Errorf("vat_return ID is required: %w", domain.ErrInvalidInput)
	}
	vr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching vat_return for XML data: %w", err)
	}
	return vr.XMLData, nil
}

// MarkFiled marks a VAT return as filed and records the timestamp.
func (s *VATReturnService) MarkFiled(ctx context.Context, id int64) (*domain.VATReturn, error) {
	if id == 0 {
		return nil, fmt.Errorf("vat_return ID is required: %w", domain.ErrInvalidInput)
	}

	vr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching vat_return for marking as filed: %w", err)
	}
	if vr.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}

	now := time.Now()
	vr.Status = domain.FilingStatusFiled
	vr.FiledAt = &now

	if err := s.repo.Update(ctx, vr); err != nil {
		return nil, fmt.Errorf("marking vat_return as filed: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "vat_return", id, "update", nil, vr)
	}
	return vr, nil
}

// periodDateRange returns the start and end dates for a tax period.
func periodDateRange(p domain.TaxPeriod) (time.Time, time.Time) {
	if p.Month > 0 {
		from := time.Date(p.Year, time.Month(p.Month), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
	// Quarter.
	startMonth := (p.Quarter-1)*3 + 1
	from := time.Date(p.Year, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 3, -1)
	return from, to
}
