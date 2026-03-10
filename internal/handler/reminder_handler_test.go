package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/service/email"
	"github.com/zajca/zfaktury/internal/testutil"
)

// testEmailSender is a mock email sender for handler tests.
type testEmailSender struct {
	sent []email.EmailMessage
}

func (s *testEmailSender) Send(_ context.Context, msg email.EmailMessage) error {
	s.sent = append(s.sent, msg)
	return nil
}

func setupReminderRouter(t *testing.T) (*chi.Mux, *domain.Invoice) {
	t.Helper()
	db := testutil.NewTestDB(t)

	invoiceRepo := repository.NewInvoiceRepository(db)
	reminderRepo := repository.NewReminderRepository(db)

	emailSender := &testEmailSender{}

	// Seed a customer with email.
	customer := testutil.SeedContact(t, db, &domain.Contact{
		Name:  "Reminder Test Customer",
		Email: "customer@example.com",
	})

	// Seed an overdue invoice.
	inv := testutil.SeedInvoice(t, db, customer.ID, []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	})
	// Mark as overdue.
	_, err := db.ExecContext(context.Background(),
		`UPDATE invoices SET status = ?, due_date = ?, bank_account = ?, bank_code = ?, variable_symbol = ? WHERE id = ?`,
		domain.InvoiceStatusOverdue,
		time.Now().AddDate(0, 0, -10).Format("2006-01-02"),
		"1234567890", "0100", "20260001",
		inv.ID,
	)
	if err != nil {
		t.Fatalf("updating invoice to overdue: %v", err)
	}

	reminderSvc := service.NewReminderService(reminderRepo, invoiceRepo, emailSender, "Jan Novak")
	reminderHandler := NewReminderHandler(reminderSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/invoices", reminderHandler.Routes())

	return r, inv
}

func TestReminderHandler_SendReminder(t *testing.T) {
	router, inv := setupReminderRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+itoa(inv.ID)+"/remind", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("SendReminder status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp reminderResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected non-zero reminder ID")
	}
	if resp.InvoiceID != inv.ID {
		t.Errorf("InvoiceID = %d, want %d", resp.InvoiceID, inv.ID)
	}
	if resp.ReminderNumber != 1 {
		t.Errorf("ReminderNumber = %d, want 1", resp.ReminderNumber)
	}
	if resp.SentTo != "customer@example.com" {
		t.Errorf("SentTo = %q, want %q", resp.SentTo, "customer@example.com")
	}
}

func TestReminderHandler_ListReminders_Empty(t *testing.T) {
	router, inv := setupReminderRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+itoa(inv.ID)+"/reminders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListReminders status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []reminderResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d reminders", len(resp))
	}
}

func TestReminderHandler_ListReminders_AfterSend(t *testing.T) {
	router, inv := setupReminderRouter(t)
	invIDStr := itoa(inv.ID)

	// Send a reminder first.
	sendReq := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invIDStr+"/remind", nil)
	sendW := httptest.NewRecorder()
	router.ServeHTTP(sendW, sendReq)

	if sendW.Code != http.StatusCreated {
		t.Fatalf("SendReminder status = %d, body = %s", sendW.Code, sendW.Body.String())
	}

	// Now list.
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invIDStr+"/reminders", nil)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("ListReminders status = %d, body = %s", listW.Code, listW.Body.String())
	}

	var resp []reminderResponse
	if err := json.NewDecoder(listW.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 reminder, got %d", len(resp))
	}
}

func TestReminderHandler_SendReminder_InvalidID(t *testing.T) {
	router, _ := setupReminderRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/abc/remind", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func itoa(id int64) string {
	return fmt.Sprintf("%d", id)
}
