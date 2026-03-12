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

// setupImportRouter creates a test router wired with real SQLite repos and a temp data dir.
func setupImportRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)
	dataDir := t.TempDir()

	expenseRepo := repository.NewExpenseRepository(db)
	docRepo := repository.NewDocumentRepository(db)

	expenseSvc := service.NewExpenseService(expenseRepo, nil)
	docSvc := service.NewDocumentService(docRepo, dataDir)
	importSvc := service.NewImportService(expenseSvc, docSvc, nil)

	h := NewImportHandler(importSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/expenses/import", h.Import)

	return r
}

// buildImportRequest creates a multipart request with a file part for the import endpoint.
func buildImportRequest(t *testing.T, filename string, fileContentType string, content []byte) *http.Request {
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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestImportHandler_Import_ValidFile(t *testing.T) {
	r := setupImportRouter(t)

	req := buildImportRequest(t, "receipt.pdf", "application/pdf", []byte("%PDF-1.4 test content"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp importResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Expense.ID == 0 {
		t.Error("expected non-zero expense ID")
	}
	if resp.Document.ID == 0 {
		t.Error("expected non-zero document ID")
	}
	if resp.Document.Filename != "receipt.pdf" {
		t.Errorf("Document.Filename = %q, want receipt.pdf", resp.Document.Filename)
	}
	if resp.Document.ContentType != "application/pdf" {
		t.Errorf("Document.ContentType = %q, want application/pdf", resp.Document.ContentType)
	}
	if resp.OCR != nil {
		t.Error("expected OCR to be nil when no OCR service is configured")
	}
}

func TestImportHandler_Import_MissingFileField(t *testing.T) {
	r := setupImportRouter(t)

	// Send a valid multipart form but without the "file" field.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	// Write a different field name so "file" is missing.
	fw, err := mw.CreateFormField("other")
	if err != nil {
		t.Fatalf("creating form field: %v", err)
	}
	fw.Write([]byte("some value"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestImportHandler_Import_InvalidMultipart(t *testing.T) {
	r := setupImportRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses/import", strings.NewReader("not a valid multipart body"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}
