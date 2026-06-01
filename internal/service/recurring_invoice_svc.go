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

	return s.createInvoiceFromTemplate(ctx, companyID, ri, ri.NextIssueDate)
}

// ProcessDue finds all due recurring invoices, creates draft invoices, advances
// the next_issue_date, and deactivates recurring invoices that are past their
// end_date, scoped to the given company. Returns the count of generated
// invoices.
//
// Sending is decoupled from generation: ProcessDue only creates drafts. The
// scheduler calls SweepAutoSend afterwards to email the unsent drafts of
// auto-send templates (which also retries earlier failures).
func (s *RecurringInvoiceService) ProcessDue(ctx context.Context, companyID int64) (int, error) {
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

		// Generate the invoice (sending happens later via SweepAutoSend).
		_, err := s.createInvoiceFromTemplate(ctx, companyID, ri, ri.NextIssueDate)
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
// and persists it within the given company as a draft. The invoice is linked back
// to its template (RecurringInvoiceID) so SweepAutoSend can find and email it.
func (s *RecurringInvoiceService) createInvoiceFromTemplate(ctx context.Context, companyID int64, ri *domain.RecurringInvoice, issueDate time.Time) (*domain.Invoice, error) {
	invoice := &domain.Invoice{
		Type:   domain.InvoiceTypeRegular,
		Status: domain.InvoiceStatusDraft,
		// SequenceID 0 falls back to the company's auto-assigned FV sequence
		// inside InvoiceService.Create; a non-zero value uses the template's
		// chosen sequence (e.g. the company's "77" series).
		SequenceID:         ri.SequenceID,
		RecurringInvoiceID: ri.ID,
		CustomerID:         ri.CustomerID,
		IssueDate:          issueDate,
		DueDate:            issueDate.AddDate(0, 0, 14),
		DeliveryDate:       issueDate,
		CurrencyCode:       ri.CurrencyCode,
		ExchangeRate:       ri.ExchangeRate,
		PaymentMethod:      ri.PaymentMethod,
		BankAccount:        ri.BankAccount,
		BankCode:           ri.BankCode,
		IBAN:               ri.IBAN,
		SWIFT:              ri.SWIFT,
		ConstantSymbol:     ri.ConstantSymbol,
		Notes:              ri.Notes,
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
	// Sending is left to SweepAutoSend (called by the scheduler).
	if err := s.invoices.Create(ctx, companyID, invoice); err != nil {
		return nil, fmt.Errorf("creating invoice from template: %w", err)
	}

	return invoice, nil
}

// SweepAutoSend emails every still-unsent (draft) invoice that was generated
// from an auto-send template within the company, then marks it sent. It is the
// systematic counterpart to generation: a send that fails leaves the invoice a
// draft, so the next sweep retries it; manually-created invoices are untouched.
// Per-invoice failures are logged and skipped (never abort the sweep). Returns
// the number of invoices successfully sent.
func (s *RecurringInvoiceService) SweepAutoSend(ctx context.Context, companyID int64) (int, error) {
	if s.emailer == nil {
		return 0, nil
	}
	drafts, err := s.repo.ListUnsentAutoSendDrafts(ctx, companyID)
	if err != nil {
		return 0, fmt.Errorf("listing unsent auto-send drafts: %w", err)
	}

	sent := 0
	for _, d := range drafts {
		if d.Recipient == "" {
			slog.Warn("auto-send skipped: no recipient email", "invoice_id", d.InvoiceID, "invoice_number", d.InvoiceNumber)
			continue
		}
		defaults, err := s.emailer.Defaults(ctx, companyID, d.InvoiceNumber)
		if err != nil {
			slog.Warn("auto-send skipped: failed to resolve email defaults", "error", err, "invoice_id", d.InvoiceID)
			continue
		}
		opts := EmailOptions{
			To:          d.Recipient,
			Subject:     defaults.Subject,
			Body:        defaults.Body,
			AttachPDF:   defaults.AttachPDF,
			AttachISDOC: defaults.AttachISDOC,
		}
		// Bound each send so a hung SMTP server can't stall the run. The
		// recipient is not logged (customer PII).
		sendCtx, cancel := context.WithTimeout(ctx, autoSendTimeout)
		err = s.emailer.SendByID(sendCtx, companyID, d.InvoiceID, opts)
		cancel()
		if err != nil {
			slog.Warn("auto-send failed; invoice left as draft, will retry next run", "error", err, "invoice_id", d.InvoiceID, "invoice_number", d.InvoiceNumber)
			continue
		}
		sent++
		slog.Info("recurring invoice auto-sent", "invoice_id", d.InvoiceID, "invoice_number", d.InvoiceNumber)
	}
	return sent, nil
}
