package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// RecurringInvoiceService provides business logic for recurring invoice management.
type RecurringInvoiceService struct {
	repo     repository.RecurringInvoiceRepo
	invoices *InvoiceService
}

// NewRecurringInvoiceService creates a new RecurringInvoiceService.
func NewRecurringInvoiceService(repo repository.RecurringInvoiceRepo, invoices *InvoiceService) *RecurringInvoiceService {
	return &RecurringInvoiceService{
		repo:     repo,
		invoices: invoices,
	}
}

// Create validates and persists a new recurring invoice.
func (s *RecurringInvoiceService) Create(ctx context.Context, ri *domain.RecurringInvoice) error {
	if ri.Name == "" {
		return errors.New("name is required")
	}
	if ri.CustomerID == 0 {
		return errors.New("customer is required")
	}
	if len(ri.Items) == 0 {
		return errors.New("at least one line item is required")
	}
	if ri.NextIssueDate.IsZero() {
		return errors.New("next issue date is required")
	}

	// Set defaults.
	if ri.Frequency == "" {
		ri.Frequency = domain.FrequencyMonthly
	}
	if ri.CurrencyCode == "" {
		ri.CurrencyCode = domain.CurrencyCZK
	}

	if err := s.repo.Create(ctx, ri); err != nil {
		return fmt.Errorf("creating recurring invoice: %w", err)
	}
	return nil
}

// Update validates and updates an existing recurring invoice.
func (s *RecurringInvoiceService) Update(ctx context.Context, ri *domain.RecurringInvoice) error {
	if ri.ID == 0 {
		return errors.New("recurring invoice ID is required")
	}
	if ri.Name == "" {
		return errors.New("name is required")
	}
	if ri.CustomerID == 0 {
		return errors.New("customer is required")
	}
	if len(ri.Items) == 0 {
		return errors.New("at least one line item is required")
	}

	// Verify it exists.
	_, err := s.repo.GetByID(ctx, ri.ID)
	if err != nil {
		return fmt.Errorf("fetching recurring invoice for update: %w", err)
	}

	if err := s.repo.Update(ctx, ri); err != nil {
		return fmt.Errorf("updating recurring invoice: %w", err)
	}
	return nil
}

// Delete removes a recurring invoice by ID (soft delete).
func (s *RecurringInvoiceService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("recurring invoice ID is required")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting recurring invoice: %w", err)
	}
	return nil
}

// GetByID retrieves a recurring invoice by its ID.
func (s *RecurringInvoiceService) GetByID(ctx context.Context, id int64) (*domain.RecurringInvoice, error) {
	if id == 0 {
		return nil, errors.New("recurring invoice ID is required")
	}
	ri, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching recurring invoice: %w", err)
	}
	return ri, nil
}

// List retrieves all recurring invoices.
func (s *RecurringInvoiceService) List(ctx context.Context) ([]domain.RecurringInvoice, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing recurring invoices: %w", err)
	}
	return items, nil
}

// GenerateInvoice creates a single invoice from a recurring invoice template.
func (s *RecurringInvoiceService) GenerateInvoice(ctx context.Context, id int64) (*domain.Invoice, error) {
	ri, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching recurring invoice for generation: %w", err)
	}

	return s.createInvoiceFromTemplate(ctx, ri, ri.NextIssueDate)
}

// ProcessDue finds all due recurring invoices, creates draft invoices, advances
// the next_issue_date, and deactivates recurring invoices that are past their end_date.
// Returns the count of generated invoices.
func (s *RecurringInvoiceService) ProcessDue(ctx context.Context) (int, error) {
	today := time.Now().Truncate(24 * time.Hour)
	dueList, err := s.repo.ListDue(ctx, today)
	if err != nil {
		return 0, fmt.Errorf("listing due recurring invoices: %w", err)
	}

	count := 0
	for i := range dueList {
		ri := &dueList[i]

		// Check if past end_date - deactivate instead of generating.
		if ri.EndDate != nil && today.After(*ri.EndDate) {
			if err := s.repo.Deactivate(ctx, ri.ID); err != nil {
				return count, fmt.Errorf("deactivating expired recurring invoice %d: %w", ri.ID, err)
			}
			continue
		}

		// Generate the invoice.
		_, err := s.createInvoiceFromTemplate(ctx, ri, ri.NextIssueDate)
		if err != nil {
			return count, fmt.Errorf("generating invoice from recurring %d: %w", ri.ID, err)
		}
		count++

		// Advance next_issue_date.
		ri.NextIssueDate = ri.NextDate()

		// Check if the new next_issue_date is past the end_date - deactivate.
		if ri.EndDate != nil && ri.NextIssueDate.After(*ri.EndDate) {
			ri.IsActive = false
		}

		if err := s.repo.Update(ctx, ri); err != nil {
			return count, fmt.Errorf("updating recurring invoice %d next date: %w", ri.ID, err)
		}
	}

	return count, nil
}

// createInvoiceFromTemplate builds a domain.Invoice from a RecurringInvoice template.
func (s *RecurringInvoiceService) createInvoiceFromTemplate(ctx context.Context, ri *domain.RecurringInvoice, issueDate time.Time) (*domain.Invoice, error) {
	invoice := &domain.Invoice{
		Type:           domain.InvoiceTypeRegular,
		Status:         domain.InvoiceStatusDraft,
		CustomerID:     ri.CustomerID,
		IssueDate:      issueDate,
		DueDate:        issueDate.AddDate(0, 0, 14),
		DeliveryDate:   issueDate,
		CurrencyCode:   ri.CurrencyCode,
		ExchangeRate:   ri.ExchangeRate,
		PaymentMethod:  ri.PaymentMethod,
		BankAccount:    ri.BankAccount,
		BankCode:       ri.BankCode,
		IBAN:           ri.IBAN,
		SWIFT:          ri.SWIFT,
		ConstantSymbol: ri.ConstantSymbol,
		Notes:          ri.Notes,
	}

	for _, item := range ri.Items {
		invoice.Items = append(invoice.Items, domain.InvoiceItem{
			Description:    item.Description,
			Quantity:       item.Quantity,
			Unit:           item.Unit,
			UnitPrice:      item.UnitPrice,
			VATRatePercent: item.VATRatePercent,
			SortOrder:      item.SortOrder,
		})
	}

	// Create assigns invoice number from sequence and calculates totals.
	if err := s.invoices.Create(ctx, invoice); err != nil {
		return nil, fmt.Errorf("creating invoice from template: %w", err)
	}

	return invoice, nil
}
