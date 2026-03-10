package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupCategoryHandler(t *testing.T) (*CategoryHandler, *repository.CategoryRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewCategoryRepository(db)
	svc := service.NewCategoryService(repo)
	h := NewCategoryHandler(svc)
	return h, repo
}

func TestCategoryHandler_List(t *testing.T) {
	h, _ := setupCategoryHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var categories []categoryResponse
	if err := json.NewDecoder(w.Body).Decode(&categories); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(categories) < 16 {
		t.Errorf("expected at least 16 categories, got %d", len(categories))
	}
}

func TestCategoryHandler_Create(t *testing.T) {
	h, _ := setupCategoryHandler(t)

	body := `{"key":"test_create","label_cs":"Testovaci","label_en":"Test","color":"#FF0000","sort_order":50}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp categoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Key != "test_create" {
		t.Errorf("Key = %q, want %q", resp.Key, "test_create")
	}
	if resp.Color != "#FF0000" {
		t.Errorf("Color = %q, want %q", resp.Color, "#FF0000")
	}
}

func TestCategoryHandler_Create_InvalidKey(t *testing.T) {
	h, _ := setupCategoryHandler(t)

	body := `{"key":"INVALID KEY","label_cs":"Test","label_en":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestCategoryHandler_Update(t *testing.T) {
	h, repo := setupCategoryHandler(t)
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()

	cat := &domain.ExpenseCategory{
		Key:     "handler_update",
		LabelCS: "Pred",
		LabelEN: "Before",
	}
	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := `{"key":"handler_update","label_cs":"Po","label_en":"After","color":"#00FF00","sort_order":5}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/%d", cat.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp categoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.LabelCS != "Po" {
		t.Errorf("LabelCS = %q, want %q", resp.LabelCS, "Po")
	}
}

func TestCategoryHandler_Delete_Custom(t *testing.T) {
	h, repo := setupCategoryHandler(t)
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()

	cat := &domain.ExpenseCategory{
		Key:     "handler_delete",
		LabelCS: "Smazat",
		LabelEN: "Delete",
	}
	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/%d", cat.ID), nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestCategoryHandler_Delete_Default_Forbidden(t *testing.T) {
	h, repo := setupCategoryHandler(t)
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()

	// Get a default category.
	categories, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	var defaultCat *domain.ExpenseCategory
	for i := range categories {
		if categories[i].IsDefault {
			defaultCat = &categories[i]
			break
		}
	}
	if defaultCat == nil {
		t.Fatal("no default categories found")
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/%d", defaultCat.ID), nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestCategoryHandler_Create_InvalidBody(t *testing.T) {
	h, _ := setupCategoryHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
