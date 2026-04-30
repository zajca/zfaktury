package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/codebooks"
)

// CodebooksHandler exposes static EPO reference codebooks (financial offices,
// CZ-NACE 2025) as read-only JSON endpoints.
type CodebooksHandler struct{}

// NewCodebooksHandler returns a CodebooksHandler.
func NewCodebooksHandler() *CodebooksHandler {
	return &CodebooksHandler{}
}

// Routes registers codebook endpoints on the given router.
func (h *CodebooksHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/financial-offices", h.FinancialOffices)
	r.Get("/nace", h.NACE)
	return r
}

// FinancialOffices handles GET /api/v1/codebooks/financial-offices.
func (h *CodebooksHandler) FinancialOffices(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, codebooks.FinancialOffices())
}

// NACE handles GET /api/v1/codebooks/nace.
func (h *CodebooksHandler) NACE(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, codebooks.NACE())
}
