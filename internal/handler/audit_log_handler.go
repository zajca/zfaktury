package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// AuditLogHandler handles HTTP requests for audit log entries.
type AuditLogHandler struct {
	auditSvc *service.AuditService
}

// NewAuditLogHandler creates a new AuditLogHandler.
func NewAuditLogHandler(auditSvc *service.AuditService) *AuditLogHandler {
	return &AuditLogHandler{auditSvc: auditSvc}
}

// Routes returns a chi.Router with audit log routes.
func (h *AuditLogHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	return r
}

// auditLogEntryResponse is the JSON response for an audit log entry.
type auditLogEntryResponse struct {
	ID         int64  `json:"id"`
	EntityType string `json:"entity_type"`
	EntityID   int64  `json:"entity_id"`
	Action     string `json:"action"`
	OldValues  string `json:"old_values"`
	NewValues  string `json:"new_values"`
	CreatedAt  string `json:"created_at"`
}

// auditLogListResponse is the paginated JSON response for audit log entries.
type auditLogListResponse struct {
	Items []auditLogEntryResponse `json:"items"`
	Total int                     `json:"total"`
}

// validEntityTypes is the allowlist of entity types accepted for filtering.
var validEntityTypes = map[string]bool{
	"contact": true, "invoice": true, "expense": true,
	"recurring_invoice": true, "recurring_expense": true,
	"settings": true, "category": true, "sequence": true,
	"vat_return": true, "vat_control_statement": true, "vies_summary": true,
	"income_tax_return": true, "social_insurance": true, "health_insurance": true,
	"tax_year_settings": true, "tax_spouse_credit": true, "tax_child_credit": true,
	"tax_personal_credits": true, "tax_deduction": true,
	"document": true, "tax_deduction_document": true, "investment_document": true,
	"capital_income": true, "security_transaction": true,
}

// validActions is the allowlist of actions accepted for filtering.
var validActions = map[string]bool{
	"create": true, "update": true, "delete": true,
	"activate": true, "deactivate": true,
	"set": true, "set_bulk": true,
	"generate_xml": true, "mark_filed": true, "recalculate": true,
}

// List returns audit log entries with optional filtering.
func (h *AuditLogHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var filter domain.AuditLogFilter

	if v := q.Get("entity_type"); v != "" && validEntityTypes[v] {
		filter.EntityType = v
	}
	if v := q.Get("action"); v != "" && validActions[v] {
		filter.Action = v
	}

	if v := parseOptionalInt64(r, "entity_id"); v != nil {
		filter.EntityID = v
	}

	if v := q.Get("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid 'from' date format, expected YYYY-MM-DD")
			return
		}
		filter.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid 'to' date format, expected YYYY-MM-DD")
			return
		}
		// Include the entire "to" day.
		filter.To = t.Add(24*time.Hour - time.Second)
	}

	limit, offset := parsePagination(r)
	if limit > 200 {
		limit = 200
	}
	filter.Limit = limit
	filter.Offset = offset

	entries, total, err := h.auditSvc.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list audit log")
		return
	}

	items := make([]auditLogEntryResponse, 0, len(entries))
	for _, e := range entries {
		items = append(items, auditLogEntryResponse{
			ID:         e.ID,
			EntityType: e.EntityType,
			EntityID:   e.EntityID,
			Action:     e.Action,
			OldValues:  e.OldValues,
			NewValues:  e.NewValues,
			CreatedAt:  e.CreatedAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, auditLogListResponse{
		Items: items,
		Total: total,
	})
}
