package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupInvoiceRouter(t *testing.T) (*chi.Mux, int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	contactSvc := service.NewContactService(contactRepo, nil)
	sequenceSvc := service.NewSequenceService(sequenceRepo)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc)

	contactHandler := NewContactHandler(contactSvc)
	invoiceHandler := NewInvoiceHandler(invoiceSvc, nil, nil, nil)

	r := chi.NewRouter()
	r.Mount("/api/v1/contacts", contactHandler.Routes())
	r.Mount("/api/v1/invoices", invoiceHandler.Routes())

	// Seed an invoice sequence.
	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	return r, seqID
}

func createInvoiceCustomer(t *testing.T, r *chi.Mux) int64 {
	t.Helper()
	body := `{"type":"company","name":"Invoice Customer"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating customer: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp contactResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.ID
}

var handlerInvoiceCounter int

func invoiceBody(customerID, seqID int64) string {
	handlerInvoiceCounter++
	return fmt.Sprintf(`{
		"sequence_id": %d,
		"invoice_number": "FV-TEST-%03d",
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
	}`, seqID, handlerInvoiceCounter, customerID)
}

func TestInvoiceHandler_Create(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(invoiceBody(customerID, seqID)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Status != "draft" {
		t.Errorf("Status = %q, want %q", resp.Status, "draft")
	}
	if resp.SubtotalAmount == 0 {
		t.Error("expected non-zero SubtotalAmount")
	}
}

func TestInvoiceHandler_Create_InvalidBody(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_Create_NoItems(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	body := fmt.Sprintf(`{"sequence_id":%d,"invoice_number":"FV-NOITEMS","customer_id":%d,"issue_date":"2026-03-01","due_date":"2026-03-15","items":[]}`, seqID, customerID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func createTestInvoice(t *testing.T, r *chi.Mux, customerID, seqID int64) invoiceResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(invoiceBody(customerID, seqID)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating invoice: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

func TestInvoiceHandler_GetByID(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(resp.Items))
	}
}

func TestInvoiceHandler_GetByID_NotFound(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvoiceHandler_GetByID_InvalidID(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_List(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	createTestInvoice(t, r, customerID, seqID)
	createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp listResponse[invoiceResponse]
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 2 {
		t.Errorf("Total = %d, want 2", resp.Total)
	}
}

func TestInvoiceHandler_Delete(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/invoices/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestInvoiceHandler_MarkAsSent(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/send", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "sent" {
		t.Errorf("Status = %q, want %q", resp.Status, "sent")
	}
}

func TestInvoiceHandler_MarkAsPaid(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	paidBody := fmt.Sprintf(`{"amount":%d,"paid_at":"2026-03-10"}`, created.TotalAmount)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/mark-paid", created.ID), bytes.NewBufferString(paidBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "paid" {
		t.Errorf("Status = %q, want %q", resp.Status, "paid")
	}
}

func TestInvoiceHandler_Duplicate(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/duplicate", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID == created.ID {
		t.Error("duplicate should have different ID")
	}
	if resp.Status != "draft" {
		t.Errorf("Status = %q, want %q", resp.Status, "draft")
	}
	if len(resp.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(resp.Items))
	}
}

func TestInvoiceHandler_Update(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	updateBody := fmt.Sprintf(`{
		"sequence_id": %d,
		"invoice_number": "%s",
		"type": "regular",
		"customer_id": %d,
		"issue_date": "2026-03-01",
		"due_date": "2026-03-15",
		"delivery_date": "2026-03-01",
		"payment_method": "bank_transfer",
		"notes": "Updated notes",
		"items": [
			{"description": "Updated item", "quantity": 200, "unit": "hod", "unit_price": 100000, "vat_rate_percent": 21}
		]
	}`, seqID, created.InvoiceNumber, customerID)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/invoices/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}
