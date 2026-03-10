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
	repo      repository.InvoiceRepo
	contacts  *ContactService
	sequences *SequenceService
}

// NewInvoiceService creates a new InvoiceService.
func NewInvoiceService(repo repository.InvoiceRepo, contacts *ContactService, sequences *SequenceService) *InvoiceService {
	return &InvoiceService{
		repo:      repo,
		contacts:  contacts,
		sequences: sequences,
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

	// Auto-assign sequence if none provided.
	if invoice.SequenceID == 0 && invoice.InvoiceNumber == "" && s.sequences != nil {
		prefix := "FV"
		switch invoice.Type {
		case domain.InvoiceTypeProforma:
			prefix = "ZF"
		case domain.InvoiceTypeCreditNote:
			prefix = "DN"
		}
		year := invoice.IssueDate.Year()
		seq, err := s.sequences.GetOrCreateForYear(ctx, prefix, year)
		if err != nil {
			return err
		}
		invoice.SequenceID = seq.ID
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

// SettleProforma creates a settlement regular invoice from a paid proforma invoice.
// The proforma must have status=paid. The new invoice is a draft linked to the proforma.
func (s *InvoiceService) SettleProforma(ctx context.Context, proformaID int64) (*domain.Invoice, error) {
	if proformaID == 0 {
		return nil, errors.New("proforma ID is required")
	}

	proforma, err := s.repo.GetByID(ctx, proformaID)
	if err != nil {
		return nil, err
	}
	if proforma.Type != domain.InvoiceTypeProforma {
		return nil, errors.New("invoice is not a proforma")
	}
	if proforma.Status != domain.InvoiceStatusPaid {
		return nil, errors.New("only paid proformas can be settled")
	}

	today := time.Now()
	settlement := &domain.Invoice{
		Type:             domain.InvoiceTypeRegular,
		RelatedInvoiceID: &proforma.ID,
		RelationType:     domain.RelationTypeSettlement,
		CustomerID:       proforma.CustomerID,
		IssueDate:        today,
		DueDate:          today,
		DeliveryDate:     today,
		CurrencyCode:     proforma.CurrencyCode,
		ExchangeRate:     proforma.ExchangeRate,
		PaymentMethod:    proforma.PaymentMethod,
		BankAccount:      proforma.BankAccount,
		BankCode:         proforma.BankCode,
		IBAN:             proforma.IBAN,
		SWIFT:            proforma.SWIFT,
		ConstantSymbol:   proforma.ConstantSymbol,
		Notes:            proforma.Notes,
		Status:           domain.InvoiceStatusDraft,
	}

	// Copy line items without IDs.
	for _, item := range proforma.Items {
		settlement.Items = append(settlement.Items, domain.InvoiceItem{
			Description:    item.Description,
			Quantity:       item.Quantity,
			Unit:           item.Unit,
			UnitPrice:      item.UnitPrice,
			VATRatePercent: item.VATRatePercent,
			SortOrder:      item.SortOrder,
		})
	}

	// Create assigns invoice number from "FV" sequence and calculates totals.
	if err := s.Create(ctx, settlement); err != nil {
		return nil, err
	}

	return settlement, nil
}

// CreateCreditNote creates a credit note for an existing regular invoice.
// If items is nil/empty, a full credit note is created (all items negated).
// If items are provided, a partial credit note is created with those items.
func (s *InvoiceService) CreateCreditNote(ctx context.Context, originalID int64, items []domain.InvoiceItem, reason string) (*domain.Invoice, error) {
	if originalID == 0 {
		return nil, errors.New("original invoice ID is required")
	}

	original, err := s.repo.GetByID(ctx, originalID)
	if err != nil {
		return nil, err
	}

	if original.Type != domain.InvoiceTypeRegular {
		return nil, errors.New("credit notes can only be created for regular invoices")
	}
	if original.Status != domain.InvoiceStatusSent && original.Status != domain.InvoiceStatusPaid {
		return nil, errors.New("credit notes can only be created for sent or paid invoices")
	}

	today := time.Now()
	creditNote := &domain.Invoice{
		Type:             domain.InvoiceTypeCreditNote,
		RelatedInvoiceID: &original.ID,
		RelationType:     domain.RelationTypeCreditNote,
		CustomerID:       original.CustomerID,
		IssueDate:        today,
		DueDate:          today,
		DeliveryDate:     today,
		CurrencyCode:     original.CurrencyCode,
		ExchangeRate:     original.ExchangeRate,
		PaymentMethod:    original.PaymentMethod,
		BankAccount:      original.BankAccount,
		BankCode:         original.BankCode,
		IBAN:             original.IBAN,
		SWIFT:            original.SWIFT,
		ConstantSymbol:   original.ConstantSymbol,
		Notes:            reason,
		Status:           domain.InvoiceStatusDraft,
	}

	if len(items) == 0 {
		// Full credit note: copy all items with negated unit prices.
		for _, item := range original.Items {
			creditNote.Items = append(creditNote.Items, domain.InvoiceItem{
				Description:    item.Description,
				Quantity:       item.Quantity,
				Unit:           item.Unit,
				UnitPrice:      item.UnitPrice * -1,
				VATRatePercent: item.VATRatePercent,
				SortOrder:      item.SortOrder,
			})
		}
	} else {
		// Partial credit note: use provided items with negated unit prices.
		for _, item := range items {
			creditNote.Items = append(creditNote.Items, domain.InvoiceItem{
				Description:    item.Description,
				Quantity:       item.Quantity,
				Unit:           item.Unit,
				UnitPrice:      item.UnitPrice * -1,
				VATRatePercent: item.VATRatePercent,
				SortOrder:      item.SortOrder,
			})
		}
	}

	if err := s.Create(ctx, creditNote); err != nil {
		return nil, err
	}

	return creditNote, nil
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
