package service

import (
	"context"
	"errors"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// InvoiceService provides business logic for invoice management.
type InvoiceService struct {
	repo     repository.InvoiceRepo
	contacts *ContactService
}

// NewInvoiceService creates a new InvoiceService.
func NewInvoiceService(repo repository.InvoiceRepo, contacts *ContactService) *InvoiceService {
	return &InvoiceService{
		repo:     repo,
		contacts: contacts,
	}
}

// Create validates, calculates totals, assigns a number, and persists a new invoice.
func (s *InvoiceService) Create(ctx context.Context, invoice *domain.Invoice) error {
	if invoice.CustomerID == 0 {
		return errors.New("customer is required")
	}
	if len(invoice.Items) == 0 {
		return errors.New("at least one line item is required")
	}

	// Verify customer exists.
	_, err := s.contacts.GetByID(ctx, invoice.CustomerID)
	if err != nil {
		return errors.New("customer not found")
	}

	// Set defaults.
	if invoice.Status == "" {
		invoice.Status = domain.InvoiceStatusDraft
	}
	if invoice.Type == "" {
		invoice.Type = domain.InvoiceTypeRegular
	}
	if invoice.CurrencyCode == "" {
		invoice.CurrencyCode = domain.CurrencyCZK
	}
	if invoice.IssueDate.IsZero() {
		invoice.IssueDate = time.Now()
	}
	if invoice.DeliveryDate.IsZero() {
		invoice.DeliveryDate = invoice.IssueDate
	}

	// Assign invoice number from sequence.
	if invoice.InvoiceNumber == "" && invoice.SequenceID > 0 {
		number, err := s.repo.GetNextNumber(ctx, invoice.SequenceID)
		if err != nil {
			return err
		}
		invoice.InvoiceNumber = number
	}

	// Set variable symbol to invoice number if not set.
	if invoice.VariableSymbol == "" {
		invoice.VariableSymbol = invoice.InvoiceNumber
	}

	// Calculate totals from line items.
	invoice.CalculateTotals()

	return s.repo.Create(ctx, invoice)
}

// Update validates, recalculates totals, and updates an existing invoice.
func (s *InvoiceService) Update(ctx context.Context, invoice *domain.Invoice) error {
	if invoice.ID == 0 {
		return errors.New("invoice ID is required")
	}
	if invoice.CustomerID == 0 {
		return errors.New("customer is required")
	}
	if len(invoice.Items) == 0 {
		return errors.New("at least one line item is required")
	}

	// Verify the invoice exists and is editable.
	existing, err := s.repo.GetByID(ctx, invoice.ID)
	if err != nil {
		return err
	}
	if existing.Status == domain.InvoiceStatusPaid {
		return errors.New("cannot update a paid invoice")
	}

	// Preserve existing status if not explicitly set in the update request.
	if invoice.Status == "" {
		invoice.Status = existing.Status
	}

	// Recalculate totals.
	invoice.CalculateTotals()

	return s.repo.Update(ctx, invoice)
}

// Delete removes an invoice by ID (soft delete).
func (s *InvoiceService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("invoice ID is required")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.Status == domain.InvoiceStatusPaid {
		return errors.New("cannot delete a paid invoice")
	}

	return s.repo.Delete(ctx, id)
}

// GetByID retrieves an invoice by its ID.
func (s *InvoiceService) GetByID(ctx context.Context, id int64) (*domain.Invoice, error) {
	if id == 0 {
		return nil, errors.New("invoice ID is required")
	}
	return s.repo.GetByID(ctx, id)
}

// List retrieves invoices matching the given filter.
// Returns the invoices, total count, and any error.
func (s *InvoiceService) List(ctx context.Context, filter domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}

// MarkAsSent updates the invoice status to sent and records the timestamp.
func (s *InvoiceService) MarkAsSent(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("invoice ID is required")
	}

	invoice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if invoice.Status != domain.InvoiceStatusDraft {
		return errors.New("only draft invoices can be marked as sent")
	}

	now := time.Now()
	invoice.Status = domain.InvoiceStatusSent
	invoice.SentAt = &now

	return s.repo.Update(ctx, invoice)
}

// MarkAsPaid updates the invoice status to paid and records the payment details.
func (s *InvoiceService) MarkAsPaid(ctx context.Context, id int64, amount domain.Amount, paidAt time.Time) error {
	if id == 0 {
		return errors.New("invoice ID is required")
	}

	invoice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if invoice.Status == domain.InvoiceStatusPaid {
		return errors.New("invoice is already paid")
	}
	if invoice.Status == domain.InvoiceStatusCancelled {
		return errors.New("cannot pay a cancelled invoice")
	}

	invoice.PaidAmount = amount
	invoice.PaidAt = &paidAt
	invoice.Status = domain.InvoiceStatusPaid

	return s.repo.Update(ctx, invoice)
}

// Duplicate creates a copy of an existing invoice as a new draft.
func (s *InvoiceService) Duplicate(ctx context.Context, id int64) (*domain.Invoice, error) {
	if id == 0 {
		return nil, errors.New("invoice ID is required")
	}

	original, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Create a copy with reset status and dates.
	dup := &domain.Invoice{
		SequenceID:     original.SequenceID,
		Type:           original.Type,
		CustomerID:     original.CustomerID,
		IssueDate:      time.Now(),
		DueDate:        time.Now().AddDate(0, 0, 14),
		DeliveryDate:   time.Now(),
		CurrencyCode:   original.CurrencyCode,
		ExchangeRate:   original.ExchangeRate,
		PaymentMethod:  original.PaymentMethod,
		BankAccount:    original.BankAccount,
		BankCode:       original.BankCode,
		IBAN:           original.IBAN,
		SWIFT:          original.SWIFT,
		ConstantSymbol: original.ConstantSymbol,
		Notes:          original.Notes,
		Status:         domain.InvoiceStatusDraft,
	}

	// Copy line items without IDs.
	for _, item := range original.Items {
		dup.Items = append(dup.Items, domain.InvoiceItem{
			Description:    item.Description,
			Quantity:       item.Quantity,
			Unit:           item.Unit,
			UnitPrice:      item.UnitPrice,
			VATRatePercent: item.VATRatePercent,
			SortOrder:      item.SortOrder,
		})
	}

	// Assign number and calculate totals via Create.
	if err := s.Create(ctx, dup); err != nil {
		return nil, err
	}

	return dup, nil
}
