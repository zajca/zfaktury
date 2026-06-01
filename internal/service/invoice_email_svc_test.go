package service

import (
	"context"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/isdoc"
	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service/email"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newInvoiceEmailService(t *testing.T, smtp config.SMTPConfig) *InvoiceEmailService {
	t.Helper()
	db := testutil.NewTestDB(t)
	invoiceRepo := repository.NewInvoiceRepository(db)
	contactRepo := repository.NewContactRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	companyRepo := repository.NewCompanyRepository(db)

	contactSvc := NewContactService(contactRepo, nil, nil)
	sequenceSvc := NewSequenceService(sequenceRepo, nil)
	invoiceSvc := NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, nil)
	settingsSvc := NewSettingsService(settingsRepo, nil)

	return NewInvoiceEmailService(invoiceSvc, settingsSvc, companyRepo, pdf.NewInvoicePDFGenerator(), isdoc.NewISDOCGenerator(), email.NewEmailSender(smtp))
}

func TestInvoiceEmailService_Defaults_SubstitutesInvoiceNumber(t *testing.T) {
	svc := newInvoiceEmailService(t, config.SMTPConfig{})

	d, err := svc.Defaults(context.Background(), 1, "FV-2026-001")
	if err != nil {
		t.Fatalf("Defaults() error: %v", err)
	}
	if !strings.Contains(d.Subject, "FV-2026-001") {
		t.Errorf("subject %q should contain the invoice number", d.Subject)
	}
	if !strings.Contains(d.Body, "FV-2026-001") {
		t.Errorf("body %q should contain the invoice number", d.Body)
	}
	if !d.AttachPDF {
		t.Error("AttachPDF should default to true")
	}
}

func TestInvoiceEmailService_Send_NotConfigured(t *testing.T) {
	svc := newInvoiceEmailService(t, config.SMTPConfig{}) // no host -> not configured
	err := svc.Send(context.Background(), &domain.Company{ID: 1}, &domain.Invoice{ID: 1, InvoiceNumber: "X"}, EmailOptions{
		To: "a@b.com", Subject: "S", Body: "B", AttachPDF: true,
	})
	if err == nil {
		t.Fatal("expected error when SMTP is not configured")
	}
}

func TestInvoiceEmailService_Send_Validation(t *testing.T) {
	configured := config.SMTPConfig{Host: "smtp.example.com", Port: 587, From: "n@e.com"}
	svc := newInvoiceEmailService(t, configured)
	company := &domain.Company{ID: 1}
	invoice := &domain.Invoice{ID: 1, InvoiceNumber: "X"}

	tests := []struct {
		name string
		opts EmailOptions
	}{
		{"empty recipient", EmailOptions{To: "", Subject: "S", AttachPDF: true}},
		{"invalid recipient", EmailOptions{To: "nope", Subject: "S", AttachPDF: true}},
		{"no attachments", EmailOptions{To: "a@b.com", Subject: "S"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.Send(context.Background(), company, invoice, tt.opts); err == nil {
				t.Errorf("expected validation error for %s", tt.name)
			}
		})
	}
}
