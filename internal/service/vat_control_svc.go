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

// VATControlStatementService provides business logic for VAT control statements.
type VATControlStatementService struct {
	repo     repository.VATControlStatementRepo
	invoices repository.InvoiceRepo
	expenses repository.ExpenseRepo
	contacts repository.ContactRepo
	xmlGen   *vatxml.ControlStatementGenerator
	audit    *AuditService
}

// NewVATControlStatementService creates a new VATControlStatementService.
func NewVATControlStatementService(
	repo repository.VATControlStatementRepo,
	invoices repository.InvoiceRepo,
	expenses repository.ExpenseRepo,
	contacts repository.ContactRepo,
	audit *AuditService,
) *VATControlStatementService {
	return &VATControlStatementService{
		repo:     repo,
		invoices: invoices,
		expenses: expenses,
		contacts: contacts,
		xmlGen:   vatxml.NewControlStatementGenerator(),
		audit:    audit,
	}
}

// Create validates and persists a new control statement.
func (s *VATControlStatementService) Create(ctx context.Context, cs *domain.VATControlStatement) error {
	if cs.Period.Year < 2000 || cs.Period.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if cs.Period.Month < 1 || cs.Period.Month > 12 {
		return fmt.Errorf("month must be 1-12: %w", domain.ErrInvalidInput)
	}
	if cs.FilingType == "" {
		cs.FilingType = domain.FilingTypeRegular
	}
	switch cs.FilingType {
	case domain.FilingTypeRegular, domain.FilingTypeCorrective, domain.FilingTypeSupplementary:
		// ok
	default:
		return fmt.Errorf("invalid filing_type: %w", domain.ErrInvalidInput)
	}

	// Check for existing statement in same period with same filing type.
	existing, err := s.repo.GetByPeriod(ctx, cs.Period.Year, cs.Period.Month, cs.FilingType)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("checking existing control statement: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("control statement already exists for %d/%d: %w", cs.Period.Year, cs.Period.Month, domain.ErrDuplicateNumber)
	}

	cs.Status = domain.FilingStatusDraft

	if err := s.repo.Create(ctx, cs); err != nil {
		return fmt.Errorf("creating control statement: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "vat_control_statement", cs.ID, "create", nil, cs)
	}
	return nil
}

// GetByID retrieves a control statement by ID.
func (s *VATControlStatementService) GetByID(ctx context.Context, id int64) (*domain.VATControlStatement, error) {
	cs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching control statement: %w", err)
	}
	return cs, nil
}

// List retrieves all control statements for a given year.
func (s *VATControlStatementService) List(ctx context.Context, year int) ([]domain.VATControlStatement, error) {
	statements, err := s.repo.List(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing control statements: %w", err)
	}
	return statements, nil
}

// Delete removes a control statement. Cannot delete filed statements.
func (s *VATControlStatementService) Delete(ctx context.Context, id int64) error {
	cs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching control statement for delete: %w", err)
	}
	if cs.Status == domain.FilingStatusFiled {
		return fmt.Errorf("cannot delete a filed control statement: %w", domain.ErrInvalidInput)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting control statement: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "vat_control_statement", id, "delete", nil, nil)
	}
	return nil
}

// GetLines retrieves lines for a control statement.
func (s *VATControlStatementService) GetLines(ctx context.Context, controlStatementID int64) ([]domain.VATControlStatementLine, error) {
	lines, err := s.repo.GetLines(ctx, controlStatementID)
	if err != nil {
		return nil, fmt.Errorf("fetching control statement lines: %w", err)
	}
	return lines, nil
}

// Recalculate rebuilds all lines for a control statement from invoices and expenses.
func (s *VATControlStatementService) Recalculate(ctx context.Context, id int64) error {
	cs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching control statement for recalculate: %w", err)
	}
	if cs.Status == domain.FilingStatusFiled {
		return fmt.Errorf("cannot recalculate a filed control statement: %w", domain.ErrInvalidInput)
	}

	// Date range for this month.
	monthStart := time.Date(cs.Period.Year, time.Month(cs.Period.Month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	var lines []domain.VATControlStatementLine

	// Process output invoices (by delivery_date).
	invoiceLines, err := s.buildInvoiceLines(ctx, id, monthStart, monthEnd)
	if err != nil {
		return fmt.Errorf("building invoice lines: %w", err)
	}
	lines = append(lines, invoiceLines...)

	// Process input expenses (by issue_date).
	expenseLines, err := s.buildExpenseLines(ctx, id, monthStart, monthEnd)
	if err != nil {
		return fmt.Errorf("building expense lines: %w", err)
	}
	lines = append(lines, expenseLines...)

	// Delete old lines and create new ones.
	if err := s.repo.DeleteLines(ctx, id); err != nil {
		return fmt.Errorf("deleting old control statement lines: %w", err)
	}
	if len(lines) > 0 {
		if err := s.repo.CreateLines(ctx, lines); err != nil {
			return fmt.Errorf("creating control statement lines: %w", err)
		}
	}

	// Update status to ready.
	cs.Status = domain.FilingStatusReady
	if err := s.repo.Update(ctx, cs); err != nil {
		return fmt.Errorf("updating control statement status: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "vat_control_statement", id, "recalculate", nil, nil)
	}
	return nil
}

// buildInvoiceLines processes invoices for the given period and builds control statement lines.
func (s *VATControlStatementService) buildInvoiceLines(ctx context.Context, csID int64, monthStart, monthEnd time.Time) ([]domain.VATControlStatementLine, error) {
	// Get invoices with delivery_date in the month, status sent/paid/overdue.
	filter := domain.InvoiceFilter{
		DateFrom: &monthStart,
		DateTo:   &monthEnd,
		Limit:    10000,
		Offset:   0,
	}

	allInvoices, _, err := s.invoices.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("listing invoices: %w", err)
	}

	// Filter by delivery_date in month and valid status.
	var invoices []domain.Invoice
	for _, inv := range allInvoices {
		if inv.Status != domain.InvoiceStatusSent &&
			inv.Status != domain.InvoiceStatusPaid &&
			inv.Status != domain.InvoiceStatusOverdue {
			continue
		}
		if inv.DeliveryDate.IsZero() {
			continue
		}
		if !inv.DeliveryDate.Before(monthStart) && !inv.DeliveryDate.After(monthEnd) {
			invoices = append(invoices, inv)
		}
	}

	var lines []domain.VATControlStatementLine

	// A5 aggregation: base+vat by rate for small invoices.
	a5Agg := make(map[int]struct{ base, vat domain.Amount })

	for _, inv := range invoices {
		// Load full invoice with items.
		fullInv, err := s.invoices.GetByID(ctx, inv.ID)
		if err != nil {
			return nil, fmt.Errorf("fetching invoice %d: %w", inv.ID, err)
		}

		// Load customer contact to check DIC.
		contact, err := s.contacts.GetByID(ctx, fullInv.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("fetching contact for invoice %d: %w", inv.ID, err)
		}

		// Only include CZ DIC partners.
		if !contact.HasCZDIC() {
			continue
		}

		// Calculate total amount (absolute value for threshold check).
		totalAbs := fullInv.TotalAmount
		if totalAbs.IsNegative() {
			totalAbs = domain.Amount(-int64(totalAbs))
		}

		invID := fullInv.ID
		dppd := fullInv.DeliveryDate.Format("2006-01-02")

		if totalAbs > domain.ControlStatementThreshold {
			// A4: individual lines per invoice item rate.
			rateAgg := make(map[int]struct{ base, vat domain.Amount })
			for _, item := range fullInv.Items {
				if item.VATRatePercent == 0 {
					continue
				}
				agg := rateAgg[item.VATRatePercent]
				// Item subtotal = quantity * unit_price / 100.
				itemBase := domain.Amount(int64(item.Quantity) * int64(item.UnitPrice) / 100)
				agg.base = agg.base.Add(itemBase)
				agg.vat = agg.vat.Add(item.VATAmount)
				rateAgg[item.VATRatePercent] = agg
			}
			for rate, agg := range rateAgg {
				lines = append(lines, domain.VATControlStatementLine{
					ControlStatementID: csID,
					Section:            "A4",
					PartnerDIC:         contact.DIC,
					DocumentNumber:     fullInv.InvoiceNumber,
					DPPD:               dppd,
					Base:               agg.base,
					VAT:                agg.vat,
					VATRatePercent:     rate,
					InvoiceID:          &invID,
				})
			}
		} else {
			// A5: aggregate by rate.
			for _, item := range fullInv.Items {
				if item.VATRatePercent == 0 {
					continue
				}
				agg := a5Agg[item.VATRatePercent]
				itemBase := domain.Amount(int64(item.Quantity) * int64(item.UnitPrice) / 100)
				agg.base = agg.base.Add(itemBase)
				agg.vat = agg.vat.Add(item.VATAmount)
				a5Agg[item.VATRatePercent] = agg
			}
		}
	}

	// Create A5 aggregated lines.
	for rate, agg := range a5Agg {
		lines = append(lines, domain.VATControlStatementLine{
			ControlStatementID: csID,
			Section:            "A5",
			Base:               agg.base,
			VAT:                agg.vat,
			VATRatePercent:     rate,
		})
	}

	return lines, nil
}

// buildExpenseLines processes expenses for the given period and builds control statement lines.
func (s *VATControlStatementService) buildExpenseLines(ctx context.Context, csID int64, monthStart, monthEnd time.Time) ([]domain.VATControlStatementLine, error) {
	filter := domain.ExpenseFilter{
		DateFrom: &monthStart,
		DateTo:   &monthEnd,
		Limit:    10000,
		Offset:   0,
	}

	allExpenses, _, err := s.expenses.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("listing expenses: %w", err)
	}

	// Filter to tax deductible expenses.
	var expenses []domain.Expense
	for _, exp := range allExpenses {
		if exp.IsTaxDeductible {
			expenses = append(expenses, exp)
		}
	}

	var lines []domain.VATControlStatementLine

	// B3 aggregation: base+vat by rate for small expenses.
	b3Agg := make(map[int]struct{ base, vat domain.Amount })

	for _, exp := range expenses {
		if exp.VATRatePercent == 0 {
			continue
		}

		// Check vendor DIC if vendor exists.
		var vendorDIC string
		if exp.VendorID != nil {
			contact, err := s.contacts.GetByID(ctx, *exp.VendorID)
			if err != nil {
				return nil, fmt.Errorf("fetching vendor for expense %d: %w", exp.ID, err)
			}
			if !contact.HasCZDIC() {
				continue
			}
			vendorDIC = contact.DIC
		} else {
			// No vendor, cannot include in control statement.
			continue
		}

		// Expense base = amount - vat_amount.
		base := exp.Amount.Sub(exp.VATAmount)
		totalAbs := exp.Amount
		if totalAbs.IsNegative() {
			totalAbs = domain.Amount(-int64(totalAbs))
		}

		expID := exp.ID
		dppd := exp.IssueDate.Format("2006-01-02")

		if totalAbs > domain.ControlStatementThreshold {
			// B2: individual line.
			lines = append(lines, domain.VATControlStatementLine{
				ControlStatementID: csID,
				Section:            "B2",
				PartnerDIC:         vendorDIC,
				DocumentNumber:     exp.ExpenseNumber,
				DPPD:               dppd,
				Base:               base,
				VAT:                exp.VATAmount,
				VATRatePercent:     exp.VATRatePercent,
				ExpenseID:          &expID,
			})
		} else {
			// B3: aggregate by rate.
			agg := b3Agg[exp.VATRatePercent]
			agg.base = agg.base.Add(base)
			agg.vat = agg.vat.Add(exp.VATAmount)
			b3Agg[exp.VATRatePercent] = agg
		}
	}

	// Create B3 aggregated lines.
	for rate, agg := range b3Agg {
		lines = append(lines, domain.VATControlStatementLine{
			ControlStatementID: csID,
			Section:            "B3",
			Base:               agg.base,
			VAT:                agg.vat,
			VATRatePercent:     rate,
		})
	}

	return lines, nil
}

// GenerateXML generates and stores the XML for a control statement.
// The dic parameter is the taxpayer's DIC (e.g. "CZ12345678").
func (s *VATControlStatementService) GenerateXML(ctx context.Context, id int64, dic string) ([]byte, error) {
	cs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching control statement for XML generation: %w", err)
	}

	lines, err := s.repo.GetLines(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching lines for XML generation: %w", err)
	}

	xmlData, err := s.xmlGen.Generate(cs, lines, dic)
	if err != nil {
		return nil, fmt.Errorf("generating control statement XML: %w", err)
	}

	cs.XMLData = xmlData
	if err := s.repo.Update(ctx, cs); err != nil {
		return nil, fmt.Errorf("storing generated XML: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "vat_control_statement", id, "generate_xml", nil, nil)
	}
	return xmlData, nil
}

// MarkFiled marks a control statement as filed.
func (s *VATControlStatementService) MarkFiled(ctx context.Context, id int64) error {
	cs, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching control statement for mark filed: %w", err)
	}

	now := time.Now()
	cs.Status = domain.FilingStatusFiled
	cs.FiledAt = &now

	if err := s.repo.Update(ctx, cs); err != nil {
		return fmt.Errorf("marking control statement as filed: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "vat_control_statement", id, "mark_filed", nil, nil)
	}
	return nil
}
