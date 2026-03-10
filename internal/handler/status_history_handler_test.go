package handler

import (
	"context"
	"database/sql"
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

func setupStatusHistoryRouter(t *testing.T) (*chi.Mux, *repository.StatusHistoryRepository, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)

	invoiceRepo := repository.NewInvoiceRepository(db)
	historyRepo := repository.NewStatusHistoryRepository(db)
	overdueSvc := service.NewOverdueService(invoiceRepo, historyRepo)
	handler := NewStatusHistoryHandler(overdueSvc)

	r := chi.NewRouter()
	r.Get("/api/v1/invoices/{id}/history", handler.GetHistory)
	r.Post("/api/v1/invoices/check-overdue", handler.CheckOverdue)

	return r, historyRepo, db
}

func TestStatusHistoryHandler_GetHistory(t *testing.T) {
	r, historyRepo, db := setupStatusHistoryRouter(t)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Handler Test Customer"})
	inv := testutil.SeedInvoice(t, db, customer.ID, []domain.InvoiceItem{
		{Description: "Work", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	})

	// Seed a history entry directly.
	change := &domain.InvoiceStatusChange{
		InvoiceID: inv.ID,
		OldStatus: domain.InvoiceStatusDraft,
		NewStatus: domain.InvoiceStatusSent,
		ChangedAt: inv.CreatedAt,
		Note:      "sent via email",
	}
	if err := historyRepo.Create(ctx, change); err != nil {
		t.Fatalf("creating history: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoices/%d/history", inv.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp []statusChangeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp))
	}
	if resp[0].NewStatus != domain.InvoiceStatusSent {
		t.Errorf("new_status = %q, want %q", resp[0].NewStatus, domain.InvoiceStatusSent)
	}
}

func TestStatusHistoryHandler_GetHistory_InvalidID(t *testing.T) {
	r, _, _ := setupStatusHistoryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/abc/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestStatusHistoryHandler_CheckOverdue(t *testing.T) {
	r, _, db := setupStatusHistoryRouter(t)

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Overdue Handler Customer"})
	inv := testutil.SeedInvoice(t, db, customer.ID, []domain.InvoiceItem{
		{Description: "Work", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	})
	_, err := db.ExecContext(context.Background(), `UPDATE invoices SET status = 'sent', due_date = '2026-01-01' WHERE id = ?`, inv.ID)
	if err != nil {
		t.Fatalf("updating invoice: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/check-overdue", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp checkOverdueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Marked != 1 {
		t.Errorf("marked = %d, want 1", resp.Marked)
	}
}

func TestStatusHistoryHandler_CheckOverdue_NoCandidates(t *testing.T) {
	r, _, _ := setupStatusHistoryRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/check-overdue", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var resp checkOverdueResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Marked != 0 {
		t.Errorf("marked = %d, want 0", resp.Marked)
	}
}
