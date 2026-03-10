package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service/email"
)

// Sentinel errors for reminder business logic validation.
var (
	ErrInvoiceNotOverdue = errors.New("invoice is not overdue")
	ErrNoCustomerEmail   = errors.New("customer has no email address")
)

// reminderRepository defines the persistence operations needed by the reminder service.
type reminderRepository interface {
	Create(ctx context.Context, reminder *domain.PaymentReminder) error
	ListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.PaymentReminder, error)
	CountByInvoiceID(ctx context.Context, invoiceID int64) (int, error)
}

// reminderInvoiceRepo defines the invoice lookup needed by the reminder service.
type reminderInvoiceRepo interface {
	GetByID(ctx context.Context, id int64) (*domain.Invoice, error)
}

// reminderEmailSender abstracts email sending for testability.
type reminderEmailSender interface {
	Send(ctx context.Context, msg email.EmailMessage) error
}

// ReminderService provides business logic for payment reminders.
type ReminderService struct {
	reminderRepo reminderRepository
	invoiceRepo  reminderInvoiceRepo
	emailSender  reminderEmailSender
	userName     string
}

// NewReminderService creates a new ReminderService.
func NewReminderService(
	reminderRepo reminderRepository,
	invoiceRepo reminderInvoiceRepo,
	emailSender reminderEmailSender,
	userName string,
) *ReminderService {
	return &ReminderService{
		reminderRepo: reminderRepo,
		invoiceRepo:  invoiceRepo,
		emailSender:  emailSender,
		userName:     userName,
	}
}

// SendReminder sends a payment reminder for the given invoice.
// It validates the invoice is overdue, determines the escalation level,
// generates the appropriate email template, sends the email, and records the reminder.
func (s *ReminderService) SendReminder(ctx context.Context, invoiceID int64) (*domain.PaymentReminder, error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("getting invoice: %w", err)
	}

	// Validate invoice is overdue.
	if !isOverdue(inv) {
		return nil, ErrInvoiceNotOverdue
	}

	// Get customer email.
	if inv.Customer == nil || inv.Customer.Email == "" {
		return nil, ErrNoCustomerEmail
	}
	customerEmail := inv.Customer.Email

	// Determine reminder level (1-3, capped at 3).
	count, err := s.reminderRepo.CountByInvoiceID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("counting reminders: %w", err)
	}
	level := count + 1
	if level > 3 {
		level = 3
	}

	// Calculate days overdue.
	now := time.Now()
	daysOverdue := int(now.Sub(inv.DueDate).Hours() / 24)
	if daysOverdue < 0 {
		daysOverdue = 0
	}

	// Build template data.
	data := email.ReminderData{
		CustomerName:   inv.Customer.Name,
		InvoiceNumber:  inv.InvoiceNumber,
		TotalAmount:    formatAmountCZK(inv.TotalAmount),
		DueDate:        formatDateCzech(inv.DueDate),
		DaysOverdue:    daysOverdue,
		BankAccount:    formatBankAccount(inv.BankAccount, inv.BankCode),
		VariableSymbol: inv.VariableSymbol,
		UserName:       s.userName,
	}

	subject, bodyHTML, bodyText := email.ReminderTemplate(level, data)

	// Send email.
	msg := email.EmailMessage{
		To:       []string{customerEmail},
		Subject:  subject,
		BodyHTML: bodyHTML,
		BodyText: bodyText,
	}
	if err := s.emailSender.Send(ctx, msg); err != nil {
		return nil, fmt.Errorf("sending reminder email: %w", err)
	}

	// Record the reminder.
	reminder := &domain.PaymentReminder{
		InvoiceID:      invoiceID,
		ReminderNumber: level,
		SentAt:         now,
		SentTo:         customerEmail,
		Subject:        subject,
		BodyPreview:    truncate(bodyText, 200),
	}
	if err := s.reminderRepo.Create(ctx, reminder); err != nil {
		return nil, fmt.Errorf("recording reminder: %w", err)
	}

	return reminder, nil
}

// GetReminders returns all reminders for the given invoice.
func (s *ReminderService) GetReminders(ctx context.Context, invoiceID int64) ([]domain.PaymentReminder, error) {
	return s.reminderRepo.ListByInvoiceID(ctx, invoiceID)
}

// isOverdue checks whether the invoice qualifies for a reminder.
func isOverdue(inv *domain.Invoice) bool {
	if inv.Status == domain.InvoiceStatusOverdue {
		return true
	}
	if inv.Status == domain.InvoiceStatusSent && time.Now().After(inv.DueDate) {
		return true
	}
	return false
}

// formatAmountCZK formats a domain.Amount as Czech currency string (e.g. "12 345,67 Kc").
func formatAmountCZK(amount domain.Amount) string {
	whole := int64(amount) / 100
	fraction := int64(amount) % 100
	if fraction < 0 {
		fraction = -fraction
	}

	// Format whole part with space as thousands separator.
	wholeStr := formatWithThousandsSep(whole)
	return fmt.Sprintf("%s,%02d Kc", wholeStr, fraction)
}

// formatWithThousandsSep formats an integer with spaces as thousands separators.
func formatWithThousandsSep(n int64) string {
	negative := n < 0
	if negative {
		n = -n
	}

	s := fmt.Sprintf("%d", n)
	// Insert spaces from the right.
	var result []byte
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ' ')
		}
		result = append(result, byte(ch))
	}

	if negative {
		return "-" + string(result)
	}
	return string(result)
}

// formatDateCzech formats a time as Czech date string (e.g. "10. 3. 2026").
func formatDateCzech(t time.Time) string {
	return fmt.Sprintf("%d. %d. %d", t.Day(), int(t.Month()), t.Year())
}

// formatBankAccount combines bank account and bank code into "account/code" format.
func formatBankAccount(account, code string) string {
	if code != "" {
		return account + "/" + code
	}
	return account
}

// truncate limits a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
