package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

// setupInvoiceDocumentRouter creates a test router wired with real SQLite repos and a temp data dir.
// It returns the router, the invoice ID, and the data directory path.
func setupInvoiceDocumentRouter(t *testing.T) (*chi.Mux, int64, string) {
	t.Helper()
	db := testutil.NewTestDB(t)
	dataDir := t.TempDir()

	contact := testutil.SeedContact(t, db, nil)
	invoice := testutil.SeedInvoice(t, db, contact.ID, nil)

	docRepo := repository.NewInvoiceDocumentRepository(db)
	docSvc := service.NewInvoiceDocumentService(docRepo, dataDir, nil)
	h := NewInvoiceDocumentHandler(docSvc)

	r := chi.NewRouter()
	r.Route("/api/v1", func(api chi.Router) {
		// Invoice-scoped document routes
		api.Get("/invoices/{id}/documents", h.ListByInvoice)
		// Standalone document routes
		api.Mount("/", h.Routes())
	})

	return r, invoice.ID, dataDir
}

// uploadInvoiceDocument is a helper that inserts a document directly via service for handler tests.
func uploadInvoiceDocument(t *testing.T, db interface{}, svc *service.InvoiceDocumentService, invoiceID int64, filename string, data []byte) int64 {
	t.Helper()
	doc, err := svc.Upload(t.Context(), invoiceID, filename, "application/pdf", data)
	if err != nil {
		t.Fatalf("uploading test document: %v", err)
	}
	return doc.ID
}

// setupInvoiceDocumentRouterWithDoc creates a test router and uploads one document, returning the document ID.
func setupInvoiceDocumentRouterWithDoc(t *testing.T) (*chi.Mux, int64, int64, string) {
	t.Helper()
	db := testutil.NewTestDB(t)
	dataDir := t.TempDir()

	contact := testutil.SeedContact(t, db, nil)
	invoice := testutil.SeedInvoice(t, db, contact.ID, nil)

	docRepo := repository.NewInvoiceDocumentRepository(db)
	docSvc := service.NewInvoiceDocumentService(docRepo, dataDir, nil)

	// Upload a test document via the service layer.
	doc, err := docSvc.Upload(t.Context(), invoice.ID, "test.pdf", "application/pdf", []byte("%PDF-1.4 test content"))
	if err != nil {
		t.Fatalf("uploading test document: %v", err)
	}

	h := NewInvoiceDocumentHandler(docSvc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/invoices/{id}/documents", h.ListByInvoice)
		api.Mount("/", h.Routes())
	})

	return r, invoice.ID, doc.ID, dataDir
}

func TestInvoiceDocumentHandler_ListByInvoice(t *testing.T) {
	r, invoiceID, _, _ := setupInvoiceDocumentRouterWithDoc(t)

	url := fmt.Sprintf("/api/v1/invoices/%d/documents", invoiceID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []invoiceDocumentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("len = %d, want 1", len(resp))
	}
	if len(resp) > 0 && resp[0].Filename != "test.pdf" {
		t.Errorf("Filename = %q, want test.pdf", resp[0].Filename)
	}
}

func TestInvoiceDocumentHandler_ListByInvoice_InvalidID(t *testing.T) {
	r, _, _ := setupInvoiceDocumentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/notanumber/documents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceDocumentHandler_ListByInvoice_Empty(t *testing.T) {
	r, invoiceID, _ := setupInvoiceDocumentRouter(t)

	url := fmt.Sprintf("/api/v1/invoices/%d/documents", invoiceID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []invoiceDocumentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("len = %d, want 0", len(resp))
	}
}

func TestInvoiceDocumentHandler_GetByID(t *testing.T) {
	r, _, docID, _ := setupInvoiceDocumentRouterWithDoc(t)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoice-documents/%d", docID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp invoiceDocumentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID != docID {
		t.Errorf("ID = %d, want %d", resp.ID, docID)
	}
	if resp.Filename != "test.pdf" {
		t.Errorf("Filename = %q, want test.pdf", resp.Filename)
	}
}

func TestInvoiceDocumentHandler_GetByID_NotFound(t *testing.T) {
	r, _, _ := setupInvoiceDocumentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoice-documents/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvoiceDocumentHandler_GetByID_InvalidID(t *testing.T) {
	r, _, _ := setupInvoiceDocumentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoice-documents/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceDocumentHandler_Download(t *testing.T) {
	r, _, docID, _ := setupInvoiceDocumentRouterWithDoc(t)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoice-documents/%d/download", docID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if ct == "" {
		t.Error("expected Content-Type header to be set")
	}

	cd := w.Header().Get("Content-Disposition")
	if cd == "" {
		t.Error("expected Content-Disposition header to be set")
	}
}

func TestInvoiceDocumentHandler_Download_NotFound(t *testing.T) {
	r, _, _ := setupInvoiceDocumentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoice-documents/99999/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvoiceDocumentHandler_Download_FileMissing(t *testing.T) {
	r, _, docID, dataDir := setupInvoiceDocumentRouterWithDoc(t)

	// Remove the file from disk to simulate a missing file.
	entries, _ := filepath.Glob(filepath.Join(dataDir, "documents", "invoices", "*", "*"))
	for _, e := range entries {
		os.Remove(e)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoice-documents/%d/download", docID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvoiceDocumentHandler_Download_InvalidID(t *testing.T) {
	r, _, _ := setupInvoiceDocumentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoice-documents/abc/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestInvoiceDocumentHandler_Delete(t *testing.T) {
	r, _, docID, _ := setupInvoiceDocumentRouterWithDoc(t)

	// Delete.
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/invoice-documents/%d", docID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify deleted.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoice-documents/%d", docID), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", getW.Code, http.StatusNotFound)
	}
}

func TestInvoiceDocumentHandler_Delete_NotFound(t *testing.T) {
	r, _, _ := setupInvoiceDocumentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/invoice-documents/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInvoiceDocumentHandler_Delete_InvalidID(t *testing.T) {
	r, _, _ := setupInvoiceDocumentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/invoice-documents/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
