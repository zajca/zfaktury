package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

// setupDocumentRouter creates a test router wired with real SQLite repos and a temp data dir.
func setupDocumentRouter(t *testing.T) (*chi.Mux, int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	dataDir := t.TempDir()

	expense := testutil.SeedExpense(t, db, nil)

	docRepo := repository.NewDocumentRepository(db)
	docSvc := service.NewDocumentService(docRepo, dataDir)
	h := NewDocumentHandler(docSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1", h.Routes())

	return r, expense.ID
}

// buildUploadRequestWithContentType creates a multipart request with an explicit content-type header on the file part.
func buildUploadRequestWithContentType(t *testing.T, url string, filename string, fileContentType string, content []byte) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

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
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestDocumentHandler_Upload_ValidPDF(t *testing.T) {
	r, expenseID := setupDocumentRouter(t)

	url := fmt.Sprintf("/api/v1/expenses/%d/documents", expenseID)
	req := buildUploadRequestWithContentType(t, url, "receipt.pdf", "application/pdf", []byte("%PDF-1.4 test content"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp documentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Filename != "receipt.pdf" {
		t.Errorf("Filename = %q, want receipt.pdf", resp.Filename)
	}
	if resp.ExpenseID != expenseID {
		t.Errorf("ExpenseID = %d, want %d", resp.ExpenseID, expenseID)
	}
	if resp.ContentType != "application/pdf" {
		t.Errorf("ContentType = %q, want application/pdf", resp.ContentType)
	}
}

func TestDocumentHandler_Upload_InvalidExpenseID(t *testing.T) {
	r, _ := setupDocumentRouter(t)

	url := "/api/v1/expenses/abc/documents"
	req := buildUploadRequestWithContentType(t, url, "file.pdf", "application/pdf", []byte("%PDF-1.4 d"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDocumentHandler_Upload_InvalidContentType(t *testing.T) {
	r, expenseID := setupDocumentRouter(t)

	url := fmt.Sprintf("/api/v1/expenses/%d/documents", expenseID)
	req := buildUploadRequestWithContentType(t, url, "malware.exe", "application/octet-stream", []byte("MZ\x90\x00binary executable content"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestDocumentHandler_Upload_MissingFileField(t *testing.T) {
	r, expenseID := setupDocumentRouter(t)

	url := fmt.Sprintf("/api/v1/expenses/%d/documents", expenseID)
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader("no form data"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDocumentHandler_ListByExpense(t *testing.T) {
	r, expenseID := setupDocumentRouter(t)

	// Upload two documents.
	for _, name := range []string{"doc1.pdf", "doc2.pdf"} {
		url := fmt.Sprintf("/api/v1/expenses/%d/documents", expenseID)
		req := buildUploadRequestWithContentType(t, url, name, "application/pdf", []byte("%PDF-1.4 test"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("upload %s: status = %d, body = %s", name, w.Code, w.Body.String())
		}
	}

	url := fmt.Sprintf("/api/v1/expenses/%d/documents", expenseID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []documentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("len = %d, want 2", len(resp))
	}
}

func TestDocumentHandler_ListByExpense_InvalidID(t *testing.T) {
	r, _ := setupDocumentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/notanumber/documents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDocumentHandler_GetByID(t *testing.T) {
	r, expenseID := setupDocumentRouter(t)

	// Upload a document.
	uploadURL := fmt.Sprintf("/api/v1/expenses/%d/documents", expenseID)
	uploadReq := buildUploadRequestWithContentType(t, uploadURL, "invoice.pdf", "application/pdf", []byte("%PDF-1.4"))
	uploadW := httptest.NewRecorder()
	r.ServeHTTP(uploadW, uploadReq)

	if uploadW.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, body = %s", uploadW.Code, uploadW.Body.String())
	}

	var uploaded documentResponse
	json.NewDecoder(uploadW.Body).Decode(&uploaded)

	// Get by ID.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/documents/%d", uploaded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp documentResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != uploaded.ID {
		t.Errorf("ID = %d, want %d", resp.ID, uploaded.ID)
	}
}

func TestDocumentHandler_GetByID_NotFound(t *testing.T) {
	r, _ := setupDocumentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDocumentHandler_GetByID_InvalidID(t *testing.T) {
	r, _ := setupDocumentRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDocumentHandler_Delete(t *testing.T) {
	r, expenseID := setupDocumentRouter(t)

	// Upload a document.
	uploadURL := fmt.Sprintf("/api/v1/expenses/%d/documents", expenseID)
	uploadReq := buildUploadRequestWithContentType(t, uploadURL, "todelete.pdf", "application/pdf", []byte("%PDF-1.4 d"))
	uploadW := httptest.NewRecorder()
	r.ServeHTTP(uploadW, uploadReq)

	if uploadW.Code != http.StatusCreated {
		t.Fatalf("upload status = %d", uploadW.Code)
	}

	var uploaded documentResponse
	json.NewDecoder(uploadW.Body).Decode(&uploaded)

	// Delete.
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/documents/%d", uploaded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify deleted.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/documents/%d", uploaded.ID), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", getW.Code, http.StatusNotFound)
	}
}

func TestDocumentHandler_Delete_NotFound(t *testing.T) {
	r, _ := setupDocumentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDocumentHandler_Delete_InvalidID(t *testing.T) {
	r, _ := setupDocumentRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
