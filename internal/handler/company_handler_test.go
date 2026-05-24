package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
)

// stubCompanyService is a hand-rolled stub for the CompanyHandlerService
// interface. Each field lets a test override one method without dragging in
// a real repo + DB; unset methods return zero values, which is fine for the
// paths each individual test exercises.
type stubCompanyService struct {
	createFn func(ctx context.Context, c domain.Company) (int64, error)
	getFn    func(ctx context.Context, id int64) (domain.Company, error)
	listFn   func(ctx context.Context) ([]domain.Company, error)
	updateFn func(ctx context.Context, c domain.Company) error
	deleteFn func(ctx context.Context, id int64) error
}

func (s *stubCompanyService) Create(ctx context.Context, c domain.Company) (int64, error) {
	if s.createFn != nil {
		return s.createFn(ctx, c)
	}
	return 0, nil
}

func (s *stubCompanyService) Get(ctx context.Context, id int64) (domain.Company, error) {
	if s.getFn != nil {
		return s.getFn(ctx, id)
	}
	return domain.Company{}, nil
}

func (s *stubCompanyService) List(ctx context.Context) ([]domain.Company, error) {
	if s.listFn != nil {
		return s.listFn(ctx)
	}
	return nil, nil
}

func (s *stubCompanyService) Update(ctx context.Context, c domain.Company) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, c)
	}
	return nil
}

func (s *stubCompanyService) Delete(ctx context.Context, id int64) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

// newCompanyTestRouter wires the handler under a fresh chi router so URL
// params (`{id}`) are parsed by chi exactly the way they are at runtime.
func newCompanyTestRouter(svc CompanyHandlerService) chi.Router {
	h := NewCompanyHandler(svc)
	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	return r
}

func TestCompanyHandler_Create_Returns201WithLocation(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	svc := &stubCompanyService{
		createFn: func(_ context.Context, _ domain.Company) (int64, error) {
			return 42, nil
		},
		getFn: func(_ context.Context, id int64) (domain.Company, error) {
			return domain.Company{
				ID:        id,
				Name:      "Acme",
				LegalName: "Acme s.r.o.",
				ICO:       "12345678",
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	body := `{"name":"Acme","legal_name":"Acme s.r.o.","ico":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "/api/v1/companies/42" {
		t.Errorf("Location = %q, want %q", loc, "/api/v1/companies/42")
	}

	var resp CompanyDTO
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ID != 42 {
		t.Errorf("ID = %d, want 42", resp.ID)
	}
	if resp.Name != "Acme" {
		t.Errorf("Name = %q, want %q", resp.Name, "Acme")
	}
	if resp.CreatedAt == "" {
		t.Errorf("CreatedAt should be populated from the re-fetched record")
	}
}

func TestCompanyHandler_Create_InvalidInput_Returns400(t *testing.T) {
	svc := &stubCompanyService{
		createFn: func(_ context.Context, _ domain.Company) (int64, error) {
			return 0, domain.ErrInvalidInput
		},
	}

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCompanyHandler_Create_InvalidJSON_Returns400(t *testing.T) {
	svc := &stubCompanyService{}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCompanyHandler_Get_NotFound_Returns404(t *testing.T) {
	svc := &stubCompanyService{
		getFn: func(_ context.Context, _ int64) (domain.Company, error) {
			return domain.Company{}, domain.ErrNotFound
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/123", nil)
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestCompanyHandler_Get_OtherError_Returns500(t *testing.T) {
	svc := &stubCompanyService{
		getFn: func(_ context.Context, _ int64) (domain.Company, error) {
			return domain.Company{}, errors.New("boom")
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/7", nil)
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestCompanyHandler_Get_InvalidID_Returns400(t *testing.T) {
	svc := &stubCompanyService{}

	req := httptest.NewRequest(http.MethodGet, "/abc", nil)
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCompanyHandler_List_OK(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	svc := &stubCompanyService{
		listFn: func(_ context.Context) ([]domain.Company, error) {
			return []domain.Company{
				{ID: 1, Name: "A", LegalName: "A s.r.o.", ICO: "11111111", CreatedAt: now, UpdatedAt: now},
				{ID: 2, Name: "B", LegalName: "B s.r.o.", ICO: "22222222", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp []CompanyDTO
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("len = %d, want 2", len(resp))
	}
}

func TestCompanyHandler_Update_Returns204(t *testing.T) {
	var receivedID int64
	svc := &stubCompanyService{
		updateFn: func(_ context.Context, c domain.Company) error {
			receivedID = c.ID
			return nil
		},
	}

	body := `{"id":999,"name":"Acme","legal_name":"Acme s.r.o.","ico":"12345678"}`
	req := httptest.NewRequest(http.MethodPut, "/42", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
	// Path ID must override the body's id field.
	if receivedID != 42 {
		t.Errorf("service received ID = %d, want 42 (path ID must win over body)", receivedID)
	}
}

func TestCompanyHandler_Update_NotFound_Returns404(t *testing.T) {
	svc := &stubCompanyService{
		updateFn: func(_ context.Context, _ domain.Company) error {
			return domain.ErrNotFound
		},
	}

	body := `{"name":"Acme","legal_name":"Acme s.r.o.","ico":"12345678"}`
	req := httptest.NewRequest(http.MethodPut, "/42", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestCompanyHandler_Update_InvalidInput_Returns400(t *testing.T) {
	svc := &stubCompanyService{
		updateFn: func(_ context.Context, _ domain.Company) error {
			return domain.ErrInvalidInput
		},
	}

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPut, "/42", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCompanyHandler_Delete_LastCompany_Returns409(t *testing.T) {
	svc := &stubCompanyService{
		deleteFn: func(_ context.Context, _ int64) error {
			return domain.ErrLastCompany
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestCompanyHandler_Delete_InUse_Returns409(t *testing.T) {
	svc := &stubCompanyService{
		deleteFn: func(_ context.Context, _ int64) error {
			return domain.ErrInUse
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestCompanyHandler_Delete_NotFound_Returns404(t *testing.T) {
	svc := &stubCompanyService{
		deleteFn: func(_ context.Context, _ int64) error {
			return domain.ErrNotFound
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestCompanyHandler_Delete_OK_Returns204(t *testing.T) {
	svc := &stubCompanyService{
		deleteFn: func(_ context.Context, _ int64) error {
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	w := httptest.NewRecorder()

	newCompanyTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
