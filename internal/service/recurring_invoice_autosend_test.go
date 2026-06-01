package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

// fakeEmailer records SendByID calls and can be configured to fail, so the
// recurring auto-send behaviour can be asserted without a real SMTP server.
type fakeEmailer struct {
	defaults EmailDefaults
	sendErr  error
	sent     []fakeSend
}

type fakeSend struct {
	companyID int64
	invoiceID int64
	opts      EmailOptions
}

func (f *fakeEmailer) Defaults(_ context.Context, _ int64, _ string) (EmailDefaults, error) {
	return f.defaults, nil
}

func (f *fakeEmailer) SendByID(_ context.Context, companyID, invoiceID int64, opts EmailOptions) error {
	f.sent = append(f.sent, fakeSend{companyID: companyID, invoiceID: invoiceID, opts: opts})
	return f.sendErr
}

// newAutoSendStack wires a recurring service with a fake emailer and seeds a
// customer that has an email address. It returns the service, the fake, and the
// seeded customer ID.
func newAutoSendStack(t *testing.T, fake *fakeEmailer) (*RecurringInvoiceService, *InvoiceService, int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	recurringRepo := repository.NewRecurringInvoiceRepository(db)
	contactSvc := NewContactService(contactRepo, nil, nil)
	sequenceSvc := NewSequenceService(sequenceRepo, nil)
	invoiceSvc := NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, nil)
	recurringSvc := NewRecurringInvoiceService(recurringRepo, invoiceSvc, fake, nil)

	testutil.SeedInvoiceSequence(t, db, 1, "FV", 2026)

	c := &domain.Contact{Name: "Auto Customer", Type: domain.ContactTypeCompany, Email: "customer@example.com"}
	if err := contactSvc.Create(context.Background(), 1, c); err != nil {
		t.Fatalf("creating customer: %v", err)
	}
	return recurringSvc, invoiceSvc, c.ID
}

func dueAutoSendTemplate(customerID int64) *domain.RecurringInvoice {
	ri := makeTestRecurringInvoice(customerID)
	ri.NextIssueDate = time.Now().Truncate(24 * time.Hour)
	ri.AutoSend = true
	return ri
}

func TestProcessDue_AutoSend_FiresWithCustomerEmail(t *testing.T) {
	fake := &fakeEmailer{defaults: EmailDefaults{AttachPDF: true, Subject: "Faktura", Body: "body"}}
	svc, _, customerID := newAutoSendStack(t, fake)
	ctx := context.Background()

	ri := dueAutoSendTemplate(customerID)
	if err := svc.Create(ctx, 1, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	count, err := svc.ProcessDue(ctx, 1, true)
	if err != nil {
		t.Fatalf("ProcessDue() error: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if len(fake.sent) != 1 {
		t.Fatalf("len(sent) = %d, want 1", len(fake.sent))
	}
	if fake.sent[0].opts.To != "customer@example.com" {
		t.Errorf("recipient = %q, want customer's email", fake.sent[0].opts.To)
	}
	if !fake.sent[0].opts.AttachPDF {
		t.Error("expected AttachPDF from resolved defaults")
	}
	if fake.sent[0].invoiceID == 0 {
		t.Error("expected a non-zero generated invoice ID")
	}
}

func TestProcessDue_AutoSend_UsesRecipientOverride(t *testing.T) {
	fake := &fakeEmailer{defaults: EmailDefaults{AttachPDF: true}}
	svc, _, customerID := newAutoSendStack(t, fake)
	ctx := context.Background()

	ri := dueAutoSendTemplate(customerID)
	ri.AutoSendRecipient = "override@example.com"
	if err := svc.Create(ctx, 1, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, err := svc.ProcessDue(ctx, 1, true); err != nil {
		t.Fatalf("ProcessDue() error: %v", err)
	}
	if len(fake.sent) != 1 || fake.sent[0].opts.To != "override@example.com" {
		t.Fatalf("expected one send to override@example.com, got %+v", fake.sent)
	}
}

func TestProcessDue_NoAutoSend_WhenSchedulerFlagFalse(t *testing.T) {
	fake := &fakeEmailer{}
	svc, _, customerID := newAutoSendStack(t, fake)
	ctx := context.Background()

	ri := dueAutoSendTemplate(customerID)
	if err := svc.Create(ctx, 1, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Manual "process due" button passes autoSend=false.
	if _, err := svc.ProcessDue(ctx, 1, false); err != nil {
		t.Fatalf("ProcessDue() error: %v", err)
	}
	if len(fake.sent) != 0 {
		t.Errorf("expected no sends when scheduler flag is false, got %d", len(fake.sent))
	}
}

func TestProcessDue_NoAutoSend_WhenTemplateOptedOut(t *testing.T) {
	fake := &fakeEmailer{}
	svc, _, customerID := newAutoSendStack(t, fake)
	ctx := context.Background()

	ri := dueAutoSendTemplate(customerID)
	ri.AutoSend = false
	if err := svc.Create(ctx, 1, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, err := svc.ProcessDue(ctx, 1, true); err != nil {
		t.Fatalf("ProcessDue() error: %v", err)
	}
	if len(fake.sent) != 0 {
		t.Errorf("expected no sends when template auto_send is off, got %d", len(fake.sent))
	}
}

func TestGenerateInvoice_NeverAutoSends(t *testing.T) {
	fake := &fakeEmailer{}
	svc, _, customerID := newAutoSendStack(t, fake)
	ctx := context.Background()

	ri := dueAutoSendTemplate(customerID)
	if err := svc.Create(ctx, 1, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, err := svc.GenerateInvoice(ctx, 1, ri.ID); err != nil {
		t.Fatalf("GenerateInvoice() error: %v", err)
	}
	if len(fake.sent) != 0 {
		t.Errorf("manual generate must not auto-send, got %d sends", len(fake.sent))
	}
}

func TestProcessDue_AutoSendFailure_StillGeneratesAndAdvances(t *testing.T) {
	fake := &fakeEmailer{sendErr: errors.New("smtp down")}
	svc, invoiceSvc, customerID := newAutoSendStack(t, fake)
	ctx := context.Background()

	ri := dueAutoSendTemplate(customerID)
	startDate := ri.NextIssueDate
	if err := svc.Create(ctx, 1, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	count, err := svc.ProcessDue(ctx, 1, true)
	if err != nil {
		t.Fatalf("ProcessDue() must not fail when auto-send fails: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	// Invoice was still generated.
	invoices, total, err := invoiceSvc.List(ctx, 1, domain.InvoiceFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Fatalf("total invoices = %d, want 1", total)
	}
	// It stays a draft because the (fake) send failed before MarkAsSent.
	if invoices[0].Status != domain.InvoiceStatusDraft {
		t.Errorf("status = %q, want draft after failed auto-send", invoices[0].Status)
	}

	// Next issue date still advanced.
	updated, err := svc.GetByID(ctx, 1, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if !updated.NextIssueDate.After(startDate) {
		t.Errorf("NextIssueDate %v not advanced past %v", updated.NextIssueDate, startDate)
	}
}

func TestCreate_AutoSend_InvalidRecipientRejected(t *testing.T) {
	fake := &fakeEmailer{}
	svc, _, customerID := newAutoSendStack(t, fake)
	ctx := context.Background()

	ri := dueAutoSendTemplate(customerID)
	ri.AutoSendRecipient = "not-an-email"
	err := svc.Create(ctx, 1, ri)
	if err == nil {
		t.Fatal("expected error for invalid auto-send recipient")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("error = %v, want ErrInvalidInput", err)
	}
}

func TestCreate_AutoSend_BlankRecipientAllowed(t *testing.T) {
	fake := &fakeEmailer{}
	svc, _, customerID := newAutoSendStack(t, fake)
	ctx := context.Background()

	ri := dueAutoSendTemplate(customerID)
	ri.AutoSendRecipient = "" // falls back to customer email at send time
	if err := svc.Create(ctx, 1, ri); err != nil {
		t.Fatalf("Create() with blank recipient should succeed: %v", err)
	}
}
