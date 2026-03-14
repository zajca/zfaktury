package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupAuditLogHandler(t *testing.T) (*AuditLogHandler, *service.AuditService) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewAuditLogRepository(db)
	svc := service.NewAuditService(repo)
	h := NewAuditLogHandler(svc)
	return h, svc
}

func serveAuditLog(h *AuditLogHandler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := chi.NewRouter()
	r.Mount("/audit-log", h.Routes())
	r.ServeHTTP(w, req)
	return w
}

func TestAuditLogHandler_List_Empty(t *testing.T) {
	h, _ := setupAuditLogHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/audit-log", nil)
	w := serveAuditLog(h, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(resp.Items) != 0 {
		t.Errorf("items length = %d, want 0", len(resp.Items))
	}
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
}

func TestAuditLogHandler_List_WithEntries(t *testing.T) {
	h, svc := setupAuditLogHandler(t)
	ctx := context.Background()

	svc.Log(ctx, "contact", 1, "create", nil, map[string]string{"name": "Test"})
	svc.Log(ctx, "invoice", 2, "update", map[string]int{"amount": 100}, map[string]int{"amount": 200})
	svc.Log(ctx, "expense", 3, "delete", map[string]string{"note": "old"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/audit-log", nil)
	w := serveAuditLog(h, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("total = %d, want 3", resp.Total)
	}
	if len(resp.Items) != 3 {
		t.Errorf("items length = %d, want 3", len(resp.Items))
	}

	// Verify fields are populated.
	for _, item := range resp.Items {
		if item.ID == 0 {
			t.Error("expected non-zero ID")
		}
		if item.EntityType == "" {
			t.Error("expected non-empty entity_type")
		}
		if item.Action == "" {
			t.Error("expected non-empty action")
		}
		if item.CreatedAt == "" {
			t.Error("expected non-empty created_at")
		}
	}
}

func TestAuditLogHandler_List_FilterByEntityType(t *testing.T) {
	h, svc := setupAuditLogHandler(t)
	ctx := context.Background()

	svc.Log(ctx, "contact", 1, "create", nil, map[string]string{"name": "A"})
	svc.Log(ctx, "invoice", 2, "create", nil, map[string]string{"number": "1"})
	svc.Log(ctx, "contact", 3, "update", nil, map[string]string{"name": "B"})

	req := httptest.NewRequest(http.MethodGet, "/audit-log?entity_type=contact", nil)
	w := serveAuditLog(h, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
	for _, item := range resp.Items {
		if item.EntityType != "contact" {
			t.Errorf("entity_type = %q, want %q", item.EntityType, "contact")
		}
	}
}

func TestAuditLogHandler_List_FilterByAction(t *testing.T) {
	h, svc := setupAuditLogHandler(t)
	ctx := context.Background()

	svc.Log(ctx, "contact", 1, "create", nil, map[string]string{"name": "A"})
	svc.Log(ctx, "contact", 1, "update", nil, map[string]string{"name": "B"})
	svc.Log(ctx, "invoice", 2, "create", nil, map[string]string{"number": "1"})

	req := httptest.NewRequest(http.MethodGet, "/audit-log?action=update", nil)
	w := serveAuditLog(h, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
	if len(resp.Items) != 1 {
		t.Errorf("items length = %d, want 1", len(resp.Items))
	}
	if resp.Items[0].Action != "update" {
		t.Errorf("action = %q, want %q", resp.Items[0].Action, "update")
	}
}

func TestAuditLogHandler_List_InvalidFromDate(t *testing.T) {
	h, _ := setupAuditLogHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/audit-log?from=not-a-date", nil)
	w := serveAuditLog(h, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestAuditLogHandler_List_InvalidToDate(t *testing.T) {
	h, _ := setupAuditLogHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/audit-log?to=31-12-2025", nil)
	w := serveAuditLog(h, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestAuditLogHandler_List_ValidDateRange(t *testing.T) {
	h, svc := setupAuditLogHandler(t)
	ctx := context.Background()

	svc.Log(ctx, "contact", 1, "create", nil, map[string]string{"name": "A"})
	svc.Log(ctx, "invoice", 2, "update", nil, map[string]string{"number": "1"})

	// Use a wide date range that covers today.
	req := httptest.NewRequest(http.MethodGet, "/audit-log?from=2020-01-01&to=2030-12-31", nil)
	w := serveAuditLog(h, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}

	// Use a date range in the past that should return nothing.
	req2 := httptest.NewRequest(http.MethodGet, "/audit-log?from=2000-01-01&to=2000-01-02", nil)
	w2 := serveAuditLog(h, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var resp2 auditLogListResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp2.Total != 0 {
		t.Errorf("total = %d, want 0 for past date range", resp2.Total)
	}
}

func TestAuditLogHandler_List_InvalidEntityTypeIgnored(t *testing.T) {
	h, svc := setupAuditLogHandler(t)
	ctx := context.Background()

	svc.Log(ctx, "contact", 1, "create", nil, map[string]string{"name": "A"})

	// Invalid entity_type should be ignored (no filter applied), returning all entries.
	req := httptest.NewRequest(http.MethodGet, "/audit-log?entity_type=invalid_type", nil)
	w := serveAuditLog(h, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("total = %d, want 1 (invalid entity_type should be ignored)", resp.Total)
	}
}

func TestAuditLogHandler_List_InvalidActionIgnored(t *testing.T) {
	h, svc := setupAuditLogHandler(t)
	ctx := context.Background()

	svc.Log(ctx, "contact", 1, "create", nil, map[string]string{"name": "A"})

	// Invalid action should be ignored (no filter applied), returning all entries.
	req := httptest.NewRequest(http.MethodGet, "/audit-log?action=invalid_action", nil)
	w := serveAuditLog(h, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("total = %d, want 1 (invalid action should be ignored)", resp.Total)
	}
}
