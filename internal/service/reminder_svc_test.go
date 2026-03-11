package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service/email"
)

// mockReminderRepo is a test double for reminderRepository.
type mockReminderRepo struct {
	reminders []domain.PaymentReminder
}

func (m *mockReminderRepo) Create(_ context.Context, r *domain.PaymentReminder) error {
	r.ID = int64(len(m.reminders) + 1)
	r.CreatedAt = time.Now()
	m.reminders = append(m.reminders, *r)
	return nil
}

func (m *mockReminderRepo) ListByInvoiceID(_ context.Context, invoiceID int64) ([]domain.PaymentReminder, error) {
	var result []domain.PaymentReminder
	for _, r := range m.reminders {
		if r.InvoiceID == invoiceID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockReminderRepo) CountByInvoiceID(_ context.Context, invoiceID int64) (int, error) {
	count := 0
	for _, r := range m.reminders {
		if r.InvoiceID == invoiceID {
			count++
		}
	}
	return count, nil
}

// mockInvoiceRepo is a test double for reminderInvoiceRepo.
type mockInvoiceRepo struct {
	invoice *domain.Invoice
	err     error
}

func (m *mockInvoiceRepo) GetByID(_ context.Context, _ int64) (*domain.Invoice, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.invoice, nil
}

// mockSettingsReader is a test double for reminderSettingsReader.
type mockSettingsReader struct {
	settings map[string]string
}

func (m *mockSettingsReader) Get(_ context.Context, key string) (string, error) {
	if v, ok := m.settings[key]; ok {
		return v, nil
	}
	return "", nil
}

// mockEmailSender is a test double for reminderEmailSender.
type mockEmailSender struct {
	sentMessages []email.EmailMessage
	err          error
}

func (m *mockEmailSender) Send(_ context.Context, msg email.EmailMessage) error {
	if m.err != nil {
		return m.err
	}
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

func makeOverdueInvoice() *domain.Invoice {
	return &domain.Invoice{
		ID:             1,
		InvoiceNumber:  "FV20260001",
		Status:         domain.InvoiceStatusOverdue,
		DueDate:        time.Now().AddDate(0, 0, -10),
		TotalAmount:    1234567,
		BankAccount:    "1234567890",
		BankCode:       "0100",
		VariableSymbol: "20260001",
		Customer: &domain.Contact{
			ID:    1,
			Name:  "Acme s.r.o.",
			Email: "billing@acme.cz",
		},
	}
}

func TestReminderService_SendReminder_Success(t *testing.T) {
	reminderRepo := &mockReminderRepo{}
	inv := makeOverdueInvoice()
	invoiceRepo := &mockInvoiceRepo{invoice: inv}
	emailSender := &mockEmailSender{}

	svc := NewReminderService(reminderRepo, invoiceRepo, emailSender, &mockSettingsReader{settings: map[string]string{"company_name": "Jan Novak"}})
	ctx := context.Background()

	rem, err := svc.SendReminder(ctx, 1)
	if err != nil {
		t.Fatalf("SendReminder() error: %v", err)
	}

	if rem.ID == 0 {
		t.Error("expected non-zero reminder ID")
	}
	if rem.ReminderNumber != 1 {
		t.Errorf("ReminderNumber = %d, want 1", rem.ReminderNumber)
	}
	if rem.SentTo != "billing@acme.cz" {
		t.Errorf("SentTo = %q, want %q", rem.SentTo, "billing@acme.cz")
	}
	if !strings.Contains(rem.Subject, "FV20260001") {
		t.Errorf("Subject missing invoice number: %q", rem.Subject)
	}

	// Verify email was sent.
	if len(emailSender.sentMessages) != 1 {
		t.Fatalf("expected 1 sent email, got %d", len(emailSender.sentMessages))
	}
	sentMsg := emailSender.sentMessages[0]
	if sentMsg.To[0] != "billing@acme.cz" {
		t.Errorf("email To = %q, want %q", sentMsg.To[0], "billing@acme.cz")
	}
}

func TestReminderService_SendReminder_EscalatesLevel(t *testing.T) {
	reminderRepo := &mockReminderRepo{}
	inv := makeOverdueInvoice()
	invoiceRepo := &mockInvoiceRepo{invoice: inv}
	emailSender := &mockEmailSender{}

	svc := NewReminderService(reminderRepo, invoiceRepo, emailSender, &mockSettingsReader{settings: map[string]string{"company_name": "Jan Novak"}})
	ctx := context.Background()

	// First reminder: level 1.
	rem1, err := svc.SendReminder(ctx, 1)
	if err != nil {
		t.Fatalf("first SendReminder() error: %v", err)
	}
	if rem1.ReminderNumber != 1 {
		t.Errorf("first ReminderNumber = %d, want 1", rem1.ReminderNumber)
	}

	// Second reminder: level 2.
	rem2, err := svc.SendReminder(ctx, 1)
	if err != nil {
		t.Fatalf("second SendReminder() error: %v", err)
	}
	if rem2.ReminderNumber != 2 {
		t.Errorf("second ReminderNumber = %d, want 2", rem2.ReminderNumber)
	}

	// Third reminder: level 3.
	rem3, err := svc.SendReminder(ctx, 1)
	if err != nil {
		t.Fatalf("third SendReminder() error: %v", err)
	}
	if rem3.ReminderNumber != 3 {
		t.Errorf("third ReminderNumber = %d, want 3", rem3.ReminderNumber)
	}

	// Fourth reminder: still level 3 (capped).
	rem4, err := svc.SendReminder(ctx, 1)
	if err != nil {
		t.Fatalf("fourth SendReminder() error: %v", err)
	}
	if rem4.ReminderNumber != 3 {
		t.Errorf("fourth ReminderNumber = %d, want 3 (capped)", rem4.ReminderNumber)
	}
}

func TestReminderService_SendReminder_NotOverdue(t *testing.T) {
	reminderRepo := &mockReminderRepo{}
	inv := &domain.Invoice{
		ID:      1,
		Status:  domain.InvoiceStatusDraft,
		DueDate: time.Now().AddDate(0, 0, 14),
		Customer: &domain.Contact{
			Email: "test@test.com",
		},
	}
	invoiceRepo := &mockInvoiceRepo{invoice: inv}
	emailSender := &mockEmailSender{}

	svc := NewReminderService(reminderRepo, invoiceRepo, emailSender, &mockSettingsReader{settings: map[string]string{"company_name": "Test"}})
	ctx := context.Background()

	_, err := svc.SendReminder(ctx, 1)
	if err == nil {
		t.Fatal("expected error for non-overdue invoice")
	}
	if !strings.Contains(err.Error(), "not overdue") {
		t.Errorf("error = %q, want 'not overdue'", err.Error())
	}
}

func TestReminderService_SendReminder_NoEmail(t *testing.T) {
	reminderRepo := &mockReminderRepo{}
	inv := makeOverdueInvoice()
	inv.Customer.Email = ""
	invoiceRepo := &mockInvoiceRepo{invoice: inv}
	emailSender := &mockEmailSender{}

	svc := NewReminderService(reminderRepo, invoiceRepo, emailSender, &mockSettingsReader{settings: map[string]string{"company_name": "Test"}})
	ctx := context.Background()

	_, err := svc.SendReminder(ctx, 1)
	if err == nil {
		t.Fatal("expected error for missing email")
	}
	if !strings.Contains(err.Error(), "no email") {
		t.Errorf("error = %q, want 'no email'", err.Error())
	}
}

func TestReminderService_SendReminder_EmailError(t *testing.T) {
	reminderRepo := &mockReminderRepo{}
	inv := makeOverdueInvoice()
	invoiceRepo := &mockInvoiceRepo{invoice: inv}
	emailSender := &mockEmailSender{err: errors.New("SMTP connection refused")}

	svc := NewReminderService(reminderRepo, invoiceRepo, emailSender, &mockSettingsReader{settings: map[string]string{"company_name": "Test"}})
	ctx := context.Background()

	_, err := svc.SendReminder(ctx, 1)
	if err == nil {
		t.Fatal("expected error when email sending fails")
	}

	// Should not have recorded the reminder.
	if len(reminderRepo.reminders) != 0 {
		t.Error("reminder should not be saved when email fails")
	}
}

func TestReminderService_SendReminder_SentAndPastDue(t *testing.T) {
	reminderRepo := &mockReminderRepo{}
	inv := makeOverdueInvoice()
	inv.Status = domain.InvoiceStatusSent // not "overdue" but past due_date
	invoiceRepo := &mockInvoiceRepo{invoice: inv}
	emailSender := &mockEmailSender{}

	svc := NewReminderService(reminderRepo, invoiceRepo, emailSender, &mockSettingsReader{settings: map[string]string{"company_name": "Test"}})
	ctx := context.Background()

	rem, err := svc.SendReminder(ctx, 1)
	if err != nil {
		t.Fatalf("SendReminder() error: %v", err)
	}
	if rem.ReminderNumber != 1 {
		t.Errorf("ReminderNumber = %d, want 1", rem.ReminderNumber)
	}
}

func TestReminderService_GetReminders(t *testing.T) {
	reminderRepo := &mockReminderRepo{
		reminders: []domain.PaymentReminder{
			{ID: 1, InvoiceID: 42, ReminderNumber: 1},
			{ID: 2, InvoiceID: 42, ReminderNumber: 2},
		},
	}
	invoiceRepo := &mockInvoiceRepo{}
	emailSender := &mockEmailSender{}

	svc := NewReminderService(reminderRepo, invoiceRepo, emailSender, &mockSettingsReader{settings: map[string]string{"company_name": "Test"}})
	ctx := context.Background()

	reminders, err := svc.GetReminders(ctx, 42)
	if err != nil {
		t.Fatalf("GetReminders() error: %v", err)
	}
	if len(reminders) != 2 {
		t.Errorf("GetReminders() returned %d reminders, want 2", len(reminders))
	}
}

func TestFormatAmountCZK(t *testing.T) {
	tests := []struct {
		amount domain.Amount
		want   string
	}{
		{1234567, "12 345,67 Kc"},
		{10000, "100,00 Kc"},
		{99, "0,99 Kc"},
		{0, "0,00 Kc"},
	}
	for _, tt := range tests {
		got := formatAmountCZK(tt.amount)
		if got != tt.want {
			t.Errorf("formatAmountCZK(%d) = %q, want %q", tt.amount, got, tt.want)
		}
	}
}

func TestFormatDateCzech(t *testing.T) {
	d := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	got := formatDateCzech(d)
	want := "10. 3. 2026"
	if got != want {
		t.Errorf("formatDateCzech() = %q, want %q", got, want)
	}
}

func TestTruncate(t *testing.T) {
	long := strings.Repeat("a", 300)
	got := truncate(long, 200)
	if len(got) != 200 {
		t.Errorf("truncate() length = %d, want 200", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("truncated string should end with '...'")
	}

	short := "hello"
	if truncate(short, 200) != short {
		t.Error("short strings should not be truncated")
	}
}
