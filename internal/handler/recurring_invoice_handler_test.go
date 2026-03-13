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

func setupRecurringInvoiceRouter(t *testing.T) (*chi.Mux, int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	recurringRepo := repository.NewRecurringInvoiceRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	sequenceSvc := service.NewSequenceService(sequenceRepo, nil)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, nil)
	recurringInvoiceSvc := service.NewRecurringInvoiceService(recurringRepo, invoiceSvc, nil)

	contactHandler := NewContactHandler(contactSvc)
	h := NewRecurringInvoiceHandler(recurringInvoiceSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/contacts", contactHandler.Routes())
	r.Mount("/api/v1/recurring-invoices", h.Routes())

	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)
	return r, seqID
}

// createTestCustomer creates a customer contact via the API and returns its ID.
func createTestCustomer(t *testing.T, r *chi.Mux) int64 {
	t.Helper()
	body := `{"type":"company","name":"Recurring Test Customer s.r.o.","ico":"12345678"}`
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

func validRecurringInvoiceBody(customerID int64) string {
	return fmt.Sprintf(`{
		"name": "Monthly hosting",
		"customer_id": %d,
		"frequency": "monthly",
		"next_issue_date": "2026-04-01",
		"currency_code": "CZK",
		"exchange_rate": 100,
		"payment_method": "bank_transfer",
		"is_active": true,
		"items": [{"description": "Hosting", "quantity": 100, "unit": "ks", "unit_price": 50000, "vat_rate_percent": 21, "sort_order": 0}]
	}`, customerID)
}

func createRecurringInvoice(t *testing.T, r *chi.Mux, customerID int64) recurringInvoiceResponse {
	t.Helper()
	body := validRecurringInvoiceBody(customerID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating recurring invoice: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp recurringInvoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

func TestRecurringInvoiceHandler_Create(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)

	body := validRecurringInvoiceBody(customerID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp recurringInvoiceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Name != "Monthly hosting" {
		t.Errorf("Name = %q, want %q", resp.Name, "Monthly hosting")
	}
	if resp.Frequency != "monthly" {
		t.Errorf("Frequency = %q, want %q", resp.Frequency, "monthly")
	}
	if resp.NextIssueDate != "2026-04-01" {
		t.Errorf("NextIssueDate = %q, want %q", resp.NextIssueDate, "2026-04-01")
	}
	if len(resp.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(resp.Items))
	}
}

func TestRecurringInvoiceHandler_Create_InvalidJSON(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-invoices", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRecurringInvoiceHandler_Create_MissingName(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)

	body := fmt.Sprintf(`{
		"name": "",
		"customer_id": %d,
		"frequency": "monthly",
		"next_issue_date": "2026-04-01",
		"items": [{"description": "Hosting", "quantity": 100, "unit": "ks", "unit_price": 50000, "vat_rate_percent": 21, "sort_order": 0}]
	}`, customerID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestRecurringInvoiceHandler_List(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)

	createRecurringInvoice(t, r, customerID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-invoices", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []recurringInvoiceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("len(resp) = %d, want 1", len(resp))
	}
}

func TestRecurringInvoiceHandler_GetByID(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)
	created := createRecurringInvoice(t, r, customerID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/recurring-invoices/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp recurringInvoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "Monthly hosting" {
		t.Errorf("Name = %q, want %q", resp.Name, "Monthly hosting")
	}
}

func TestRecurringInvoiceHandler_GetByID_NotFound(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-invoices/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRecurringInvoiceHandler_GetByID_InvalidID(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-invoices/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRecurringInvoiceHandler_Update(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)
	created := createRecurringInvoice(t, r, customerID)

	updateBody := fmt.Sprintf(`{
		"name": "Updated hosting",
		"customer_id": %d,
		"frequency": "quarterly",
		"next_issue_date": "2026-07-01",
		"currency_code": "CZK",
		"exchange_rate": 100,
		"payment_method": "bank_transfer",
		"is_active": true,
		"items": [{"description": "Premium Hosting", "quantity": 100, "unit": "ks", "unit_price": 75000, "vat_rate_percent": 21, "sort_order": 0}]
	}`, customerID)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/recurring-invoices/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp recurringInvoiceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "Updated hosting" {
		t.Errorf("Name = %q, want %q", resp.Name, "Updated hosting")
	}
}

func TestRecurringInvoiceHandler_Delete(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)
	created := createRecurringInvoice(t, r, customerID)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/recurring-invoices/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify deleted.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/recurring-invoices/%d", created.ID), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", getW.Code, http.StatusNotFound)
	}
}

func TestRecurringInvoiceHandler_Delete_NotFound(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/recurring-invoices/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRecurringInvoiceHandler_GenerateInvoice(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)
	created := createRecurringInvoice(t, r, customerID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/recurring-invoices/%d/generate", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp invoiceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero invoice ID")
	}
	if resp.CustomerID != customerID {
		t.Errorf("CustomerID = %d, want %d", resp.CustomerID, customerID)
	}
	if resp.InvoiceNumber == "" {
		t.Error("expected non-empty invoice number")
	}
	if len(resp.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(resp.Items))
	}
}

func TestRecurringInvoiceHandler_ProcessDue(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-invoices/process-due", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp processDueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.GeneratedCount != 0 {
		t.Errorf("GeneratedCount = %d, want 0 (no due invoices)", resp.GeneratedCount)
	}
}

func TestRecurringInvoiceHandler_Update_InvalidID(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/recurring-invoices/abc", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRecurringInvoiceHandler_Update_InvalidJSON(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)
	created := createRecurringInvoice(t, r, customerID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/recurring-invoices/%d", created.ID), bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRecurringInvoiceHandler_Update_MissingDate(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)
	created := createRecurringInvoice(t, r, customerID)

	// Missing next_issue_date should fail in toDomain.
	body := fmt.Sprintf(`{
		"name": "Updated",
		"customer_id": %d,
		"frequency": "monthly",
		"next_issue_date": "",
		"items": []
	}`, customerID)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/recurring-invoices/%d", created.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestRecurringInvoiceHandler_Update_FullSuccess(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)
	created := createRecurringInvoice(t, r, customerID)

	endDate := "2026-12-31"
	updateBody := fmt.Sprintf(`{
		"name": "Updated name",
		"customer_id": %d,
		"frequency": "quarterly",
		"next_issue_date": "2026-07-01",
		"end_date": "%s",
		"currency_code": "EUR",
		"exchange_rate": 2500,
		"payment_method": "bank_transfer",
		"bank_account": "1234567890",
		"bank_code": "0100",
		"iban": "CZ6508000000192000145399",
		"swift": "KOMBCZPP",
		"constant_symbol": "0308",
		"notes": "Updated notes",
		"is_active": false,
		"items": [
			{"description": "Premium Hosting", "quantity": 200, "unit": "ks", "unit_price": 75000, "vat_rate_percent": 21, "sort_order": 0},
			{"description": "Support", "quantity": 100, "unit": "hod", "unit_price": 100000, "vat_rate_percent": 21, "sort_order": 1}
		]
	}`, customerID, endDate)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/recurring-invoices/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp recurringInvoiceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Name != "Updated name" {
		t.Errorf("Name = %q, want %q", resp.Name, "Updated name")
	}
	if resp.Frequency != "quarterly" {
		t.Errorf("Frequency = %q, want %q", resp.Frequency, "quarterly")
	}
	if resp.EndDate == nil || *resp.EndDate != endDate {
		t.Errorf("EndDate = %v, want %q", resp.EndDate, endDate)
	}
	if resp.CurrencyCode != "EUR" {
		t.Errorf("CurrencyCode = %q, want %q", resp.CurrencyCode, "EUR")
	}
	if resp.IsActive {
		t.Error("IsActive = true, want false")
	}
	if len(resp.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(resp.Items))
	}
}

func TestRecurringInvoiceHandler_GenerateInvoice_InvalidID(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-invoices/abc/generate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRecurringInvoiceHandler_GenerateInvoice_NotFound(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-invoices/99999/generate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestRecurringInvoiceHandler_Delete_InvalidID(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/recurring-invoices/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRecurringInvoiceHandler_Create_MissingDate(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)

	body := fmt.Sprintf(`{
		"name": "Test",
		"customer_id": %d,
		"frequency": "monthly",
		"next_issue_date": "",
		"items": []
	}`, customerID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestRecurringInvoiceHandler_Create_WithEndDate(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)

	body := fmt.Sprintf(`{
		"name": "With end date",
		"customer_id": %d,
		"frequency": "monthly",
		"next_issue_date": "2026-04-01",
		"end_date": "2026-12-31",
		"currency_code": "CZK",
		"exchange_rate": 100,
		"payment_method": "bank_transfer",
		"is_active": true,
		"items": [{"description": "Service", "quantity": 100, "unit": "ks", "unit_price": 50000, "vat_rate_percent": 21, "sort_order": 0}]
	}`, customerID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp recurringInvoiceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.EndDate == nil {
		t.Error("expected EndDate to be set")
	} else if *resp.EndDate != "2026-12-31" {
		t.Errorf("EndDate = %q, want %q", *resp.EndDate, "2026-12-31")
	}
}

func TestRecurringInvoiceHandler_GenerateInvoice_VerifyResponse(t *testing.T) {
	r, _ := setupRecurringInvoiceRouter(t)
	customerID := createTestCustomer(t, r)
	created := createRecurringInvoice(t, r, customerID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/recurring-invoices/%d/generate", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp invoiceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Type != "regular" {
		t.Errorf("Type = %q, want %q", resp.Type, "regular")
	}
	if resp.CurrencyCode != "CZK" {
		t.Errorf("CurrencyCode = %q, want %q", resp.CurrencyCode, "CZK")
	}
}
