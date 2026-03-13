package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/fakturoid"
	"github.com/zajca/zfaktury/internal/service"
)

// FakturoidHandler handles Fakturoid import HTTP requests.
type FakturoidHandler struct {
	svc *service.FakturoidImportService
}

// NewFakturoidHandler creates a new FakturoidHandler.
func NewFakturoidHandler(svc *service.FakturoidImportService) *FakturoidHandler {
	return &FakturoidHandler{svc: svc}
}

// Routes returns the chi router for Fakturoid import endpoints.
func (h *FakturoidHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/import", h.Import)
	return r
}

// --- DTOs ---

type fakturoidImportRequest struct {
	Slug                string `json:"slug"`
	Email               string `json:"email"`
	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret"`
	DownloadAttachments bool   `json:"download_attachments"`
}

type fakturoidImportResponse struct {
	ContactsCreated      int      `json:"contacts_created"`
	ContactsSkipped      int      `json:"contacts_skipped"`
	InvoicesCreated      int      `json:"invoices_created"`
	InvoicesSkipped      int      `json:"invoices_skipped"`
	ExpensesCreated      int      `json:"expenses_created"`
	ExpensesSkipped      int      `json:"expenses_skipped"`
	AttachmentsDownloaded int     `json:"attachments_downloaded"`
	AttachmentsSkipped   int      `json:"attachments_skipped"`
	Errors               []string `json:"errors"`
}

// Import performs a full import from Fakturoid using credentials from the request body.
func (h *FakturoidHandler) Import(w http.ResponseWriter, r *http.Request) {
	var req fakturoidImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Slug == "" || req.Email == "" || req.ClientID == "" || req.ClientSecret == "" {
		respondError(w, http.StatusBadRequest, "slug, email, client_id, and client_secret are required")
		return
	}

	client := fakturoid.NewClient(req.Slug, req.Email, req.ClientID, req.ClientSecret)
	if err := client.Authenticate(r.Context()); err != nil {
		respondError(w, http.StatusUnauthorized, fmt.Sprintf("Fakturoid authentication failed: %v", err))
		return
	}
	result, err := h.svc.ImportAll(r.Context(), client, req.DownloadAttachments)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, fakturoidImportResponse{
		ContactsCreated:      result.ContactsCreated,
		ContactsSkipped:      result.ContactsSkipped,
		InvoicesCreated:      result.InvoicesCreated,
		InvoicesSkipped:      result.InvoicesSkipped,
		ExpensesCreated:      result.ExpensesCreated,
		ExpensesSkipped:      result.ExpensesSkipped,
		AttachmentsDownloaded: result.AttachmentsDownloaded,
		AttachmentsSkipped:   result.AttachmentsSkipped,
		Errors:               result.Errors,
	})
}
