package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/domain"
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

// startFakeSMTP starts a loopback SMTP server. When rejectMail is true the server
// answers MAIL FROM with a permanent failure so that every send fails.
func startFakeSMTP(t *testing.T, rejectMail bool, onData func()) (host string, port int) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening for fake SMTP: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveFakeSMTP(conn, rejectMail, onData)
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port
}

func serveFakeSMTP(conn net.Conn, rejectMail bool, onData func()) {
	defer func() { _ = conn.Close() }()
	tp := textproto.NewConn(conn)
	_ = tp.PrintfLine("220 fake ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		cmd := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			_ = tp.PrintfLine("250 fake")
		case strings.HasPrefix(cmd, "MAIL"):
			if rejectMail {
				_ = tp.PrintfLine("550 rejected")
				continue
			}
			_ = tp.PrintfLine("250 ok")
		case strings.HasPrefix(cmd, "RCPT"):
			_ = tp.PrintfLine("250 ok")
		case strings.HasPrefix(cmd, "DATA"):
			_ = tp.PrintfLine("354 go ahead")
			if _, err := tp.ReadDotBytes(); err != nil {
				return
			}
			if onData != nil {
				onData()
			}
			_ = tp.PrintfLine("250 queued")
		case strings.HasPrefix(cmd, "QUIT"):
			_ = tp.PrintfLine("221 bye")
			return
		default:
			_ = tp.PrintfLine("250 ok")
		}
	}
}

// setupEmailStatusRouter wires the email handler to a fake SMTP server and
// returns the router, the invoice service and a seeded sequence ID.
func setupEmailStatusRouter(t *testing.T, rejectMail bool) (*chi.Mux, *service.InvoiceService, int64) {
	t.Helper()
	return setupEmailStatusRouterWithHook(t, rejectMail, nil)
}

// setupEmailStatusRouterWithHook is setupEmailStatusRouter with a callback that
// runs on the fake SMTP server after the message body was received.
func setupEmailStatusRouterWithHook(t *testing.T, rejectMail bool, onData func()) (*chi.Mux, *service.InvoiceService, int64) {
	t.Helper()

	host, port := startFakeSMTP(t, rejectMail, onData)

	db := testutil.NewTestDB(t)
	invoiceRepo := repository.NewInvoiceRepository(db)
	contactRepo := repository.NewContactRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	sequenceSvc := service.NewSequenceService(sequenceRepo, nil)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, nil)
	settingsSvc := service.NewSettingsService(settingsRepo, nil)
	sender := email.NewEmailSender(config.SMTPConfig{Host: host, Port: port, From: "noreply@example.com"})

	h := NewEmailHandler(invoiceSvc, settingsSvc, pdf.NewInvoicePDFGenerator(), isdoc.NewISDOCGenerator(), sender)

	r := chi.NewRouter()
	r.Mount("/api/v1/contacts", NewContactHandler(contactSvc).Routes())
	r.Mount("/api/v1/invoices", NewInvoiceHandler(invoiceSvc, nil, nil, nil).Routes())
	r.Post("/api/v1/invoices/{id}/send-email", h.SendEmail)

	return r, invoiceSvc, testutil.SeedInvoiceSequence(t, db, "FV", 2026)
}

func sendTestEmail(t *testing.T, r *chi.Mux, invoiceID int64) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"to": "customer@example.com", "subject": "Invoice", "body": "Attached."})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/send-email", invoiceID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func getInvoiceJSON(t *testing.T, r *chi.Mux, invoiceID int64) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d", invoiceID), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("getting invoice: status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding invoice: %v", err)
	}
	return resp
}

func TestEmailHandler_SendEmail_DraftBecomesSent(t *testing.T) {
	r, _, seqID := setupEmailStatusRouter(t, false)
	invoiceID := createEmailTestInvoice(t, r, createEmailTestCustomer(t, r), seqID)

	if got := getInvoiceJSON(t, r, invoiceID)["status"]; got != "draft" {
		t.Fatalf("precondition: status = %v, want draft", got)
	}

	rr := sendTestEmail(t, r, invoiceID)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != `{"status":"sent"}` {
		t.Errorf("body = %s, want {\"status\":\"sent\"}", rr.Body.String())
	}

	inv := getInvoiceJSON(t, r, invoiceID)
	if inv["status"] != "sent" {
		t.Errorf("status = %v, want sent", inv["status"])
	}
	if inv["sent_at"] == nil || inv["sent_at"] == "" {
		t.Errorf("sent_at is not set: %v", inv["sent_at"])
	}
}

func TestEmailHandler_SendEmail_PaidInvoiceKeepsStatus(t *testing.T) {
	r, invoiceSvc, seqID := setupEmailStatusRouter(t, false)
	invoiceID := createEmailTestInvoice(t, r, createEmailTestCustomer(t, r), seqID)

	if err := invoiceSvc.MarkAsSent(context.Background(), invoiceID); err != nil {
		t.Fatalf("marking sent: %v", err)
	}
	if err := invoiceSvc.MarkAsPaid(context.Background(), invoiceID, domain.NewAmount(1000, 0), time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("marking paid: %v", err)
	}

	rr := sendTestEmail(t, r, invoiceID)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if got := getInvoiceJSON(t, r, invoiceID)["status"]; got != "paid" {
		t.Errorf("status = %v, want paid", got)
	}
}

func TestEmailHandler_SendEmail_StatusUpdateFailureAfterSend(t *testing.T) {
	var deliveredInvoiceID atomic.Int64
	var invoiceSvc atomic.Pointer[service.InvoiceService]
	// The invoice leaves draft while the message is being delivered, so
	// MarkAsSent fails after the email went out.
	onData := func() {
		if err := invoiceSvc.Load().MarkAsSent(context.Background(), deliveredInvoiceID.Load()); err != nil {
			t.Errorf("marking sent during delivery: %v", err)
		}
	}
	r, svc, seqID := setupEmailStatusRouterWithHook(t, false, onData)
	invoiceSvc.Store(svc)
	invoiceID := createEmailTestInvoice(t, r, createEmailTestCustomer(t, r), seqID)
	deliveredInvoiceID.Store(invoiceID)

	rr := sendTestEmail(t, r, invoiceID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500, body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp["code"] != EmailSentStatusNotUpdatedCode {
		t.Errorf("code = %q, want %q", resp["code"], EmailSentStatusNotUpdatedCode)
	}
	if !strings.Contains(resp["error"], "email was sent") {
		t.Errorf("error = %q, want it to say the email was sent", resp["error"])
	}
}

func TestEmailHandler_SendEmail_FailedSendLeavesDraft(t *testing.T) {
	r, _, seqID := setupEmailStatusRouter(t, true)
	invoiceID := createEmailTestInvoice(t, r, createEmailTestCustomer(t, r), seqID)

	rr := sendTestEmail(t, r, invoiceID)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500, body = %s", rr.Code, rr.Body.String())
	}

	inv := getInvoiceJSON(t, r, invoiceID)
	if inv["status"] != "draft" {
		t.Errorf("status = %v, want draft", inv["status"])
	}
	if inv["sent_at"] != nil {
		t.Errorf("sent_at = %v, want nil", inv["sent_at"])
	}
}
