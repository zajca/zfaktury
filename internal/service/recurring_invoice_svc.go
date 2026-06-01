package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// invoiceEmailer is the subset of InvoiceEmailService the recurring auto-send
// path needs. It is an interface so tests can substitute a fake.
type invoiceEmailer interface {
	Defaults(ctx context.Context, companyID int64, invoiceNumber string) (EmailDefaults, error)
	SendByID(ctx context.Context, companyID, invoiceID int64, opts EmailOptions) error
}

// RecurringInvoiceService provides business logic for recurring invoice management.
type RecurringInvoiceService struct {
	repo     repository.RecurringInvoiceRepo
	invoices *InvoiceService
	emailer  invoiceEmailer
	audit    *AuditService
}

// NewRecurringInvoiceService creates a new RecurringInvoiceService. The emailer
// may be nil (e.g. in tests); when nil, auto-send is skipped.
func NewRecurringInvoiceService(repo repository.RecurringInvoiceRepo, invoices *InvoiceService, emailer invoiceEmailer, audit *AuditService) *RecurringInvoiceService {
	return &RecurringInvoiceService{
		repo:     repo,
		invoices: invoices,
		emailer:  emailer,
		audit:    audit,
	}
}

// autoSendTimeout bounds a single auto-send email so a hung SMTP server can't
// stall the rest of a scheduler run.
const autoSendTimeout = 30 * time.Second

// validateAutoSend ensures a configured recipient override (when present) is a
// valid email address. A blank override is allowed -- the customer's contact
// email is used at send time.
func validateAutoSend(ri *domain.RecurringInvoice) error {
	if ri.AutoSend && ri.AutoSendRecipient != "" {
		if _, err := mail.ParseAddress(ri.AutoSendRecipient); err != nil {
			return fmt.Errorf("invalid auto-send recipient email %q: %w", ri.AutoSendRecipient, domain.ErrInvalidInput)
		}
	}
	return nil
}

// Create validates and persists a new recurring invoice under the given company.
func (s *RecurringInvoiceService) Create(ctx context.Context, companyID int64, ri *domain.RecurringInvoice) error {
	if ri.Name == "" {
		return fmt.Errorf("name is required: %w", domain.ErrInvalidInput)
	}
	if ri.CustomerID == 0 {
		return fmt.Errorf("customer is required: %w", domain.ErrInvalidInput)
	}
	if len(ri.Items) == 0 {
		return fmt.Errorf("at least one line item is required: %w", domain.ErrNoItems)
	}
	if ri.NextIssueDate.IsZero() {
		return fmt.Errorf("next issue date is required: %w", domain.ErrInvalidInput)
	}
	if err := validateAutoSend(ri); err != nil {
		return err
	}

	// Set defaults.
	if ri.Frequency == "" {
		ri.Frequency = domain.FrequencyMonthly
	}
	if ri.CurrencyCode == "" {
		ri.CurrencyCode = domain.CurrencyCZK
	}

	if err := s.repo.Create(ctx, companyID, ri); err != nil {
		return fmt.Errorf("creating recurring invoice: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "recurring_invoice", ri.ID, "create", nil, ri)
	}
	return nil
}

// Update validates and updates an existing recurring invoice within the given company.
func (s *RecurringInvoiceService) Update(ctx context.Context, companyID int64, ri *domain.RecurringInvoice) error {
	if ri.ID == 0 {
		return fmt.Errorf("recurring invoice ID is required: %w", domain.ErrInvalidInput)
	}
	if ri.Name == "" {
		return fmt.Errorf("name is required: %w", domain.ErrInvalidInput)
	}
	if ri.CustomerID == 0 {
		return fmt.Errorf("customer is required: %w", domain.ErrInvalidInput)
	}
	if len(ri.Items) == 0 {
		return fmt.Errorf("at least one line item is required: %w", domain.ErrNoItems)
	}
	if err := validateAutoSend(ri); err != nil {
		return err
	}

	// Fetch existing for audit trail.
	existing, err := s.repo.GetByID(ctx, companyID, ri.ID)
	if err != nil {
		return fmt.Errorf("fetching recurring invoice for update: %w", err)
	}

	if err := s.repo.Update(ctx, companyID, ri); err != nil {
		return fmt.Errorf("updating recurring invoice: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "recurring_invoice", ri.ID, "update", existing, ri)
	}
	return nil
}

// Delete removes a recurring invoice by ID (soft delete) within the given company.
func (s *RecurringInvoiceService) Delete(ctx context.Context, companyID, id int64) error {
	if id == 0 {
		return fmt.Errorf("recurring invoice ID is required: %w", domain.ErrInvalidInput)
	}
	if err := s.repo.Delete(ctx, companyID, id); err != nil {
		return fmt.Errorf("deleting recurring invoice: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "recurring_invoice", id, "delete", nil, nil)
	}
	return nil
}

// GetByID retrieves a recurring invoice by its ID within the given company.
func (s *RecurringInvoiceService) GetByID(ctx context.Context, companyID, id int64) (*domain.RecurringInvoice, error) {
	if id == 0 {
		return nil, fmt.Errorf("recurring invoice ID is required: %w", domain.ErrInvalidInput)
	}
	ri, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("fetching recurring invoice: %w", err)
	}
	return ri, nil
}

// List retrieves all recurring invoices within the given company.
func (s *RecurringInvoiceService) List(ctx context.Context, companyID int64) ([]domain.RecurringInvoice, error) {
	items, err := s.repo.List(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("listing recurring invoices: %w", err)
	}
	return items, nil
}

// GenerateInvoice creates a single invoice from a recurring invoice template within the given company.
func (s *RecurringInvoiceService) GenerateInvoice(ctx context.Context, companyID, id int64) (*domain.Invoice, error) {
	ri, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("fetching recurring invoice for generation: %w", err)
	}

	// Manual single-generate never auto-sends: the user gets a draft to review.
	return s.createInvoiceFromTemplate(ctx, companyID, ri, ri.NextIssueDate, false)
}

// ProcessDue finds all due recurring invoices, creates draft invoices, advances
// the next_issue_date, and deactivates recurring invoices that are past their end_date,
// scoped to the given company. Returns the count of generated invoices.
//
// When autoSend is true (the daily scheduler), invoices whose template has
// AutoSend enabled are emailed after generation; auto-send failures are logged
// but never abort processing (the draft remains for manual sending). Callers on
// the manual "process due" button pass false.
func (s *RecurringInvoiceService) ProcessDue(ctx context.Context, companyID int64, autoSend bool) (int, error) {
	today := time.Now().Truncate(24 * time.Hour)
	dueList, err := s.repo.ListDue(ctx, companyID, today)
	if err != nil {
		return 0, fmt.Errorf("listing due recurring invoices: %w", err)
	}

	count := 0
	for i := range dueList {
		ri := &dueList[i]

		// Check if past end_date - deactivate instead of generating.
		if ri.EndDate != nil && today.After(*ri.EndDate) {
			if err := s.repo.Deactivate(ctx, companyID, ri.ID); err != nil {
				return count, fmt.Errorf("deactivating expired recurring invoice %d: %w", ri.ID, err)
			}
			if s.audit != nil {
				s.audit.Log(ctx, "recurring_invoice", ri.ID, "deactivate", nil, nil)
			}
			continue
		}

		// Generate the invoice (and auto-send it when requested).
		_, err := s.createInvoiceFromTemplate(ctx, companyID, ri, ri.NextIssueDate, autoSend)
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

		if err := s.repo.Update(ctx, companyID, ri); err != nil {
			return count, fmt.Errorf("updating recurring invoice %d next date: %w", ri.ID, err)
		}
	}

	return count, nil
}

// createInvoiceFromTemplate builds a domain.Invoice from a RecurringInvoice template
// and persists it within the given company. When autoSend is true and the
// template opts into auto-send, the generated invoice is emailed on a best-effort
// basis (failures are logged, not returned, so the draft is preserved).
func (s *RecurringInvoiceService) createInvoiceFromTemplate(ctx context.Context, companyID int64, ri *domain.RecurringInvoice, issueDate time.Time, autoSend bool) (*domain.Invoice, error) {
	invoice := &domain.Invoice{
		Type:   domain.InvoiceTypeRegular,
		Status: domain.InvoiceStatusDraft,
		// SequenceID 0 falls back to the company's auto-assigned FV sequence
		// inside InvoiceService.Create; a non-zero value uses the template's
		// chosen sequence (e.g. the company's "77" series).
		SequenceID:     ri.SequenceID,
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
	if err := s.invoices.Create(ctx, companyID, invoice); err != nil {
		return nil, fmt.Errorf("creating invoice from template: %w", err)
	}

	if autoSend && ri.AutoSend {
		s.autoSendInvoice(ctx, companyID, ri, invoice)
	}

	return invoice, nil
}

// autoSendInvoice emails a freshly generated invoice for templates that opt into
// auto-send. It is best-effort: any problem (no emailer, SMTP unconfigured,
// missing/invalid recipient, send failure) is logged and the invoice is left as
// a draft for manual sending. It never returns an error so generation and the
// next-issue-date advance always complete.
func (s *RecurringInvoiceService) autoSendInvoice(ctx context.Context, companyID int64, ri *domain.RecurringInvoice, invoice *domain.Invoice) {
	if s.emailer == nil {
		slog.Warn("recurring auto-send skipped: no email sender wired", "recurring_id", ri.ID, "invoice_id", invoice.ID)
		return
	}

	recipient := ri.AutoSendRecipient
	if recipient == "" && ri.Customer != nil {
		recipient = ri.Customer.Email
	}
	if recipient == "" {
		slog.Warn("recurring auto-send skipped: no recipient email", "recurring_id", ri.ID, "invoice_id", invoice.ID, "invoice_number", invoice.InvoiceNumber)
		return
	}

	defaults, err := s.emailer.Defaults(ctx, companyID, invoice.InvoiceNumber)
	if err != nil {
		slog.Warn("recurring auto-send skipped: failed to resolve email defaults", "error", err, "recurring_id", ri.ID, "invoice_id", invoice.ID)
		return
	}

	opts := EmailOptions{
		To:          recipient,
		Subject:     defaults.Subject,
		Body:        defaults.Body,
		AttachPDF:   defaults.AttachPDF,
		AttachISDOC: defaults.AttachISDOC,
	}
	// Bound the send so a hung SMTP server can't stall the rest of the
	// scheduler run. Recipient is not logged (it is customer PII).
	sendCtx, cancel := context.WithTimeout(ctx, autoSendTimeout)
	defer cancel()
	if err := s.emailer.SendByID(sendCtx, companyID, invoice.ID, opts); err != nil {
		slog.Warn("recurring auto-send failed; invoice left as draft", "error", err, "recurring_id", ri.ID, "invoice_id", invoice.ID, "invoice_number", invoice.InvoiceNumber)
		return
	}
	slog.Info("recurring invoice auto-sent", "recurring_id", ri.ID, "invoice_id", invoice.ID, "invoice_number", invoice.InvoiceNumber)
}
