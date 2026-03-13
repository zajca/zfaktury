package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/isdoc"
	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/service/email"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupEmailRouter(t *testing.T) *chi.Mux {
	t.Helper()

	db := testutil.NewTestDB(t)
	invoiceRepo := repository.NewInvoiceRepository(db)
	contactRepo := repository.NewContactRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	sequenceSvc := service.NewSequenceService(sequenceRepo, nil)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, nil)
	settingsSvc := service.NewSettingsService(settingsRepo, nil)
	pdfGen := pdf.NewInvoicePDFGenerator()
	// Empty SMTPConfig means IsConfigured() returns false.
	sender := email.NewEmailSender(config.SMTPConfig{})

	isdocGen := isdoc.NewISDOCGenerator()
	h := NewEmailHandler(invoiceSvc, settingsSvc, pdfGen, isdocGen, sender)

	r := chi.NewRouter()
	r.Post("/api/v1/invoices/{id}/send-email", h.SendEmail)
	return r
}

func TestEmailHandler_SendEmail_InvalidID(t *testing.T) {
	r := setupEmailRouter(t)

	body := map[string]string{
		"to":      "test@example.com",
		"subject": "Test",
		"body":    "Hello",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/abc/send-email", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp errorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Error != "invalid invoice ID" {
		t.Errorf("expected error %q, got %q", "invalid invoice ID", resp.Error)
	}
}

func TestEmailHandler_SendEmail_UnconfiguredSMTP(t *testing.T) {
	r := setupEmailRouter(t)

	body := map[string]string{
		"to":      "test@example.com",
		"subject": "Test Invoice",
		"body":    "Please find attached.",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/1/send-email", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rr.Code)
	}

	var resp errorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Error != "SMTP is not configured. Configure [smtp] section in config.toml" {
		t.Errorf("expected error %q, got %q", "SMTP is not configured. Configure [smtp] section in config.toml", resp.Error)
	}
}

// setupEmailConfiguredRouter creates a router with a configured SMTP sender (host set)
// so that validation tests beyond the IsConfigured() check can be reached.
func setupEmailConfiguredRouter(t *testing.T) (*chi.Mux, int64) {
	t.Helper()

	db := testutil.NewTestDB(t)
	invoiceRepo := repository.NewInvoiceRepository(db)
	contactRepo := repository.NewContactRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	sequenceSvc := service.NewSequenceService(sequenceRepo, nil)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, nil)
	settingsSvc := service.NewSettingsService(settingsRepo, nil)
	pdfGen := pdf.NewInvoicePDFGenerator()
	// Configured sender -- has SMTP host set so IsConfigured() returns true.
	sender := email.NewEmailSender(config.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user@example.com",
		Password: "secret",
		From:     "noreply@example.com",
	})

	isdocGen := isdoc.NewISDOCGenerator()
	h := NewEmailHandler(invoiceSvc, settingsSvc, pdfGen, isdocGen, sender)

	contactHandler := NewContactHandler(contactSvc)
	invoiceHandler := NewInvoiceHandler(invoiceSvc, nil, nil, nil)

	r := chi.NewRouter()
	r.Mount("/api/v1/contacts", contactHandler.Routes())
	r.Mount("/api/v1/invoices", invoiceHandler.Routes())
	r.Post("/api/v1/invoices/{id}/send-email", h.SendEmail)

	// Seed an invoice sequence.
	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	return r, seqID
}

func createEmailTestCustomer(t *testing.T, r *chi.Mux) int64 {
	t.Helper()
	body := `{"type":"company","name":"Email Test Customer"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating customer: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return int64(resp["id"].(float64))
}

var emailInvoiceCounter int

func createEmailTestInvoice(t *testing.T, r *chi.Mux, customerID, seqID int64) int64 {
	t.Helper()
	emailInvoiceCounter++
	body := fmt.Sprintf(`{
		"sequence_id": %d,
		"invoice_number": "FV-EMAIL-%03d",
		"type": "regular",
		"customer_id": %d,
		"issue_date": "2026-03-01",
		"due_date": "2026-03-15",
		"delivery_date": "2026-03-01",
		"currency_code": "CZK",
		"payment_method": "bank_transfer",
		"items": [
			{
				"description": "Web dev",
				"quantity": 100,
				"unit": "hod",
				"unit_price": 150000,
				"vat_rate_percent": 21,
				"sort_order": 0
			}
		]
	}`, seqID, emailInvoiceCounter, customerID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating invoice: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return int64(resp["id"].(float64))
}

func TestEmailHandler_SendEmail_MissingTo(t *testing.T) {
	r, seqID := setupEmailConfiguredRouter(t)
	customerID := createEmailTestCustomer(t, r)
	invoiceID := createEmailTestInvoice(t, r, customerID, seqID)

	body := map[string]string{
		"to":      "",
		"subject": "Test Invoice",
		"body":    "Hello",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/send-email", invoiceID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	var resp errorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Error != "recipient email (to) is required" {
		t.Errorf("expected error %q, got %q", "recipient email (to) is required", resp.Error)
	}
}

func TestEmailHandler_SendEmail_InvalidEmailFormat(t *testing.T) {
	r, seqID := setupEmailConfiguredRouter(t)
	customerID := createEmailTestCustomer(t, r)
	invoiceID := createEmailTestInvoice(t, r, customerID, seqID)

	body := map[string]string{
		"to":      "not-an-email",
		"subject": "Test Invoice",
		"body":    "Hello",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/send-email", invoiceID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	var resp errorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Error != "invalid recipient email address" {
		t.Errorf("expected error %q, got %q", "invalid recipient email address", resp.Error)
	}
}

func TestEmailHandler_SendEmail_MissingSubject(t *testing.T) {
	r, seqID := setupEmailConfiguredRouter(t)
	customerID := createEmailTestCustomer(t, r)
	invoiceID := createEmailTestInvoice(t, r, customerID, seqID)

	body := map[string]string{
		"to":      "test@example.com",
		"subject": "",
		"body":    "Hello",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/send-email", invoiceID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	var resp errorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Error != "subject is required" {
		t.Errorf("expected error %q, got %q", "subject is required", resp.Error)
	}
}

func TestEmailHandler_SendEmail_InvoiceNotFound(t *testing.T) {
	r, _ := setupEmailConfiguredRouter(t)

	body := map[string]string{
		"to":      "test@example.com",
		"subject": "Test Invoice",
		"body":    "Hello",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/99999/send-email", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}

	var resp errorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Error != "invoice not found" {
		t.Errorf("expected error %q, got %q", "invoice not found", resp.Error)
	}
}

func TestEmailHandler_SendEmail_InvalidRequestBody(t *testing.T) {
	r, _ := setupEmailConfiguredRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/1/send-email", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp errorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Error != "invalid request body" {
		t.Errorf("expected error %q, got %q", "invalid request body", resp.Error)
	}
}
