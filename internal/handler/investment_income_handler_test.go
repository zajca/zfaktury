package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/service/ocr"
	"github.com/zajca/zfaktury/internal/testutil"
)

// setupInvestmentRouter creates a test router wired with real SQLite repos.
func setupInvestmentRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)
	capitalRepo := repository.NewCapitalIncomeRepository(db)
	securityRepo := repository.NewSecurityTransactionRepository(db)

	investmentSvc := service.NewInvestmentIncomeService(capitalRepo, securityRepo, nil)

	docRepo := repository.NewInvestmentDocumentRepository(db)
	dataDir := t.TempDir()
	docSvc := service.NewInvestmentDocumentService(docRepo, capitalRepo, securityRepo, dataDir, nil)

	h := NewInvestmentIncomeHandler(investmentSvc, docSvc, nil)

	r := chi.NewRouter()
	r.Route("/api/v1/investments", func(api chi.Router) {
		api.Mount("/", h.Routes())
	})

	return r
}

// --- Capital Income Handler Tests ---

func TestInvestmentHandler_CreateCapitalIncome_Valid(t *testing.T) {
	r := setupInvestmentRouter(t)

	body := `{
		"year": 2025,
		"category": "dividend_cz",
		"description": "Test dividend",
		"income_date": "2025-06-15",
		"gross_amount": 10000,
		"withheld_tax_cz": 1500,
		"withheld_tax_foreign": 0,
		"country_code": "CZ",
		"needs_declaring": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/capital-income", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp capitalIncomeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.NetAmount != 8500 {
		t.Errorf("NetAmount = %d, want 8500", resp.NetAmount)
	}
}

func TestInvestmentHandler_CreateCapitalIncome_InvalidCategory(t *testing.T) {
	r := setupInvestmentRouter(t)

	body := `{
		"year": 2025,
		"category": "invalid",
		"income_date": "2025-01-01",
		"gross_amount": 1000
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/capital-income", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestInvestmentHandler_CreateCapitalIncome_InvalidDateFormat(t *testing.T) {
	r := setupInvestmentRouter(t)

	body := `{
		"year": 2025,
		"category": "dividend_cz",
		"income_date": "15/06/2025",
		"gross_amount": 1000
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/capital-income", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_CreateCapitalIncome_InvalidBody(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/capital-income", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_ListCapitalIncome(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Create two entries.
	for i := 0; i < 2; i++ {
		body := fmt.Sprintf(`{
			"year": 2025,
			"category": "dividend_cz",
			"income_date": "2025-0%d-01",
			"gross_amount": %d
		}`, i+1, (i+1)*1000)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/capital-income", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %d: status = %d, body = %s", i, w.Code, w.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/capital-income?year=2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []capitalIncomeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("len = %d, want 2", len(resp))
	}
}

func TestInvestmentHandler_ListCapitalIncome_MissingYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/capital-income", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_ListCapitalIncome_InvalidYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/capital-income?year=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_UpdateCapitalIncome(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Create an entry.
	body := `{
		"year": 2025,
		"category": "interest",
		"income_date": "2025-01-01",
		"gross_amount": 5000,
		"withheld_tax_cz": 750
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/capital-income", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created capitalIncomeResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Update it.
	updateBody := `{
		"year": 2025,
		"category": "interest",
		"income_date": "2025-01-15",
		"gross_amount": 8000,
		"withheld_tax_cz": 1200
	}`
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/investments/capital-income/%d", created.ID), strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", updateW.Code, http.StatusOK, updateW.Body.String())
	}

	var updated capitalIncomeResponse
	json.NewDecoder(updateW.Body).Decode(&updated)
	if updated.GrossAmount != 8000 {
		t.Errorf("GrossAmount = %d, want 8000", updated.GrossAmount)
	}
	if updated.NetAmount != 6800 {
		t.Errorf("NetAmount = %d, want 6800", updated.NetAmount)
	}
}

func TestInvestmentHandler_UpdateCapitalIncome_InvalidID(t *testing.T) {
	r := setupInvestmentRouter(t)

	body := `{"year":2025,"category":"interest","income_date":"2025-01-01","gross_amount":1000}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/investments/capital-income/abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_DeleteCapitalIncome(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Create an entry.
	body := `{
		"year": 2025,
		"category": "dividend_cz",
		"income_date": "2025-01-01",
		"gross_amount": 1000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/capital-income", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created capitalIncomeResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Delete it.
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/investments/capital-income/%d", created.ID), nil)
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", delW.Code, http.StatusNoContent)
	}
}

func TestInvestmentHandler_DeleteCapitalIncome_InvalidID(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/investments/capital-income/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Security Transaction Handler Tests ---

func TestInvestmentHandler_CreateSecurityTransaction_Valid(t *testing.T) {
	r := setupInvestmentRouter(t)

	body := `{
		"year": 2025,
		"asset_type": "stock",
		"asset_name": "AAPL",
		"isin": "US0378331005",
		"transaction_type": "buy",
		"transaction_date": "2025-03-15",
		"quantity": 10000,
		"unit_price": 1500000,
		"total_amount": 1500000,
		"fees": 5000,
		"currency_code": "CZK",
		"exchange_rate": 10000
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/security-transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp securityTransactionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.AssetName != "AAPL" {
		t.Errorf("AssetName = %q, want AAPL", resp.AssetName)
	}
}

func TestInvestmentHandler_CreateSecurityTransaction_InvalidAssetType(t *testing.T) {
	r := setupInvestmentRouter(t)

	body := `{
		"year": 2025,
		"asset_type": "invalid",
		"transaction_type": "buy",
		"transaction_date": "2025-01-01",
		"quantity": 10000
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/security-transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestInvestmentHandler_CreateSecurityTransaction_InvalidDateFormat(t *testing.T) {
	r := setupInvestmentRouter(t)

	body := `{
		"year": 2025,
		"asset_type": "stock",
		"transaction_type": "buy",
		"transaction_date": "15-03-2025",
		"quantity": 10000
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/security-transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_CreateSecurityTransaction_InvalidBody(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/security-transactions", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_ListSecurityTransactions(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Create an entry.
	body := `{
		"year": 2025,
		"asset_type": "stock",
		"asset_name": "MSFT",
		"transaction_type": "buy",
		"transaction_date": "2025-01-01",
		"quantity": 10000,
		"total_amount": 500000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/security-transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", w.Code, w.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/investments/security-transactions?year=2025", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", listW.Code, http.StatusOK)
	}

	var resp []securityTransactionResponse
	if err := json.NewDecoder(listW.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("len = %d, want 1", len(resp))
	}
}

func TestInvestmentHandler_ListSecurityTransactions_MissingYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/security-transactions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_UpdateSecurityTransaction(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Create.
	body := `{
		"year": 2025,
		"asset_type": "etf",
		"asset_name": "VWCE",
		"transaction_type": "buy",
		"transaction_date": "2025-01-01",
		"quantity": 50000,
		"total_amount": 4000000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/security-transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created securityTransactionResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Update.
	updateBody := `{
		"year": 2025,
		"asset_type": "etf",
		"asset_name": "VWCE.DE",
		"transaction_type": "buy",
		"transaction_date": "2025-01-15",
		"quantity": 50000,
		"total_amount": 4100000
	}`
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/investments/security-transactions/%d", created.ID), strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", updateW.Code, http.StatusOK, updateW.Body.String())
	}

	var updated securityTransactionResponse
	json.NewDecoder(updateW.Body).Decode(&updated)
	if updated.AssetName != "VWCE.DE" {
		t.Errorf("AssetName = %q, want VWCE.DE", updated.AssetName)
	}
}

func TestInvestmentHandler_UpdateSecurityTransaction_InvalidID(t *testing.T) {
	r := setupInvestmentRouter(t)

	body := `{"year":2025,"asset_type":"stock","transaction_type":"buy","transaction_date":"2025-01-01","quantity":10000}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/investments/security-transactions/abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_DeleteSecurityTransaction(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Create.
	body := `{
		"year": 2025,
		"asset_type": "stock",
		"asset_name": "GOOG",
		"transaction_type": "buy",
		"transaction_date": "2025-01-01",
		"quantity": 10000,
		"total_amount": 200000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/security-transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created securityTransactionResponse
	json.NewDecoder(w.Body).Decode(&created)

	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/investments/security-transactions/%d", created.ID), nil)
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", delW.Code, http.StatusNoContent)
	}
}

func TestInvestmentHandler_DeleteSecurityTransaction_InvalidID(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/investments/security-transactions/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Summary and FIFO Handler Tests ---

func TestInvestmentHandler_GetYearSummary(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Create some data. needs_declaring must be true for SumByYear to count it.
	body := `{
		"year": 2025,
		"category": "dividend_foreign",
		"income_date": "2025-06-01",
		"gross_amount": 10000,
		"withheld_tax_cz": 1500,
		"needs_declaring": true
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/capital-income", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", w.Code, w.Body.String())
	}

	summaryReq := httptest.NewRequest(http.MethodGet, "/api/v1/investments/summary/2025", nil)
	summaryW := httptest.NewRecorder()
	r.ServeHTTP(summaryW, summaryReq)

	if summaryW.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", summaryW.Code, http.StatusOK, summaryW.Body.String())
	}

	var resp investmentYearSummaryResponse
	if err := json.NewDecoder(summaryW.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.CapitalIncomeGross != 10000 {
		t.Errorf("CapitalIncomeGross = %d, want 10000", resp.CapitalIncomeGross)
	}
}

func TestInvestmentHandler_GetYearSummary_InvalidYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/summary/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_RecalculateFIFO(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Create buy and sell.
	buyBody := `{
		"year": 2025,
		"asset_type": "stock",
		"asset_name": "AAPL",
		"transaction_type": "buy",
		"transaction_date": "2020-01-01",
		"quantity": 10000,
		"total_amount": 1000000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/security-transactions", strings.NewReader(buyBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("buy create: status = %d, body = %s", w.Code, w.Body.String())
	}

	sellBody := `{
		"year": 2025,
		"asset_type": "stock",
		"asset_name": "AAPL",
		"transaction_type": "sell",
		"transaction_date": "2025-06-01",
		"quantity": 10000,
		"total_amount": 2000000,
		"fees": 5000
	}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/investments/security-transactions", strings.NewReader(sellBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("sell create: status = %d, body = %s", w2.Code, w2.Body.String())
	}

	// Recalculate FIFO.
	fifoReq := httptest.NewRequest(http.MethodPost, "/api/v1/investments/recalculate-fifo/2025", nil)
	fifoW := httptest.NewRecorder()
	r.ServeHTTP(fifoW, fifoReq)

	if fifoW.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", fifoW.Code, http.StatusOK, fifoW.Body.String())
	}

	var resp investmentYearSummaryResponse
	if err := json.NewDecoder(fifoW.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
}

func TestInvestmentHandler_RecalculateFIFO_InvalidYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/recalculate-fifo/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Document Handler Tests ---

func TestInvestmentHandler_ListDocuments_MissingYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/documents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_ListDocuments_InvalidYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/documents?year=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_DeleteDocument_InvalidID(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/investments/documents/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_ExtractDocument_NilExtraction(t *testing.T) {
	r := setupInvestmentRouter(t)

	// extraction service is nil, should return 501.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/documents/1/extract", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotImplemented)
	}
}

// --- investmentMockOCRProvider implements ocr.Provider for handler tests ---

type investmentHandlerMockOCR struct {
	promptResponse string
	promptErr      error
}

func (m *investmentHandlerMockOCR) ProcessImage(_ context.Context, _ []byte, _ string) (*domain.OCRResult, error) {
	return nil, nil
}

func (m *investmentHandlerMockOCR) ProcessWithPrompt(_ context.Context, _ []byte, _ string, _, _ string) (string, error) {
	return m.promptResponse, m.promptErr
}

func (m *investmentHandlerMockOCR) Name() string {
	return "investment-handler-mock"
}

var _ ocr.Provider = (*investmentHandlerMockOCR)(nil)

// setupInvestmentRouterWithExtraction creates a test router with a real extraction service using a mock OCR provider.
func setupInvestmentRouterWithExtraction(t *testing.T, provider ocr.Provider) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)
	capitalRepo := repository.NewCapitalIncomeRepository(db)
	securityRepo := repository.NewSecurityTransactionRepository(db)

	investmentSvc := service.NewInvestmentIncomeService(capitalRepo, securityRepo, nil)

	docRepo := repository.NewInvestmentDocumentRepository(db)
	dataDir := t.TempDir()
	docSvc := service.NewInvestmentDocumentService(docRepo, capitalRepo, securityRepo, dataDir, nil)

	extractionSvc := service.NewInvestmentExtractionService(provider, docSvc, capitalRepo, securityRepo, docRepo)

	h := NewInvestmentIncomeHandler(investmentSvc, docSvc, extractionSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/investments", func(api chi.Router) {
		api.Mount("/", h.Routes())
	})

	return r
}

// buildInvestmentUploadRequest creates a multipart request for uploading an investment document.
func buildInvestmentUploadRequest(t *testing.T, filename, fileContentType string, content []byte, year, platform string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// Write the file part with explicit Content-Type header.
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)}
	h["Content-Type"] = []string{fileContentType}
	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("creating form part: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("copying file content: %v", err)
	}

	// Write year and platform fields.
	if year != "" {
		if err := mw.WriteField("year", year); err != nil {
			t.Fatalf("writing year field: %v", err)
		}
	}
	if platform != "" {
		if err := mw.WriteField("platform", platform); err != nil {
			t.Fatalf("writing platform field: %v", err)
		}
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/documents", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// uploadInvestmentDocument uploads a PDF document and returns its ID.
func uploadInvestmentDocument(t *testing.T, r *chi.Mux) int64 {
	t.Helper()
	pdfContent := append([]byte("%PDF-1.4 "), bytes.Repeat([]byte{0x00}, 503)...)
	req := buildInvestmentUploadRequest(t, "statement.pdf", "application/pdf", pdfContent, "2025", "portu")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("upload: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp investmentDocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.ID
}

// --- UploadDocument tests ---

func TestInvestmentHandler_UploadDocument_ValidPDF(t *testing.T) {
	r := setupInvestmentRouter(t)

	pdfContent := append([]byte("%PDF-1.4 "), bytes.Repeat([]byte{0x00}, 503)...)
	req := buildInvestmentUploadRequest(t, "statement.pdf", "application/pdf", pdfContent, "2025", "portu")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp investmentDocumentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.Platform != "portu" {
		t.Errorf("Platform = %q, want portu", resp.Platform)
	}
	if resp.Filename != "statement.pdf" {
		t.Errorf("Filename = %q, want statement.pdf", resp.Filename)
	}
	if resp.ExtractionStatus != "pending" {
		t.Errorf("ExtractionStatus = %q, want pending", resp.ExtractionStatus)
	}
}

func TestInvestmentHandler_UploadDocument_MissingFile(t *testing.T) {
	r := setupInvestmentRouter(t)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("year", "2025")
	mw.WriteField("platform", "portu")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/documents", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestInvestmentHandler_UploadDocument_MissingYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	pdfContent := append([]byte("%PDF-1.4 "), bytes.Repeat([]byte{0x00}, 503)...)
	req := buildInvestmentUploadRequest(t, "statement.pdf", "application/pdf", pdfContent, "", "portu")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_UploadDocument_InvalidYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	pdfContent := append([]byte("%PDF-1.4 "), bytes.Repeat([]byte{0x00}, 503)...)
	req := buildInvestmentUploadRequest(t, "statement.pdf", "application/pdf", pdfContent, "abc", "portu")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_UploadDocument_MissingPlatform(t *testing.T) {
	r := setupInvestmentRouter(t)

	pdfContent := append([]byte("%PDF-1.4 "), bytes.Repeat([]byte{0x00}, 503)...)
	req := buildInvestmentUploadRequest(t, "statement.pdf", "application/pdf", pdfContent, "2025", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_UploadDocument_InvalidMultipart(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/documents", strings.NewReader("not a valid multipart body"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- DownloadDocument tests ---

func TestInvestmentHandler_DownloadDocument_Success(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Upload a document first.
	docID := uploadInvestmentDocument(t, r)

	// Download it.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/investments/documents/%d/download", docID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check Content-Disposition header contains the filename.
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "statement.pdf") {
		t.Errorf("Content-Disposition = %q, want to contain statement.pdf", cd)
	}

	// Check body has content (the PDF bytes).
	if w.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestInvestmentHandler_DownloadDocument_InvalidID(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/documents/abc/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_DownloadDocument_NotFound(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/documents/99999/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- ListDocuments success path ---

func TestInvestmentHandler_ListDocuments_Success(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Upload two documents.
	uploadInvestmentDocument(t, r)
	uploadInvestmentDocument(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/documents?year=2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []investmentDocumentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("len = %d, want 2", len(resp))
	}
}

func TestInvestmentHandler_ListDocuments_EmptyYear(t *testing.T) {
	r := setupInvestmentRouter(t)

	// No documents for year 2020, should return empty list.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/investments/documents?year=2020", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []investmentDocumentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("len = %d, want 0", len(resp))
	}
}

// --- DeleteDocument success path ---

func TestInvestmentHandler_DeleteDocument_Success(t *testing.T) {
	r := setupInvestmentRouter(t)

	docID := uploadInvestmentDocument(t, r)

	// Delete the document.
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/investments/documents/%d", docID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify document is gone by listing.
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/investments/documents?year=2025", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)

	var docs []investmentDocumentResponse
	json.NewDecoder(listW.Body).Decode(&docs)
	if len(docs) != 0 {
		t.Errorf("expected 0 documents after delete, got %d", len(docs))
	}
}

func TestInvestmentHandler_DeleteDocument_NotFound(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/investments/documents/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- ExtractDocument tests ---

func TestInvestmentHandler_ExtractDocument_InvalidID(t *testing.T) {
	provider := &investmentHandlerMockOCR{}
	r := setupInvestmentRouterWithExtraction(t, provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/documents/abc/extract", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvestmentHandler_ExtractDocument_Success(t *testing.T) {
	mockJSON := `{
		"platform": "portu",
		"capital_entries": [
			{
				"category": "dividend_cz",
				"description": "Dividend from Portu",
				"income_date": "2025-06-15",
				"gross_amount": 100.00,
				"withheld_tax_cz": 15.00,
				"withheld_tax_foreign": 0,
				"country_code": "CZ",
				"needs_declaring": false
			}
		],
		"transactions": [
			{
				"asset_type": "etf",
				"asset_name": "VWCE",
				"isin": "IE00BK5BQT80",
				"transaction_type": "buy",
				"transaction_date": "2025-03-01",
				"quantity": 1.5,
				"unit_price": 100.0,
				"total_amount": 150.0,
				"fees": 1.5,
				"currency_code": "CZK",
				"exchange_rate": 1.0
			}
		],
		"confidence": 0.92
	}`

	provider := &investmentHandlerMockOCR{promptResponse: mockJSON}
	r := setupInvestmentRouterWithExtraction(t, provider)

	// Upload a document first.
	docID := uploadInvestmentDocument(t, r)

	// Extract it.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/investments/documents/%d/extract", docID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Decode the extraction response.
	var resp struct {
		Platform       string                        `json:"platform"`
		CapitalEntries []capitalIncomeResponse       `json:"capital_entries"`
		Transactions   []securityTransactionResponse `json:"transactions"`
		Confidence     float64                       `json:"confidence"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if resp.Platform != "portu" {
		t.Errorf("Platform = %q, want portu", resp.Platform)
	}
	if resp.Confidence != 0.92 {
		t.Errorf("Confidence = %f, want 0.92", resp.Confidence)
	}
	if len(resp.CapitalEntries) != 1 {
		t.Fatalf("CapitalEntries len = %d, want 1", len(resp.CapitalEntries))
	}
	if resp.CapitalEntries[0].Category != "dividend_cz" {
		t.Errorf("CapitalEntries[0].Category = %q, want dividend_cz", resp.CapitalEntries[0].Category)
	}
	if len(resp.Transactions) != 1 {
		t.Fatalf("Transactions len = %d, want 1", len(resp.Transactions))
	}
	if resp.Transactions[0].AssetName != "VWCE" {
		t.Errorf("Transactions[0].AssetName = %q, want VWCE", resp.Transactions[0].AssetName)
	}
}

func TestInvestmentHandler_ExtractDocument_NotFound(t *testing.T) {
	provider := &investmentHandlerMockOCR{}
	r := setupInvestmentRouterWithExtraction(t, provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/investments/documents/99999/extract", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- mapInvestmentError additional cases ---

func TestInvestmentHandler_MapInvestmentError_NotFound(t *testing.T) {
	r := setupInvestmentRouter(t)

	// Request a non-existent capital income entry for update.
	body := `{"year":2025,"category":"interest","income_date":"2025-01-01","gross_amount":1000}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/investments/capital-income/99999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvestmentHandler_MapInvestmentError_DeleteNotFound(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/investments/capital-income/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvestmentHandler_MapInvestmentError_SecurityNotFound(t *testing.T) {
	r := setupInvestmentRouter(t)

	body := `{"year":2025,"asset_type":"stock","transaction_type":"buy","transaction_date":"2025-01-01","quantity":10000}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/investments/security-transactions/99999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvestmentHandler_MapInvestmentError_DeleteSecurityNotFound(t *testing.T) {
	r := setupInvestmentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/investments/security-transactions/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
