package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/isdoc"
	"github.com/zajca/zfaktury/internal/pdf"
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

func TestInvoiceHandler_Create_MissingIssueDate(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	body := fmt.Sprintf(`{"sequence_id":%d,"invoice_number":"FV-NODATE","customer_id":%d,"issue_date":"","due_date":"2026-03-15","items":[{"description":"Test","quantity":100,"unit":"ks","unit_price":10000,"vat_rate_percent":21}]}`, seqID, customerID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "issue_date is required") {
		t.Errorf("error = %q, want to contain %q", resp.Error, "issue_date is required")
	}
}

func TestInvoiceHandler_Create_MissingDueDate(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	body := fmt.Sprintf(`{"sequence_id":%d,"invoice_number":"FV-NODUE","customer_id":%d,"issue_date":"2026-03-01","due_date":"","items":[{"description":"Test","quantity":100,"unit":"ks","unit_price":10000,"vat_rate_percent":21}]}`, seqID, customerID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "due_date is required") {
		t.Errorf("error = %q, want to contain %q", resp.Error, "due_date is required")
	}
}

func TestInvoiceHandler_Update_PreservesInvoiceNumber(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	originalInvoiceNumber := created.InvoiceNumber

	// PUT without invoice_number or sequence_id - simulates what frontend sends.
	updateBody := fmt.Sprintf(`{
		"customer_id": %d,
		"issue_date": "2026-03-01",
		"due_date": "2026-03-15",
		"delivery_date": "2026-03-01",
		"payment_method": "bank_transfer",
		"notes": "Updated",
		"items": [{"description": "Updated", "quantity": 100, "unit": "hod", "unit_price": 100000, "vat_rate_percent": 21}]
	}`, customerID)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/invoices/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PUT status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// GET the invoice and verify invoice_number is preserved.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d", created.ID), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getW.Code, http.StatusOK)
	}

	var got invoiceResponse
	json.NewDecoder(getW.Body).Decode(&got)
	if got.InvoiceNumber != originalInvoiceNumber {
		t.Errorf("InvoiceNumber = %q, want %q (original should be preserved)", got.InvoiceNumber, originalInvoiceNumber)
	}
}

func proformaInvoiceBody(customerID, seqID int64) string {
	handlerInvoiceCounter++
	return fmt.Sprintf(`{
		"sequence_id": %d,
		"invoice_number": "PF-TEST-%03d",
		"type": "proforma",
		"customer_id": %d,
		"issue_date": "2026-03-01",
		"due_date": "2026-03-15",
		"delivery_date": "2026-03-01",
		"currency_code": "CZK",
		"payment_method": "bank_transfer",
		"items": [
			{
				"description": "Proforma web dev",
				"quantity": 100,
				"unit": "hod",
				"unit_price": 150000,
				"vat_rate_percent": 21,
				"sort_order": 0
			}
		]
	}`, seqID, handlerInvoiceCounter, customerID)
}

func createTestProforma(t *testing.T, r *chi.Mux, customerID, seqID int64) invoiceResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(proformaInvoiceBody(customerID, seqID)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating proforma: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

func markInvoiceAsPaid(t *testing.T, r *chi.Mux, id int64, amount int64) invoiceResponse {
	t.Helper()
	paidBody := fmt.Sprintf(`{"amount":%d,"paid_at":"2026-03-10"}`, amount)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/mark-paid", id), bytes.NewBufferString(paidBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("marking invoice as paid: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

func markInvoiceAsSent(t *testing.T, r *chi.Mux, id int64) invoiceResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/send", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("marking invoice as sent: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

func TestInvoiceHandler_SettleProforma(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	// Create a proforma invoice.
	proforma := createTestProforma(t, r, customerID, seqID)
	if proforma.Type != "proforma" {
		t.Fatalf("expected proforma type, got %q", proforma.Type)
	}

	// Mark it as paid (required before settling).
	markInvoiceAsPaid(t, r, proforma.ID, proforma.TotalAmount)

	// Settle the proforma.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/settle", proforma.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.ID == 0 {
		t.Error("expected non-zero ID for settlement invoice")
	}
	if resp.Type != "regular" {
		t.Errorf("Type = %q, want %q", resp.Type, "regular")
	}
	if resp.Status != "draft" {
		t.Errorf("Status = %q, want %q", resp.Status, "draft")
	}
	if resp.RelatedInvoiceID == nil || *resp.RelatedInvoiceID != proforma.ID {
		t.Errorf("RelatedInvoiceID = %v, want %d", resp.RelatedInvoiceID, proforma.ID)
	}
	if len(resp.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(resp.Items))
	}
}

func TestInvoiceHandler_SettleProforma_NotPaid(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	// Create a proforma but do NOT mark as paid.
	proforma := createTestProforma(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/settle", proforma.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestInvoiceHandler_SettleProforma_RegularInvoice(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	// Try to settle a regular invoice (should fail).
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/settle", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestInvoiceHandler_SettleProforma_InvalidID(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/abc/settle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_SettleProforma_NotFound(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/99999/settle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestInvoiceHandler_SettleProforma_Idempotent(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	proforma := createTestProforma(t, r, customerID, seqID)
	markInvoiceAsPaid(t, r, proforma.ID, proforma.TotalAmount)

	// Settle once.
	req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/settle", proforma.ID), nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("first settle: status = %d, body = %s", w1.Code, w1.Body.String())
	}

	var first invoiceResponse
	json.NewDecoder(w1.Body).Decode(&first)

	// Settle again -- should return the same settlement.
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/settle", proforma.ID), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("second settle: status = %d, body = %s", w2.Code, w2.Body.String())
	}

	var second invoiceResponse
	json.NewDecoder(w2.Body).Decode(&second)

	if first.ID != second.ID {
		t.Errorf("idempotent settle: first ID = %d, second ID = %d, expected same", first.ID, second.ID)
	}
}

func TestInvoiceHandler_CreateCreditNote(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	// Create a regular invoice and mark as sent (required for credit note).
	created := createTestInvoice(t, r, customerID, seqID)
	markInvoiceAsSent(t, r, created.ID)

	// Create a credit note with items.
	creditBody := `{
		"items": [
			{
				"description": "Refund for web dev",
				"quantity": 100,
				"unit": "hod",
				"unit_price": 150000,
				"vat_rate_percent": 21,
				"sort_order": 0
			}
		],
		"reason": "Customer requested refund"
	}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/credit-note", created.ID), bytes.NewBufferString(creditBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.ID == 0 {
		t.Error("expected non-zero ID for credit note")
	}
	if resp.Type != "credit_note" {
		t.Errorf("Type = %q, want %q", resp.Type, "credit_note")
	}
	if resp.RelatedInvoiceID == nil || *resp.RelatedInvoiceID != created.ID {
		t.Errorf("RelatedInvoiceID = %v, want %d", resp.RelatedInvoiceID, created.ID)
	}
	if len(resp.Items) == 0 {
		t.Error("expected at least one item in credit note")
	}
}

func TestInvoiceHandler_CreateCreditNote_FullCreditNote(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	created := createTestInvoice(t, r, customerID, seqID)
	markInvoiceAsSent(t, r, created.ID)

	// Create a full credit note (no items, empty reason).
	creditBody := `{"items": [], "reason": ""}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/credit-note", created.ID), bytes.NewBufferString(creditBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Type != "credit_note" {
		t.Errorf("Type = %q, want %q", resp.Type, "credit_note")
	}
	// Full credit note should have items copied from original.
	if len(resp.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1 (copied from original)", len(resp.Items))
	}
}

func TestInvoiceHandler_CreateCreditNote_DraftInvoice(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	// Create a regular invoice but do NOT mark as sent.
	created := createTestInvoice(t, r, customerID, seqID)

	creditBody := `{"items": [], "reason": "test"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/credit-note", created.ID), bytes.NewBufferString(creditBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestInvoiceHandler_CreateCreditNote_ProformaInvoice(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	// Try to create credit note for a proforma (should fail).
	proforma := createTestProforma(t, r, customerID, seqID)

	creditBody := `{"items": [], "reason": "test"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/credit-note", proforma.ID), bytes.NewBufferString(creditBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestInvoiceHandler_CreateCreditNote_InvalidID(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	creditBody := `{"items": [], "reason": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/abc/credit-note", bytes.NewBufferString(creditBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_CreateCreditNote_InvalidBody(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)
	markInvoiceAsSent(t, r, created.ID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/credit-note", created.ID), bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_CreateCreditNote_PaidInvoice(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	// Create and pay a regular invoice, then create credit note.
	created := createTestInvoice(t, r, customerID, seqID)
	markInvoiceAsPaid(t, r, created.ID, created.TotalAmount)

	creditBody := `{"items": [], "reason": "refund"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/credit-note", created.ID), bytes.NewBufferString(creditBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Paid invoices should also allow credit notes.
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Type != "credit_note" {
		t.Errorf("Type = %q, want %q", resp.Type, "credit_note")
	}
}

func TestInvoiceHandler_MarkAsSent_InvalidID(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/abc/send", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_MarkAsSent_NotFound(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/99999/send", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestInvoiceHandler_MarkAsPaid_InvalidID(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	paidBody := `{"amount":100,"paid_at":"2026-03-10"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/abc/mark-paid", bytes.NewBufferString(paidBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_MarkAsPaid_InvalidBody(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/mark-paid", created.ID), bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_MarkAsPaid_InvalidDateFormat(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	paidBody := `{"amount":100,"paid_at":"not-a-date"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/mark-paid", created.ID), bytes.NewBufferString(paidBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestInvoiceHandler_MarkAsPaid_RFC3339Date(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	paidBody := fmt.Sprintf(`{"amount":%d,"paid_at":"2026-03-10T14:30:00Z"}`, created.TotalAmount)
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

func TestInvoiceHandler_Duplicate_NotFound(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/99999/duplicate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestInvoiceHandler_Duplicate_InvalidID(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/abc/duplicate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_Duplicate_PreservesItems(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/duplicate", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.CustomerID != created.CustomerID {
		t.Errorf("CustomerID = %d, want %d", resp.CustomerID, created.CustomerID)
	}
	if len(resp.Items) != len(created.Items) {
		t.Errorf("len(Items) = %d, want %d", len(resp.Items), len(created.Items))
	}
	if resp.InvoiceNumber == created.InvoiceNumber {
		t.Error("duplicate should have a different invoice number")
	}
}

// setupInvoiceRouterWithSettings creates an invoice router that includes settingsSvc, pdfGen, and isdocGen.
func setupInvoiceRouterWithSettings(t *testing.T) (*chi.Mux, *sql.DB, int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil)
	sequenceSvc := service.NewSequenceService(sequenceRepo)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc)
	settingsSvc := service.NewSettingsService(settingsRepo)
	pdfGen := pdf.NewInvoicePDFGenerator()
	isdocGen := isdoc.NewISDOCGenerator()

	contactHandler := NewContactHandler(contactSvc)
	invoiceHandler := NewInvoiceHandler(invoiceSvc, settingsSvc, pdfGen, isdocGen)

	r := chi.NewRouter()
	r.Mount("/api/v1/contacts", contactHandler.Routes())
	r.Mount("/api/v1/invoices", invoiceHandler.Routes())

	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)
	return r, db, seqID
}

// seedSettings inserts required settings for PDF/ISDOC generation into the database.
func seedSettings(t *testing.T, db *sql.DB) {
	t.Helper()
	settings := map[string]string{
		"company_name": "Test Company s.r.o.",
		"ico":          "12345678",
		"dic":          "CZ12345678",
		"street":       "Testovaci 123",
		"city":         "Praha",
		"zip":          "11000",
		"email":        "test@example.com",
		"phone":        "+420123456789",
		"bank_account": "1234567890",
		"bank_code":    "0100",
		"iban":         "CZ6508000000192000145399",
		"swift":        "KOMBCZPP",
	}
	for key, value := range settings {
		_, err := db.ExecContext(context.Background(), "INSERT INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))", key, value)
		if err != nil {
			t.Fatalf("seeding setting %s: %v", key, err)
		}
	}
}

func TestInvoiceHandler_DownloadPDF_InvalidID(t *testing.T) {
	r, _, _ := setupInvoiceRouterWithSettings(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/abc/pdf", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_DownloadPDF_NotFound(t *testing.T) {
	r, _, _ := setupInvoiceRouterWithSettings(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/99999/pdf", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvoiceHandler_DownloadPDF_Success(t *testing.T) {
	r, db, seqID := setupInvoiceRouterWithSettings(t)
	seedSettings(t, db)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d/pdf", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/pdf")
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty PDF body")
	}
}

func TestInvoiceHandler_QRPayment_InvalidID(t *testing.T) {
	r, _, _ := setupInvoiceRouterWithSettings(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/abc/qr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_QRPayment_NotFound(t *testing.T) {
	r, _, _ := setupInvoiceRouterWithSettings(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/99999/qr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvoiceHandler_QRPayment_NoIBAN(t *testing.T) {
	r, _, seqID := setupInvoiceRouterWithSettings(t)
	// No settings seeded, so IBAN is empty.
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d/qr", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestInvoiceHandler_QRPayment_Success(t *testing.T) {
	r, db, seqID := setupInvoiceRouterWithSettings(t)
	seedSettings(t, db)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d/qr", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want %q", ct, "image/png")
	}
}

func TestInvoiceHandler_ExportISDOC_InvalidID(t *testing.T) {
	r, _, _ := setupInvoiceRouterWithSettings(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/abc/isdoc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_ExportISDOC_NotFound(t *testing.T) {
	r, _, _ := setupInvoiceRouterWithSettings(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/99999/isdoc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvoiceHandler_ExportISDOC_Success(t *testing.T) {
	r, db, seqID := setupInvoiceRouterWithSettings(t)
	seedSettings(t, db)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d/isdoc", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/xml" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/xml")
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty ISDOC body")
	}
}

func TestInvoiceHandler_ExportISDOCBatch_InvalidBody(t *testing.T) {
	r, _, _ := setupInvoiceRouterWithSettings(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/export/isdoc", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_ExportISDOCBatch_EmptyIDs(t *testing.T) {
	r, _, _ := setupInvoiceRouterWithSettings(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/export/isdoc", bytes.NewBufferString(`{"invoice_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_ExportISDOCBatch_Success(t *testing.T) {
	r, db, seqID := setupInvoiceRouterWithSettings(t)
	seedSettings(t, db)
	customerID := createInvoiceCustomer(t, r)
	inv1 := createTestInvoice(t, r, customerID, seqID)
	inv2 := createTestInvoice(t, r, customerID, seqID)

	body := fmt.Sprintf(`{"invoice_ids":[%d,%d]}`, inv1.ID, inv2.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/export/isdoc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/zip")
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty ZIP body")
	}
}

func TestInvoiceHandler_ExportISDOCBatch_WithNonExistentIDs(t *testing.T) {
	r, db, seqID := setupInvoiceRouterWithSettings(t)
	seedSettings(t, db)
	customerID := createInvoiceCustomer(t, r)
	inv1 := createTestInvoice(t, r, customerID, seqID)

	// Include a non-existent ID; it should be skipped but the batch should still succeed.
	body := fmt.Sprintf(`{"invoice_ids":[%d,99999]}`, inv1.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/export/isdoc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestInvoiceHandler_Update_InvalidID(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/invoices/abc", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_Update_InvalidBody(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/invoices/%d", created.ID), bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_Update_InvalidDates(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	// Missing issue_date should fail.
	body := fmt.Sprintf(`{
		"customer_id": %d,
		"issue_date": "",
		"due_date": "2026-03-15",
		"items": [{"description": "Test", "quantity": 100, "unit": "ks", "unit_price": 10000, "vat_rate_percent": 21}]
	}`, customerID)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/invoices/%d", created.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestInvoiceHandler_Update_NotFound(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	body := `{
		"customer_id": 1,
		"issue_date": "2026-03-01",
		"due_date": "2026-03-15",
		"items": [{"description": "Test", "quantity": 100, "unit": "ks", "unit_price": 10000, "vat_rate_percent": 21}]
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/invoices/99999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestInvoiceHandler_Delete_InvalidID(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/invoices/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_Delete_NotFound(t *testing.T) {
	r, _ := setupInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/invoices/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvoiceHandler_List_WithFilters(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	createTestInvoice(t, r, customerID, seqID)

	// Filter by status.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices?status=draft", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Filter by type.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/invoices?type=regular", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Invalid status filter should be ignored (not error).
	req = httptest.NewRequest(http.MethodGet, "/api/v1/invoices?status=invalid_status", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Invalid type filter should be ignored.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/invoices?type=invalid_type", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Pagination.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/invoices?limit=5&offset=0", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Date filters.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/invoices?date_from=2026-01-01&date_to=2026-12-31", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Search.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/invoices?search=FV", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Filter by customer_id.
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices?customer_id=%d", customerID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("customer_id filter: status = %d, want %d", w.Code, http.StatusOK)
	}

	var listResp struct {
		Data  []invoiceResponse `json:"data"`
		Total int               `json:"total"`
	}
	json.NewDecoder(w.Body).Decode(&listResp)
	if listResp.Total == 0 {
		t.Error("expected at least 1 invoice when filtering by customer_id")
	}
}

func TestInvoiceHandler_GetByID_WithRelatedInvoices(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)

	// Create a regular invoice.
	created := createTestInvoice(t, r, customerID, seqID)

	// Mark as sent, then paid so we can create a credit note.
	markInvoiceAsSent(t, r, created.ID)
	markInvoiceAsPaid(t, r, created.ID, 18150000) // 150000 * 1.21 = 181500 CZK

	// Create a credit note referencing this invoice.
	cnBody := `{
		"items": [{"description":"Credit","quantity":100,"unit":"hod","unit_price":150000,"vat_rate_percent":21}],
		"reason": "Test credit"
	}`
	cnReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/credit-note", created.ID), bytes.NewBufferString(cnBody))
	cnReq.Header.Set("Content-Type", "application/json")
	cnW := httptest.NewRecorder()
	r.ServeHTTP(cnW, cnReq)
	if cnW.Code != http.StatusCreated {
		t.Fatalf("creating credit note: status = %d, want %d, body = %s", cnW.Code, http.StatusCreated, cnW.Body.String())
	}

	// Now GET the original invoice -- it should have related invoices.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d", created.ID), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", getW.Code, http.StatusOK)
	}

	var resp invoiceResponse
	json.NewDecoder(getW.Body).Decode(&resp)
	if len(resp.RelatedInvoices) == 0 {
		t.Error("expected related invoices (credit note) to be present")
	}
}

func TestInvoiceHandler_MarkAsPaid_WithDate(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	// Mark as sent first.
	markInvoiceAsSent(t, r, created.ID)

	// Mark as paid with specific date.
	paidBody := `{"amount":18150000,"paid_at":"2026-03-10"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/mark-paid", created.ID), bytes.NewBufferString(paidBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp invoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.PaidAt == nil {
		t.Error("expected PaidAt to be set")
	}
	if resp.Status != "paid" {
		t.Errorf("Status = %q, want %q", resp.Status, "paid")
	}
}

func TestInvoiceHandler_MarkAsPaid_NoBody(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	// Mark as sent first.
	markInvoiceAsSent(t, r, created.ID)

	// Mark as paid with empty body (defaults to now).
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/invoices/%d/mark-paid", created.ID), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestInvoiceHandler_ExportISDOCBatch_TooManyIDs(t *testing.T) {
	r, _, _ := setupInvoiceRouterWithSettings(t)

	// Generate 501 IDs.
	ids := make([]int64, 501)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	body, _ := json.Marshal(isdocBatchRequest{InvoiceIDs: ids})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/export/isdoc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_List_WithCustomerIDFilter(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	createTestInvoice(t, r, customerID, seqID)

	// Filter by this specific customer.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices?customer_id=%d", customerID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data  []invoiceResponse `json:"data"`
		Total int               `json:"total"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}

	// Filter by non-existent customer.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/invoices?customer_id=99999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 0 {
		t.Errorf("Total = %d, want 0", resp.Total)
	}
}

func TestInvoiceHandler_DownloadPDF_NoSettings(t *testing.T) {
	r, _, seqID := setupInvoiceRouterWithSettings(t)
	// Don't seed settings -- loadPDFSupplierInfo should still work (empty settings).
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d/pdf", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should still succeed even without settings (supplier info will be empty but generation works).
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestInvoiceHandler_ExportISDOC_WithSettings(t *testing.T) {
	r, db, seqID := setupInvoiceRouterWithSettings(t)
	seedSettings(t, db)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d/isdoc", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/xml" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/xml")
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty ISDOC body")
	}
}
